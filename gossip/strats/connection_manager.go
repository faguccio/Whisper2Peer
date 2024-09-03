package strats

import (
	"errors"
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

// Some function are marked as unsafe. This means that they do not provide locking. Whoever is calling
// them should make sure to lock the mutex, weather it is for writing or for reading.
// They are needed so more complex function can chain them after

// Return a new instance of the Connection Manager.
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

// Perform a function f on every To Be Proved connection
func (manager *ConnectionManager) ActionOnToBeProved(f func(x gossipConnection)) {
	manager.connMutex.RLock()
	defer manager.connMutex.RUnlock()

	for _, conn := range manager.toBeProvedConnections {
		f(conn)
	}
}

// Perform an function f on every valid (open) connection
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

// Returns the connection with matching ID from the to be proved connections and a boolean indicating
// the presence of the value
func (manager *ConnectionManager) FindToBeProved(id horizontalapi.ConnectionId) (gossipConnection, bool) {
	manager.connMutex.RLock()
	defer manager.connMutex.RUnlock()

	value, ok := manager.toBeProvedConnections[id]
	return value, ok
}

// Returns the connection with matching ID from the to be in progress connections and a boolean indicating
// the presence of the value
func (manager *ConnectionManager) FindInProgress(id horizontalapi.ConnectionId) (gossipConnection, bool) {
	manager.connMutex.RLock()
	defer manager.connMutex.RUnlock()

	value, ok := manager.powInProgress[id]
	return value, ok
}

// Returns the connection with matching ID from the to valid (open) connections and a boolean indicating
// the presence of the value
func (manager *ConnectionManager) FindValid(id horizontalapi.ConnectionId) (gossipConnection, bool) {
	manager.connMutex.RLock()
	defer manager.connMutex.RUnlock()

	value, ok := manager.openConnectionsMap[id]
	return value, ok
}

// Find the connection with matching ID, else return the zero value
func (manager *ConnectionManager) unsafeFind(id horizontalapi.ConnectionId) gossipConnection {
	// Search in the ToBeProved
	peer, ok := manager.toBeProvedConnections[id]
	if ok {
		return peer
	}

	// Search in the In Progress connections
	peer, ok = manager.powInProgress[id]
	if ok {
		return peer
	}

	// Search in the open (valid) connections
	peer, ok = manager.openConnectionsMap[id]
	if ok {
		return peer
	}

	return gossipConnection{}
}

// Wrapper around unsafeRemove with locking for thread safety
func (manager *ConnectionManager) Remove(id horizontalapi.ConnectionId) (gossipConnection, error) {
	manager.connMutex.Lock()
	defer manager.connMutex.Unlock()
	return manager.unsafeRemove(id)
}

// Remove the connection with a specific ID, without locking resources
func (manager *ConnectionManager) unsafeRemove(id horizontalapi.ConnectionId) (gossipConnection, error) {
	// Remove from ToBeProved if is there
	peer, ok := manager.toBeProvedConnections[id]
	if ok {
		delete(manager.toBeProvedConnections, id)
		return peer, nil
	}

	// Remove from the In Progress connections
	peer, ok = manager.powInProgress[id]
	if ok {
		delete(manager.powInProgress, id)
		return peer, nil
	}

	// Remove it from openConnectionMap and Slice
	peer, ok = manager.openConnectionsMap[id]
	if ok {
		delete(manager.openConnectionsMap, id)

		//remove it from slice as well
		for i, conn := range manager.openConnections {
			if conn.connection.Id == id {
				manager.openConnections[i] = manager.openConnections[len(manager.openConnections)-1]
				manager.openConnections = manager.openConnections[:len(manager.openConnections)-1]
				return peer, nil
			}
		}
	}

	return gossipConnection{}, errors.New("No element found when removing connection")
}

// Move a connection from weather it was and make it valid with the current timestamp
func (manager *ConnectionManager) MakeValid(id horizontalapi.ConnectionId, timestamp time.Time) {
	manager.connMutex.Lock()
	defer manager.connMutex.Unlock()

	peer := manager.unsafeFind(id)
	manager.unsafeRemove(id)

	peer.timestamp = timestamp

	manager.openConnections = append(manager.openConnections, peer)
	manager.openConnectionsMap[id] = peer

}

// Add a gossip connection to the In Progress ones
func (manager *ConnectionManager) AddInProgress(peer gossipConnection) {
	manager.connMutex.Lock()
	defer manager.connMutex.Unlock()

	manager.powInProgress[peer.connection.Id] = peer
}

// Remove all valid connection on which f return true
func (manager *ConnectionManager) CullConnections(f func(x gossipConnection) bool) {
	manager.connMutex.Lock()
	defer manager.connMutex.Unlock()

	toRemove := make([]int, 0)
	for i, peer := range manager.openConnections {
		if f(peer) {
			//Remove it from the map
			delete(manager.openConnectionsMap, peer.connection.Id)
			toRemove = append(toRemove, i)
		}
	}

	for i := len(toRemove)-1; i >= 0; i-- {
		//remove it from slice as well
		manager.openConnections[toRemove[i]] = manager.openConnections[len(manager.openConnections)-1]
		manager.openConnections = manager.openConnections[:len(manager.openConnections)-1]
	}

}
