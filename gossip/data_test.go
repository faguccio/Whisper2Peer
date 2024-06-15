package main

import (
	"gossip/common"
	"testing"
	"time"
)

func TestConcurrentLoading(test *testing.T) {
	store := NewNotifyMap()
	vert_type := common.GossipType(42)

	var modules []*common.RegisteredModule

	for i := 0; i < 100; i++ {
		modules = append(modules, &common.RegisteredModule{
			MainToVert: make(chan common.ToVert),
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
