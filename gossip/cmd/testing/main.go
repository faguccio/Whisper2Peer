package main

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

// bookkeeping structure for known/started peers
type peer struct {
	idx    uint
	id     common.ConnectionId
	a      args.Args
	conn   net.Conn
	dialer *net.Dialer
}

func (p *peer) close() {
	p.conn.Close()
}

func (p *peer) connect() error {
	var err error
	p.conn, err = p.dialer.Dial("tcp", p.a.Vert_addr)
	return err
}

func (p *peer) sendMsg(v Marshaler) error {
	msg, err := v.Marshal(nil)
	if err != nil {
		return err
	}

	if n, err := p.conn.Write(msg); err != nil {
		return err
	} else if n != len(msg) {
		return errors.New("Message could not be written entirely")
	}
	return err
}

func (p *peer) markAllValid() {
	var msgHdr vtypes.MessageHeader
	buf := make([]byte, msgHdr.CalcSize())
	var gn vtypes.GossipNotification

	for {
		// TODO code duplication with vertAPI -> maybe some refactoring later
		buf = buf[0:msgHdr.CalcSize()]
		// read the message header
		nRead, err := io.ReadFull(p.conn, buf)
		if errors.Is(err, io.EOF) {
			return
		}

		_, err = msgHdr.Unmarshal(buf)
		if err != nil {
			continue
		}

		// allocate space for the message body
		buf = slices.Grow(buf, int(msgHdr.Size)-nRead)
		buf = buf[0:int(msgHdr.Size)]

		// read the message body
		_, err = io.ReadFull(p.conn, buf[nRead:])
		if msgHdr.Type != vtypes.GossipNotificationType {
			continue
		}
		gn.MessageHeader = msgHdr

		_, err = gn.Unmarshal(buf)
		if err != nil {
			print("markAllValid: ", err.Error(), "\n")
			continue
		}

		msg := vtypes.GossipValidation{
			MessageHeader: vtypes.MessageHeader{
				Type: vtypes.GossipValidationType,
			},
			Gv: common.GossipValidation{
				MessageId: gn.Gn.MessageId,
			},
		}
		msg.Gv.SetValid(true)
		msg.MessageHeader.RecalcSize(&msg)
		if err = p.sendMsg(&msg); err != nil {
			print("markAllValid: ", err.Error(), "\n")
			return
		}
	}
}

type Marshaler interface {
	Marshal(buf []byte) ([]byte, error)
}

type Tester struct {
	g        graph
	peers    map[uint]*peer
	peersLut map[common.ConnectionId]uint
	// the logger of each peer will indirectly write it's testing events onto this channel
	logChan chan event
	// all events are collected here (mostly because there are no unlimited buffered channels)
	events  []event
	closers []io.Closer
}

func NewTester() *Tester {
	return &Tester{
		peers:    make(map[uint]*peer),
		peersLut: make(map[common.ConnectionId]uint),
		logChan:  make(chan event, 64),
		events:   make([]event, 0),
		closers:  make([]io.Closer, 0),
	}
}

// reads all the config stuff
func (t *Tester) Init() error {
	var err error

	// parse graph from provided file
	// TODO graph file should be hardcoded in the end
	t.g, err = readGraph(os.Args[1])
	if err != nil {
		return err
	}

	return nil
}

// starts all the peers etc
func (t *Tester) Startup() error {
	// iterator for ip address
	ip := netip.MustParseAddr("127.0.0.1")

	// goroutine simply copies the events over to the events slice
	// NOTE: when closing the channel will also terminate the goroutine
	go func(logChan <-chan event) {
		for e := range logChan {
			t.events = append(t.events, e)
		}
	}(t.logChan)

	// create peers
	for nodeIdx, node := range t.g.Nodes {
		nodeIdx := uint(nodeIdx)

		args := args.Args{
			Hz_addr:    ip.String() + ":7001",
			Vert_addr:  ip.String() + ":6001",
			Peer_addrs: []string{},
		}

		// read config, use config from json but use passed config as default
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
		for _, edge := range t.g.Edges {
			// make sure edge "tuple" is ordered
			if edge[1] < edge[0] {
				tmp := edge[0]
				edge[0] = edge[1]
				edge[1] = tmp
			}
			if edge[1] == nodeIdx {
				// add neighbor
				args.Peer_addrs = append(args.Peer_addrs, t.peers[edge[0]].a.Hz_addr)
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
		t.peers[nodeIdx] = p
		t.peersLut[p.id] = nodeIdx

		// create a pipe to process the generates json logs
		var wPipe io.Writer
		var rPipe io.Reader
		{
			_rPipe, _wPipe := io.Pipe()
			t.closers = append(t.closers, _rPipe, _wPipe)
			wPipe, rPipe = _wPipe, _rPipe
		}
		// print logs on stdout as well
		wPipe = io.MultiWriter(os.Stdout, wPipe)

		// start the peer
		go filterLog(t.logChan, rPipe)
		m := gossip.NewMainWithArgs(args, logInit(wPipe, p.id))
		go m.Run()
		t.closers = append(t.closers, m)

		time.Sleep(500 * time.Millisecond)
		ip = ip.Next()
	}

	// connect on the vertival api to all peers
	for _, p := range t.peers {
		err := p.connect()
		if err != nil {
			return err
		}
		go p.markAllValid()
	}

	return nil
}

func (t *Tester) ProcessLogs() error {
	// sort logs after time
	slices.SortFunc(t.events, func(a event, b event) int { return a.Time.Compare(b.Time) })

	// check during which time the test ran
	tmin := time.Now()
	tmax := time.Now()
	for _, e := range t.events {
		if e.Time.Before(tmin) {
			tmin = e.Time
		}
		// if e.Time.After(tmax) {
		// 	tmax = e.Time
		// }
	}

	f, err := os.Create(".css")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	// generate a css style sheet to draw colored graph
	for _, e := range t.events {
		if !(e.Msg == "received" || e.Msg == "announce") {
			continue
		}
		// calculate when the node received the message, relative to the duration of the test [0,200]
		// we extend percentage [0,100] to use more than two colors
		tRel := (200*e.Time.UnixMilli() - 200*tmin.UnixMilli()) / (tmax.UnixMilli() - tmin.UnixMilli())
		id := t.peersLut[e.Id]
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

	// generate timeseries with cnt received after distance
	f, err = os.Create("reached_dist.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	// some orga stuff to set up the distances
	// TODO fixed start node
	nodeToDist := t.g.calcDistances(1)
	distCnt := make(map[uint]uint)
	distMaxCnt := make(map[uint]uint)
	var distOrd []uint
	for _, v := range nodeToDist {
		if _, ok := distCnt[v]; !ok {
			distCnt[v] = 0
			distMaxCnt[v] = 0
			distOrd = append(distOrd, v)
		}
		distMaxCnt[v] += 1
	}
	slices.Sort(distOrd)

	fmt.Fprintf(f, "time")
	for _, d := range distOrd {
		fmt.Fprintf(f, ";%d", d)
	}
	fmt.Fprintf(f, "\n")

	// print the actual data (amount of nodes with specific distance received)
	for _, e := range t.events {
		if !(e.Msg == "received" || e.Msg == "announce") {
			continue
		}
		nodeIdx := t.peersLut[e.Id]
		dist := nodeToDist[nodeIdx]
		distCnt[dist] += 1
		fmt.Fprintf(f, "%f", float64(e.Time.UnixMilli()-tmin.UnixMilli())/1000)
		for _, d := range distOrd {
			fmt.Fprintf(f, ";%d", distCnt[d])
		}
		fmt.Fprintf(f, "\n")
	}

	// print how many nodes exist with a specific distance
	f, err = os.Create("cnt_dist.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	fmt.Fprintf(f, "time")
	for _, d := range distOrd {
		fmt.Fprintf(f, ";%d", d)
	}
	fmt.Fprintf(f, "\n%f", 0.0)
	for _, d := range distOrd {
		fmt.Fprintf(f, ";%d", distMaxCnt[d])
	}
	fmt.Fprintf(f, "\n%f", float64(tmax.UnixMilli()-tmin.UnixMilli())/1000)
	for _, d := range distOrd {
		fmt.Fprintf(f, ";%d", distMaxCnt[d])
	}
	fmt.Fprintf(f, "\n")

	// generate timeseries with amount of packets sent over time
	f, err = os.Create("packets_sent.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	sentEvents := make([]event, 0, len(t.events))
	for _, e := range t.events {
		if !(e.Msg == "hz packet sent") {
			continue
		}
		sentEvents = append(sentEvents, e)
	}
	slices.SortFunc(sentEvents, func(a event, b event) int { return a.TimeBucket.Compare(b.TimeBucket) })

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
			fmt.Fprintf(f, "%f;%d\n", float64(currTime.UnixMilli()-tmin.UnixMilli())/1000, cnt)
		}
		currTime = e.TimeBucket
		cnt = e.Cnt
	}
	fmt.Fprintf(f, "%f;%f\n", float64(tmax.UnixMilli()-tmin.UnixMilli()+50)/1000, 0.0)

	// print the stats once again
	for _, e := range t.events {
		fmt.Printf("%+v\n", e)
	}

	return nil
}

func (t *Tester) Teardown() {
	for _, p := range t.peers {
		p.close()
	}
	close(t.logChan)
	for _, c := range t.closers {
		c.Close()
	}
}

func main() {
	var err error

	t := NewTester()

	if err = t.Init(); err != nil {
		panic(err)
	}

	if err = t.Startup(); err != nil {
		panic(err)
	}

	// register all peers for message type
	for _, p := range t.peers {
		msg := vtypes.GossipNotify{
			Gn: common.GossipNotify{
				Reserved: 0,
				DataType: 1337,
			},
			MessageHeader: vtypes.MessageHeader{
				Type: vtypes.GossipNotifyType,
			},
		}
		msg.MessageHeader.RecalcSize(&msg)

		if err = p.sendMsg(&msg); err != nil {
			panic(err)
		}
	}

	// send an announcement
	// TODO fixed start node
	p := t.peers[1]
	msg := vtypes.GossipAnnounce{
		Ga: common.GossipAnnounce{
			TTL:      3,
			Reserved: 0,
			DataType: 1337,
			Data:     []byte{1},
		},
		MessageHeader: vtypes.MessageHeader{
			Type: vtypes.GossipAnnounceType,
		},
	}
	msg.MessageHeader.RecalcSize(&msg)
	if err = p.sendMsg(&msg); err != nil {
		panic(err)
	}

	time.Sleep(60 * time.Second)

	t.Teardown()

	if err = t.ProcessLogs(); err != nil {
		panic(err)
	}

}
