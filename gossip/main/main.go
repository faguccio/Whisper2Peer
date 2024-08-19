package gossip

import (
	"context"
	"errors"
	"fmt"
	"gossip/common"
	"gossip/internal/args"
	gs "gossip/strats"
	verticalapi "gossip/verticalAPI"
	"sync"

	"log/slog"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/lmittmann/tint"
	"gopkg.in/ini.v1"
)

// Arguments read using go-arg https://github.com/alexflint/go-arg. The annotation instruct the library on
// the type of comment and optionally the help message.
type UserArgs struct {
	Degree      *uint     `ini:"degree" arg:"-d,--degree" help:"Gossip parameter degree: Number of peers the current peer has to exchange information with"`
	Cache_size  *uint     `ini:"cache_size" arg:"-c,--cache" help:"Gossip parameter cache_size: Maximum number of data items to be held as part of the peer’s knowledge base. Older items will be removed to ensure space for newer items if the peer’s knowledge base exceeds this limit"`
	GossipTimer *uint     `ini:"gtimer" arg:"-t,--gtimer" help:"How often the gossip strategy should perform a strategy cycle, if applicable"`
	Hz_addr     *string   `ini:"p2p address" arg:"-h,--haddr" help:"Address to listen for incoming peer connections, ip:port"`
	Vert_addr   *string   `ini:"api address" arg:"-v,--vaddr" help:"Address to listen for incoming peer connections, ip:port"`
	Peer_addrs  []string  `ini:"hconns" arg:"positional" help:"List of horizontal peers to connect to, [ip]:port"`
	ConfigFile  *string   `arg:"-c" help:"Path to the configuration file (cli arguments always take predecence)"`
	// Strategy string ``
}

// uses the values set in arg as defaults and overwrites the values which are
// set (!= nil) in uarg
func (uarg *UserArgs) Merge(arg args.Args) args.Args {
	if uarg.Degree != nil {
		arg.Degree = *uarg.Degree
	}
	if uarg.Degree != nil {
		arg.Degree = *uarg.Degree
	}
	if uarg.Cache_size != nil {
		arg.Cache_size = *uarg.Cache_size
	}
	if uarg.GossipTimer != nil {
		arg.GossipTimer = *uarg.GossipTimer
	}
	if uarg.Hz_addr != nil {
		arg.Hz_addr = *uarg.Hz_addr
	}
	if uarg.Vert_addr != nil {
		arg.Vert_addr = *uarg.Vert_addr
	}
	if uarg.Peer_addrs != nil {
		arg.Peer_addrs = uarg.Peer_addrs
	}

	return arg
}

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
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

func NewMainWithArgs(args args.Args, log *slog.Logger) *Main {
	m := &Main{
		typeStorage: *NewNotifyMap(),
		args:        args,
	}

	m.log = log
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

func NewMain() *Main {
	// obtain the arguments with the default values set
	args := args.NewFromDefaults()

	// read the cli arguments
	var cargs UserArgs
	arg.MustParse(&cargs)

	// if set also read the ini arguments
	if cargs.ConfigFile != nil {
		cfg, err := ini.Load(cargs.ConfigFile)
		if err != nil {
			panic(err)
		}
		var iargs UserArgs
		if err = cfg.Section("gossip").MapTo(iargs); err != nil {
			panic(err)
		}

		// use args as defaults and overwrite those values which were set by
		// the ini config file
		args = iargs.Merge(args)
	}

	// merge in the end as cli takes predecence
	args = cargs.Merge(args)


	fmt.Printf("%+v\n", args)

	panic(nil)

	return NewMainWithArgs(args, logInit(args.Hz_addr))
}

func (m *Main) Run(initFinished chan<- error) {
	var err error
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())
	m.wg.Add(1)

	vInitFin := make(chan struct{}, 1)
	gsInitFin := make(chan struct{}, 1)

	va := verticalapi.NewVerticalApi(m.log, m.vertToMain)
	err = va.Listen(m.args.Vert_addr, vInitFin)
	if err != nil {
		m.mlog.Error("Error on listening on vertAPI", "err", err)
		initFinished <- err
		return
	}
	defer va.Close()

	strategy, err := gs.New(m.log, m.args, m.strategyChannels, gsInitFin)
	if err != nil {
		m.mlog.Error("Error on instantiating the strategy", "err", err)
		initFinished <- err
		return
	}

	strategy.Listen()
	defer strategy.Close()

	// wait asynchronously until strat and vertApi are initialized, to notify caller
	go func(initFinished chan<- error, vInitFin <-chan struct{}, gsInitFin <-chan struct{}) {
		<-vInitFin
		<-gsInitFin
		initFinished <- nil
	}(initFinished, vInitFin, gsInitFin)

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
		case <-ctx.Done():
			break loop
		}
	}
	m.mlog.Info("Main terminating")
	m.wg.Done()
}

func (m *Main) Close() error {
	m.cancel()
	m.wg.Wait()
	return nil
}

func (m *Main) handleModuleUnregister(msg common.GossipUnRegister) {
	m.typeStorage.RemoveChannel(common.ConnectionId(msg))
	m.mlog.Info("Unregistered module", "module", msg)
	// TODO talk about invalidating messages (not feasible I think, also this should only concern a rather short timeframe)
}

// Handle incoming Gossip Registration (Notify) Messages
func (m *Main) handleTypeRegistration(msg common.GossipRegister) {
	typeToRegister := common.GossipType(msg.Data.DataType)
	err := m.typeStorage.AddChannelToType(typeToRegister, msg.Module)
	if err != nil {
		m.mlog.Warn("Skipped registration of module", "type", typeToRegister, "module", msg.Module.Id)
	} else {
		m.mlog.Info("Registered module", "type", typeToRegister, "module", msg.Module.Id)
	}
}

// Handle incoming Gossip Validation messages.
func (m *Main) handleGossipValidation(msg common.GossipValidation) {
	m.mlog.Info("Validation data handled", "Message", msg)
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
	m.mlog.Info("Gossip Announce", "Message", announce_data)
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
