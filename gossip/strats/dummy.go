package strats

import (
	"context"
	"errors"
	"fmt"
	"gossip/common"
	horizontalapi "gossip/horizontalAPI"
	ringbuffer "gossip/internal/ringbuffer"
	pow "gossip/pow"
	"reflect"
	"sync"

	"crypto/cipher"
	"crypto/rand"
	"math/big"
	mrand "math/rand"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

// define potential errors
var (
	ErrNoMessageFound error = errors.New("no message found when extracting")
)

var (
	POW_TIMEOUT      = 7 * time.Second
	POW_REQUEST_TIME = 2 * time.Second
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
	// Connection that this peer needs to prove
	toBeProvedConnections map[horizontalapi.ConnectionId]horizontalapi.Conn[chan<- horizontalapi.ToHz]
	// Array of peers channels, where messages can be sent (stored as slice for easing permutations)
	openConnections []gossipConnection
	// Map of valid connection (for fast access). If a connection is present in the openConnections
	// It is also present here
	openConnectionMap map[horizontalapi.ConnectionId]gossipConnection
	// Mutex to synchronize between proving connections and validating connections
	connMutex sync.RWMutex
	// Map of invalid connection (for fast access) that needs to be validated
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
// strategy must be the baseStrategy. toBeProvedConnections a list of ToHz channels, one for each peer, that
// current peer needs to send PoWs to
func NewDummy(strategy Strategy, fromHz <-chan horizontalapi.FromHz, hzConnection <-chan horizontalapi.NewConn, toBeProvedConnections map[horizontalapi.ConnectionId]horizontalapi.Conn[chan<- horizontalapi.ToHz]) dummyStrat {
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		panic(err)
	}

	return dummyStrat{
		rootStrat:             strategy,
		fromHz:                fromHz,
		hzConnection:          hzConnection,
		toBeProvedConnections: toBeProvedConnections,
		openConnectionMap:     make(map[horizontalapi.ConnectionId]gossipConnection),
		powInProgress:         make(map[horizontalapi.ConnectionId]gossipConnection),
		invalidMessages:       ringbuffer.NewRingbuffer[*storedMessage](strategy.stratArgs.Cache_size),
		validMessages:         ringbuffer.NewRingbuffer[*storedMessage](strategy.stratArgs.Cache_size),
		sentMessages:          ringbuffer.NewRingbuffer[*storedMessage](strategy.stratArgs.Cache_size),
		cipher:                aead,
	}
}

// Listen for messages incoming on either StrategyChannels (from the base strategy, such as the
// vertical API) or horizontal API
//
// This function spawn a new goroutine. Incoming messages will be processed by the Dummy Strategy.
func (dummy *dummyStrat) Listen() {
	go func() {
		// Sending out initial challenges requests
		dummy.RequestInitialChallenges()
		// A repeating signal to trigger a recurrent behavior.
		ticker := time.NewTicker(time.Duration(dummy.rootStrat.stratArgs.GossipTimer) * time.Second)
		// A repeating signal for the renewing of connections
		renewalTicker := time.NewTicker(POW_REQUEST_TIME)
		// A repeating signal for the checking (and culling) all open connections
		timeoutTicker := time.NewTicker(POW_TIMEOUT)

		// Keep listening on all channels
		for {
			select {
			// Message received from a peer.
			case x := <-dummy.fromHz:
				switch msg := x.(type) {
				case horizontalapi.Unregister:
					dummy.removeConnection(horizontalapi.ConnectionId(msg))

				case horizontalapi.Push:
					dummy.connMutex.RLock()
					peer := dummy.openConnectionMap[msg.Id]
					dummy.connMutex.RUnlock()

					if !dummy.checkConnectionValidity(peer) {
						dummy.rootStrat.log.Debug("PUSH message not processed because peer was not PoW valid", "Peer", peer)

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
						Id:     msg.Id,
						Cookie: cookie.CreateCookie(dummy.cipher),
					}

					// If request is coming from a new peer, get it from the right Map
					conn := dummy.powInProgress[msg.Id]
					if conn.connection.Id == msg.Id {
						// It is a new connection requesting a pow
						conn.connection.Data <- m
						continue
					}

					// If request is coming from a valid peer, get it from the right Map
					dummy.connMutex.RLock()
					peer := dummy.openConnectionMap[msg.Id]
					dummy.connMutex.RUnlock()

					if !peer.timestamp.IsZero() {
						// It is an already established connection, renewing the pow
						peer.connection.Data <- m
						continue
					}

					// No peer found for such connection Id
					dummy.rootStrat.log.Error("No open connection found for challReq", "conn Id", msg.Id)

				case horizontalapi.ConnChall:
					// Checks weather the Chall is coming from a toBeProvedConnection
					conn := dummy.toBeProvedConnections[msg.Id]
					if conn.Id == msg.Id {
						delete(dummy.toBeProvedConnections, conn.Id)
						go func() {
							pow := ComputePoW(msg.Cookie)
							conn.Data <- pow

							newConnection := gossipConnection{
								connection: conn,
								timestamp:  time.Now(),
							}

							dummy.connMutex.Lock()
							defer dummy.connMutex.Unlock()
							dummy.openConnections = append(dummy.openConnections, newConnection)
							dummy.openConnectionMap[conn.Id] = newConnection
						}()
						continue
					}

					// Checks weather the Chall is from an openConnection (renewal)
					peer := dummy.openConnectionMap[msg.Id]
					if peer.connection.Id == msg.Id {
						// if too little time has passed from last PoW request, this might be a
						// DoS attack, but I will be lenient and just skip it
						// diff := time.Now().Sub(peer.timestamp)
						// if diff < POW_REQUEST_TIME/2 {
						// 	dummy.rootStrat.log.Debug("Too many pow request, this one was rejected", "connection ID", msg.Id, "diff", diff)
						// 	continue
						// }
						go func() {
							pow := ComputePoW(msg.Cookie)
							peer.connection.Data <- pow
						}()
						continue
					}

					dummy.rootStrat.log.Error("No open connection found for connChall", "conn Id", msg.Id)

				// Checks incoming PoWs
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
						dummy.rootStrat.log.Debug("Invalid pow, dropping connection", "actual Id", msg.Id)
						dummy.removeConnection(msg.Id)
						continue
					}

					// check if dest is valid
					if cookieRead.dest != msg.Id {
						dummy.rootStrat.log.Debug("Mismatched connectionId between received connPow and sender", "expected conn Id", cookieRead.dest, "actual Id", msg.Id)
						dummy.removeConnection(msg.Id)
						continue
					}

					peer := dummy.powInProgress[msg.Id]
					if peer.connection.Id == msg.Id {
						// check if time taken for giving pow is within the limits
						diff := time.Now().Sub(cookieRead.timestamp)

						if diff > POW_TIMEOUT {
							dummy.rootStrat.log.Debug("POW for accepting connection was given not within the time limit", "expected conn Id", cookieRead.dest, "actual Id", msg.Id)
							dummy.removeConnection(msg.Id)
							continue
						}

						// Validate connection and remove it from inProgress
						peer.timestamp = cookieRead.timestamp

						dummy.removeConnection(msg.Id)
						dummy.connMutex.Lock()
						dummy.openConnectionMap[msg.Id] = peer
						dummy.openConnections = append(dummy.openConnections, peer)
						dummy.connMutex.Unlock()

						continue
					}

					// Checks weather the PoW is from an openConnection (renewal)
					peer = dummy.openConnectionMap[msg.Id]
					if peer.connection.Id == msg.Id {
						newPeer := gossipConnection{connection: peer.connection, timestamp: cookieRead.timestamp}
						dummy.connMutex.Lock()
						dummy.openConnectionMap[msg.Id] = newPeer
						for i := 0; i < len(dummy.openConnections); i++ {
							if dummy.openConnections[i].connection.Id == msg.Id {
								dummy.openConnections[i] = newPeer
							}
						}
						dummy.connMutex.Unlock()
						continue
					}

					dummy.rootStrat.log.Error("No open connection found for connPoW", "conn Id", msg.Id)

				}
			// Accept any connection and put it in the inProgress slice.
			case newPeer := <-dummy.hzConnection:
				conn := gossipConnection{
					connection: horizontalapi.Conn[chan<- horizontalapi.ToHz](newPeer),
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
				dummy.connMutex.RLock()
				// Create a permutation over the length of valid connections
				perm := mrand.Perm(len(dummy.openConnections))
				amount := min(len(dummy.openConnections), int(dummy.rootStrat.stratArgs.Degree))

				validMessages := dummy.validMessages.ExtractToSlice()
				for _, msg := range validMessages {
					for i := 0; i < amount; i++ {
						idx := perm[i]

						//dummy.rootStrat.log.Debug("DST conn", "conn", dummy.openConnections[idx])
						dummy.openConnections[idx].connection.Data <- msg.message
						dummy.rootStrat.log.Debug("HZ Message sent:", "dst", dummy.openConnections[idx].connection.Id, "Message", msg)
						msg.counter++

						// If message was sent to args.Degree neighboughrs delete it from the set of messages
						if msg.counter >= int(dummy.rootStrat.stratArgs.Degree) {
							err := dummy.validMessages.Remove(msg)
							if err != nil {
								panic(err)
							}
							dummy.sentMessages.Insert(msg)
							break
						}
					}
				}
				dummy.connMutex.RUnlock()

			case <-renewalTicker.C:
				dummy.RequestChallenges()

			case <-timeoutTicker.C:
				dummy.CullConnections()

			case <-dummy.rootStrat.ctx.Done():
				// should terminate
				return
			}
		}
	}()
}

// Returns weather the connection is valid or not
func (dummy *dummyStrat) checkConnectionValidity(peer gossipConnection) bool {
	diff := time.Now().Sub(peer.timestamp)
	return diff < POW_TIMEOUT
}

// Request Challenge for initial proof of work
func (dummy *dummyStrat) RequestInitialChallenges() {
	// This is not parallelized as there is very little computation overhead
	for _, conn := range dummy.toBeProvedConnections {
		req := horizontalapi.ConnReq{}
		conn.Data <- req
	}
}

// Remove all connection which did not provide a PoW lately
func (dummy *dummyStrat) CullConnections() {
	dummy.connMutex.Lock()
	defer dummy.connMutex.Unlock()

	for _, conn := range dummy.openConnections {
		if !dummy.checkConnectionValidity(conn) {
			dummy.unsafeRemoveConnection(conn.connection.Id)
			dummy.rootStrat.log.Debug("Removed a connection since no pow was received in time", "Conn Id", conn.connection.Id)
			fmt.Println("Removed a connection since no pow was received in time", "Conn Id", conn.connection.Id)
		}
	}

}

// Request challenges for renewal of connections
func (dummy *dummyStrat) RequestChallenges() {
	// This is not parallelized as there is very little computation overhead
	for _, conn := range dummy.openConnections {
		req := horizontalapi.ConnReq{}
		conn.connection.Data <- req
	}
}

// Remove a connection with id from all the data structures. Has to be locked
func (dummy *dummyStrat) unsafeRemoveConnection(id horizontalapi.ConnectionId) {
	// Remove it from powInProgress connections
	peer := dummy.powInProgress[id]
	if peer.connection.Id == id {
		delete(dummy.powInProgress, id)
		return
	}

	// Remove it from openConnectionMap and slice
	peer = dummy.openConnectionMap[id]
	if !peer.timestamp.IsZero() {
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

// Wrapper on removeConnection to provide synchronization
func (dummy *dummyStrat) removeConnection(id horizontalapi.ConnectionId) {
	dummy.connMutex.Lock()
	defer dummy.connMutex.Unlock()

	dummy.unsafeRemoveConnection(id)
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
