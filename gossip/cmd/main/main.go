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
	initFin := make(chan struct{}, 1)
	go m.Run(initFin)
	<- initFin

	// teardown
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c
	m.Close()
}
