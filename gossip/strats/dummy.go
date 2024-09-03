package strats

import (
	"context"
	"errors"
	"gossip/common"
	horizontalapi "gossip/horizontalAPI"
	ringbuffer "gossip/internal/ringbuffer"
	pow "gossip/pow"
	"reflect"

	"crypto/cipher"
	"crypto/rand"
	"math/big"
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
	// Connection Manager object
	connManager *ConnectionManager

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
func NewDummy(strategy Strategy, fromHz <-chan horizontalapi.FromHz, connManager *ConnectionManager) dummyStrat {
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		panic(err)
	}

	return dummyStrat{
		rootStrat:       strategy,
		fromHz:          fromHz,
		connManager:     connManager,
		invalidMessages: ringbuffer.NewRingbuffer[*storedMessage](strategy.stratArgs.Cache_size),
		validMessages:   ringbuffer.NewRingbuffer[*storedMessage](strategy.stratArgs.Cache_size),
		sentMessages:    ringbuffer.NewRingbuffer[*storedMessage](strategy.stratArgs.Cache_size),
		cipher:          aead,
	}
}

// Listen for messages incoming on either StrategyChannels (from the base strategy, such as the
// vertical API) or horizontal API
//
// This function spawn a new goroutine. Incoming messages will be processed by the Dummy Strategy.
func (dummy *dummyStrat) Listen() {
	go func() {
		// Sending out initial challenges requests
		dummy.connManager.ActionOnToBeProved(func(x gossipConnection) {
			req := horizontalapi.ConnReq{}
			x.connection.Data <- req
		})
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
					peer, err := dummy.connManager.Remove(horizontalapi.ConnectionId(msg))
					if err == nil {
						// now after removing the peer from all internal datastructures it is safe to fully close it
						peer.connection.Cfunc()
					}

				case horizontalapi.Push:
					_, isValid := dummy.connManager.FindValid(msg.Id)

					if !isValid {
						dummy.rootStrat.log.Debug("PUSH message not processed because peer was not PoW valid", "Peer ID", msg.Id)
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
					// Create ConnChall message with the encrypted cookie
					cookie := NewConnCookie(msg.Id)

					peer, IsInProgress := dummy.connManager.FindInProgress(msg.Id)

					if !IsInProgress {
						dummy.rootStrat.log.Debug("ConnReq received from a connection not present in the inProgress connections", "Id", msg.Id)
						continue
					}

					m := horizontalapi.ConnChall{
						Id:     msg.Id,
						Cookie: cookie.CreateCookie(dummy.cipher),
					}

					peer.connection.Data <- m

				case horizontalapi.ConnChall:
					// Checks weather the Chall is coming from a toBeProvedConnection
					peer, isToBeProved := dummy.connManager.FindToBeProved(msg.Id)
					if !isToBeProved {
						dummy.rootStrat.log.Debug("ConnChall received from a not toBeProved connection", "id", msg.Id, "Message", msg)
						dummy.connManager.Remove(msg.Id)
						continue
					}

					go func() {
						nonce := ComputePoW(msg.Cookie)
						pow := horizontalapi.ConnPoW{PowNonce: nonce, Cookie: msg.Cookie}
						select {
						case <-peer.connection.Ctx.Done():
						// connection was already closed in the meantime
						default:
							peer.connection.Data <- pow
							dummy.connManager.MakeValid(peer.connection.Id, time.Now())
						}
					}()

				// Checks incoming PoWs
				case horizontalapi.ConnPoW:
					_, connValidty := dummy.connManager.FindInProgress(msg.Id)
					if !connValidty {
						dummy.rootStrat.log.Debug("ConnPow received was from a connection which is not actually in Progress", "Id", msg.Id)
						dummy.connManager.Remove(msg.Id)
						continue
					}

					mypow := powMarsh{PowNonce: msg.PowNonce, Cookie: msg.Cookie}
					cookieRead, err := ReadCookie(dummy.cipher, mypow.Cookie)

					if err != nil {
						dummy.rootStrat.log.Debug("Failed to decrypt cookie, dropping connection", "connection ID", msg.Id)
						dummy.connManager.Remove(msg.Id)
						continue
					}

					// Check proof of work
					powValidity := pow.CheckProofOfWork(func(digest []byte) bool {
						return pow.First8bits0(digest)
					}, &mypow)

					if !powValidity {
						dummy.rootStrat.log.Debug("Invalid pow, dropping connection", "actual Id", msg.Id)
						dummy.connManager.Remove(msg.Id)
						continue
					}

					// check if dest is valid
					if cookieRead.dest != msg.Id {
						dummy.rootStrat.log.Debug("Mismatched connectionId between received connPow and sender", "expected conn Id", cookieRead.dest, "actual Id", msg.Id)
						dummy.connManager.Remove(msg.Id)
						continue
					}

					// check if time taken for giving pow is within the limits
					diff := time.Now().Sub(cookieRead.timestamp)

					if diff > POW_TIMEOUT {
						dummy.rootStrat.log.Debug("POW for accepting connection was given not within the time limit", "expected conn Id", cookieRead.dest, "actual Id", msg.Id)
						dummy.connManager.Remove(msg.Id)
						continue
					}

					dummy.connManager.MakeValid(msg.Id, cookieRead.timestamp)

				case horizontalapi.PowReq:
					// Create PowChall message with the encrypted cookie
					cookie := NewConnCookie(msg.Id)

					peer, isValid := dummy.connManager.FindValid(msg.Id)
					if !isValid {
						dummy.rootStrat.log.Debug("Id not found in the connection manager", "Id", msg.Id)
						continue
					}

					m := horizontalapi.PowChall{
						Id:     msg.Id,
						Cookie: cookie.CreateCookie(dummy.cipher),
					}

					peer.connection.Data <- m

				case horizontalapi.PowChall:
					// Checks weather the Chall is from an openConnection (renewal)
					peer, isValid := dummy.connManager.FindValid(msg.Id)

					if !isValid {
						dummy.rootStrat.log.Debug("PowChall received from a not valid connection", "id", msg.Id)
						dummy.connManager.Remove(msg.Id)
						continue
					}

					// if too little time has passed from last PoW request, this might be a
					// DoS attack, but I will be lenient and just skip it
					// diff := time.Now().Sub(peer.timestamp)
					// if diff < POW_REQUEST_TIME/2 {
					// 	dummy.rootStrat.log.Debug("Too many pow request, this one was rejected", "connection ID", msg.Id, "diff", diff)
					// 	continue
					// }

					go func() {
						nonce := ComputePoW(msg.Cookie)
						pow := horizontalapi.PowPoW{PowNonce: nonce, Cookie: msg.Cookie}
						select {
						case <-peer.connection.Ctx.Done():
						// connection was already closed in the meantime
						default:
							peer.connection.Data <- pow
						}
					}()

				case horizontalapi.PowPoW:
					// Checks weather the PoW is from an openConnection (renewal)
					_, connValidty := dummy.connManager.FindValid(msg.Id)
					if !connValidty {
						dummy.rootStrat.log.Debug("PoWPoW received was from a connection which is not actually valid", "Id", msg.Id)
						dummy.connManager.Remove(msg.Id)
						continue
					}

					mypow := powMarsh{PowNonce: msg.PowNonce, Cookie: msg.Cookie}
					cookieRead, err := ReadCookie(dummy.cipher, mypow.Cookie)

					if err != nil {
						dummy.rootStrat.log.Debug("Failed to decrypt cookie, dropping connection", "connection ID", msg.Id)
						dummy.connManager.Remove(msg.Id)
						continue
					}

					// Check proof of work
					powValidity := pow.CheckProofOfWork(func(digest []byte) bool {
						return pow.First8bits0(digest)
					}, &mypow)

					if !powValidity {
						dummy.rootStrat.log.Debug("Invalid pow, dropping connection", "actual Id", msg.Id)
						dummy.connManager.Remove(msg.Id)
						continue
					}

					// check if dest is valid
					if cookieRead.dest != msg.Id {
						dummy.rootStrat.log.Debug("Mismatched connectionId between received connPow and sender", "expected conn Id", cookieRead.dest, "actual Id", msg.Id)
						dummy.connManager.Remove(msg.Id)
						continue
					}

					dummy.connManager.MakeValid(msg.Id, cookieRead.timestamp)

				case horizontalapi.NewConn:
					// Accept any connection and put it in the inProgress slice.
					conn := gossipConnection{
						connection: horizontalapi.Conn[chan<- horizontalapi.ToHz](msg),
					}
					dummy.connManager.AddInProgress(conn)
				}

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
				validMessages := dummy.validMessages.ExtractToSlice()

				dummy.connManager.ActionOnPermutedValid(func(peer gossipConnection) {
					for _, msg := range validMessages {
						peer.connection.Data <- msg.message
						dummy.rootStrat.log.Debug("HZ Message sent:", "dst", peer.connection.Id, "Message", msg)
						msg.counter++

						// If message was sent to args.Degree neighboughrs delete it from the set of messages
						if msg.counter >= int(dummy.rootStrat.stratArgs.Degree) {
							dummy.validMessages.Remove(msg)
							dummy.sentMessages.Insert(msg)
							break
						}
					}
				}, int(dummy.rootStrat.stratArgs.Degree))

			case <-renewalTicker.C:
				dummy.connManager.ActionOnValid(func(x gossipConnection) {
					req := horizontalapi.PowReq{}
					x.connection.Data <- req
				})

			case <-timeoutTicker.C:
				dummy.connManager.CullConnections(isConnectionInvalid)

			case <-dummy.rootStrat.ctx.Done():
				// should terminate
				return
			}
		}
	}()
}

// Returns weather the connection is valid or not
func isConnectionInvalid(peer gossipConnection) bool {
	diff := time.Now().Sub(peer.timestamp)
	return !(diff < POW_TIMEOUT)
}

// Go through a ringbuffer of messages and return the one with a matching ID, error if none is found
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
