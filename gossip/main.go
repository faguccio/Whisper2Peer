package main

import (
	"errors"
	horizontalapi "gossip/horizontalAPI"
	verticalapi "gossip/verticalAPI"
	vertTypes "gossip/verticalAPI/types"

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

var mainLogger *slog.Logger

func main() {
	// var err error
	// set up logging
	slog.SetDefault(logInit())
	mainLogger = slog.With("module", "main")

	// just skip the ini parsing etc for now and only start/listen on the vertical api using constants as address
	// then later if you want to maybe extract those from the ini config (at a fixed path to skip the argument parsing stuff for now)

	// Arguments read using go-arg https://github.com/alexflint/go-arg. The annotation instruct the library on
	// the type of comment and optionally the help message.
	var args struct {
		Degree     int      `arg:"-d,--degree" default:"30" help:"Gossip parameter degree: Number of peers the current peer has to exchange information with"`
		Cache_size int      `arg:"-c,--cache" default:"50" help:"Gossip parameter cache_size: Maximum number of data items to be held as part of the peer’s knowledge base. Older items will be removed to ensure space for newer items if the peer’s knowledge base exceeds this limit"`
		Hz_addr    string   `arg:"-h,--haddr" default:"127.0.0.1:6001" help:"Address to listen for incoming peer connections, ip:port"`
		Vert_addr  string   `arg:"-v,--vaddr" default:"127.0.0.1:7001" help:"Address to listen for incoming peer connections, ip:port"`
		Peer_addrs []string `arg:"positional" help:"List of horizontal peers to connect to, [ip]:port"`
	}
	arg.MustParse(&args)

	mainLogger.Debug("CMD ARGS mandatory",
		"cache size", args.Cache_size,
		"degree", args.Degree,
	)

	mainLogger.Debug("CMD ARGS mandatory",
		"Horizontal addr", args.Hz_addr,
		"Vertical addr", args.Vert_addr,
		"Peers addresses", args.Peer_addrs,
	)

	vertToMain := verticalapi.VertToMainChans{
		Register:   make(chan verticalapi.VertToMainRegister),
		Announce:   make(chan verticalapi.VertToMainAnnounce),
		Validation: make(chan verticalapi.VertToMainValidation),
	}

	va := verticalapi.NewVerticalApi(slog.With("module", "vertAPI"), vertToMain)
	va.Listen(args.Vert_addr)
	defer va.Close()

	fromHz := make(chan horizontalapi.FromHz, 1)
	hz := horizontalapi.NewHorizontalApi(slog.With("module", "horzAPI"), fromHz)
	hzConnection := make(chan horizontalapi.NewConn, 1)
	hz.Listen(args.Hz_addr, hzConnection)
	defer hz.Close()

	hz.AddNeighbors(args.Peer_addrs...)

	typeStorage := NewNotifyMap()

	for {
		select {
		case x := <-vertToMain.Validation:
			handleGossipValidation(x)
		case x := <-vertToMain.Announce:
			_ = handleGossipAnnounce(x, typeStorage)
		case x := <-vertToMain.Register:
			handleTypeRegistration(x, typeStorage)
		case x := <-fromHz:
			// This will be done by the GOSSIP STRATEGY module I am not sure if we will listen fromHz or pass
			// the channel to the GOSSIP STRATEGY module
			handlePeerMessage(x)
		}
	}

}

func handlePeerMessage(msg horizontalapi.FromHz) {
	mainLogger.Debug("Horizntal Message", "msg", msg)
}

// Handle incoming Gossip Registration (Notify) Messages
func handleTypeRegistration(msg verticalapi.VertToMainRegister, storage *notifyMap) {
	typeToRegister := vertTypes.GossipType(msg.Data.DataType)
	listeningModule := msg.Module
	storage.AddChannelToType(typeToRegister, listeningModule)
	//mainLogger.Debug("Just registered: %d with val %v\n", "msg", typeToRegister, storage.Load(typeToRegister))
}

// Handle incoming Gossip Validation messages.
func handleGossipValidation(msg verticalapi.VertToMainValidation) {
	validation_data := msg.Data
	mainLogger.Debug("Validation data handled: ", "msg", validation_data)
}

// Handle incoming Gossip Announce messages. This function sould call the GOSSIP STRATEGY module
// and use that to spread the message. The type registering is done here.
func handleGossipAnnounce(msg verticalapi.VertToMainAnnounce, storage *notifyMap) error {
	typeToCheck := vertTypes.GossipType(msg.Data.DataType)
	res := storage.Load(typeToCheck)
	if len(res) == 0 {
		return errors.New("Gossip Type not registered, cannot accept message.")
	}

	announce_data := msg.Data
	mainLogger.Debug("Gossip Announce", "msg", announce_data)
	return nil
}
