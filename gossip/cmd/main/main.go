package main

import (
	"context"
	gossip "gossip/main"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// init
	m := gossip.NewMain()
	ctx, cancel := context.WithCancel(context.Background())

	// run
	m.Run(ctx)

	// teardown
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	<-c
	cancel()
}
