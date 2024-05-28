package horizontalapi

import (
	"context"
	"errors"
	vertTypes "gossip/verticalAPI/types"
	"log/slog"
	"sync"
)

var (
	// Developing reasons
	ErrMethodNotYetImplemented = errors.New("method is not implemented by that specific message type")
)

type MainToHorzChans struct {
	RelayAnnounce chan MainToHorzAnnounce
}

type MainToHorzAnnounce struct {
	Data vertTypes.GossipAnnounce
}

// This struct represents the horizontal api and is the main interface to/from
// the vertical api.
//
// TODO: Since it was not discussed yet, so far the Horizontal Api is a copycat of the Vertical
// one, implementing one toy method just to supply an interface to the main. It does not have
// nore handle connections
type HorizontalApi struct {
	// internally uses a context to signal when the goroutines shall terminate
	cancel context.CancelFunc
	// internally uses a context to signal when the goroutines shall terminate
	ctx context.Context
	// collection of channels from the main package to this package
	mainToHorzChans MainToHorzChans
	// logging for this module
	log *slog.Logger
	// waitgroup to wait for all goroutines to terminate in the end
	wg sync.WaitGroup
}

// Use this function to instantiate the horizontal api
func NewHorizontalApi(log *slog.Logger, mainToHorzChans MainToHorzChans) *HorizontalApi {
	// context is only used internally -> no need to pass it to the constructor
	ctx, cancel := context.WithCancel(context.Background())
	return &HorizontalApi{
		cancel:          cancel,
		ctx:             ctx,
		mainToHorzChans: mainToHorzChans,
		log:             log,
	}
}

// This function will try to spread (relay) horizontally any GossipAnnounce coming from the main
//
// TODO: I tried to model this function as a Listen from Vertical API. Tbh
// it might be better wrapping it in a Start() function (along other functions)
func (hz *HorizontalApi) SpreadMessages() (err error) {
	hz.wg.Add(1)
	go func() {
		defer hz.wg.Done()
		for {

			msg := <-hz.mainToHorzChans.RelayAnnounce
			hz.log.Debug("Spreading announce messages:", "msg", msg.Data)
		}
	}()

	return nil
}

// Empty Close function, clean up connection still has to be done since there are no connection yer :)
func (hz *HorizontalApi) Close() (err error) {
	hz.cancel()
	hz.wg.Wait()
	return nil
}
