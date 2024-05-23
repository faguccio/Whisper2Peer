package main

import (
	"fmt"
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

func handleTypeRegistration() {
	fmt.Println()
}

func handleGossipNotification() {
	fmt.Println("Handle Notification")
}

func handleGossipAnnounce() {
	fmt.Println("Handle Announce")
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

	fmt.Println("MAIN ABOUT TO WAIT")
	select {
	case x := <-vertToMain.Validation:
		fmt.Println(x)
		fmt.Println("Handle Validation???")
	case x := <-vertToMain.Announce:
		handleGossipAnnounce()
		fmt.Println(x)
	case x := <-vertToMain.Register:
		fmt.Println(x)
		handleTypeRegistration()

	}
	fmt.Println("FINISHED")
	va.Close()
}
