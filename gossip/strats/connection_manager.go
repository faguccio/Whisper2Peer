package strats

import (
	"errors"
	"fmt"
	horizontalapi "gossip/horizontalAPI"
	mrand "math/rand"
	"sync"
	"time"
)

type gossipConnection struct {
	connection horizontalapi.Conn[chan<- horizontalapi.ToHz]
	timestamp  time.Time
}

// This object is used to manage the connection used by the gossip strategy
//
// It has to maintain 3 types of connection
// 1. Valid ones: peers that have provided a PoW
// 2. In progress ones: peers that are about to provide a PoW
// 3. To be proved ones: peers to which we have to give a PoW

type ConnectionManager struct {
	// Connection that this peer needs to prove
	toBeProvedConnections map[horizontalapi.ConnectionId]gossipConnection
	// Array of peers channels, where messages can be sent (stored as slice for easing permutations)
	openConnections []gossipConnection
	// Map of valid connection (for fast access). If a connection is present in the openConnections
	openConnectionsMap map[horizontalapi.ConnectionId]gossipConnection
	// Map of invalid connection (for fast access) that needs to be validated
	powInProgress map[horizontalapi.ConnectionId]gossipConnection

	// Mutex to synchronize between proving connections and validating connections
	connMutex sync.RWMutex
}

func NewConnectionManager(toBeProved []horizontalapi.Conn[chan<- horizontalapi.ToHz]) ConnectionManager {
	toBeProvedMap := make(map[horizontalapi.ConnectionId]gossipConnection)
	for _, conn := range toBeProved {
		toBeProvedMap[conn.Id] = gossipConnection{connection: conn}
	}

	return ConnectionManager{
		toBeProvedConnections: toBeProvedMap,
		openConnectionsMap:    make(map[horizontalapi.ConnectionId]gossipConnection),
		powInProgress:         make(map[horizontalapi.ConnectionId]gossipConnection),
	}
}

func (manager *ConnectionManager) ActionOnToBeProved(f func(x gossipConnection)) {
	manager.connMutex.RLock()
	defer manager.connMutex.RUnlock()

	for _, conn := range manager.toBeProvedConnections {
		f(conn)
	}
}

func (manager *ConnectionManager) ActionOnValid(f func(x gossipConnection)) {
	manager.connMutex.RLock()
	defer manager.connMutex.RUnlock()

	for _, conn := range manager.openConnections {
		f(conn)
	}
}

// Function which perform a function f on a permutation of the valid connections.
// Max is the number of elements we want to perform the action on
func (manager *ConnectionManager) ActionOnPermutedValid(f func(x gossipConnection), max int) {
	manager.connMutex.RLock()
	defer manager.connMutex.RUnlock()

	perm := mrand.Perm(len(manager.openConnections))
	amount := min(max, len(manager.openConnections))

	for i := 0; i < amount; i++ {
		idx := perm[i]
		f(manager.openConnections[idx])
	}
}

// To check weather the map has such element, we compare the ID.
// If there is no element, Id will be the zero value
func (manager *ConnectionManager) unsafeFind(id horizontalapi.ConnectionId) gossipConnection {
	// Search in the ToBeProved
	peer := manager.toBeProvedConnections[id]
	if peer.connection.Id == id {
		return peer
	}

	// Search in the In Progress connections
	peer = manager.powInProgress[id]
	if peer.connection.Id == id {
		return peer
	}

	peer = manager.openConnectionsMap[id]
	if peer.connection.Id == id {
		return peer
	}

	// Should I actually return an error? This should never happen, so I am throwing a panic
	fmt.Println("You searched", id, manager.openConnectionsMap)
	panic("Error, gossipConnection searched but not found")
}

func (manager *ConnectionManager) Find(id horizontalapi.ConnectionId) gossipConnection {
	manager.connMutex.RLock()
	defer manager.connMutex.RUnlock()

	return manager.unsafeFind(id)
}

// Function to check weather the ID is of a connection which is to be proved
func (manager *ConnectionManager) IsToBeProved(id horizontalapi.ConnectionId) bool {
	manager.connMutex.RLock()
	defer manager.connMutex.RUnlock()

	// Search in the ToBeProved
	peer := manager.toBeProvedConnections[id]
	if peer.connection.Id == id {
		return true
	}
	return false
}

// Function to check weather the ID is of a connection which is in progress
func (manager *ConnectionManager) IsInProgress(id horizontalapi.ConnectionId) bool {
	manager.connMutex.RLock()
	defer manager.connMutex.RUnlock()

	// Search in the inProgress
	peer := manager.powInProgress[id]
	if peer.connection.Id == id {
		return true
	}
	return false
}

// Function to check weather the ID is of a already valid connection
func (manager *ConnectionManager) IsValid(id horizontalapi.ConnectionId) bool {
	manager.connMutex.RLock()
	defer manager.connMutex.RUnlock()

	peer := manager.openConnectionsMap[id]
	if peer.connection.Id == id {
		return true
	}
	return false
}

// Wrapper around unsafeRemove with locking
func (manager *ConnectionManager) Remove(id horizontalapi.ConnectionId) error {
	manager.connMutex.Lock()
	defer manager.connMutex.Unlock()
	manager.unsafeRemove(id)

	return errors.New("No element found when removing connection")
}

// Remove the connection with a specific ID, without locking resources
func (manager *ConnectionManager) unsafeRemove(id horizontalapi.ConnectionId) error {
	// Remove from ToBeProved if is there
	peer := manager.toBeProvedConnections[id]
	if peer.connection.Id == id {
		delete(manager.toBeProvedConnections, id)
		return nil
	}

	// Remove from the In Progress connections
	peer = manager.powInProgress[id]
	if peer.connection.Id == id {
		delete(manager.powInProgress, id)
		return nil
	}

	// Remove it from openConnectionMap and Slice
	peer = manager.openConnectionsMap[id]
	if peer.connection.Id == id {
		delete(manager.openConnectionsMap, id)

		//remove it from slice as well
		for i, conn := range manager.openConnections {
			if conn.connection.Id == id {
				manager.openConnections[i] = manager.openConnections[len(manager.openConnections)-1]
				manager.openConnections = manager.openConnections[:len(manager.openConnections)-1]
				return nil
			}
		}
	}

	return errors.New("No element found when removing connection")
}

// Move a connection from weather it was and make it valid with the current timestamp
func (manager *ConnectionManager) MakeValid(id horizontalapi.ConnectionId, timestamp time.Time) {
	manager.connMutex.Lock()
	defer manager.connMutex.Unlock()

	peer := manager.unsafeFind(id)
	manager.unsafeRemove(id)

	newConnection := gossipConnection{
		connection: peer.connection,
		timestamp:  timestamp,
	}

	manager.openConnections = append(manager.openConnections, newConnection)
	manager.openConnectionsMap[id] = newConnection
}

func (manager *ConnectionManager) AddInProgress(peer gossipConnection) {
	manager.connMutex.Lock()
	defer manager.connMutex.Unlock()

	manager.powInProgress[peer.connection.Id] = peer
}

// Remove all valid connection on which f return true
func (manager *ConnectionManager) CullConnections(f func(x gossipConnection) bool) {
	manager.connMutex.Lock()
	defer manager.connMutex.Unlock()

	for i, peer := range manager.openConnections {
		if f(peer) {
			//Remove it from the map
			delete(manager.openConnectionsMap, peer.connection.Id)

			//remove it from slice as well
			manager.openConnections[i] = manager.openConnections[len(manager.openConnections)-1]
			manager.openConnections = manager.openConnections[:len(manager.openConnections)-1]

			fmt.Println("Removed a connection since no pow was received in time", "Conn Id", peer.connection.Id)
		}
	}

}
