/*
 * gossip
 * Copyright (C) 2024 Fabio Gaiba and Lukas Heindl
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package horizontalapi

import (
	"context"
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
	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()

	hz.wg.Add(2)
	go hz.handleConnection(cRead, Conn[chan<- ToHz]{Data: toHz, Ctx: ctx, Cfunc: cfunc})
	go hz.writeToConnection(cWrite, Conn[<-chan ToHz]{Data: toHz, Ctx: ctx, Cfunc: cfunc})

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
	// create the vertical api with above setup values
	hz1 := NewHorizontalApi(testLog, fromHz1)
	initFin := make(chan struct{}, 1)
	if err := hz1.Listen("localhost:13376", initFin); err != nil {
		test.Fatalf("listen on horizontalApi 1 failed with %v", err)
	}
	defer hz1.Close()
	<-initFin

	fromHz2 := make(chan FromHz, 1)
	// create the vertical api with above setup values
	hz2 := NewHorizontalApi(testLog, fromHz2)
	initFin2 := make(chan struct{}, 1)
	if err := hz2.Listen("localhost:13378", initFin2); err != nil {
		test.Fatalf("listen on horizontalApi 2 failed with %v", err)
	}
	defer hz2.Close()
	<-initFin2

	ns, err := hz1.AddNeighbors(&net.Dialer{}, "localhost:13378")
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
	ns[0].Data <- t

	push := 0
	newConn := 0

	// receive first message
	var u FromHz
	select {
	case u = <-fromHz2:
	case <-time.After(1 * time.Second):
		test.Fatalf("timeout for reading the to be received message after 1 second")
	}

	switch u := u.(type) {
	case Push:
		u.Id = ""
		if !reflect.DeepEqual(t, u) {
			test.Fatalf("didn't reveice the message previously sent. Sent %+v rcved%+v", t, u)
		}
		push += 1
	case NewConn:
		newConn += 1
	default:
		test.Fatalf("received message is of wrong type")
	}

	// receive second message
	select {
	case u = <-fromHz2:
	case <-time.After(1 * time.Second):
		test.Fatalf("timeout for reading the to be received message after 1 second")
	}

	switch u := u.(type) {
	case Push:
		u.Id = ""
		if !reflect.DeepEqual(t, u) {
			test.Fatalf("didn't reveice the message previously sent. Sent %+v rcved%+v", t, u)
		}
		push += 1
	case NewConn:
		newConn += 1
	default:
		test.Fatalf("received message is of wrong type")
	}

	if newConn != 1 || push != 1 {
		test.Fatalf("did not receive one push (%d) and one newConn (%d) message", push, newConn)
	}

	// sleep to make sure the connections are established
	time.Sleep(2 * time.Second)
	// check channels passed on listen for newly established channels
	select {
	case <-fromHz1:
		test.Fatalf("fromHz1 received too often")
	case <-fromHz2:
		test.Fatalf("fromHz2 received too often")
	default:
	}
}
