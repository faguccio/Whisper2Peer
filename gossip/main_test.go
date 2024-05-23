package main

import (
	"fmt"
	vertTypes "gossip/verticalAPI/types"
	"net"
	"testing"

	_ "gopkg.in/ini.v1"
)

// abstraction for message types
// everything implementing below methods is considered a msgType
type msgType interface {
	Marshal(buf []byte) error
	CalcSize() int
}

// collection of msgType and the associated buffer
// needed because these types do not implement unmarshal at the moment
type tester struct {
	msg  msgType
	buf  []byte
	name string
}

// Test main
func TestMain(test *testing.T) {
	// use table driven testing: https://go.dev/wiki/TableDrivenTests
	// define the messages that should be received via the socket
	ts := []tester{
		{
			&vertTypes.GossipAnnounce{
				MessageHeader: vertTypes.MessageHeader{
					Size: 8 + 2,
					Type: vertTypes.GossipAnnounceType,
				},
				TTL:      32,
				Reserved: 0,
				DataType: 24,
				Data:     []byte{0x20, 0x50},
			},
			[]byte{0x0, 0x0a, 0x01, 0xf4, 32, 0, 0x0, 0x18, 0x20, 0x50},
			"announce",
		},
		{
			&vertTypes.GossipNotify{
				MessageHeader: vertTypes.MessageHeader{
					Size: 8,
					Type: vertTypes.GossipNotifyType,
				},
				Reserved: 0,
				DataType: 42,
			},
			[]byte{0x0, 0x08, 0x01, 0xf5, 0, 0, 0x0, 0x2a},
			"notify",
		},
		{
			&vertTypes.GossipValidation{
				MessageHeader: vertTypes.MessageHeader{
					Size: 8,
					Type: vertTypes.GossipValidationType,
				},
				MessageId: 1337,
				// setting to 0b..1 does not work since the valid flag is not
				// imported (-> or do not use reflect.DeepEqual later)
				Bitfield: 0,
			},
			[]byte{0x0, 0x08, 0x01, 0xf7, 0x05, 0x39, 0, 0},
			"validation",
		},
	}

	go main()

	conn, err := net.Dial("tcp", "localhost:13379")
	if err != nil {
		fmt.Println("Error connecting, should not happen:", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write(ts[0].buf)
	if err != nil {
		fmt.Println("Error sending data:", err)
		return
	}

	// _, cTest := net.Pipe()
	//time.Sleep(5 * time.Second)
	// //vert.conns[cVert] = struct{}{}
	// fmt.Println("Test about to send")

	// if n, err := cTest.Write(ts[0].buf); err != nil {
	// 	test.Fatalf("failed sending: %v", err)
	// } else if n != len(ts[0].buf) {
	// 	test.Fatalf("buffer wasn't sent completely (might need to adjust the test)")
	// }

	// // run one test for each message that should be received -> those can run
	// // in parallel -> speedup for the tests
	// for _, t := range ts {
	// 	t := t // NOTE: /wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
	// 	test.Run(t.name, func(test *testing.T) {
	// 		test.Parallel()
	// 		// use this for logging so that messages are not shown in general,
	// 		// only if the test fails
	// 		var testLog *slog.Logger = slogt.New(test)
	// 		// create the network pipe
	// 		cVert, cTest := net.Pipe()
	// 		vertToMainChans := VertToMainChans{
	// 			Register:   make(chan VertToMainRegister, 1),
	// 			Announce:   make(chan VertToMainAnnounce, 1),
	// 			Validation: make(chan VertToMainValidation, 1),
	// 		}
	// 		// create the vertical api with above setup values
	// 		vert := NewVerticalApi(testLog, vertToMainChans)

	// 		// pretend the connection just got established and all the channels
	// 		// have been created
	// 		vert.conns[cVert] = struct{}{}
	// 		mainToVert := make(chan MainToVertNotification)
	// 		regMod := RegisteredModule{
	// 			MainToVert: mainToVert,
	// 		}
	// 		// start the handler
	// 		go vert.handleConnection(cVert, regMod)

	// 		// write the message to the socket
	// 		if n, err := cTest.Write(t.buf); err != nil {
	// 			test.Fatalf("failed sending: %v %+v", err, t)
	// 		} else if n != len(t.buf) {
	// 			test.Fatalf("buffer wasn't sent completely (might need to adjust the test) %v", t)
	// 		}

	// 		// sleep to make sure the message had time to arrive
	// 		// TODO are 5 seconds too long? (makes these tests a bit slow)
	// 		time.Sleep(5 * time.Second)

	// 		// check all channels on which unmarshaled messages could be sent
	// 		// to by the handler
	// 		// then check if that is the right channel and if the message contains the right information
	// 		select {
	// 		case x := <-vertToMainChans.Validation:
	// 			t, ok := t.msg.(*vertTypes.GossipValidation)
	// 			if !ok {
	// 				test.Fatalf("handler sent to wrong channel")
	// 			}
	// 			if !reflect.DeepEqual(x.Data, *t) {
	// 				test.Fatalf("handler didn't receive the sent message correctly. Was %+v should %+v", x.Data, *t)
	// 			}
	// 		case x := <-vertToMainChans.Announce:
	// 			t, ok := t.msg.(*vertTypes.GossipAnnounce)
	// 			if !ok {
	// 				test.Fatalf("handler sent to wrong channel")
	// 			}
	// 			if !reflect.DeepEqual(x.Data, *t) {
	// 				test.Fatalf("handler didn't receive the sent message correctly. Was %+v should %+v", x.Data, *t)
	// 			}
	// 		case x := <-vertToMainChans.Register:
	// 			t, ok := t.msg.(*vertTypes.GossipNotify)
	// 			if !ok {
	// 				test.Fatalf("handler sent to wrong channel")
	// 			}
	// 			if !reflect.DeepEqual(x.Data, *t) {
	// 				test.Fatalf("handler didn't receive the sent message correctly. Was %+v should %+v", x.Data, *t)
	// 			}
	// 		default:
	// 			// nothing did arrive
	// 			test.Fatalf("handler didn't pass a message to any vertToMain channel")
	// 		}

	// 		// check if the handler also sent additional messages
	// 		if len(vertToMainChans.Announce) != 0 || len(vertToMainChans.Register) != 0 || len(vertToMainChans.Validation) != 0 {
	// 			test.Fatalf("handler sent to many messages on the vertToMain channels")
	// 		}

	// 		// close would pannic because the listener isn't setup correctly -> skip it
	// 		// if err := vert.Close(); err != nil {
	// 		// 	test.Fatalf("failed to close vertical api: %v", err)
	// 		// }
	// 	})
	// }
}
