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

// Valid messages have also a counter of how many peers the message was relayed to
type validMessage struct {
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
	openConnections []chan<- horizontalapi.ToHz
	// Collection of messages received from a peer which needs to be validated through the vertical api
	invalidMessages *ringbuffer.Ringbuffer[*horizontalapi.Push]
	// Collection of messages which need to be relayed to other peers.
	validMessages *ringbuffer.Ringbuffer[*validMessage]
	// Collection of messages already relayed to other peers
	sentMessages *ringbuffer.Ringbuffer[*horizontalapi.Push]
}

// Function to instantiate a new DummyStrategy.
//
// strategy must be the baseStrategy. openConnection a list of ToHz channels, one for each peer
func NewDummy(strategy Strategy, fromHz <-chan horizontalapi.FromHz, hzConnection <-chan horizontalapi.NewConn, openConnections []chan<- horizontalapi.ToHz) dummyStrat {
	return dummyStrat{
		rootStrat:       strategy,
		fromHz:          fromHz,
		hzConnection:    hzConnection,
		openConnections: openConnections,
		invalidMessages: ringbuffer.NewRingbuffer[*horizontalapi.Push](strategy.cacheSize),
		validMessages:   ringbuffer.NewRingbuffer[*validMessage](strategy.cacheSize),
		sentMessages:    ringbuffer.NewRingbuffer[*horizontalapi.Push](strategy.cacheSize),
	}
}

// Listen for messages incoming on either StrategyChannels (from the base strategy, such as the
// vertical API) or horizontal API
//
// This function spawn a new goroutine. Incoming messages will be processed by the Dummy Strategy.
func (dummy *dummyStrat) Listen() {
	go func() {
		// A repeating signal to trigger a recurrent behavior.
		ticker := time.NewTicker(1 * time.Second)

		// Keep listening on all channels
		for {
			select {
			// Message received from a peer.
			case x := <-dummy.fromHz:
				switch msg := x.(type) {
				case horizontalapi.Push:
					notification := convertPushToNotification(msg)
					_, err := fetchMessage(dummy.sentMessages, msg.MessageID)

					// If the message was not already sent, move it to the invalidMessages
					// and send a notification to vert API
					if err == ErrNoMessageFound {
						dummy.invalidMessages.Insert(&msg)
						dummy.rootStrat.strategyChannels.FromStrat <- notification
						dummy.rootStrat.log.Debug("HZ Message received:", "type", reflect.TypeOf(msg), "msg", msg)
					}
				}

				// New connection is established.
			case newPeer := <-dummy.hzConnection:
				dummy.openConnections = append(dummy.openConnections, newPeer.ToHz)

				// Message from the vertical API
			case x := <-dummy.rootStrat.strategyChannels.ToStrat:
				switch x := x.(type) {
				case common.GossipAnnounce:
					pushMsg := convertAnnounceToPush(x)
					// We consider Announce messages automatically valid
					dummy.validMessages.Insert(&validMessage{0, pushMsg})
				case common.GossipValidation:
					msg, err := extractMessage(dummy.invalidMessages, x.MessageId)

					if err != nil {
						dummy.rootStrat.log.Warn("Tried to validate a message which did not exists", "Message ID", x.MessageId)
						break
					}

					if x.Valid {
						dummy.validMessages.Insert(&validMessage{0, *msg})
					}
				}

				// Recurrent timer signal
			case <-ticker.C:
				// A random peer is selected and we relay all messages to that peer.
				idx := mrand.Intn(len(dummy.openConnections))
				dummy.validMessages.Do((func(msg *validMessage) {
					dummy.openConnections[idx] <- msg.message
					dummy.rootStrat.log.Debug("HZ Message sent:", "dst", "tbi", "msg", msg)
					msg.counter++

					// If message was sent to args.Degree neighboughrs delete it from the set of messages
					if msg.counter >= int(dummy.rootStrat.degree) {
						dummy.validMessages.Remove(msg)
						dummy.sentMessages.Insert(&msg.message)
					}
				}))

			}
		}
	}()
}

// Go throught a ringbuffer of messages and return the one with a matching ID, error if none is found
func fetchMessage(ring *ringbuffer.Ringbuffer[*horizontalapi.Push], messageId uint16) (*horizontalapi.Push, error) {
	result := ring.Filter(func(p *horizontalapi.Push) bool {
		return p.MessageID == messageId
	})

	if len(result) == 0 {
		return nil, ErrNoMessageFound
	}

	return result[0], nil
}

// Go throught a ringbuffer of messages and return the one with a matching ID, error if none is found.
// Also remove the message from the buffer
func extractMessage(ring *ringbuffer.Ringbuffer[*horizontalapi.Push], messageId uint16) (*horizontalapi.Push, error) {
	msg, err := fetchMessage(ring, messageId)
	if err != nil {
		return nil, err
	}

	ring.Remove(msg)
	return msg, nil
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
