package testutils

import (
	"context"
	"errors"
	"fmt"
	"gossip/common"
	"gossip/internal/args"
	gossip "gossip/main"
	vtypes "gossip/verticalAPI/types"
	"io"
	"net"
	"net/netip"
	"slices"
	"time"
)

type testState string
var (
	TestStateInit testState = "" // use zero value here
	TestStateRunning testState = "running"
	TestStateProcessing testState = "processing"
)

// implement the stringer interface
func (t testState) String() string {
	// init shouldn't be printed as "" (zero value) but as "init"
	if t == TestStateInit {
		return "init"
	}
	return string(t)
}

// structure which holds information over the graph/topology under test
// should be instanciated with [NewTesterFromJSON]
//
// The usual procedure is [NewTesterFromJSON] -> [Startup] -> some custom logic
// / sleep -> [Teardown] -> Processing with the ProcessLogsGen* functions
type Tester struct {
	// the graph on which the test is executed
	G        Graph
	// mapping from nodeIdx to peer
	Peers    map[uint]*peer
	// mapping from connectionId (ip address) to nodeIdx
	PeersLut map[common.ConnectionId]uint
	// the logger of each peer will indirectly write it's testing events onto
	// this channel
	logChan chan Event
	// the tester will send the types of observed package (on hzApi) to this
	// channel when observing it. Can be used to sleep until packets with a
	// certain type do not appear anymore. This is not completely reliable.
	// When the (buffered) channel is full, messages might get lost.
	busyChan chan common.GossipType
	// all Events are collected here (only neededb ecause there are no
	// unlimited buffered channels)
	Events  []Event
	// field for bookkeeping what needs to be closed in the end
	closers []io.Closer
	// store in which state the tester is
	state testState
	// store min/max value for time during the test.
	// guaranteed to be set when in processing state
	tmin time.Time
	tmax time.Time
	durSec float64
	// caching for distances in the processing state
	distanceBook distanceBook
	// context used to nofity spwaned goroutines about teardown
	cfunc context.CancelFunc
}

// Create a new tester
//
// reads all the config stuff from the json file fn
func NewTesterFromJSON(fn string) (*Tester,error) {
	t := &Tester{
		Peers:    make(map[uint]*peer),
		PeersLut: make(map[common.ConnectionId]uint),
		logChan:  make(chan Event, 64),
		busyChan: make(chan common.GossipType, 64),
		Events:   make([]Event, 0),
		closers:  make([]io.Closer, 0),
	}
	var err error

	// parse graph from provided file
	t.G, err = NewGraphFromJSON(fn)
	if err != nil {
		return t, err
	}

	return t, nil
}

// starts all the peers etc
// the addresses for the peers whill be allocated starting with 127.0.0.1
func (t *Tester) Startup(startIp string) error {
	if t.state != TestStateInit {
		return errors.New("cannot start a tester which is not in init state")
	}

	var ctx context.Context
	ctx, t.cfunc = context.WithCancel(context.Background())

	// iterator for ip address
	if startIp == "" {
		startIp = "127.0.0.1"
	}
	ip := netip.MustParseAddr(startIp)

	// goroutine simply copies the events over to the events slice
	// NOTE: when closing the channel will also terminate the goroutine
	go func(logChan <-chan Event, busyChan chan<- common.GossipType) {
		for e := range logChan {
			t.Events = append(t.Events, e)
			// if is packet on hz api
			if e.Msg == "hz packet sent" {
				// if eg channel is full, don't block, simply loose/skip the
				// nofitication
				select {
					case busyChan <- e.MsgType:
					default:
				}
			}
		}
	}(t.logChan, t.busyChan)

	// create peers
	for nodeIdx, node := range t.G.Nodes {
		nodeIdx := uint(nodeIdx)

		args := args.Args{
			Hz_addr:    ip.String() + ":7001",
			Vert_addr:  ip.String() + ":6001",
			Peer_addrs: []string{},
		}

		// read config, use config from json. If unset, use default values
		if node.Degree != nil {
			args.Degree = *node.Degree
		} else {
			args.Degree = 30
		}
		if node.Cache_size != nil {
			args.Cache_size = *node.Cache_size
		} else {
			args.Cache_size = 50
		}
		if node.GossipTimer != nil {
			args.GossipTimer = *node.GossipTimer
		} else {
			args.GossipTimer = 1
		}

		// add the neighbors
		for _, edge := range t.G.Edges {
			// make sure edge "tuple" is ordered
			if edge[1] < edge[0] {
				tmp := edge[0]
				edge[0] = edge[1]
				edge[1] = tmp
			}
			if edge[1] == nodeIdx {
				// add neighbor
				args.Peer_addrs = append(args.Peer_addrs, t.Peers[edge[0]].a.Hz_addr)
			}
		}

		// add item for bookkeeping
		p := &peer{
			idx: nodeIdx,
			id:  common.ConnectionId(ip.String()),
			a:   args,
			dialer: &net.Dialer{
				LocalAddr: &net.TCPAddr{
					IP:   ip.AsSlice(),
					Port: 0,
				},
			},
		}
		t.Peers[nodeIdx] = p
		t.PeersLut[p.id] = nodeIdx

		// create a pipe to process the generates json logs
		var wPipe io.Writer
		var rPipe io.Reader
		{
			_rPipe, _wPipe := io.Pipe()
			t.closers = append(t.closers, _rPipe, _wPipe)
			wPipe, rPipe = _wPipe, _rPipe
		}
		// print logs on stdout as well
		// wPipe = io.MultiWriter(os.Stdout, wPipe)

		// start the peer
		go FilterLog(ctx, t.logChan, rPipe)
		m := gossip.NewMainWithArgs(args, LogInit(wPipe, p.id))

		initFin := make(chan error, 1)
		go m.Run(initFin)
		err := <- initFin
		if err != nil {
			return err
		}

		t.closers = append(t.closers, m)

		ip = ip.Next()
	}

	// connect to all peers on the vertical api
	for _, p := range t.Peers {
		err := p.connect()
		if err != nil {
			return err
		}
		// mark all incoming notifications as valid (so that they are disseminated further)
		// NOTE: when closing the connection this will terminate
		go p.markAllValid()
	}

	t.state = TestStateRunning
	return nil
}

// register all peers for a certain GossipType (so that they will relay the
// respective messages).
func (t *Tester) RegisterAllPeersForType(gtype common.GossipType) error {
	var err error
	for _, p := range t.Peers {
		msg := vtypes.GossipNotify{
			Gn: common.GossipNotify{
				Reserved: 0,
				DataType: gtype,
			},
			MessageHeader: vtypes.MessageHeader{
				Type: vtypes.GossipNotifyType,
			},
		}
		msg.MessageHeader.RecalcSize(&msg)

		if err = p.SendMsg(&msg); err != nil {
			return err
		}
	}
	return nil
}

// monitor the hzAPI and wait until for interval, no packet of gtype (or of any
// type) was observed
//
// context can/should be used to e.g. specify a timeout ([context.WithTimeout])
func (t *Tester) WaitUntilSilent(ctx context.Context, all bool, gtype common.GossipType, interval time.Duration) error {
	timer := time.NewTimer(interval)
	for {
		select {
		case gt := <- t.busyChan:
			if all || gt == gtype {
				timer.Reset(interval)
			}
		case <-ctx.Done():
			return fmt.Errorf("Wait-context: %w", ctx.Err())
		case <- timer.C:
			return nil
		}
	}
}

// teardown the running test.
//
// This includes closing all connections, pipes and channels needed during the
// test.
func (t *Tester) Teardown() error {
	if t.state != TestStateRunning {
		return errors.New("cannot teardown a not running test")
	}
	for _, p := range t.Peers {
		p.close()
	}
	close(t.logChan)
	close(t.busyChan)
	for _, c := range t.closers {
		c.Close()
	}

	// check during which time the test ran
	t.tmin = time.Now()
	t.tmax = time.Now() // max is the current time, not when the last log came in
	for _, e := range t.Events {
		if e.Time.Before(t.tmin) {
			t.tmin = e.Time
		}
	}
	t.durSec = float64(t.tmax.UnixMilli() - t.tmin.UnixMilli())/1000

	// sort logs with time as key to ensure the right order when processing
	slices.SortFunc(t.Events, func(a Event, b Event) int { return a.Time.Compare(b.Time) })

	t.state = TestStateProcessing
	return nil
}
