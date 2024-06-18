package strats

import (
	"fmt"
	"gossip/common"
	horizontalapi "gossip/horizontalAPI"
	ringbuffer "gossip/internal/ringbuffer"

	"math/big"
	"time"

	"crypto/rand"
	mrand "math/rand"
)

type dummyStrat struct {
	rootStrat       Strategy
	fromHz          <-chan horizontalapi.FromHz
	hzConnection    <-chan horizontalapi.NewConn
	openConnections []chan<- horizontalapi.ToHz
	invalidMessages *ringbuffer.Ringbuffer[*horizontalapi.Push]
	validMessages   *ringbuffer.Ringbuffer[*horizontalapi.Push]
	sentMessages    *ringbuffer.Ringbuffer[*horizontalapi.Push]
}

func NewDummy(strategy Strategy, fromHz <-chan horizontalapi.FromHz, hzConnection <-chan horizontalapi.NewConn, openConnections []chan<- horizontalapi.ToHz) dummyStrat {
	return dummyStrat{
		rootStrat:       strategy,
		fromHz:          fromHz,
		hzConnection:    hzConnection,
		openConnections: openConnections,
		invalidMessages: ringbuffer.NewRingbuffer[*horizontalapi.Push](strategy.cacheSize),
		validMessages:   ringbuffer.NewRingbuffer[*horizontalapi.Push](strategy.cacheSize),
		sentMessages:    ringbuffer.NewRingbuffer[*horizontalapi.Push](strategy.cacheSize),
	}
}

func (dummy *dummyStrat) Listen() {
	// This will be a configuration parameter
	go func() {
		ticker := time.NewTicker(1 * time.Second)

		for {
			select {
			case x := <-dummy.fromHz:
				// Will notify the vertical api of the message
				switch msg := x.(type) {
				case horizontalapi.Push:
					notification := convertPushToNotification(msg)
					dummy.invalidMessages.Insert(&msg)
					// HANDLE MAIN TO send messages to all listener registered to that TYPE
					dummy.rootStrat.strategyChannels.FromStrat <- notification
					dummy.rootStrat.log.Debug("Message received:", "msg", msg)
				}

			case newPeer := <-dummy.hzConnection:
				dummy.openConnections = append(dummy.openConnections, newPeer.ToHz)

			case x := <-dummy.rootStrat.strategyChannels.ToStrat:
				switch x := x.(type) {
				case common.GossipAnnounce:
					// Append announce to the "to be relayed" (so validated) messages
					pushMsg := convertAnnounceToPush(x)
					dummy.validMessages.Insert(&pushMsg)
				case common.GossipValidation:
					if x.Valid {
						err := moveMessage(dummy.invalidMessages, dummy.validMessages, x.MessageId)
						if err != nil {
							dummy.rootStrat.log.Warn("Tried to validate a message which did not exists", "Message ID", x.MessageId)
						}

					} else {
						msg, err := extractMessage(dummy.invalidMessages, x.MessageId)
						if err != nil {
							dummy.rootStrat.log.Warn("Tried to invalidate a message wich did not exists", "Message ID", x.MessageId)
						}
						dummy.invalidMessages.Remove(msg)
					}
				}

			case <-ticker.C:
				// Time passed! New round:
				// Send messages to random neighbor
				//dummy.rootStrat.log.Debug("Length of openConnection", "len", len(dummy.openConnections))
				idx := mrand.Intn(len(dummy.openConnections))
				dummy.validMessages.Do((func(msg *horizontalapi.Push) {
					dummy.openConnections[idx] <- *msg
					dummy.rootStrat.log.Debug("Message sent:", "msg", msg)
					moveMessage(dummy.validMessages, dummy.sentMessages, msg.MessageID)
				}))

				// If message was sent to args.Degree neighboughrs delete it from the set of messages
			}
		}
	}()
}

func extractMessage(ring *ringbuffer.Ringbuffer[*horizontalapi.Push], messageId uint16) (*horizontalapi.Push, error) {
	result := ring.Filter(func(p *horizontalapi.Push) bool {
		return p.MessageID == messageId
	})

	if len(result) == 0 {
		return nil, fmt.Errorf("Trying to extract a message but 0 were found")
	}

	return result[0], nil

}

func moveMessage(src *ringbuffer.Ringbuffer[*horizontalapi.Push], dst *ringbuffer.Ringbuffer[*horizontalapi.Push], messageId uint16) error {
	msg, err := extractMessage(src, messageId)

	if err != nil {
		return err
	}

	src.Remove(msg)
	dst.Insert(msg)

	return nil
}

func convertAnnounceToPush(msg common.GossipAnnounce) horizontalapi.Push {
	id, _ := rand.Int(rand.Reader, big.NewInt(16))

	pushMsg := horizontalapi.Push{
		TTL:        msg.TTL,
		GossipType: msg.DataType,
		MessageID:  uint16(id.Int64()),
		Payload:    msg.Data,
	}

	return pushMsg
}

func convertPushToNotification(pushMsg horizontalapi.Push) common.GossipNotification {
	notification := common.GossipNotification{
		MessageId: pushMsg.MessageID,
		DataType:  common.GossipType(pushMsg.GossipType),
		Data:      pushMsg.Payload,
	}
	return notification
}

func (dummy *dummyStrat) Close() {
	dummy.rootStrat.Close()
}
