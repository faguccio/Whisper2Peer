package horizontalapi

import (
	"context"
	"errors"
	hzTypes "gossip/horizontalAPI/types"
	"log/slog"
	"net"
	"sync"
)

var (
	// Developing reasons
	Err = errors.New("")
)

//go:generate capnp compile -I $HOME/programme/go-capnp/std -ogo:./ types/core.capnp

type FromHoriz interface{}
type ToHoriz interface{}

// This struct represents the horizontal api and is the main interface to/from
// the vertical api.
type HorizontalApi struct {
	// internally uses a context to signal when the goroutines shall terminate
	cancel context.CancelFunc
	// internally uses a context to signal when the goroutines shall terminate
	ctx context.Context
	// store all open connections so that they can be closed in the end
	conns map[net.Conn]struct{}
	// channel on which data which was received is being passed
	fromHorizChan chan<- FromHoriz
	// logging for this module
	log *slog.Logger
	// waitgroup to wait for all goroutines to terminate in the end
	wg sync.WaitGroup
}

// Use this function to instantiate the horizontal api
func NewHorizontalApi(log *slog.Logger, fromHoriz chan<- FromHoriz) *HorizontalApi {
	// context is only used internally -> no need to pass it to the constructor
	ctx, cancel := context.WithCancel(context.Background())
	return &HorizontalApi{
		cancel:        cancel,
		ctx:           ctx,
		conns:         make(map[net.Conn]struct{}, 0),
		fromHorizChan: fromHoriz,
		log:           log,
	}
}

func (hz *HorizontalApi) AddNeighbors(addrs ...string) ([]chan<-ToHoriz, error) {
	var ret []chan<-ToHoriz
	for _,a := range addrs {
		conn, err := net.Dial("tcp", a)
		if err != nil {
			return nil, err
		}

		hz.conns[conn] = struct{}{}

		toHoriz := make(chan ToHoriz)
		ret = append(ret, toHoriz)

		hz.wg.Add(2)
		go hz.handleConnection(conn, toHoriz)
		go hz.writeToConnection(conn, toHoriz)
	}
	return ret, nil
}

// only needs toHoriz so that it can close it when the connection is closed
func (hz *HorizontalApi) handleConnection(conn net.Conn, toHoriz chan<- ToHoriz) {
	defer hz.wg.Done()
	var err error
	_ = err

	// if this read routine terminates, make sure the connection is cleaned up
	// properly
	// avoid double Close when the vertical api is being closed
	defer delete(hz.conns, conn)
	// close the > hz channel to signal the connection is closed
	// also this causes the write routine to terminate
	defer close(toHoriz)
	// close the connection
	defer conn.Close()

	for {
		var msg hzTypes.PushMsg
		// TODO capnp proto stuff here for reading from connection
		hz.fromHorizChan <- msg
	}
}

func (hz *HorizontalApi) writeToConnection(conn net.Conn, toHoriz <-chan ToHoriz) {
	defer hz.wg.Done()
	var err error
	_ = err

	for msg := range toHoriz {
		_ = msg
		// TODO capnp proto stuff here for writing to the connection
	}
}

// Empty Close function, clean up connection still has to be done since there are no connection yer :)
func (hz *HorizontalApi) Close() (err error) {
	hz.cancel()
	hz.wg.Wait()
	return nil
}
