package main

import (
	"errors"
	gs "gossip/strats"
	verticalapi "gossip/verticalAPI"
	"gossip/internal/args"
	vertTypes "gossip/verticalAPI/types"
	"os/signal"
	"syscall"

	"log/slog"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/lmittmann/tint"
	_ "gopkg.in/ini.v1"
)

func logInit() *slog.Logger {
	return slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}))
}

type Main struct {
	log *slog.Logger
	mlog *slog.Logger
	args args.Args
	typeStorage notifyMap
	vertToMain verticalapi.VertToMainChans
	strategyChannels gs.StrategyChannels
}

func NewMain() *Main {
	m := &Main{
		log:         logInit(),
		typeStorage: *NewNotifyMap(),
	}
	m.mlog = m.log.With("module", "main")

	// Arguments read using go-arg https://github.com/alexflint/go-arg. The annotation instruct the library on
	// the type of comment and optionally the help message.
	arg.MustParse(&m.args)

	m.mlog.Debug("CMD ARGS mandatory",
		"cache size", m.args.Cache_size,
		"degree", m.args.Degree,
	)

	m.mlog.Debug("CMD ARGS mandatory",
		"Horizontal addr", m.args.Hz_addr,
		"Vertical addr", m.args.Vert_addr,
		"Peers addresses", m.args.Peer_addrs,
	)

	m.vertToMain = verticalapi.VertToMainChans{
		Register:   make(chan verticalapi.VertToMainRegister),
		Announce:   make(chan verticalapi.VertToMainAnnounce),
		Validation: make(chan verticalapi.VertToMainValidation),
	}

	m.strategyChannels = gs.StrategyChannels{
		Notification: make(chan verticalapi.MainToVertNotification),
		Announce:     make(chan verticalapi.VertToMainAnnounce),
		Validation:   make(chan verticalapi.VertToMainValidation),
	}


	return m
}

func (m *Main) run() {
	va := verticalapi.NewVerticalApi(slog.With("module", "vertAPI"), m.vertToMain)
	va.Listen(m.args.Vert_addr)
	defer va.Close()

	strategy, _ := gs.New(m.args, m.strategyChannels)
	strategy.Listen()
	defer strategy.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

	loop: for {
		select {
		case x := <-m.vertToMain.Validation:
			m.handleGossipValidation(x)
		case x := <-m.vertToMain.Announce:
			_ = m.handleGossipAnnounce(x)
		case x := <-m.vertToMain.Register:
			m.handleTypeRegistration(x)
		case x := <- m.strategyChannels.Notification:
			m.handleNotification(x)
		case <- c:
			break loop
		}
	}
}

// Handle incoming Gossip Registration (Notify) Messages
func (m *Main) handleTypeRegistration(msg verticalapi.VertToMainRegister) {
	typeToRegister := vertTypes.GossipType(msg.Data.DataType)
	listeningModule := msg.Module
	m.typeStorage.AddChannelToType(typeToRegister, listeningModule)
	//m.mlog.Debug("Just registered: %d with val %v\n", "msg", typeToRegister, storage.Load(typeToRegister))
}

// Handle incoming Gossip Validation messages.
func (m *Main) handleGossipValidation(msg verticalapi.VertToMainValidation) {
	validation_data := msg.Data
	m.mlog.Debug("Validation data handled: ", "msg", validation_data)
	m.strategyChannels.Validation <- msg
}

// Handle incoming Gossip Announce messages. This function sould call the GOSSIP STRATEGY module
// and use that to spread the message.
func (m *Main) handleGossipAnnounce(msg verticalapi.VertToMainAnnounce) error {
	typeToCheck := vertTypes.GossipType(msg.Data.DataType)
	res := m.typeStorage.Load(typeToCheck)
	if len(res) == 0 {
		return errors.New("Gossip Type not registered, cannot accept message.")
	}

	announce_data := msg.Data
	m.mlog.Debug("Gossip Announce", "msg", announce_data)
	// send to gossip
	m.strategyChannels.Announce <- msg
	return nil
}

func (m *Main) handleNotification(msg verticalapi.MainToVertNotification) error {
	typeToCheck := vertTypes.GossipType(msg.Data.DataType)
	res := m.typeStorage.Load(typeToCheck)
	if len(res) == 0 {
		// if no module is registered for this type, mark this message as non-valid (don't propagate it)
		s := verticalapi.VertToMainValidation{
			Data: vertTypes.GossipValidation{
				MessageHeader: vertTypes.MessageHeader{},
				MessageId:     msg.Data.MessageId,
			},
		}
		s.Data.SetValid(false)
		s.Data.MessageHeader.Size = uint16(s.Data.CalcSize()+s.Data.MessageHeader.CalcSize())
		s.Data.MessageHeader.Type = vertTypes.GossipValidationType
		m.strategyChannels.Validation <- s
		return nil
	}

	for _,r := range res {
		r.MainToVert <- msg
	}
	return nil
}

func main() {
	m := NewMain()
	m.run()
}
