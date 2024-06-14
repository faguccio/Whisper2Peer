package strats

import (
	"fmt"
	"gossip/internal/args"
	horizontalapi "gossip/horizontalAPI"
	verticalapi "gossip/verticalAPI"

	"log/slog"
)

type Strategy struct {
	hz               *horizontalapi.HorizontalApi
	strategyChannels StrategyChannels
}

type StrategyCloser interface {
	Close()
	Listen()
}

type StrategyChannels struct {
	Notification chan verticalapi.MainToVertNotification
	Announce     chan verticalapi.VertToMainAnnounce
	Validation   chan verticalapi.VertToMainValidation
}

func New(args args.Args, stratChans StrategyChannels) (StrategyCloser, error) {

	fromHz := make(chan horizontalapi.FromHz, 1)
	hz := horizontalapi.NewHorizontalApi(slog.With("module", "horzAPI"), fromHz)
	strategy := Strategy{
		hz:               hz,
		strategyChannels: stratChans,
	}

	hzConnection := make(chan horizontalapi.NewConn, 1)
	hz.Listen(args.Hz_addr, hzConnection)

	openConnections, err := hz.AddNeighbors(args.Peer_addrs...)

	if err != nil {
		return nil, fmt.Errorf("The error occured while initiating the gossip module %w", err)
	}

	// Hardcoded strategy, later switching
	dummyStrat := NewDummy(strategy, fromHz, hzConnection, openConnections)
	return &dummyStrat, nil
}

func (strt *Strategy) Close() {
	strt.hz.Close()
}
