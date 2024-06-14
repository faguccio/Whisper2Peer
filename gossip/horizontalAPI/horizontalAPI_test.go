package horizontalapi

import (
	"log/slog"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/neilotoole/slogt"
)

func TestGorizontalApiWithPipe(test *testing.T) {
	// use this for logging so that messages are not shown in general,
	// only if the test fails
	var testLog *slog.Logger = slogt.New(test)
	// var testLog *slog.Logger = slog.Default()

	toHz := make(chan ToHz, 1)
	fromHz := make(chan FromHz, 1)
	// create the vertical api with above setup values
	hz := NewHorizontalApi(testLog, fromHz)
	defer func() {
		hz.cancel()
		hz.wg.Wait()
	}()

	cWrite, cRead := net.Pipe()
	defer cRead.Close()

	hz.wg.Add(2)
	go hz.handleConnection(cRead, toHz)
	go hz.writeToConnection(cWrite, toHz)

	// send a notify message to register to the server and get the mainToVert
	// channel in return
	t := Push{
		TTL:        42,
		GossipType: 10,
		MessageID:  99,
		Payload:    []byte{0x20, 0x40},
	}

	// send message
	testLog.Info("sending", "msg", t)
	toHz <- t
	// receive message
	var u FromHz
	select {
	case u = <-fromHz:
	case <-time.After(5 * time.Second):
		test.Fatalf("timeout for reading the to be received message after 5 seconds")
	}

	switch u := u.(type) {
	case Push:
		if !reflect.DeepEqual(t, u) {
			test.Fatalf("didn't reveice the message previously sent. Sent %+v rcved%+v", t, u)
		}
	}
}

func TestHorizontalApi(test *testing.T) {
	// use this for logging so that messages are not shown in general,
	// only if the test fails
	var testLog *slog.Logger = slogt.New(test)
	// var testLog *slog.Logger = slog.Default()

	fromHz1 := make(chan FromHz, 1)
	connHz1 := make(chan NewConn, 1)
	// create the vertical api with above setup values
	hz1 := NewHorizontalApi(testLog, fromHz1)
	defer hz1.Close()
	if err := hz1.Listen("localhost:13377", connHz1); err != nil {
		test.Fatalf("listen on horizontalApi 1 failed with %v", err)
	}

	fromHz2 := make(chan FromHz, 1)
	connHz2 := make(chan NewConn, 1)
	// create the vertical api with above setup values
	hz2 := NewHorizontalApi(testLog, fromHz2)
	defer hz2.Close()
	if err := hz2.Listen("localhost:13378", connHz2); err != nil {
		test.Fatalf("listen on horizontalApi 2 failed with %v", err)
	}

	ns, err := hz1.AddNeighbors("localhost:13378")
	if err != nil {
		test.Fatalf("adding neighbor for horizontalApi 1 failed with %v", err)
	}

	if len(ns) != 1 {
		test.Fatalf("adding neighbor returned wrong amount of channels (was %d, should: %d)", len(ns), 1)
	}

	// send a notify message to register to the server and get the mainToVert
	// channel in return
	t := Push{
		TTL:        42,
		GossipType: 10,
		MessageID:  99,
		Payload:    []byte{0x20, 0x40},
	}

	// send message
	testLog.Info("sending", "msg", t)
	ns[0] <- t

	// receive message
	var u FromHz
	select {
	case u = <-fromHz2:
	case <-time.After(1 * time.Second):
		test.Fatalf("timeout for reading the to be received message after 1 second")
	}

	switch u := u.(type) {
	case Push:
		if !reflect.DeepEqual(t, u) {
			test.Fatalf("didn't reveice the message previously sent. Sent %+v rcved%+v", t, u)
		}
	default:
		test.Fatalf("received message is of wrong type")
	}

	// sleep to make sure the connections are established
	time.Sleep(2 * time.Second)
	// check channels passed on listen for newly established channels
	_ = <-connHz2
	select {
	case <-connHz1:
		test.Fatalf("connHz1 should only receive exactly zero channels")
	case <-connHz2:
		test.Fatalf("connHz2 should only receive exactly one channel")
	default:
	}
}
