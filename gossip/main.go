package main

import (
	"flag"
	horizontalapi "gossip/horizontalAPI"
	verticalapi "gossip/verticalAPI"
	vertTypes "gossip/verticalAPI/types"

	"strings"

	"log/slog"
	"os"
	"time"

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

var (
	gossip_degree     int
	gossip_cache_size int
	hz_addr           string
	vert_addr         string
	peer_addrs_string string
)

var peer_addrs []string

func main() {
	// var err error
	// set up logging
	slog.SetDefault(logInit())
	mainLogger = slog.With("module", "main")

	// just skip the ini parsing etc for now and only start/listen on the vertical api using constants as address
	// then later if you want to maybe extract those from the ini config (at a fixed path to skip the argument parsing stuff for now)

	mainLogger.Debug("ARGS", "args", os.Args)

	// Using the flag library to read commandline arguments, in the future default values will be read from
	// the config.ini file
	flag.IntVar(&gossip_degree, "gossip", 30, "Gossip parameter degree: Number of peers the current peer has to exchange information with")
	flag.IntVar(&gossip_cache_size, "cache", 50, "Gossip parameter cahce_size: Maximum number of data items to be held as part of the peer’s knowledge base. Older items will be removed to ensure space for newer items if the peer’s knowledge base exceeds this limit")
	flag.StringVar(&hz_addr, "h_addr", "127.0.0.1:13377", "Address to listen for incoming peer connections, ip:port")
	flag.StringVar(&vert_addr, "v_addr", "127.0.0.1:13379", "Address to listen for incoming peer connections, ip:port")
	flag.StringVar(&peer_addrs_string, "peers", "", "List of horizontal peers to connect to, [ip]:port comma separated. For example 1.0.0.1:13377,1.0.0.2:13377,1.0.0.3:13377")
	flag.Parse()

	mainLogger.Debug("CMD ARGS mandatory",
		"cache size", gossip_cache_size,
		"degree", gossip_degree,
	)

	peer_addrs = strings.Split(peer_addrs_string, ",")

	mainLogger.Debug("CMD ARGS mandatory",
		"Horizontal addr", hz_addr,
		"Vertical addr", vert_addr,
		"Peers addresses", peer_addrs,
	)

	vertToMain := verticalapi.VertToMainChans{
		Register:   make(chan verticalapi.VertToMainRegister),
		Announce:   make(chan verticalapi.VertToMainAnnounce),
		Validation: make(chan verticalapi.VertToMainValidation),
	}

	va := verticalapi.NewVerticalApi(slog.With("module", "vertAPI"), vertToMain)
	va.Listen(vert_addr)
	defer va.Close()

	fromHz := make(chan horizontalapi.FromHz, 1)
	hz := horizontalapi.NewHorizontalApi(slog.With("module", "horzAPI"), fromHz)
	hzConnection := make(chan horizontalapi.NewConn, 1)
	hz.Listen(hz_addr, hzConnection)
	defer hz.Close()

	hz.AddNeighbors(peer_addrs...)

	typeStorage := NewNotifyMap()

	for {
		select {
		case x := <-vertToMain.Validation:
			handleGossipValidation(x)
		case x := <-vertToMain.Announce:
			mainLogger.Debug("a", "b", x)
			// _ = handleGossipAnnounce(x, horzToMain.RelayAnnounce, typeStorage)
		case x := <-vertToMain.Register:
			handleTypeRegistration(x, typeStorage)
		case x := <-fromHz:
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

// Handle incoming Gossip Announce messages.
// func handleGossipAnnounce(msg verticalapi.VertToMainAnnounce, targetChan chan horizontalapi.MainToHorzAnnounce, storage *notifyMap) error {
// 	typeToCheck := vertTypes.GossipType(msg.Data.DataType)
// 	res := storage.Load(typeToCheck)
// 	if len(res) == 0 {
// 		return errors.New("Gossip Type not registered, cannot accept message.")
// 	}
// 	announce_data := msg.Data
// 	enriched_announce := horizontalapi.MainToHorzAnnounce{
// 		Data: announce_data,
// 	}
// 	targetChan <- enriched_announce
// 	return nil
// }
