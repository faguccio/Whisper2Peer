package main

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
	"os"
	"time"

	"github.com/alexflint/go-arg"
)

// bookkeeping structure for known/started peers
type peer struct {
	idx  uint
	id   common.ConnectionId
	a    args.Args
	conn net.Conn
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
	msg,err := v.Marshal(nil)
	if err != nil {
		return err
	}

	if n,err := p.conn.Write(msg); err != nil {
		return err
	} else if n != len(msg) {
		return errors.New("Message could not be written entirely")
	}
	return err
}

type Marshaler interface {
	Marshal(buf []byte) ([]byte, error)
}

type Tester struct{
	a Args
	g graph
	cancel context.CancelFunc
	peers map[uint]*peer
	// the logger of each peer will indirectly write it's testing events onto this channel
	logChan chan event
	// all events are collected here (mostly because there are no unlimited buffered channels)
	events []event
	closers []io.Closer
}

func NewTester() *Tester {
	return &Tester{
		peers: make(map[uint]*peer),
		logChan: make(chan event, 64),
		events: make([]event, 0),
		closers: make([]io.Closer, 0),
	}
}

// reads all the config stuff
func (t *Tester) Init() error {
	var err error

	arg.MustParse(&t.a)

	// parse graph from provided file
	t.g, err = readGraph(t.a.Fn)
	if err != nil {
		return err
	}

	return nil
}

// starts all the peers etc
func (t *Tester) Startup() error {
	// iterator for ip address
	ip := netip.MustParseAddr("127.0.0.1")

	// make ready for terminating
	var ctx context.Context
	ctx, t.cancel = context.WithCancel(context.Background())

	// goroutine simply copies the events over to the events slice
	// NOTE: when closing the channel will also terminate the goroutine
	go func(logChan <-chan event){
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
			args.Degree = t.a.Degree
		}
		if node.Cache_size != nil {
			args.Cache_size = *node.Cache_size
		} else {
			args.Cache_size = t.a.Cache_size
		}
		if node.GossipTimer != nil {
			args.GossipTimer = *node.GossipTimer
		} else {
			args.GossipTimer = t.a.GossipTimer
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
			idx:  nodeIdx,
			id:   common.ConnectionId(ip.String()),
			a:    args,
			dialer: &net.Dialer{
				LocalAddr: &net.TCPAddr{
					IP:   ip.AsSlice(),
					Port: 0,
				},
			},
		}
		t.peers[nodeIdx] = p

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
		go m.Run(ctx)

		time.Sleep(500 * time.Millisecond)
		ip = ip.Next()
	}

	// connect on the vertival api to all peers
	for _,p := range t.peers {
		err := p.connect()
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Tester) ProcessLogs() error {
	fmt.Println(t.events)
	// TODO
	return nil
}

func (t *Tester) Teardown() {
	for _,p := range t.peers {
		p.close()
	}
	t.cancel()
	close(t.logChan)
	for _,c := range t.closers {
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
	for _,p := range t.peers {
		msg := vtypes.GossipNotify{
			Gn:            common.GossipNotify{
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
	p := t.peers[1]
	msg := vtypes.GossipAnnounce{
		Ga:            common.GossipAnnounce{
			TTL:      2,
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

	time.Sleep(60*time.Second)

	t.Teardown()

	if err = t.ProcessLogs(); err != nil {
		panic(err)
	}

}
