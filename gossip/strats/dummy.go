package strats

import (
	"errors"
	"gossip/common"
	horizontalapi "gossip/horizontalAPI"
	ringbuffer "gossip/internal/ringbuffer"
	"reflect"

	"math/big"
	"time"

	"crypto/rand"
	mrand "math/rand"
)

var (
	ErrNoMessageFound error = errors.New("No message found when extracting")
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
						dummy.rootStrat.strategyChannels.FromStrat <- notification
						dummy.rootStrat.log.Debug("HZ Message received:", "type", reflect.TypeOf(msg), "msg", msg)
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
						dummy.validMessages.Insert(msg)
					}
				}

				// Recurrent timer signal
			case <-ticker.C:
				// A random peer is selected and we relay all messages to that peer.
				idx := mrand.Intn(len(dummy.openConnections))
				dummy.validMessages.Do((func(msg *storedMessage) {
					//dummy.rootStrat.log.Debug("DST conn", "conn", dummy.openConnections[idx])
					dummy.openConnections[idx].Data <- msg.message
					dummy.rootStrat.log.Debug("HZ Message sent:", "dst", dummy.openConnections[idx].Id, "msg", msg)
					msg.counter++

					// If message was sent to args.Degree neighboughrs delete it from the set of messages
					if msg.counter >= int(dummy.rootStrat.stratArgs.Degree) {
						dummy.validMessages.Remove(msg)
						dummy.sentMessages.Insert(msg)
					}
				}))
			}
		}
	}()
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
