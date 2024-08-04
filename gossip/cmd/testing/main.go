package main

import (
	"gossip/common"
	"gossip/internal/testutils"
	vtypes "gossip/verticalAPI/types"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

const STARTNODE = 1

func logInit() *slog.Logger {
	return slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}))
}


func main() {
	var err error
	slog.SetDefault(logInit())

	fn := os.Args[1]
	slog.Info("Create tester from file", "file", fn)
	t,err := testutils.NewTesterFromJSON(fn)
	if err != nil {
		panic(err)
	}

	slog.Info("Startup tester")
	if err = t.Startup(); err != nil {
		panic(err)
	}

	// register all peers for message type
	slog.Info("Register all peers for message type", "dataType", 1337)
	for _, p := range t.Peers {
		msg := vtypes.GossipNotify{
			Gn: common.GossipNotify{
				Reserved: 0,
				DataType: 1337,
			},
			MessageHeader: vtypes.MessageHeader{
				Type: vtypes.GossipNotifyType,
			},
		}
		msg.MessageHeader.RecalcSize(&msg)

		if err = p.SendMsg(&msg); err != nil {
			panic(err)
		}
	}

	// send an announcement
	slog.Info("Send an announcement", "node", t.Peers[STARTNODE].String())
	p := t.Peers[STARTNODE]
	msg := vtypes.GossipAnnounce{
		Ga: common.GossipAnnounce{
			TTL:      3,
			Reserved: 0,
			DataType: 1337,
			Data:     []byte{1},
		},
		MessageHeader: vtypes.MessageHeader{
			Type: vtypes.GossipAnnounceType,
		},
	}
	msg.MessageHeader.RecalcSize(&msg)
	if err = p.SendMsg(&msg); err != nil {
		panic(err)
	}

	slog.Info("wait for dissemination", "dur", 60 * time.Second)
	time.Sleep(60 * time.Second)

	slog.Info("teardown test")
	t.Teardown()

	slog.Info("processing the logs -> gen", "file", "reached.css")
	if err = t.ProcessLogsGenReachedWhenCSS("reached.css"); err != nil {
		panic(err)
	}

	slog.Info("processing the logs -> gen", "file", "dist_cnt.csv")
	if err = t.ProcessLogsGenDistCntCSV("dist_cnt.csv", STARTNODE); err != nil {
		panic(err)
	}
	slog.Info("processing the logs -> gen", "file", "dist_reached.csv")
	if err = t.ProcessLogsGenDistReachedCSV("dist_reached.csv", STARTNODE); err != nil {
		panic(err)
	}

	slog.Info("processing the logs -> gen", "file", "packets_sent.csv")
	if err = t.ProcessLogsGenSentPacketsCSV("packets_sent.csv"); err != nil {
		panic(err)
	}
}
