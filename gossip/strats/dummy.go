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
	// TODO generate
	secretKey []byte = []byte("secrets_secrets_secrets_secrets_")
)

// Some messages might use a counter of how many peers the message was relayed to
type storedMessage struct {
	counter int
	message horizontalapi.Push
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
	openConnections []horizontalapi.Conn[chan<- horizontalapi.ToHz]
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
func NewDummy(strategy Strategy, fromHz <-chan horizontalapi.FromHz, hzConnection <-chan horizontalapi.NewConn, openConnections []horizontalapi.Conn[chan<- horizontalapi.ToHz]) dummyStrat {

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
					for i, j := range dummy.openConnections {
						if j.Id == horizontalapi.ConnectionId(msg) {
							dummy.openConnections[i] = dummy.openConnections[len(dummy.openConnections)-1]
							dummy.openConnections = dummy.openConnections[:len(dummy.openConnections)-1]
							break
						}
					}
				case horizontalapi.Push:
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

					// Create ConnChall message
					cookie := createCookie(chall, string(msg.Id))
					m := horizontalapi.ConnChall{
						Chall:  chall,
						Cookie: cookie,
					}

					// Send message to correct peer
					peer := findConnection(dummy.openConnections, msg.Id)
					if peer == nil {
						dummy.rootStrat.log.Error("No open connection found for challReq", "conn Id", msg.Id)
					}
					peer.Data <- m

				case horizontalapi.ConnPoW:
					mypow := powMarsh{PowNonce: msg.PowNonce, Cookie: msg.Cookie}
					flag := pow.CheckProofOfWork(func(digest []byte) bool {
						return pow.First8bits0(digest)
					}, &mypow)

					if flag {
						// Flag the connection as valid or add it to the valid Ids
					}
				}

				// New connection is established.
			case newPeer := <-dummy.hzConnection:
				dummy.openConnections = append(dummy.openConnections, horizontalapi.Conn[chan<- horizontalapi.ToHz](newPeer))

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
						//dummy.rootStrat.log.Debug("DST conn", "conn", dummy.openConnections[idx])
						dummy.openConnections[idx].Data <- msg.message
						dummy.rootStrat.log.Debug("HZ Message sent:", "dst", dummy.openConnections[idx].Id, "Message", msg)
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
func findConnection(connections []horizontalapi.Conn[chan<- horizontalapi.ToHz], id horizontalapi.ConnectionId) *horizontalapi.Conn[chan<- horizontalapi.ToHz] {
	for _, conn := range connections {
		if conn.Id == id {
			return &conn
		}
	}
	return nil
}

func createCookie(chall uint64, dest string) []byte {
	// Create chacha20 cipher

	aead, err := chacha20poly1305.New(secretKey)
	if err != nil {
		panic(err)
	}

	// Serialize the cookie as a byte array
	bchall := make([]byte, 8)
	binary.LittleEndian.PutUint64(bchall, chall)
	timestamp := make([]byte, 8)
	binary.LittleEndian.PutUint64(timestamp, uint64(time.Now().Unix()))
	plaintext := slices.Concat(bchall, timestamp, []byte(dest))

	//ciphertext := make([]byte, len(plaintext))
	//fmt.Println(plaintext)

	// TODO: I am using as the nonce the challange for the PoW (appendend with 0s, ugh).
	// I think if we want to stick with ChaCha20 we need to change the protocol messages to include the
	// ChaCha Nonce, otherwise we need to store state? 64 random bits could also be enough for uniqueness
	ciphertext := aead.Seal(nil, slices.Concat(bchall, []byte{0, 0, 0, 0}), plaintext, nil)

	//(ciphertext, plaintext)
	//fmt.Println(ciphertext)

	return ciphertext
}

func readCookie(cookie []byte, chall uint64) (readChall uint64, timestamp time.Time, dest string) {
	// Create chacha20 cipher
	aead, err := chacha20poly1305.New(secretKey)
	if err != nil {
		panic(err)
	}

	bchall := make([]byte, 8)
	binary.LittleEndian.PutUint64(bchall, chall)

	// Decrypt the cookie
	//plaintext := make([]byte, len(cookie))
	plaintext, err := aead.Open(nil, slices.Concat(bchall, []byte{0, 0, 0, 0}), cookie, nil)
	if err != nil {
		return
	}
	fmt.Println(plaintext)

	// Deserialize cookie
	readChall = binary.LittleEndian.Uint64(plaintext[0:8])
	t := binary.LittleEndian.Uint64(plaintext[8:16])
	timestamp = time.Unix(int64(t), 0)
	dest = string(plaintext[16:])

	return
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

// Close the root strategy
func (dummy *dummyStrat) Close() {
	dummy.rootStrat.Close()
}
