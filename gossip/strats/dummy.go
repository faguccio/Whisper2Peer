package strats

import (
	horizontalapi "gossip/horizontalAPI"
	"time"
)

type dummyStrat struct {
	rootStrat       Strategy
	fromHz          <-chan horizontalapi.FromHz
	hzConnection    <-chan horizontalapi.NewConn
	openConnections []chan<- horizontalapi.ToHz
}

func NewDummy(strategy Strategy, fromHz <-chan horizontalapi.FromHz, hzConnection <-chan horizontalapi.NewConn, openConnections []chan<- horizontalapi.ToHz) dummyStrat {
	return dummyStrat{rootStrat: strategy, fromHz: fromHz, hzConnection: hzConnection, openConnections: openConnections}
}

func (dummy *dummyStrat) Listen() {
	// This will be a configuration parameter
	ticker := time.NewTicker(2 * time.Second)

	for {
		select {
		case x := <-dummy.fromHz:
			// Ask for validation
			// Append to the "to be relayed" messages
			// Send Vertical notification
		case newPeer := <-dummy.hzConnection:
			dummy.openConnections = append(dummy.openConnections, newPeer.ToHz)

		case x := <-dummy.rootStrat.strategyChannels.Announce:
			// Append to the "to be relayed" messages
		case x := <-dummy.rootStrat.strategyChannels.Validation:
			// Mark the message as valid
		case _ := <-ticker.C:
			// Time passed! New round:
			// Select random neigbours
			// Send messages
			// If message was sent to args.Degree neighboughrs delete it from the set of messages
		}
	}
}

func (dummy *dummyStrat) Close() {
	dummy.rootStrat.Close()
}
