package testutils

import (
	"errors"
	"gossip/common"
	"gossip/internal/args"
	vtypes "gossip/verticalAPI/types"
	"io"
	"net"
	"slices"
)

// simple interface implemented by all types which allow marshalling to bytes
type marshaler interface {
	Marshal(buf []byte) ([]byte, error)
}

// bookkeeping structure for known/started peers
type peer struct {
	// index of the peer
	idx uint
	// identifier of the peer (most likely the ip address)
	id common.ConnectionId
	// arguments which which the peer was started
	a args.Args
	// open connection of the verticalAPI
	conn net.Conn
	// dialer used when connect to the verticalAPI to be able to set the
	// local IP address
	dialer *net.Dialer
}

// close must be called to cleanup the allocated ressources (includes closing
// the open connection)
func (p *peer) close() {
	p.conn.Close()
}

// open a connection to the verticalAPI of the peer
func (p *peer) connect() error {
	var err error
	p.conn, err = p.dialer.Dial("tcp", p.a.Vert_addr)
	return err
}

func (p *peer) String() string {
	return string(p.id)
}

// Send a message to the verticalAPI of that peer
func (p *peer) SendMsg(v marshaler) error {
	if p.conn == nil {
		return errors.New("vertAPI connection not yet opened")
	}

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

// receives any message sent to the vertAPI and sends for all notification a
// validation(valid=true) message
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
		if err = p.SendMsg(&msg); err != nil {
			print("markAllValid: ", err.Error(), "\n")
			return
		}
	}
}
