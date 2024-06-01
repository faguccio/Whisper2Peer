package horizontalapi

import (
	"context"
	"errors"
	hzTypes "gossip/horizontalAPI/types"
	"log/slog"
	"net"
	"sync"

	"capnproto.org/go/capnp/v3"
)

var (
	// Developing reasons
	Err = errors.New("")
)

//go:generate capnp compile -I $HOME/programme/go-capnp/std -ogo:./ types/core.capnp

// interfaces to implement union-like behavior
// unions diectly are sadly not provided in golang, see: https://go.dev/doc/faq#variant_types
// go-sumtype:decl FromHz
type FromHz interface {
	// add a function to the interface to avoid that arbitrary types can be
	// passed (accidentally) as FromHz
	canFromHz()
}

// go-sumtype:decl ToHz
type ToHz interface {
	// add a function to the interface to avoid that arbitrary types can be
	// passed (accidentally) as ToHz
	canToHz()
}

type Push struct {
	HzType uint16
	TTL    uint16
	// TODO use same type as in verticalAPI
	GossipType uint16
	MessageID  uint16
	Payload    []byte
}

// mark this type as being sendable via FromHz channels
func (p Push) canFromHz() {}

// mark this type as being sendable via ToHz channels
func (p Push) canToHz() {}

// This struct represents the horizontal api and is the main interface to/from
// the horizontal api.
//
// The struct contains various internal fields, thus it should only be created
// by using the [NewHorizontalApi] function!
//
// To close and cleanup the HorizontalApi, the [HorizontalApi.Close] method shall be called
// exactly once.
type HorizontalApi struct {
	// internally uses a context to signal when the goroutines shall terminate
	cancel context.CancelFunc
	// internally uses a context to signal when the goroutines shall terminate
	ctx context.Context
	// store all open connections so that they can be closed in the end
	conns map[net.Conn]struct{}
	// channel on which data which was received is being passed
	fromHzChan chan<- FromHz
	// logging for this module
	log *slog.Logger
	// waitgroup to wait for all goroutines to terminate in the end
	wg sync.WaitGroup
}

// Use this function to instantiate the horizontal api
//
// The fromHz serve as backchannel. The horizontalapi will send all messages
// read to this channel.
//
// The methods of this module all will use the logger passed here. You can use
// the [pkg/log/slog.With] function or [pkg/log/slog.Logger.With] on a
// slog-logger to set a field for all logged entries (like "module"="hzAPI").
func NewHorizontalApi(log *slog.Logger, fromHz chan<- FromHz) *HorizontalApi {
	// context is only used internally -> no need to pass it to the constructor
	ctx, cancel := context.WithCancel(context.Background())
	return &HorizontalApi{
		cancel:     cancel,
		ctx:        ctx,
		conns:      make(map[net.Conn]struct{}, 0),
		fromHzChan: fromHz,
		log:        log,
	}
}

// Use this function to add neighbor connections to the horizontalApi
//
// Returns a slice of channels (same ordering like the address-slice parameter)
// on which the horizontalApi will send incoming packets on the connection to
// the respective neighbor.
func (hz *HorizontalApi) AddNeighbors(addrs ...string) ([]chan<- ToHz, error) {
	var ret []chan<- ToHz
	for _, a := range addrs {
		conn, err := net.Dial("tcp", a)
		if err != nil {
			return nil, err
		}

		hz.conns[conn] = struct{}{}

		toHz := make(chan ToHz)
		ret = append(ret, toHz)

		hz.wg.Add(2)
		go hz.handleConnection(conn, toHz)
		go hz.writeToConnection(conn, toHz)
	}
	return ret, nil
}

// Handle an incoming connection -- read
//
// Parses incoming messages with the help of capnproto and converts them into
// [FromHz] structs. Currently these are: [Push]. These are then sent on the
// hz.fromHzChan channel.
//
// only needs toHz so that it can close it when the connection is closed
func (hz *HorizontalApi) handleConnection(conn net.Conn, toHz chan<- ToHz) {
	defer hz.wg.Done()
	var err error
	_ = err

	// if this read routine terminates, make sure the connection is cleaned up
	// properly
	// avoid double Close when the vertical api is being closed
	defer delete(hz.conns, conn)
	// close the > hz channel to signal the connection is closed
	// also this causes the write routine to terminate
	defer close(toHz)
	// close the connection
	defer conn.Close()

	decoder := capnp.NewDecoder(conn)
	for {
		msg, err := decoder.Decode()
		if err != nil {
			select {
			case <- hz.ctx.Done():
				break
			default:
			}
			hz.log.Error("decoding the message failed", "err", err)
			goto continue_read
		}
		// scoping needed for goto continue_read to ensure p or push isn't used
		// after goto
		{
			push, err := hzTypes.ReadRootPushMsg(msg)
			if err != nil {
				hz.log.Error("read the push message failed", "err", err)
				goto continue_read
			}
			p := Push{
				HzType:     push.HorizontalType(),
				TTL:        push.Ttl(),
				GossipType: push.GossipType(),
				MessageID:  push.MessageID(),
			}
			p.Payload, err = push.Payload()
			if err != nil {
				hz.log.Error("obtaining the payload failed", "err", err)
				goto continue_read
			}
			hz.fromHzChan <- p
		}
		continue_read:
	}
}

// Write messages to the connection
//
// Writes all messages sent to he toHz channel to the connection (via capnproto)
func (hz *HorizontalApi) writeToConnection(conn net.Conn, toHz <-chan ToHz) {
	defer hz.wg.Done()
	var err error
	_ = err

	encoder := capnp.NewEncoder(conn)
	arena := capnp.SingleSegment(nil)
	for msg := range toHz {
		m, seg, err := capnp.NewMessage(arena)
		if err != nil {
			hz.log.Error("creating new message failed", "err", err)
			goto continue_write
		}
		switch msg := msg.(type) {
		case Push:
			push, err := hzTypes.NewRootPushMsg(seg)
			if err != nil {
				hz.log.Error("creating new push message failed", "err", err)
				goto continue_write
			}
			push.SetHorizontalType(msg.HzType)
			push.SetTtl(msg.TTL)
			push.SetGossipType(msg.GossipType)
			push.SetMessageID(msg.MessageID)
			if err := push.SetPayload(msg.Payload); err != nil {
				hz.log.Error("setting the payload for the push message failed", "err", err)
				goto continue_write
			}
		}
		if err := encoder.Encode(m); err != nil {
			select {
			case <- hz.ctx.Done():
				break
			default:
			}
			hz.log.Error("encoding the message failed", "err", err)
			goto continue_write
		}
		continue_write:
		// reset the arena to free memory used by the last encoded message
		m.Release()
	}
}

// Close the horizontal api
//
// Always tries to close all the connections. If multiple
// fail, this function only returns the last error.
func (hz *HorizontalApi) Close() error {
	var err error

	// signal to connection goroutines that they should terminate
	hz.cancel()

	for c := range hz.conns {
		// interrupt read of the connection routine
		if e := c.Close(); e != nil {
			err = e
		}
	}

	hz.wg.Wait()
	return err
}
