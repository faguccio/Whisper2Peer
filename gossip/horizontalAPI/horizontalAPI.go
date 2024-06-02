package horizontalapi

import (
	"context"
	"fmt"
	hzTypes "gossip/horizontalAPI/types"
	"log/slog"
	"net"
	"sync"

	"capnproto.org/go/capnp/v3"
)

//go:generate capnp compile -I $HOME/programme/go-capnp/std -ogo:./ types/message.capnp types/push.capnp

// interfaces to implement union-like behavior
// unions directly are sadly not provided in golang, see: https://go.dev/doc/faq#variant_types
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
	TTL uint16
	// TODO use same type as in verticalAPI
	GossipType uint16
	MessageID  uint16
	Payload    []byte
}

// mark this type as being sendable via FromHz channels
func (p Push) canFromHz() {}

// mark this type as being sendable via ToHz channels
func (p Push) canToHz() {}

// This struct is a collection of some information about a new incoming
// connection.
//
// It includes the remote address and a channel which can be used to write on
// that connection.
//
// These structs are passed by the [HorizontalApi.Listen] function on the
// proviced channel to signal that a new connection was established.
type NewConn struct {
	ToHz chan<- ToHz
	Addr string
}

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
	// store the listener so that it can be closed in the end
	ln net.Listener
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
		ln:         nil,
		conns:      make(map[net.Conn]struct{}, 0),
		fromHzChan: fromHz,
		log:        log,
	}
}

// Listen on the specified address for incoming horizontal api connections.
//
// This function spawns a new goroutine accepting new connections and
// terminates afterwards.
//
// On the channel passed as seccond argument, the horizontalApi advertises new
// incoming connections
func (hz *HorizontalApi) Listen(addr string, newConn chan<- NewConn) error {
	var err error
	hz.ln, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen to port for vertical API: %w", err)
	}

	// accept all connections on this port
	hz.wg.Add(1)
	go func() {
		defer hz.wg.Done()
		for {
			conn, err := hz.ln.Accept()
			if err != nil {
				// check if shall terminate
				select {
				case <-hz.ctx.Done():
					return
				default:
					hz.log.Error("Accept for vertical API failed", "err", err)
				}
				continue
			}

			toHz := make(chan ToHz)
			newConn <- NewConn{
				ToHz: toHz,
				Addr: conn.RemoteAddr().String(),
			}

			hz.conns[conn] = struct{}{}
			hz.log.Info("Incoming connection from", "addr", conn.RemoteAddr().String())

			hz.wg.Add(2)
			go hz.handleConnection(conn, toHz)
			go hz.writeToConnection(conn, toHz)
		}
	}()
	return nil
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
		hz.log.Info("Added connection to", "addrArg", a, "addr", conn.RemoteAddr().String())

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

	// if this read routine terminates, make sure the connection is cleaned up
	// properly
	// avoid double Close when the horizontal api is being closed
	defer delete(hz.conns, conn)
	// close the > hz channel to signal the connection is closed
	// also this causes the write routine to terminate
	defer close(toHz)
	// close the connection
	defer conn.Close()

	// one global decoder suffices
	decoder := capnp.NewDecoder(conn)
	// the following loop uses goto continue_read instead of continue so that
	// some cleanup can be done before actually continuing
	for {
		// wait for new message
		cmsg, err := decoder.Decode()
		if err != nil {
			// error might be because the connection has closed -> check if should terminate
			select {
			case <-hz.ctx.Done():
				break
			default:
			}
			hz.log.Error("decoding the message failed", "err", err)
			goto continue_read
		}
		// scoping needed for goto continue_read to ensure p or push isn't used
		// after goto
		{
			// read the actual message
			msg, err := hzTypes.ReadRootMessage(cmsg)
			if err != nil {
				hz.log.Error("read the message failed", "err", err)
				goto continue_read
			}
			// check of which type the message is
			// using msg.Body().Which() might be more efficient...?
			switch {
			case msg.Body().HasPush():
				// retrieve the push message
				push, err := msg.Body().Push()
				if err != nil {
					hz.log.Error("read the push message failed", "err", err)
					goto continue_read
				}
				// copy the capnproto push message to an internal
				// representation to make the handling in other packages easier
				// retrieving scalar values can be done without the possibility
				// of an error
				p := Push{
					TTL:        push.Ttl(),
					GossipType: push.GossipType(),
					MessageID:  push.MessageID(),
				}
				// payload is no scalar type -> retrival might error
				p.Payload, err = push.Payload()
				if err != nil {
					hz.log.Error("obtaining the payload failed", "err", err)
					goto continue_read
				}
				// send the push message to the channel
				hz.fromHzChan <- p
			// TODO(horizontal types) add new ones here
			default:
				hz.log.Error("no valid message was sent", "type was", msg.Body().Which().String())
				goto continue_read
			}
		}
	continue_read:
		// reset the arena to free memory used by the last decoded message
		cmsg.Release()
	}
}

// Write messages to the connection
//
// Writes all messages sent to he toHz channel to the connection (via capnproto)
func (hz *HorizontalApi) writeToConnection(conn net.Conn, toHz <-chan ToHz) {
	defer hz.wg.Done()

	// one global encoder and arena suffice
	encoder := capnp.NewEncoder(conn)
	arena := capnp.SingleSegment(nil)
	// the following loop uses goto continue_write instead of continue so that
	// some cleanup can be done before actually continuing
	for rmsg := range toHz {
		// create a new capnproto message
		cmsg, seg, err := capnp.NewMessage(arena)
		if err != nil {
			hz.log.Error("creating new message failed", "err", err)
			goto continue_write
		}
		// scoping needed for goto continue_read to ensure hm isn't used
		// after goto
		{
			// create the actual message
			msg, err := hzTypes.NewRootMessage(seg)
			if err != nil {
				hz.log.Error("creating new sending message failed", "err", err)
				goto continue_write
			}
			// check of which type the remote message actually is
			switch rmsg := rmsg.(type) {
			case Push:
				// create the push message
				push, err := hzTypes.NewPushMsg(seg)
				if err != nil {
					hz.log.Error("creating new sending message failed", "err", err)
					goto continue_write
				}
				// populate the push message
				// setting scalar value cannot error
				push.SetTtl(rmsg.TTL)
				push.SetGossipType(rmsg.GossipType)
				push.SetMessageID(rmsg.MessageID)
				// payload is no scalar type -> setting might error
				if err := push.SetPayload(rmsg.Payload); err != nil {
					hz.log.Error("setting the payload for the push message failed", "err", err)
					goto continue_write
				}
				// combine push and the message
				if err := msg.Body().SetPush(push); err != nil {
					hz.log.Error("setting sending message to push failed", "err", err)
					goto continue_write
				}
				// TODO(horizontal types) add new ones here
			}
		}
		// sent the message on the channel
		if err := encoder.Encode(cmsg); err != nil {
			// might error because the connection has closed -> check if should
			// terminate
			select {
			case <-hz.ctx.Done():
				break
			default:
			}
			hz.log.Error("encoding the message failed", "err", err)
			goto continue_write
		}
	continue_write:
		// reset the arena to free memory used by the last encoded message
		cmsg.Release()
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

	// close the listener
	hz.ln.Close()

	for c := range hz.conns {
		// interrupt read of the connection routine
		if e := c.Close(); e != nil {
			err = e
		}
	}

	hz.wg.Wait()
	return err
}
