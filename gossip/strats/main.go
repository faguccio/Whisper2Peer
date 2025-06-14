/*
* gossip
* Copyright (C) 2024 Fabio Gaiba and Lukas Heindl
*
* This program is free software: you can redistribute it and/or modify
* it under the terms of the GNU General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* This program is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
* GNU General Public License for more details.
*
* You should have received a copy of the GNU General Public License
* along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

// Package strats implements the different gossip strategies that can be used.
package strats

import (
	"context"
	"encoding/binary"
	"fmt"
	"gossip/common"
	horizontalapi "gossip/horizontalAPI"
	"gossip/internal/args"
	pow "gossip/pow"
	"io"
	"net"
	"slices"

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
func New(log *slog.Logger, args args.Args, stratChans StrategyChannels, initFinished chan<- struct{}) (StrategyCloser, error) {
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

	host, _, err := net.SplitHostPort(args.Hz_addr)
	if err != nil {
		return nil, err
	}
	openConnections, err := hz.AddNeighbors(&net.Dialer{LocalAddr: &net.TCPAddr{IP: net.ParseIP(host), Port: 0}}, args.Peer_addrs...)

	hzInitFin := make(chan struct{}, 1)

	hz.Listen(args.Hz_addr, hzInitFin)

	go func(initFinished chan<- struct{}, hzInitFin <-chan struct{}) {
		<-hzInitFin
		initFinished <- struct{}{}
	}(initFinished, hzInitFin)

	if err != nil {
		return nil, fmt.Errorf("the error occured while initiating the gossip module %w", err)
	}

	connManager := NewConnectionManager(openConnections)

	// Hardcoded strategy, later switching on args argument
	dummyStrat := NewDummy(strategy, fromHz, &connManager)
	return &dummyStrat, nil
}

// Simply closes the horizontal API
func (strt *Strategy) Close() {
	strt.cancel()
	strt.hz.Close()
}

// Make the ConnPoW message implement the interface POWMarshaller
type powMarsh horizontalapi.ConnPoW

// marshal the (whole) struct to a bytes slice
func (x *powMarsh) Marshal(buf []byte) ([]byte, error) {
	buf = slices.Grow(buf, binary.Size(x.PowNonce)+len(x.Cookie))
	buf = buf[:binary.Size(x.PowNonce)+len(x.Cookie)]

	idx := 0

	copy(buf[idx:], x.Cookie[:])
	idx += len(x.Cookie)

	binary.BigEndian.PutUint64(buf[idx:], x.PowNonce)
	idx += binary.Size(x.PowNonce)

	return buf, nil
}

// obtain the nonce of the struct
func (x *powMarsh) Nonce() uint64 {
	return x.PowNonce
}

// set the nonce of the struct
func (x *powMarsh) SetNonce(n uint64) {
	x.PowNonce = n
}

func (x *powMarsh) AddToNonce(n uint64) {
	x.PowNonce += n
}

// how many bytes in the beginning of the bytes slice (from marshal) should
// be skipped (and not be included in the PoW)
func (x *powMarsh) StripPrefixLen() uint {
	return 0
}

// remaining length of the struct minus the skipped part and minus the
// length of the nonce
func (x *powMarsh) PrefixLen() uint {
	return uint(binary.Size(x.Cookie))
}

// write the nonce to an arbitrary writer (used to write the nonce to the
// hash object)
func (x *powMarsh) WriteNonce(w io.Writer) {
	w.Write(binary.BigEndian.AppendUint64(nil, x.PowNonce))
}

func (x *powMarsh) Clone() pow.POWMarshaller[uint64] {
	return &powMarsh{
		PowNonce: x.PowNonce,
		Cookie:   x.Cookie,
	}
}
