package main

import (
	"errors"
	"fmt"
	horizontalapi "gossip/horizontalAPI"
	verticalapi "gossip/verticalAPI"
	vertTypes "gossip/verticalAPI/types"

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
func handleTypeRegistration(msg verticalapi.VertToMainRegister, storage *notifyMap) {
	typeToRegister := vertTypes.GossipType(msg.Data.DataType)
	// I think we are missing a (multiple?) channel mainToVert. This is a useless placeholder
	newMainToVert := NotificationChannel{NotificationChannel: make(chan string)}
	storage.AddChannelToType(typeToRegister, newMainToVert)
	//fmt.Printf("Just registered: %d with val %v\n", typeToRegister, storage.Load(typeToRegister))
}

// Handle incoming Gossip Validation messages.
func handleGossipValidation(msg verticalapi.VertToMainValidation) {
	validation_data := msg.Data
	fmt.Println(validation_data)
}

// Handle incoming Gossip Announce messages.
func handleGossipAnnounce(msg verticalapi.VertToMainAnnounce, targetChan chan horizontalapi.MainToHorzAnnounce, storage *notifyMap) error {
	typeToCheck := vertTypes.GossipType(msg.Data.DataType)
	res := storage.Load(typeToCheck)
	//fmt.Printf("Just checked: %d with val %v\n", typeToCheck, storage.Load(typeToCheck))
	if len(res) == 0 {
		return errors.New("Gossip Type not registered, cannot accept message.")
	}
	announce_data := msg.Data
	enriched_announce := horizontalapi.MainToHorzAnnounce{
		Data: announce_data,
	}
	targetChan <- enriched_announce
	return nil
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

	typeStorage := NewNotifyMap()

	for {
		select {
		case x := <-vertToMain.Validation:
			handleGossipValidation(x)
		case x := <-vertToMain.Announce:
			_ = handleGossipAnnounce(x, horzToMain.RelayAnnounce, typeStorage)
			//fmt.Println(res)
		case x := <-vertToMain.Register:
			handleTypeRegistration(x, typeStorage)
		}
	}

	va.Close()
}
