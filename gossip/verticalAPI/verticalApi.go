package verticalapi

import (
	"context"
	"errors"
	"fmt"
	"gossip/common"
	vertTypes "gossip/verticalAPI/types"
	"io"
	"log/slog"
	"net"
	"slices"
	"sync"
)

// This struct represents the vertical api and is the main interface to/from
// the vertical api.
//
// The struct contains various internal fields, thus it should only be created
// by using the [NewVerticalApi] function!
//
// To close and cleanup the VerticalApi, the [VerticalApi.Close] method shall be called
// exactly once.
type VerticalApi struct {
	// internally uses a context to signal when the goroutines shall terminate
	cancel context.CancelFunc
	// internally uses a context to signal when the goroutines shall terminate
	ctx context.Context
	// store the listener so that it can be closed in the end
	ln net.Listener
	// store all open connections so that they can be closed in the end
	conns map[net.Conn]struct{}
	// collection of channels for the backchannel to the main package
	vertToMainChan chan<- common.FromVert
	// logging for this module
	log *slog.Logger
	// waitgroup to wait for all goroutines to terminate in the end
	wg sync.WaitGroup
}

// Use this function to instanciate the vertical api
//
// The vertToMainChans serve as backchannel to the main package. Depending on
// what message type was received, the message struct will be sent on the
// respective channel.
//
// The methods of this module all will use the logger passed here. You can use
// the [pkg/log/slog.With] function or [pkg/log/slog.Logger.With] on a
// slog-logger to set a field for all logged entries (like "module"="vertAPI").
func NewVerticalApi(log *slog.Logger, vertToMainChan chan<- common.FromVert) *VerticalApi {
	// context is only used internally -> no need to pass it to the constructor
	ctx, cancel := context.WithCancel(context.Background())
	return &VerticalApi{
		cancel:         cancel,
		ctx:            ctx,
		ln:             nil,
		conns:          make(map[net.Conn]struct{}, 0),
		vertToMainChan: vertToMainChan,
		log:            log.With("module", "vertAPI"),
	}
}

// Listen on the specified address for incoming vertical api connections.
//
// This function spawns a new goroutine accepting new connections and
// terminates afterwards.
func (v *VerticalApi) Listen(addr string) error {
	var err error
	v.ln, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen to port for vertical API: %w", err)
	}

	// accept all connections on this port
	v.wg.Add(1)
	go func() {
		defer v.wg.Done()
		for {
			conn, err := v.ln.Accept()
			if err != nil {
				// check if shall terminate
				select {
				case <-v.ctx.Done():
					return
				default:
					v.log.Error("Accept for vertical API failed", "err", err)
				}
				continue
			}

			v.conns[conn] = struct{}{}

			mainToVert := make(chan common.ToVert)
			regMod := common.RegisteredModule{
				MainToVert: mainToVert,
			}

			v.wg.Add(2)
			go v.handleConnection(conn, common.Conn[common.RegisteredModule]{Data: regMod, Id: common.ConnectionId(conn.RemoteAddr().String())})
			go v.writeToConnection(conn, common.Conn[<-chan common.ToVert]{Data: mainToVert, Id: common.ConnectionId(conn.RemoteAddr().String())})
		}
	}()
	return nil
}

// Handle an incoming connection -- read
//
// Parses the message header and body. Then sends the message via the
// respective channel to the main package.
func (v *VerticalApi) handleConnection(conn net.Conn, regMod common.Conn[common.RegisteredModule]) {
	defer v.wg.Done()
	var err error
	var nRead int

	// if this read routine terminates, make sure the connection is cleaned up
	// properly
	// avoid double Close when the vertical api is being closed
	defer delete(v.conns, conn)
	// close the main > vert channel to signal the connection is closed
	// also this causes the write routine to terminate
	defer close(regMod.Data.MainToVert)
	// signal to main that this vert module terminated -> needs to unregister it
	defer func(regMod *common.Conn[common.RegisteredModule]) {
		v.vertToMainChan <- common.GossipUnRegister(regMod.Id)
	}(&regMod)
	// close the connection
	defer conn.Close()

	var msgHdr vertTypes.MessageHeader
	buf := make([]byte, msgHdr.CalcSize())

	for {
		// set the size of the slice to make sure not to read too much
		buf = buf[0:msgHdr.CalcSize()]
		// read the message header
		nRead, err = io.ReadFull(conn, buf)
		if err != nil {
			// check if shall terminate
			select {
			case <-v.ctx.Done():
				return
			default:
			}
			if errors.Is(err, io.EOF) {
				return
			}
			v.log.Error("Read on vertical API failed", "err", err)
			continue
		}
		// it is save that the complete buffer is completely populated at this point

		// parse the message header
		_, err = msgHdr.Unmarshal(buf)
		if err != nil {
			v.log.Warn("Invalid header read", "err", err)
			continue
		}

		// allocate space for the message body
		buf = slices.Grow(buf, int(msgHdr.Size)-nRead)
		buf = buf[0:int(msgHdr.Size)]

		// read the message body
		_, err = io.ReadFull(conn, buf[nRead:])
		if err != nil {
			// check if shall terminate
			select {
			case <-v.ctx.Done():
				return
			default:
			}
			v.log.Error("Read on vertical API failed", "err", err)
			continue
		}
		// it is save that the complete buffer is completely populated at this point

		v.log.Debug("received buffer", "buf", buf)

		switch msgHdr.Type {

		case vertTypes.GossipAnnounceType:
			var ga vertTypes.GossipAnnounce
			ga.MessageHeader = msgHdr
			_, err = ga.Unmarshal(buf)
			if err != nil {
				v.log.Warn("Invalid GossipAnnounce read", "err", err)
				continue
			} else {
				v.vertToMainChan <- ga.Ga
			}

		case vertTypes.GossipNotifyType:
			var gn vertTypes.GossipNotify
			gn.MessageHeader = msgHdr
			_, err = gn.Unmarshal(buf)
			if err != nil {
				v.log.Warn("Invalid GossipNotify read", "err", err)
				continue
			} else {
				v.vertToMainChan <- common.GossipRegister{
					Data:   gn.Gn,
					Module: &regMod,
				}
			}

		case vertTypes.GossipValidationType:
			var gv vertTypes.GossipValidation
			gv.MessageHeader = msgHdr
			_, err = gv.Unmarshal(buf)
			if err != nil {
				v.log.Warn("Invalid GossipValidation read", "err", err)
				continue
			} else {
				v.vertToMainChan <- gv.Gv
			}

		default:
			v.log.Warn("vertical API received an unexpected message type", "type", msgHdr.Type)
		}
	}
}

// Write messages to the connection
//
// Writes all messages sent to he mainToVert channel to the connection
func (v *VerticalApi) writeToConnection(conn net.Conn, cData common.Conn[<-chan common.ToVert]) {
	defer v.wg.Done()
	var err error
	var nWritten int
	buf := make([]byte, 0, 4096)

	for msg := range cData.Data {
		switch msg := msg.(type) {
		case common.GossipNotification:
			vmsg := vertTypes.GossipNotification{
				Gn: msg,
				MessageHeader: vertTypes.MessageHeader{
					Type: vertTypes.GossipNotificationType,
				},
			}
			vmsg.MessageHeader.RecalcSize(&vmsg)
			buf, err = vmsg.Marshal(buf)
			if err != nil {
				v.log.Warn("Failed to marshal GossipNotification", "err", err)
				continue
			}
		}

		nWritten, err = conn.Write(buf)
		if err != nil {
			// check if shall terminate
			select {
			case <-v.ctx.Done():
				return
			default:
			}
			v.log.Warn("Failed to send GossipNotification", "err", err)
			continue
		}
		_ = nWritten
	}
}

// Close the vertical api
//
// Always tries to close the listener and all the connections. If multiple
// fail, this function only returns the last error.
func (v *VerticalApi) Close() error {
	var err error
	// signal to listener and connection goroutines that they should terminate
	v.cancel()
	// interrupt accept of the listener routine
	if e := v.ln.Close(); e != nil {
		err = e
	}
	for c := range v.conns {
		// interrupt read of the connection routine
		if e := c.Close(); e != nil {
			err = e
		}
	}
	v.wg.Wait()
	return err
}
