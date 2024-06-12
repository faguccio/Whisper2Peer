package main

import (
	"fmt"
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

func TestExceedingCache(test *testing.T) {
	store := NewNotifyMap(3)

	for i := 0; i < 3; i++ {
		vert_type := vertTypes.GossipType(42 + i)
		module := &verticalapi.RegisteredModule{
			MainToVert: make(chan verticalapi.MainToVertNotification),
		}

		store.AddChannelToType(vert_type, module)
	}

	registered := store.Load(42)
	fmt.Println(registered)
	if len(registered) != 1 {
		test.Fatalf("Error while registering module, %d channels are registered, instead of 1", len(registered))
	}

	vert_type := vertTypes.GossipType(420)
	module := &verticalapi.RegisteredModule{
		MainToVert: make(chan verticalapi.MainToVertNotification),
	}

	store.AddChannelToType(vert_type, module)

	registered = store.Load(42)
	fmt.Println(registered)
	if len(registered) != 0 {
		test.Fatalf("Exceding cache should delete old registered type, but associated channel can still be found")
	}
}
