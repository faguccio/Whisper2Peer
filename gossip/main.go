package main

import (
	"errors"
	"gossip/common"
	"gossip/internal/args"
	gs "gossip/strats"
	verticalapi "gossip/verticalAPI"
	"os/signal"
	"syscall"

	"log/slog"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/lmittmann/tint"
	_ "gopkg.in/ini.v1"
)

func logInit(identifier any) *slog.Logger {
	return slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	})).With("id", identifier)
}

type Main struct {
	log              *slog.Logger
	mlog             *slog.Logger
	args             args.Args
	typeStorage      notifyMap
	vertToMain       chan common.FromVert
	strategyChannels gs.StrategyChannels
}

func NewMain() *Main {
	m := &Main{
		typeStorage: *NewNotifyMap(),
	}

	// Arguments read using go-arg https://github.com/alexflint/go-arg. The annotation instruct the library on
	// the type of comment and optionally the help message.
	arg.MustParse(&m.args)

	m.log = logInit(m.args.Hz_addr)
	m.mlog = m.log.With("module", "main")

	m.mlog.Debug("CMD ARGS mandatory",
		"cache size", m.args.Cache_size,
		"degree", m.args.Degree,
	)

	m.mlog.Debug("CMD ARGS mandatory",
		"Horizontal addr", m.args.Hz_addr,
		"Vertical addr", m.args.Vert_addr,
		"Peers addresses", m.args.Peer_addrs,
	)

	m.vertToMain = make(chan common.FromVert)

	m.strategyChannels = gs.StrategyChannels{
		FromStrat: make(chan common.FromStrat),
		ToStrat:   make(chan common.ToStrat),
	}

	return m
}

func (m *Main) run() {
	va := verticalapi.NewVerticalApi(m.log.With("module", "vertAPI"), m.vertToMain)
	va.Listen(m.args.Vert_addr)
	defer va.Close()

	strategy, err := gs.New(m.log.With("module", "strategy"), m.args, m.strategyChannels)
	if err != nil {
		m.mlog.Error("Error on instantiating the strategy", "err", err)
	}

	strategy.Listen()
	defer strategy.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

loop:
	for {
		select {
		case x := <-m.vertToMain:
			switch x := x.(type) {
			case common.GossipValidation:
				m.handleGossipValidation(x)
			case common.GossipAnnounce:
				_ = m.handleGossipAnnounce(x)
			case common.GossipRegister:
				m.handleTypeRegistration(x)
			case common.GossipUnRegister:
				m.handleModuleUnregister(x)
			}
		case x := <-m.strategyChannels.FromStrat:
			switch x := x.(type) {
			case common.GossipNotification:
				m.handleNotification(x)
			}
		case <-c:
			break loop
		}
	}
	m.mlog.Info("Main terminating")
}

func (m *Main) handleModuleUnregister(msg common.GossipUnRegister) {
	m.typeStorage.RemoveChannel(msg)
	m.mlog.Info("Unregistered module", "module", msg)
	// TODO talk about invalidating messages (not feasible I think, also this should only concern a rather short timeframe)
}

// Handle incoming Gossip Registration (Notify) Messages
func (m *Main) handleTypeRegistration(msg common.GossipRegister) {
	typeToRegister := common.GossipType(msg.Data.DataType)
	listeningModule := msg.Module
	m.typeStorage.AddChannelToType(typeToRegister, listeningModule)
	m.mlog.Info("Registered module", "module", typeToRegister, "type", m.typeStorage.Load(typeToRegister))
}

// Handle incoming Gossip Validation messages.
func (m *Main) handleGossipValidation(msg common.GossipValidation) {
	m.mlog.Info("Validation data handled", "msg", msg)
	m.strategyChannels.ToStrat <- msg
}

// Handle incoming Gossip Announce messages. This function sould call the GOSSIP STRATEGY module
// and use that to spread the message.
func (m *Main) handleGossipAnnounce(msg common.GossipAnnounce) error {
	typeToCheck := common.GossipType(msg.DataType)
	res := m.typeStorage.Load(typeToCheck)
	if len(res) == 0 {
		return errors.New("Gossip Type not registered, cannot accept message.")
	}

	announce_data := msg.Data
	m.mlog.Info("Gossip Announce", "msg", announce_data)
	// send to gossip
	m.strategyChannels.ToStrat <- msg
	return nil
}

func (m *Main) handleNotification(msg common.GossipNotification) error {
	typeToCheck := common.GossipType(msg.DataType)
	res := m.typeStorage.Load(typeToCheck)
	if len(res) == 0 {
		// if no module is registered for this type, mark this message as non-valid (don't propagate it)
		s := common.GossipValidation{
			MessageId: msg.MessageId,
		}
		s.SetValid(false)
		m.strategyChannels.ToStrat <- s
		return nil
	}

	for _, r := range res {
		r.Data.MainToVert <- msg
	}
	return nil
}

func main() {
	m := NewMain()
	m.run()
}
