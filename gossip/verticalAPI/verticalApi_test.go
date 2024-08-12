package verticalapi

import (
	"errors"
	"gossip/common"
	"io"
	"log/slog"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/neilotoole/slogt"
)

// abstraction for message types
// everything implementing below methods is considered a msgType
type msgType interface {
	// Marshal(buf []byte) error
	// CalcSize() int
}

// collection of msgType and the associated buffer
// needed because these types do not implement unmarshal at the moment
type tester struct {
	msg  msgType
	buf  []byte
	name string
}

// test for internal handleConnection function
// idea is to simulate the network via a net.Pipe -> isolated -> tests can run
// in parallel
func TestHandleConnection(test *testing.T) {
	test.Parallel()
	// use table driven testing: https://go.dev/wiki/TableDrivenTests
	// define the messages that should be received via the socket
	ts := []tester{
		{
			msg: common.GossipAnnounce{
				TTL:      32,
				Reserved: 0,
				DataType: 24,
				Data:     []byte{0x20, 0x50},
			},
			buf:  []byte{0x0, 0x0a, 0x01, 0xf4, 32, 0, 0x0, 0x18, 0x20, 0x50},
			name: "announce",
		},
		{
			msg: common.GossipNotify{
				Reserved: 0,
				DataType: 42,
			},
			buf:  []byte{0x0, 0x08, 0x01, 0xf5, 0, 0, 0x0, 0x2a},
			name: "notify",
		},
		{
			msg: common.GossipValidation{
				MessageId: 1337,
				// setting to 0b..1 does not work since the valid flag is not
				// imported (-> or do not use reflect.DeepEqual later)
				Bitfield: 0,
			},
			buf:  []byte{0x0, 0x08, 0x01, 0xf7, 0x05, 0x39, 0, 0},
			name: "validation",
		},
	}

	// run one test for each message that should be received -> those can run
	// in parallel -> speedup for the tests
	for _, t := range ts {
		t := t // NOTE: /wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		test.Run(t.name, func(test *testing.T) {
			test.Parallel()
			// use this for logging so that messages are not shown in general,
			// only if the test fails
			var testLog *slog.Logger = slogt.New(test)
			// create the network pipe
			cVert, cTest := net.Pipe()
			vertToMainChan := make(chan common.FromVert)
			// create the vertical api with above setup values
			vert := NewVerticalApi(testLog, vertToMainChan)

			// pretend the connection just got established and all the channels
			// have been created
			vert.conns[cVert] = struct{}{}
			mainToVert := make(chan common.ToVert)
			regMod := common.RegisteredModule{
				MainToVert: mainToVert,
			}
			// start the handler
			go vert.handleConnection(cVert, common.Conn[common.RegisteredModule]{Data: regMod})

			// write the message to the socket
			if n, err := cTest.Write(t.buf); err != nil {
				test.Fatalf("failed sending: %v %+v", err, t)
			} else if n != len(t.buf) {
				test.Fatalf("buffer wasn't sent completely (might need to adjust the test) %v", t)
			}

			// sleep to make sure the message had time to arrive
			time.Sleep(1 * time.Second)

			// check all channels on which unmarshaled messages could be sent
			// to by the handler
			// then check if that is the right channel and if the message contains the right information
			select {
			case msg := <-vertToMainChan:
				switch x := msg.(type) {
				case common.GossipValidation:
					t, ok := t.msg.(common.GossipValidation)
					if !ok {
						test.Fatalf("handler sent to wrong channel")
					}
					if !reflect.DeepEqual(x, t) {
						test.Fatalf("handler didn't receive the sent message correctly. Was %+v should %+v", x, t)
					}
				case common.GossipAnnounce:
					t, ok := t.msg.(common.GossipAnnounce)
					if !ok {
						test.Fatalf("handler sent to wrong channel")
					}
					if !reflect.DeepEqual(x, t) {
						test.Fatalf("handler didn't receive the sent message correctly. Was %+v should %+v", x, t)
					}
				case common.GossipRegister:
					t, ok := t.msg.(common.GossipNotify)
					if !ok {
						test.Fatalf("handler sent to wrong channel")
					}
					if !reflect.DeepEqual(x.Data, t) {
						test.Fatalf("handler didn't receive the sent message correctly. Was %+v should %+v", x.Data, t)
					}
				}
			default:
				// nothing did arrive
				test.Fatalf("handler didn't pass a message to any vertToMain channel")
			}

			// check if the handler also sent additional messages
			if len(vertToMainChan) != 0 {
				test.Fatalf("handler sent to many messages on the vertToMain channel")
			}

			// close would pannic because the listener isn't setup correctly -> skip it
			// if err := vert.Close(); err != nil {
			// 	test.Fatalf("failed to close vertical api: %v", err)
			// }
		})
	}
}

// test the writeHandler in an isolated way
// again as with the other handler, we're using net.Pipe as network stub which
// allows for isolation and run in parallel
func TestWriteToConnection(test *testing.T) {
	test.Parallel()
	test.Run("notification", func(test *testing.T) {
		test.Parallel()
		// use this for logging so that messages are not shown in general,
		// only if the test fails
		var testLog *slog.Logger = slogt.New(test)
		// create the network pipe
		cVert, cTest := net.Pipe()
		vertToMainChan := make(chan common.FromVert)
		// create the vertical api with above setup values
		vert := NewVerticalApi(testLog, vertToMainChan)

		// start the handler
		mainToVert := make(chan common.ToVert, 1)
		go vert.writeToConnection(cVert, common.Conn[<-chan common.ToVert]{Data: mainToVert})

		// send a message to the handler which shall be sent on the network
		mainToVert <- common.GossipNotification{
			MessageId: 1337,
			DataType:  42,
			Data:      []byte{0x50, 0x20},
		}
		bufReal := []byte{0x0, 0x0a, 0x01, 0xf6, 0x05, 0x39, 0x0, 0x2a, 0x50, 0x20}

		// allow the message arrive within 1 second to avoid hanging up the test
		if err := cTest.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			test.Fatalf("Setting readDeadline failed: %v", err)
		}
		// receive the message sent on the network
		buf := make([]byte, len(bufReal))
		if n, err := io.ReadFull(cTest, buf); err != nil {
			test.Fatalf("Error reading from network. %v", err)
		} else if n != len(buf) {
			test.Fatalf("read too few bytes from the line. was: %d should: %d", n, len(buf))
		}
		if !reflect.DeepEqual(buf, bufReal) {
			test.Fatalf("Sent buffer is wrong. was: %v should: %v", buf, bufReal)
		}

		// check if an additional message was sent
		if err := cTest.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			test.Fatalf("Setting readDeadline failed: %v", err)
		}
		buf = make([]byte, 1)
		if n, err := cTest.Read(buf); err == nil || !errors.Is(err, os.ErrDeadlineExceeded) || n != 0 {
			test.Fatalf("There shouldn't be any data left on the socket: %v", err)
		}
	})
}

// mostly a combined version of the other two tests which also tests the tcp
// server and has a more black-box approach
func TestVerticalApi(test *testing.T) {
	// use this for logging so that messages are not shown in general,
	// only if the test fails
	var testLog *slog.Logger = slogt.New(test)
	vertToMainChan := make(chan common.FromVert, 1)
	// create the vertical api with above setup values
	vert := NewVerticalApi(testLog, vertToMainChan)
	// start the server on localhost:13377

	initFin := make(chan struct{}, 1)
	if err := vert.Listen("localhost:13377", initFin); err != nil {
		test.Fatalf("Error starting server: %v", err)
	}
	<-initFin

	// establish a connection to the server
	cTest, err := net.Dial("tcp", "localhost:13377")
	if err != nil {
		test.Fatalf("Error connecting to server: %v", err)
	}

	// send a notify message to register to the server and get the mainToVert
	// channel in return
	t := tester{
		msg: common.GossipNotify{
			Reserved: 0,
			DataType: 42,
		},
		buf:  []byte{0x0, 0x08, 0x01, 0xf5, 0, 0, 0x0, 0x2a},
		name: "notify",
	}

	// write the message to the network
	if n, err := cTest.Write(t.buf); err != nil {
		test.Fatalf("failed sending: %v %+v", err, t)
	} else if n != len(t.buf) {
		test.Fatalf("buffer wasn't sent completely (might need to adjust the test) %v", t)
	}

	// sleep to make sure the message had time to arrive
	time.Sleep(1 * time.Second)

	// read the message sent to the mainModule
	var reg common.GossipRegister
	select {
	case msg := <-vertToMainChan:
		var ok bool
		reg, ok = msg.(common.GossipRegister)
		if !ok {
			test.Fatalf("did not receive a register message")
		}
		t, ok := t.msg.(common.GossipNotify)
		if !ok {
			test.Fatalf("handler sent to wrong channel")
		}
		if !reflect.DeepEqual(reg.Data, t) {
			test.Fatalf("handler didn't receive the sent message correctly. Was %+v should %+v", reg.Data, t)
		}
	default:
		test.Fatalf("handler didn't pass a message to the right vertToMain channel")
	}
	// check if additional/other messages got sent as well
	if len(vertToMainChan) != 0 {
		test.Fatalf("handler sent to many messages on the vertToMain channel")
	}

	// send a message on the newly established channel and check if it is
	// written to the network
	reg.Module.Data.MainToVert <- common.GossipNotification{
		MessageId: 1337,
		DataType:  42,
		Data:      []byte{0x50, 0x20},
	}
	bufReal := []byte{0x0, 0x0a, 0x01, 0xf6, 0x05, 0x39, 0x0, 0x2a, 0x50, 0x20}

	// allow the message arrive within 1 second to avoid hanging up the test
	if err := cTest.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		test.Fatalf("Setting readDeadline failed: %v", err)
	}
	// receive the message sent on the network
	buf := make([]byte, len(bufReal))
	if n, err := io.ReadFull(cTest, buf); err != nil {
		test.Fatalf("Error reading from network. %v", err)
	} else if n != len(buf) {
		test.Fatalf("read too few bytes from the line. was: %d should: %d", n, len(buf))
	}
	// check if it is the right message
	if !reflect.DeepEqual(buf, bufReal) {
		test.Fatalf("Sent buffer is wrong. was: %v should: %v", buf, bufReal)
	}

	// check if additional messages also get send
	if err := cTest.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
		test.Fatalf("Setting readDeadline failed: %v", err)
	}
	buf = make([]byte, 1)
	if n, err := cTest.Read(buf); err == nil || !errors.Is(err, os.ErrDeadlineExceeded) || n != 0 {
		test.Fatalf("There shouldn't be any data left on the socket: %v", err)
	}

	// terminate the server (this time the server got properly setup)
	if err := vert.Close(); err != nil {
		test.Fatalf("Failed to close server: %v", err)
	}

	// sleep to make sure the message had time to arrive
	time.Sleep(1 * time.Second)

	// check if the unregistering message was sent
	select {
	case msg := <-vertToMainChan:
		var ok bool
		reg, ok := msg.(common.GossipUnRegister)
		if !ok {
			test.Fatalf("did not receive an unregister message")
		}
		t := common.GossipUnRegister(cTest.LocalAddr().String())
		if !reflect.DeepEqual(reg, t) {
			test.Fatalf("handler didn't receive the sent message correctly. Was %+v should %+v", reg, t)
		}
	default:
		test.Fatalf("no unregister message sent on closing")
	}
}
