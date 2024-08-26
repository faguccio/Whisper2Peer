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
	"strings"
	"time"

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

type gossipConnection struct {
	connection horizontalapi.Conn[chan<- horizontalapi.ToHz]
	timestamp  time.Time
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

	openConnections, err := hz.AddNeighbors(&net.Dialer{LocalAddr: &net.TCPAddr{IP: net.ParseIP(strings.SplitN(args.Hz_addr, ":", 2)[0]), Port: 0}}, args.Peer_addrs...)

	hzInitFin := make(chan struct{}, 1)

	hzConnection := make(chan horizontalapi.NewConn, 1)
	hz.Listen(args.Hz_addr, hzConnection, hzInitFin)

	go func(initFinished chan<- struct{}, hzInitFin <-chan struct{}) {
		<-hzInitFin
		initFinished <- struct{}{}
	}(initFinished, hzInitFin)

	var gossipConnections []gossipConnection

	// Request PoW (and compute it) here
	for _, conn := range openConnections {
		req := horizontalapi.ConnReq{}
		conn.Data <- req
		x := <-fromHz
		switch chall := x.(type) {
		case horizontalapi.ConnChall:

			mypow := powMarsh{
				PowNonce: 0,
				Cookie:   chall.Cookie,
			}

			nonce := pow.ProofOfWork(func(digest []byte) bool {
				return pow.First8bits0(digest)
			}, &mypow)

			mypow.PowNonce = nonce
			conn.Data <- horizontalapi.ConnPoW{PowNonce: mypow.PowNonce, Cookie: mypow.Cookie}
			gossipConnections = append(gossipConnections, gossipConnection{
				connection: conn,
				timestamp:  time.Now(),
			})

		default:
			log.Debug("Second message during validation is not a ConnChall", "message", "chall")
			continue
		}
	}

	if err != nil {
		return nil, fmt.Errorf("The error occured while initiating the gossip module %w", err)
	}

	// Hardcoded strategy, later switching on args argument
	dummyStrat := NewDummy(strategy, fromHz, hzConnection, gossipConnections)
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
