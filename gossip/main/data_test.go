package gossip

import (
	"fmt"
	"gossip/common"
	"sync"
	"testing"
)

func TestConcurrentAdding(test *testing.T) {
	store := NewNotifyMap()
	vert_type := common.GossipType(42)

	var modules []*common.RegisteredModule

	for i := 0; i < 100; i++ {
		modules = append(modules, &common.RegisteredModule{
			MainToVert: make(chan common.ToVert),
		})
	}

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		i := i
		go func() {
			defer wg.Done()
			store.AddChannelToType(vert_type, &common.Conn[common.RegisteredModule]{Data: *modules[i], Id: common.ConnectionId(fmt.Sprint(i))})
		}()
	}

	// otherwise the insertion below can happen before the one above -> error
	// is returned in invocation in for loop, not in the invocation below =>
	// test fails
	wg.Wait()

	err := store.AddChannelToType(vert_type, &common.Conn[common.RegisteredModule]{Data: *modules[5], Id: common.ConnectionId(fmt.Sprint(5))})

	if err == nil {
		test.Fatalf("Registered error multiple times but no error returned")

	}

	registered := store.Load(vert_type)
	if len(registered) != 100 {
		test.Fatalf("A race condition happend while doing concurrent writes to the type store")
	}
}

func TestRemoveChannel(t *testing.T) {
	var wg sync.WaitGroup
	errorCh := make(chan error) // Buffer size for potential errors

	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store := NewNotifyMap()

			var modules []*common.RegisteredModule
			vert_type1 := common.GossipType(42)
			vert_type2 := common.GossipType(420)

			for i := 0; i < 3; i++ {
				modules = append(modules, &common.RegisteredModule{
					MainToVert: make(chan common.ToVert),
				})
			}

			store.AddChannelToType(vert_type1, &common.Conn[common.RegisteredModule]{Data: *modules[0], Id: "a"})
			store.AddChannelToType(vert_type1, &common.Conn[common.RegisteredModule]{Data: *modules[1], Id: "b"})
			store.AddChannelToType(vert_type1, &common.Conn[common.RegisteredModule]{Data: *modules[1], Id: "b"})
			store.AddChannelToType(vert_type2, &common.Conn[common.RegisteredModule]{Data: *modules[2], Id: "c"})
			store.AddChannelToType(vert_type2, &common.Conn[common.RegisteredModule]{Data: *modules[1], Id: "b"})

			store.RemoveChannel("b")

			res1 := store.Load(vert_type1)
			for _, conn := range res1 {
				if conn.Id == "b" {
					errorCh <- fmt.Errorf("Connection was not deleted from registerd type %v", vert_type1)
					return
				}
			}

			res2 := store.Load(vert_type2)
			for _, conn := range res2 {
				if conn.Id == "b" {
					errorCh <- fmt.Errorf("Connection was not deleted from registerd type %v", vert_type2)
					return
				}
			}
		}()
	}

	done := make(chan struct{}) // Channel to signal when the waitgroup is done
	go func() {
		defer close(done)
		wg.Wait()
		close(errorCh)
	}()

	for {
		select {
		case err := <-errorCh:
			if err != nil {
				t.Fatal(err)
			}
		case <-done:
			return // Exit when all goroutines have finished
		}
	}
}
