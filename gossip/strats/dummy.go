package strats

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"gossip/common"
	horizontalapi "gossip/horizontalAPI"
	ringbuffer "gossip/internal/ringbuffer"
	pow "gossip/pow"
	"math"
	"reflect"
	"slices"

	"crypto/rand"
	"math/big"
	mrand "math/rand"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

var (
	ErrNoMessageFound error = errors.New("No message found when extracting")
)

var (
	// TODO generate
	secretKey []byte = []byte("secrets_secrets_secrets_secrets_")
	POW_TIME         = 5 * time.Second
)

// Some messages might use a counter of how many peers the message was relayed to
type storedMessage struct {
	counter int
	message horizontalapi.Push
}

type gossipConnection struct {
	connection horizontalapi.Conn[chan<- horizontalapi.ToHz]
	timestamp  time.Time
	flag       bool
}

type connCookie struct {
	chall     uint64
	timestamp time.Time
	dest      horizontalapi.ConnectionId
}

// This struct contains all the fields used by the Dummy Strategy.
type dummyStrat struct {
	// The base strategy, which takes care of instantiating the HZ API and contains many common fields
	rootStrat Strategy
	// Channel where peer messages arrive
	fromHz <-chan horizontalapi.FromHz
	// Channel were new connection are notified
	hzConnection <-chan horizontalapi.NewConn
	// Array of peers channels, where messages can be sent
	openConnections []gossipConnection
	// Collection of messages received from a peer which needs to be validated through the vertical api
	invalidMessages *ringbuffer.Ringbuffer[*storedMessage]
	// Collection of messages which need to be relayed to other peers.
	validMessages *ringbuffer.Ringbuffer[*storedMessage]
	// Collection of messages already relayed to other peers
	sentMessages *ringbuffer.Ringbuffer[*storedMessage]
}

// Function to instantiate a new DummyStrategy.
//
// strategy must be the baseStrategy. openConnection a list of ToHz channels, one for each peer
func NewDummy(strategy Strategy, fromHz <-chan horizontalapi.FromHz, hzConnection <-chan horizontalapi.NewConn, openConnections []gossipConnection) dummyStrat {

	return dummyStrat{
		rootStrat:       strategy,
		fromHz:          fromHz,
		hzConnection:    hzConnection,
		openConnections: openConnections,
		invalidMessages: ringbuffer.NewRingbuffer[*storedMessage](strategy.stratArgs.Cache_size),
		validMessages:   ringbuffer.NewRingbuffer[*storedMessage](strategy.stratArgs.Cache_size),
		sentMessages:    ringbuffer.NewRingbuffer[*storedMessage](strategy.stratArgs.Cache_size),
	}
}

// Listen for messages incoming on either StrategyChannels (from the base strategy, such as the
// vertical API) or horizontal API
//
// This function spawn a new goroutine. Incoming messages will be processed by the Dummy Strategy.
func (dummy *dummyStrat) Listen() {
	go func() {
		// A repeating signal to trigger a recurrent behavior.
		ticker := time.NewTicker(time.Duration(dummy.rootStrat.stratArgs.GossipTimer) * time.Second)

		// Keep listening on all channels
		for {
			select {
			// Message received from a peer.
			case x := <-dummy.fromHz:
				switch msg := x.(type) {
				case horizontalapi.Unregister:
					// remove the connection with the respective ID from the list of connections
					dummy.removeConnection(horizontalapi.ConnectionId(msg))

				case horizontalapi.Push:
					if !dummy.checkConnectionValidity(msg.Id) {
						continue
					}

					notification := convertPushToNotification(msg)
					_, err1 := findFirstMessage(dummy.sentMessages, msg.MessageID)
					_, err2 := findFirstMessage(dummy.validMessages, msg.MessageID)
					_, err3 := findFirstMessage(dummy.invalidMessages, msg.MessageID)

					// If the message was not already received, move it to the invalidMessages
					// and send a notification to vert API
					if err1 != nil && err2 != nil && err3 != nil {
						dummy.invalidMessages.Insert(&storedMessage{0, msg})
						dummy.rootStrat.log.Log(context.Background(), common.LevelTest, "received", "msgId", notification.MessageId, "msgType", notification.DataType)
						dummy.rootStrat.strategyChannels.FromStrat <- notification
						dummy.rootStrat.log.Debug("HZ Message received:", "type", reflect.TypeOf(msg), "Message", msg)
					}

				case horizontalapi.ConnReq:

					n, err := rand.Int(rand.Reader, big.NewInt(int64(math.MaxInt64)))
					if err != nil {
						panic(err)
					}

					chall := n.Uint64()

					cookie := connCookie{
						chall:     chall,
						timestamp: time.Now(),
						dest:      msg.Id,
					}

					// Create ConnChall message
					m := horizontalapi.ConnChall{
						Chall:  chall,
						Cookie: cookie.createCookie(),
					}

					// Send message to correct peer
					peer := findConnection(dummy.openConnections, msg.Id)
					if peer == nil {
						dummy.rootStrat.log.Error("No open connection found for challReq", "conn Id", msg.Id)
						continue
					}
					peer.connection.Data <- m

				case horizontalapi.ConnPoW:
					mypow := powMarsh{PowNonce: msg.PowNonce, Cookie: msg.Cookie}

					flag := pow.CheckProofOfWork(func(digest []byte) bool {
						return pow.First8bits0(digest)
					}, &mypow)

					// The following code is commented as for now HZ messages do not include the ChaCha20 nonce
					// which could also be the same as the chall argument, but has to be sent as plaintext
					// var cookie connCookie

					// cookie.readCookie(msg.Cookie, chall)

					// // check dest is valid
					// if cookie.dest != msg.Id {
					// 	dummy.rootStrat.log.Debug("Mismatched connectionId between received connPow and sender", "expected conn Id", cookie.dest, "actual Id", msg.Id)
					// }

					// diff := time.Now().Sub(cookie.timestamp)
					// if diff > POW_TIME {
					// 	dummy.rootStrat.log.Debug("POW for accepting connection was given not within the time limit", "expected conn Id", cookie.dest, "actual Id", msg.Id)
					// }

					// check if time taken for giving pow is within the limits
					peer := findConnection(dummy.openConnections, msg.Id)
					if peer == nil {
						dummy.rootStrat.log.Error("No open connection found for challPoW", "conn Id", msg.Id)
						continue
					}

					// Validate connection or remove it
					if flag {
						peer.flag = true
						peer.timestamp = time.Now()
					} else {
						dummy.removeConnection(msg.Id)
					}
				}

				// Accept any connection and flag it as invalid.
			case newPeer := <-dummy.hzConnection:
				conn := gossipConnection{
					connection: horizontalapi.Conn[chan<- horizontalapi.ToHz](newPeer),
					timestamp:  time.Now(),
					flag:       false,
				}
				dummy.openConnections = append(dummy.openConnections, conn)

				// Message from the vertical API
			case x := <-dummy.rootStrat.strategyChannels.ToStrat:
				switch x := x.(type) {
				case common.GossipAnnounce:
					pushMsg := convertAnnounceToPush(x)
					dummy.rootStrat.log.Log(context.Background(), common.LevelTest, "announce", "msgId", pushMsg.MessageID, "msgType", pushMsg.GossipType)
					// We consider Announce messages automatically valid
					dummy.validMessages.Insert(&storedMessage{0, pushMsg})
				case common.GossipValidation:
					msg, err := findFirstMessage(dummy.invalidMessages, x.MessageId)
					dummy.invalidMessages.Remove(msg)

					if err != nil {
						dummy.rootStrat.log.Warn("Tried to validate a message which did not exists", "Message ID", x.MessageId)
						break
					}

					if x.Valid {
						if msg.message.TTL == 1 {
							dummy.sentMessages.Insert(msg)
						} else {
							msg.message.TTL = max(msg.message.TTL-1, 0)

							dummy.validMessages.Insert(msg)
						}
					}
				}

				// Recurrent timer signal
			case <-ticker.C:
				// A random peer is selected and we relay all messages to that peer.
				amount := min(len(dummy.openConnections), int(dummy.rootStrat.stratArgs.Degree))
				perm := mrand.Perm(amount)

				//idx := mrand.Intn(len(dummy.openConnections))
				dummy.validMessages.Do((func(msg *storedMessage) {
					for i := 0; i < amount; i++ {
						idx := perm[i]
						if !dummy.openConnections[idx].flag {
							continue
						}
						//dummy.rootStrat.log.Debug("DST conn", "conn", dummy.openConnections[idx])
						dummy.openConnections[idx].connection.Data <- msg.message
						dummy.rootStrat.log.Debug("HZ Message sent:", "dst", dummy.openConnections[idx].connection.Id, "Message", msg)
						msg.counter++

						// If message was sent to args.Degree neighboughrs delete it from the set of messages
						if msg.counter >= int(dummy.rootStrat.stratArgs.Degree) {
							dummy.validMessages.Remove(msg)
							dummy.sentMessages.Insert(msg)
						}
					}
				}))
			case <-dummy.rootStrat.ctx.Done():
				// should terminate
				return
			}
		}
	}()
}

// Return the corresponding connection given an Id and a list of connection
func findConnection(connections []gossipConnection, id horizontalapi.ConnectionId) *gossipConnection {
	for i, conn := range connections {
		if conn.connection.Id == id {
			return &connections[i]
		}
	}
	return nil
}

func (x *connCookie) Marshal() []byte {
	timestamp := x.timestamp.UnixNano()
	buf := make([]byte, binary.Size(x.chall)+binary.Size(timestamp)+len(x.dest))

	idx := 0

	binary.BigEndian.PutUint64(buf[idx:], x.chall)
	idx += binary.Size(x.chall)

	binary.BigEndian.PutUint64(buf[idx:], uint64(timestamp))
	idx += binary.Size(timestamp)

	copy(buf[idx:], x.dest[:])
	idx += len(x.dest)

	return buf
}

func (x *connCookie) Unmarshal(buf []byte) {
	idx := 0

	cookieChall := binary.BigEndian.Uint64(buf[idx:binary.Size(x.chall)])
	idx += binary.Size(x.chall)

	t := binary.BigEndian.Uint64(buf[idx : idx+8])
	timestamp := time.Unix(0, int64(t))
	idx += 8

	dest := make([]byte, len(buf[idx:]))

	copy(dest, buf[idx:])
	x.chall = cookieChall
	x.timestamp = timestamp
	x.dest = horizontalapi.ConnectionId(string(dest))
}

func (x *connCookie) createCookie() []byte {
	// Create chacha20 cipher
	aead, err := chacha20poly1305.New(secretKey)
	if err != nil {
		panic(err)
	}

	payload := x.Marshal()

	// TODO: I am using as the nonce the challange for the PoW (appendend with 0s, ugh).
	// I think if we want to stick with ChaCha20 we need to change the protocol messages to include the
	// ChaCha Nonce, otherwise we need to store state? 64 random bits could also be enough for uniqueness
	ciphertext := aead.Seal(nil, slices.Concat(payload[0:8], []byte{0, 0, 0, 0}), payload, nil)
	return ciphertext
}

func (x *connCookie) readCookie(cookie []byte, chall uint64) {
	// Create chacha20 cipher
	aead, err := chacha20poly1305.New(secretKey)
	if err != nil {
		panic(err)
	}

	bchall := make([]byte, 8)
	binary.BigEndian.PutUint64(bchall, chall)

	// Decrypt the cookie
	//plaintext := make([]byte, len(cookie))
	plaintext, err := aead.Open(nil, slices.Concat(bchall, []byte{0, 0, 0, 0}), cookie, nil)
	if err != nil {
		fmt.Println("Error while decrypting the cookie", "err", err)
		return
	}

	x.Unmarshal(plaintext)
}

// Go throught a ringbuffer of messages and return the one with a matching ID, error if none is found
// This function is needed just for a closure
func findFirstMessage(ring *ringbuffer.Ringbuffer[*storedMessage], messageId uint16) (*storedMessage, error) {
	res, err := ring.FindFirst(func(p *storedMessage) bool {
		return p.message.MessageID == messageId
	})
	return res, err
}

// Convert a Gossip Announce message to a Horizontal Push message. Message ID is chosen at random.
func convertAnnounceToPush(msg common.GossipAnnounce) horizontalapi.Push {
	// Hardcoded 65535 as the max value of a uint16
	id, _ := rand.Int(rand.Reader, big.NewInt(65535))

	pushMsg := horizontalapi.Push{
		TTL:        msg.TTL,
		GossipType: msg.DataType,
		MessageID:  uint16(id.Int64()),
		Payload:    msg.Data,
	}

	return pushMsg
}

// Convert a Horizontal Push message to a Gossip Notification one
func convertPushToNotification(pushMsg horizontalapi.Push) common.GossipNotification {
	notification := common.GossipNotification{
		MessageId: pushMsg.MessageID,
		DataType:  common.GossipType(pushMsg.GossipType),
		Data:      pushMsg.Payload,
	}
	return notification
}

func (dummy *dummyStrat) removeConnection(id horizontalapi.ConnectionId) {
	for i, conn := range dummy.openConnections {
		if conn.connection.Id == id {
			dummy.openConnections[i] = dummy.openConnections[len(dummy.openConnections)-1]
			dummy.openConnections = dummy.openConnections[:len(dummy.openConnections)-1]
			break
		}
	}
}

func (dummy *dummyStrat) checkConnectionValidity(id horizontalapi.ConnectionId) bool {
	conn := findConnection(dummy.openConnections, id)
	if conn == nil {
		return false
	}
	return conn.flag
}

// Close the root strategy
func (dummy *dummyStrat) Close() {
	dummy.rootStrat.Close()
}
