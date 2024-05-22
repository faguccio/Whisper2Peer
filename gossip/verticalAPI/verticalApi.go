package verticalapi

import (
	"context"
	"log/slog"
	"net"
)

// This struct represents the vertical api and is the main interface to/from
// the vertical api.
//
// The struct contains various internal fields, thus it should only be created
// by using the [NewVerticalApi] function!
//
// To close and cleanup the VerticalApi, the [VerticalApi.Close] method shall be called
// exactly once.
type VerticalApi struct {
	// internally uses a context to signal when the goroutines shall terminate
	cancel context.CancelFunc
	// internally uses a context to signal when the goroutines shall terminate
	ctx context.Context
	// store the listener so that it can be closed in the end
	ln net.Listener
	// store all open connections so that they can be closed in the end
	conns map[net.Conn]struct{}
	// collection of channels for the backchannel to the main package
	vertToMainChans VertToMainChans
	// logging for this module
	log *slog.Logger
}

// Use this function to instanciate the vertical api
//
// The vertToMainChans serve as backchannel to the main package. Depending on
// what message type was received, the message struct will be sent on the
// respective channel.
//
// The methods of this module all will use the logger passed here. You can use
// the [pkg/log/slog.With] function or [pkg/log/slog.Logger.With] on a
// slog-logger to set a field for all logged entries (like "module"="vertAPI").
func NewVerticalApi(log *slog.Logger, vertToMainChans VertToMainChans) *VerticalApi {
	// context is only used internally -> no need to pass it to the constructor
	ctx, cancel := context.WithCancel(context.Background())
	return &VerticalApi{
		cancel:          cancel,
		ctx:             ctx,
		ln:              nil,
		conns:           make(map[net.Conn]struct{}, 0),
		vertToMainChans: vertToMainChans,
		log:             log,
	}
}

// Listen on the specified address for incoming vertical api connections.
//
// This function spawns a new goroutine accepting new connections and
// terminates afterwards.
func (v *VerticalApi) Listen(addr string) (err error) {
	return
}

// Close the vertical api
//
// Always tries to close the listener and all the connections. If multiple
// fail, this function only returns the last error.
func (v *VerticalApi) Close() (err error) {
	return
}
