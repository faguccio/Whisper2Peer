package testutils

import (
	"errors"
	"fmt"
	"gossip/common"
	"gossip/internal/args"
	gossip "gossip/main"
	vtypes "gossip/verticalAPI/types"
	"io"
	"net"
	"net/netip"
	"os"
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
	// caching for distances in the processing state
	distanceBook distanceBook
}

// Create a new tester
//
// reads all the config stuff from the json file fn
func NewTesterFromJSON(fn string) (*Tester,error) {
	t := &Tester{
		Peers:    make(map[uint]*peer),
		PeersLut: make(map[common.ConnectionId]uint),
		logChan:  make(chan Event, 64),
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
func (t *Tester) Startup() error {
	if t.state != TestStateInit {
		return errors.New("cannot start a tester which is not in init state")
	}

	// iterator for ip address
	ip := netip.MustParseAddr("127.0.0.1")

	// goroutine simply copies the events over to the events slice
	// NOTE: when closing the channel will also terminate the goroutine
	go func(logChan <-chan Event) {
		for e := range logChan {
			t.Events = append(t.Events, e)
		}
	}(t.logChan)

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
		go FilterLog(t.logChan, rPipe)
		m := gossip.NewMainWithArgs(args, LogInit(wPipe, p.id))
		go m.Run()
		t.closers = append(t.closers, m)

		time.Sleep(500 * time.Millisecond)
		ip = ip.Next()
	}

	// connect to all peers on the vertical api
	for _, p := range t.Peers {
		err := p.connect()
		if err != nil {
			return err
		}
		// mark all incoming notifications as valid (so that they are disseminated further)
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

	// sort logs with time as key to ensure the right order when processing
	slices.SortFunc(t.Events, func(a Event, b Event) int { return a.Time.Compare(b.Time) })

	t.state = TestStateProcessing
	return nil
}

// generate a css style sheet to draw colored graph
func (t *Tester) ProcessLogsGenReachedWhenCSS(fn string) error {
	if t.state != TestStateProcessing {
		return errors.New("cannot do processing if tester is not in processing state")
	}

	f, err := os.Create(fn)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for _, e := range t.Events {
		if !(e.Msg == "received" || e.Msg == "announce") {
			continue
		}
		// calculate when the node received the message, relative to the duration of the test [0,200]
		// we extend percentage [0,100] to use more than two colors
		tRel := (200*e.Time.UnixMilli() - 200*t.tmin.UnixMilli()) / (t.tmax.UnixMilli() - t.tmin.UnixMilli())
		id := t.PeersLut[e.Id]
		if tRel <= 100 {
			fmt.Fprintf(f, `._%d>ellipse {
    fill: color-mix(in srgb, yellow %d%%, green);
}
`, id, tRel)
		} else if tRel <= 200 {
			fmt.Fprintf(f, `._%d>ellipse {
    fill: color-mix(in srgb, red %d%%, yellow);
}
`, id, tRel-100)
		}
	}
	return nil
}

// generate timeseries with cnt received after distance
func (t *Tester) ProcessLogsGenDistReachedCSV(fn string, startNode uint) error {
	if t.state != TestStateProcessing {
		return errors.New("cannot do processing if tester is not in processing state")
	}

	f, err := os.Create(fn)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// setup for distances
	distCnt := t.distanceBook.processingSetupForDistance(func(u uint) map[uint]uint {return t.G.CalcDistances(u)}, startNode)

	fmt.Fprintf(f, "time")
	for _, d := range t.distanceBook.distOrd {
		fmt.Fprintf(f, ";%d", d)
	}
	fmt.Fprintf(f, "\n")

	// print the actual data (amount of nodes with specific distance received)
	for _, e := range t.Events {
		if !(e.Msg == "received" || e.Msg == "announce") {
			continue
		}
		nodeIdx := t.PeersLut[e.Id]
		dist := t.distanceBook.nodeToDist[nodeIdx]
		distCnt[dist] += 1
		fmt.Fprintf(f, "%f", float64(e.Time.UnixMilli()-t.tmin.UnixMilli())/1000)
		for _, d := range t.distanceBook.distOrd {
			fmt.Fprintf(f, ";%d", distCnt[d])
		}
		fmt.Fprintf(f, "\n")
	}
	return nil
}

// print how many nodes exist with a specific distance
func (t *Tester) ProcessLogsGenDistCntCSV(fn string, startNode uint) error {
	if t.state != TestStateProcessing {
		return errors.New("cannot do processing if tester is not in processing state")
	}

	f, err := os.Create(fn)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// setup for distances
	t.distanceBook.processingSetupForDistance(func(u uint) map[uint]uint {return t.G.CalcDistances(u)}, startNode)

	fmt.Fprintf(f, "time")
	for _, d := range t.distanceBook.distOrd {
		fmt.Fprintf(f, ";%d", d)
	}
	fmt.Fprintf(f, "\n%f", 0.0)
	for _, d := range t.distanceBook.distOrd {
		fmt.Fprintf(f, ";%d", t.distanceBook.distMaxCnt[d])
	}
	fmt.Fprintf(f, "\n%f", float64(t.tmax.UnixMilli()-t.tmin.UnixMilli())/1000)
	for _, d := range t.distanceBook.distOrd {
		fmt.Fprintf(f, ";%d", t.distanceBook.distMaxCnt[d])
	}
	fmt.Fprintf(f, "\n")
	return nil
}

// generate timeseries with amount of packets sent over time
func (t *Tester) ProcessLogsGenSentPacketsCSV(fn string) error {
	if t.state != TestStateProcessing {
		return errors.New("cannot do processing if tester is not in processing state")
	}

	f, err := os.Create(fn)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	sentEvents := make([]Event, 0, len(t.Events))
	for _, e := range t.Events {
		if !(e.Msg == "hz packet sent") {
			continue
		}
		sentEvents = append(sentEvents, e)
	}
	slices.SortFunc(sentEvents, func(a Event, b Event) int { return a.TimeBucket.Compare(b.TimeBucket) })

	var currTime time.Time
	var cnt uint
	fmt.Fprintf(f, "time;cnt\n")
	for _, e := range sentEvents {
		if !(e.Msg == "hz packet sent") {
			continue
		}
		if currTime.Equal(e.TimeBucket) {
			cnt += e.Cnt
			continue
		}
		if !currTime.IsZero() {
			fmt.Fprintf(f, "%f;%d\n", float64(currTime.UnixMilli()-t.tmin.UnixMilli())/1000, cnt)
		}
		currTime = e.TimeBucket
		cnt = e.Cnt
	}
	fmt.Fprintf(f, "%f;%f\n", float64(t.tmax.UnixMilli()-t.tmin.UnixMilli()+50)/1000, 0.0)

	// print the stats once again
	for _, e := range t.Events {
		fmt.Printf("%+v\n", e)
	}

	return nil
}

// some orga stuff to avoid calculating the distances too often
// this struct only contains entries which will/must not change during
// processing (therefore no dynamic distCnt included)
type distanceBook struct {
	// store if distanceBook was already initialized
	valid bool
	// store the startNode for which the distances were calculated
	startNode uint
	// actual generated data
	nodeToDist map[uint]uint
	distOrd []uint
	distMaxCnt map[uint]uint
}

// setup working with distances with the given start node. If the distances for
// this start node are already calculated, don't re-calculate the distances
func (db *distanceBook) processingSetupForDistance(genDistances func(uint)map[uint]uint, startNode uint) (map[uint]uint) {
	distCnt := make(map[uint]uint)
	if db.valid && db.startNode == startNode {
		// init map with known distances
		for d := range db.distMaxCnt {
			distCnt[d] = 0
		}
		return distCnt
	}

	db.nodeToDist = genDistances(startNode)
	db.distMaxCnt = make(map[uint]uint)
	// collect (and count) known distances
	for _, v := range db.nodeToDist {
		if _, ok := distCnt[v]; !ok {
			distCnt[v] = 0
			db.distMaxCnt[v] = 0
			db.distOrd = append(db.distOrd, v)
		}
		db.distMaxCnt[v] += 1
	}
	slices.Sort(db.distOrd)

	db.valid = true
	db.startNode = startNode

	fmt.Printf("%+v\n", db)

	return distCnt
}

