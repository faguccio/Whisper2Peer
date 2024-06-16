package strats

import (
	"errors"
	"gossip/common"
	horizontalapi "gossip/horizontalAPI"

	"math/big"
	"time"

	"crypto/rand"
	mrand "math/rand"
)

type dummyStrat struct {
	rootStrat        Strategy
	fromHz           <-chan horizontalapi.FromHz
	hzConnection     <-chan horizontalapi.NewConn
	openConnections  []chan<- horizontalapi.ToHz
	unvalidatedCache []horizontalapi.Push
	validatedCache   []horizontalapi.Push
}

func NewDummy(strategy Strategy, fromHz <-chan horizontalapi.FromHz, hzConnection <-chan horizontalapi.NewConn, openConnections []chan<- horizontalapi.ToHz) dummyStrat {

	return dummyStrat{
		rootStrat:        strategy,
		fromHz:           fromHz,
		hzConnection:     hzConnection,
		openConnections:  openConnections,
		unvalidatedCache: make([]horizontalapi.Push, 0),
		validatedCache:   make([]horizontalapi.Push, 0),
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
					dummy.unvalidatedCache = append(dummy.unvalidatedCache, msg)
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
					dummy.validatedCache = append(dummy.validatedCache, pushMsg)
				case common.GossipValidation:
					if x.Valid {
						err := dummy.validateMessage(x.MessageId)
						if err != nil {
							dummy.rootStrat.log.Warn("Tried to validate a message which did not exists", "Message ID", x.MessageId)
						}

					} else {
						dummy.removeMessage(x.MessageId)
					}
				}

			case <-ticker.C:
				// Time passed! New round:
				// Send messages to random neighbor
				//dummy.rootStrat.log.Debug("Length of openConnection", "len", len(dummy.openConnections))
				idx := mrand.Intn(len(dummy.openConnections))
				for _, message := range dummy.validatedCache {
					dummy.openConnections[idx] <- message
					dummy.rootStrat.log.Debug("Message sent:", "msg", message)
				}
				dummy.validatedCache = make([]horizontalapi.Push, 0)

				// If message was sent to args.Degree neighboughrs delete it from the set of messages
			}
		}
	}()
}

func (dummy *dummyStrat) validateMessage(messageId uint16) error {
	valid, err := dummy.removeMessage(messageId)
	// What should I do if there is no message to be validated?
	if err != nil {
		return err
	}
	dummy.validatedCache = append(dummy.validatedCache, *valid)
	return nil
}

func (dummy *dummyStrat) removeMessage(messageId uint16) (*horizontalapi.Push, error) {
	index := -1
	for i, message := range dummy.unvalidatedCache {
		if message.MessageID == messageId {
			index = i
			break
		}
	}

	if index == -1 {
		return nil, errors.New("Tried to remove a non existent message")
	}

	removedItem := dummy.unvalidatedCache[index]
	dummy.unvalidatedCache[index] = dummy.unvalidatedCache[len(dummy.unvalidatedCache)-1]
	dummy.unvalidatedCache = dummy.unvalidatedCache[:len(dummy.unvalidatedCache)-1]
	return &removedItem, nil
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
