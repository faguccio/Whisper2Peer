package strats

import (
	"fmt"
	horizontalapi "gossip/horizontalAPI"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestConnectoinManager(test *testing.T) {
	conns := make([]horizontalapi.Conn[chan<- horizontalapi.ToHz], 4)
	for i, _ := range conns {
		conns[i].Id = horizontalapi.ConnectionId("ciao" + fmt.Sprint(i))
	}
	manager := NewConnectionManager(conns)

	var ids string
	manager.ActionOnToBeProved(func(x *gossipConnection) {
		ids += string(x.connection.Id)
	})

	if !strings.Contains(ids, "ciao0") {
		test.Fatalf("Action on to be proved behaves in a wrong manner")
	}

	ciao0firstTime := time.Now()
	manager.MakeValid("ciao0", ciao0firstTime)

	var validConn *gossipConnection
	manager.ActionOnValid(func(x *gossipConnection) {
		validConn = x
	})

	if validConn.connection.Id != "ciao0" {
		test.Fatalf("Performing action on valid behaves in a wrong manner")
	}

	manager.MakeValid("ciao1", time.Now())

	value, _ := manager.FindValid("ciao0")
	if !reflect.DeepEqual(value, validConn) {
		test.Fatalf("Performing action on valid behaves in a wrong manner")
	}

	ids = ""
	manager.ActionOnPermutedValid(func(x *gossipConnection) {
		ids += string(x.connection.Id)
	}, 2)

	if ids != "ciao0ciao1" && ids != "ciao1ciao0" {
		test.Fatalf("Performing action on permuted valid behaves wrongly")
	}

	_, ok := manager.FindToBeProved("ciao0")
	if ok {
		test.Fatalf("Connection should be valid but found in to be proved")
	}

	manager.MakeValid("ciao0", time.Now())
	value, _ = manager.FindValid("ciao0")

	if time.Now().Sub(ciao0firstTime) <= 0 {
		test.Fatalf("Second MakeValid does not refresh timestamp")
	}

	ciao4 := gossipConnection{
		connection: horizontalapi.Conn[chan<- horizontalapi.ToHz]{Id: horizontalapi.ConnectionId("ciao4")},
	}

	manager.AddInProgress(&ciao4)
	value, ok = manager.FindInProgress("ciao4")

	if !ok {
		test.Fatalf("Add in progress failed or faulty FindInProgress")
	}

	// Should remove all valid connections since this whole test should take less than 2 seconds
	manager.CullConnections(func(x *gossipConnection) bool {
		return time.Now().Sub(x.timestamp) < time.Second*2
	})

	if len(manager.openConnections) > 0 {
		test.Fatalf("Cull Connections did not remove the connections")
	}

}
