package main

import (
	gossip "gossip/main"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// init
	m := gossip.NewMain()

	// run
	initFin := make(chan error, 1)
	go m.Run(initFin)
	err := <-initFin
	if err != nil {
		panic(err)
	}

	// teardown
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c
	m.Close()
}
