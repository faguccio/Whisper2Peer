package main

import (
	verticalapi "gossip/verticalAPI"
	vertTypes "gossip/verticalAPI/types"
	"testing"
	"time"
)

func TestConcurrentLoading(test *testing.T) {
	store := NewNotifyMap(100)
	vert_type := vertTypes.GossipType(42)

	var modules []*verticalapi.RegisteredModule

	for i := 0; i < 100; i++ {
		modules = append(modules, &verticalapi.RegisteredModule{
			MainToVert: make(chan verticalapi.MainToVertNotification),
		})
	}

	for i := 0; i < 100; i++ {
		go func() {
			store.AddChannelToType(vert_type, modules[i])
		}()
	}

	time.Sleep(20 * time.Millisecond)

	registered := store.Load(vert_type)
	if len(registered) != 100 {
		test.Fatalf("A race condition happend while doing concurrent writes to the type store")
	}
}
