package strats

import (
	"context"
	"fmt"
	"gossip/common"
	horizontalapi "gossip/horizontalAPI"
	"gossip/internal/args"

	"log/slog"
)

// This struct represents a base strategy, which is an abstraction over common fields (and in the future, methods) to all strategies.
type Strategy struct {
	// internally uses a context to signal when the goroutines shall terminate
	cancel context.CancelFunc
	// internally uses a context to signal when the goroutines shall terminate
	ctx context.Context
	// Internally spawn and uses the horizotal API
	hz               *horizontalapi.HorizontalApi
	strategyChannels StrategyChannels
	log              *slog.Logger
	stratArgs        args.Args
}

// Any strategy should implement the strategyCloser type, so a Listen method and a Close one.
type StrategyCloser interface {
	Close()
	Listen()
}

// Ingoing and outgoing channels of any strategy
type StrategyChannels struct {
	FromStrat chan common.FromStrat
	ToStrat   chan common.ToStrat
}

// This function instantiate a new Strategy.
//
// The function internally spawn the horizontal API and connect to all given peers and start to
// listen on the given address.
// It instantiate a strategy too. The caller has to call Listen to start the strategy and Close
// to end it.
func New(log *slog.Logger, args args.Args, stratChans StrategyChannels) (StrategyCloser, error) {

	fromHz := make(chan horizontalapi.FromHz, 1)
	hz := horizontalapi.NewHorizontalApi(log, fromHz)
	// context is only used internally -> no need to pass it to the constructor
	ctx, cancel := context.WithCancel(context.Background())
	strategy := Strategy{
		cancel:           cancel,
		ctx:              ctx,
		hz:               hz,
		strategyChannels: stratChans,
		stratArgs:        args,
		log:              log.With("module", "strategy"),
	}

	hzConnection := make(chan horizontalapi.NewConn, 1)
	hz.Listen(args.Hz_addr, hzConnection)

	openConnections, err := hz.AddNeighbors(args.Peer_addrs...)

	if err != nil {
		return nil, fmt.Errorf("The error occured while initiating the gossip module %w", err)
	}

	// Hardcoded strategy, later switching on args argument
	dummyStrat := NewDummy(strategy, fromHz, hzConnection, openConnections)
	return &dummyStrat, nil
}

// Simply closes the horizontal API
func (strt *Strategy) Close() {
	strt.cancel()
	strt.hz.Close()
}
