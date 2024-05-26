package main

import (
	"fmt"
	horizontalapi "gossip/horizontalAPI"
	verticalapi "gossip/verticalAPI"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	_ "gopkg.in/ini.v1"
)

func logInit() *slog.Logger {
	return slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}))
}

// Handle incoming Gossip Registration (Notify) Messages
func handleTypeRegistration(msg verticalapi.VertToMainRegister) {
	registration_data := msg.Data
	fmt.Println(registration_data)
}

// Handle incoming Gossip Validation messages.
func handleGossipValidation(msg verticalapi.VertToMainValidation) {
	validation_data := msg.Data
	fmt.Println(validation_data)
}

// Handle incoming Gossip Announce messages.
func handleGossipAnnounce(msg verticalapi.VertToMainAnnounce, targetChan chan horizontalapi.MainToHorzAnnounce) {
	// TODO: check that the type is one we have received a NOTIFY for
	announce_data := msg.Data
	enriched_announce := horizontalapi.MainToHorzAnnounce{
		Data: announce_data,
	}
	targetChan <- enriched_announce
}

func main() {
	// var err error
	// set up logging
	slog.SetDefault(logInit())

	vertToMain := verticalapi.VertToMainChans{
		Register:   make(chan verticalapi.VertToMainRegister),
		Announce:   make(chan verticalapi.VertToMainAnnounce),
		Validation: make(chan verticalapi.VertToMainValidation),
	}
	va := verticalapi.NewVerticalApi(slog.With("module", "vertAPI"), vertToMain)
	va.Listen("localhost:13379")

	// just skip the ini parsing etc for now and only start/listen on the vertical api using constants as address
	// then later if you want to maybe extract those from the ini config (at a fixed path to skip the argument parsing stuff for now)

	horzToMain := horizontalapi.MainToHorzChans{
		RelayAnnounce: make(chan horizontalapi.MainToHorzAnnounce),
	}
	ha := horizontalapi.NewHorizontalApi(slog.With("module", "horzAPI"), horzToMain)
	ha.SpreadMessages()

	for {

		select {
		case x := <-vertToMain.Validation:
			handleGossipValidation(x)
		case x := <-vertToMain.Announce:
			handleGossipAnnounce(x, horzToMain.RelayAnnounce)
		case x := <-vertToMain.Register:
			handleTypeRegistration(x)
		}
	}

	fmt.Println("FINISHED")
	va.Close()
}
