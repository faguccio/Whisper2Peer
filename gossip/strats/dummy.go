package strats

import (
	"errors"
	horizontalapi "gossip/horizontalAPI"
	verticalapi "gossip/verticalAPI"
	vertTypes "gossip/verticalAPI/types"

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
	ticker := time.NewTicker(2 * time.Second)

	for {
		select {
		case x := <-dummy.fromHz:
			// Will notify the vertical api of the message
			switch msg := x.(type) {
			case horizontalapi.Push:
				notification := convertPushToNotification(msg)
				dummy.unvalidatedCache = append(dummy.unvalidatedCache, msg)
				notificationMsg := verticalapi.MainToVertNotification{
					Data: notification,
				}
				// HANDLE MAIN TO send messages to all listener registered to that TYPE
				dummy.rootStrat.strategyChannels.Notification <- notificationMsg
			}

		case newPeer := <-dummy.hzConnection:
			dummy.openConnections = append(dummy.openConnections, newPeer.ToHz)

		case x := <-dummy.rootStrat.strategyChannels.Announce:
			// Append announce to the "to be relayed" (so validated) messages
			pushMsg := convertAnnounceToPush(x)
			dummy.validatedCache = append(dummy.validatedCache, pushMsg)

		case x := <-dummy.rootStrat.strategyChannels.Validation:
			if x.Data.Valid {
				dummy.validateMessage(x.Data.MessageId)
			} else {
				dummy.removeMessage(x.Data.MessageId)
			}

		case <-ticker.C:
			// Time passed! New round:
			// Send messages to random neighbor
			idx := mrand.Intn(len(dummy.openConnections))
			for _, message := range dummy.validatedCache {
				dummy.openConnections[idx] <- message
				// FABIO TODO: remove message from validated!!!
			}

			// If message was sent to args.Degree neighboughrs delete it from the set of messages
		}
	}
}

func (dummy *dummyStrat) validateMessage(messageId uint16) {
	valid, _ := dummy.removeMessage(messageId)
	// What should I do if there is no message to be validated?
	dummy.validatedCache = append(dummy.validatedCache, *valid)
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

func convertAnnounceToPush(msg verticalapi.VertToMainAnnounce) horizontalapi.Push {
	id, _ := rand.Int(rand.Reader, big.NewInt(16))

	pushMsg := horizontalapi.Push{
		TTL:        msg.Data.TTL,
		GossipType: uint16(msg.Data.DataType),
		MessageID:  uint16(id.Int64()),
		Payload:    msg.Data.Data,
	}

	return pushMsg
}

func convertPushToNotification(pushMsg horizontalapi.Push) vertTypes.GossipNotification {
	notification := vertTypes.GossipNotification{
		MessageHeader: vertTypes.MessageHeader{},
		MessageId:     pushMsg.MessageID,
		DataType:      vertTypes.GossipType(pushMsg.GossipType),
		Data:          pushMsg.Payload,
	}

	notification.MessageHeader.Size = uint16(notification.CalcSize())
	notification.MessageHeader.Type = vertTypes.GossipNotificationType

	return notification
}

func (dummy *dummyStrat) Close() {
	dummy.rootStrat.Close()
}
