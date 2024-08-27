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
	"reflect"
	"slices"

	"crypto/cipher"
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
	POW_TIME = 5 * time.Second
)

// Some messages might use a counter of how many peers the message was relayed to
type storedMessage struct {
	counter int
	message horizontalapi.Push
}

type connCookie struct {
	chall     []byte
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
	// Array of peers channels, where messages can be sent (stored as slice for easing permutations)
	openConnections []gossipConnection
	// Map of valid connection (for fast access)
	openConnectionMap map[horizontalapi.ConnectionId]gossipConnection
	// Map of invalid connection (for fast access)
	powInProgress map[horizontalapi.ConnectionId]gossipConnection
	// Collection of messages received from a peer which needs to be validated through the vertical api
	invalidMessages *ringbuffer.Ringbuffer[*storedMessage]
	// Collection of messages which need to be relayed to other peers.
	validMessages *ringbuffer.Ringbuffer[*storedMessage]
	// Collection of messages already relayed to other peers
	sentMessages *ringbuffer.Ringbuffer[*storedMessage]
	// ChaCha20 cipher
	cipher cipher.AEAD
}

// Function to instantiate a new DummyStrategy.
//
// strategy must be the baseStrategy. openConnection a list of ToHz channels, one for each peer
func NewDummy(strategy Strategy, fromHz <-chan horizontalapi.FromHz, hzConnection <-chan horizontalapi.NewConn, openConnections []gossipConnection, connectionMap map[horizontalapi.ConnectionId]gossipConnection) dummyStrat {
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		panic(err)
	}

	return dummyStrat{
		rootStrat:         strategy,
		fromHz:            fromHz,
		hzConnection:      hzConnection,
		openConnections:   openConnections,
		openConnectionMap: connectionMap,
		powInProgress:     make(map[horizontalapi.ConnectionId]gossipConnection),
		invalidMessages:   ringbuffer.NewRingbuffer[*storedMessage](strategy.stratArgs.Cache_size),
		validMessages:     ringbuffer.NewRingbuffer[*storedMessage](strategy.stratArgs.Cache_size),
		sentMessages:      ringbuffer.NewRingbuffer[*storedMessage](strategy.stratArgs.Cache_size),
		cipher:            aead,
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
					cookie := NewConnCookie(msg.Id)

					// Create ConnChall message with the encrypted cookie
					m := horizontalapi.ConnChall{
						Cookie: cookie.createCookie(dummy.cipher),
					}

					// Send message to correct peer
					peer := dummy.powInProgress[msg.Id]
					if peer.timestamp.IsZero() {
						dummy.rootStrat.log.Error("No open connection found for challReq", "conn Id", msg.Id)
						continue
					}

					peer.connection.Data <- m

				case horizontalapi.ConnPoW:
					mypow := powMarsh{PowNonce: msg.PowNonce, Cookie: msg.Cookie}

					cookieRead, err := ReadCookie(dummy.cipher, mypow.Cookie)

					if err != nil {
						dummy.rootStrat.log.Debug("Failed to decrypt cookie, dropping connection", "connection ID", msg.Id)
						dummy.removeConnection(msg.Id)
						continue
					}

					// Check proof of work
					powValidity := pow.CheckProofOfWork(func(digest []byte) bool {
						return pow.First8bits0(digest)
					}, &mypow)

					if !powValidity {
						dummy.rootStrat.log.Debug("Invalid pow, dropping connection", "expected conn Id", cookieRead.dest, "actual Id", msg.Id)
						dummy.removeConnection(msg.Id)
						continue
					}

					// check dest is valid
					if cookieRead.dest != msg.Id {
						dummy.rootStrat.log.Debug("Mismatched connectionId between received connPow and sender", "expected conn Id", cookieRead.dest, "actual Id", msg.Id)
						dummy.removeConnection(msg.Id)
						continue
					}

					// check if time taken for giving pow is within the limits
					diff := time.Now().Sub(cookieRead.timestamp)
					if diff > POW_TIME {
						dummy.rootStrat.log.Debug("POW for accepting connection was given not within the time limit", "expected conn Id", cookieRead.dest, "actual Id", msg.Id)
						dummy.removeConnection(msg.Id)
						continue
					}

					peer := dummy.powInProgress[msg.Id]
					if peer.timestamp.IsZero() {
						dummy.rootStrat.log.Error("No open connection found for challPoW", "conn Id", msg.Id)
						continue
					}

					// Validate connection and remove it from inProgress
					peer.timestamp = cookieRead.timestamp
					dummy.openConnectionMap[msg.Id] = peer
					dummy.openConnections = append(dummy.openConnections, peer)
					dummy.removeConnection(msg.Id)

				}

			// Accept any connection and put it in the inProgress slice.
			case newPeer := <-dummy.hzConnection:
				conn := gossipConnection{
					connection: horizontalapi.Conn[chan<- horizontalapi.ToHz](newPeer),
					timestamp:  time.Now(),
				}
				dummy.powInProgress[conn.connection.Id] = conn

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

				// Create a permutation over the length of valid connections
				perm := mrand.Perm(len(dummy.openConnections))
				amount := min(len(dummy.openConnections), int(dummy.rootStrat.stratArgs.Degree))

				dummy.validMessages.Do((func(msg *storedMessage) {
					for i := 0; i < amount; i++ {
						idx := perm[i]

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
// func findConnection(connections []gossipConnection, id horizontalapi.ConnectionId) *gossipConnection {
// 	for i, conn := range connections {
// 		if conn.connection.Id == id {
// 			return &connections[i]
// 		}
// 	}
// 	return nil
// }

func NewConnCookie(dest horizontalapi.ConnectionId) connCookie {
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		panic(err)
	}

	return connCookie{
		chall:     nonce,
		timestamp: time.Unix(0, time.Now().UnixNano()),
		dest:      dest,
	}
}

func (x *connCookie) Marshal() []byte {
	timestamp := x.timestamp.UnixNano()
	buf := make([]byte, len(x.chall)+binary.Size(timestamp)+len(x.dest))

	idx := 0

	binary.BigEndian.PutUint64(buf[idx:], uint64(timestamp))
	idx += binary.Size(timestamp)

	copy(buf[idx:], x.chall[:])
	idx += len(x.chall)

	copy(buf[idx:], x.dest[:])
	idx += len(x.dest)

	return buf
}

func (x *connCookie) Unmarshal(buf []byte) {
	idx := 0

	t := binary.BigEndian.Uint64(buf[idx : idx+8])
	timestamp := time.Unix(0, int64(t))
	idx += 8

	x.chall = make([]byte, chacha20poly1305.NonceSizeX)
	copy(x.chall, buf[idx:idx+chacha20poly1305.NonceSizeX])
	idx += chacha20poly1305.NonceSizeX

	dest := make([]byte, len(buf[idx:]))
	copy(dest, buf[idx:])

	x.timestamp = timestamp
	x.dest = horizontalapi.ConnectionId(string(dest))
}

func (x *connCookie) createCookie(aead cipher.AEAD) []byte {
	payload := x.Marshal()

	ciphertext := aead.Seal(nil, x.chall, payload, nil)
	// Cipher nonce is appended as the first (24) bytes of the cookie
	cookie := slices.Concat(x.chall, ciphertext)
	return cookie
}

func ReadCookie(aead cipher.AEAD, cookie []byte) (*connCookie, error) {
	cipherNonce := cookie[0:chacha20poly1305.NonceSizeX]
	cookie = cookie[chacha20poly1305.NonceSizeX:]

	plaintext, err := aead.Open(nil, cipherNonce, cookie, nil)
	if err != nil {
		fmt.Println("Error while decrypting the cookie", "err", err)
		return nil, err
	}

	var c connCookie
	c.Unmarshal(plaintext)
	return &c, nil
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
	// Try to remove it from in progress connections
	peer := dummy.powInProgress[id]
	if peer.timestamp.IsZero() {
		delete(dummy.powInProgress, id)
		return
	}

	// Try to remove it from the valid connections
	peer = dummy.openConnectionMap[id]
	if peer.timestamp.IsZero() {
		delete(dummy.openConnectionMap, id)

		// Remove it from the slice as well
		for i, conn := range dummy.openConnections {
			if conn.connection.Id == id {
				dummy.openConnections[i] = dummy.openConnections[len(dummy.openConnections)-1]
				dummy.openConnections = dummy.openConnections[:len(dummy.openConnections)-1]
				break
			}
		}
		return
	}
}

// Returns weather the connection is valid or not (for now, just return if it is present in the openConnection)
func (dummy *dummyStrat) checkConnectionValidity(id horizontalapi.ConnectionId) bool {
	peer := dummy.openConnectionMap[id]
	if peer.timestamp.IsZero() {
		return false
	}
	return true
}

// Close the root strategy
func (dummy *dummyStrat) Close() {
	dummy.rootStrat.Close()
}
