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

package main

import (
	"context"
	"encoding/csv"
	"gossip/common"
	"gossip/internal/testutils"
	vtypes "gossip/verticalAPI/types"
	"log/slog"
	"os"
	"time"

	"github.com/jszwec/csvutil"
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
	t, err := testutils.NewTesterFromJSON(fn)
	if err != nil {
		panic(err)
	}

	slog.Info("Startup tester")
	if err = t.Startup("127.0.0.1"); err != nil {
		panic(err)
	}

	// register all peers for message type
	slog.Info("Register all peers for message type", "dataType", 1337)
	if err = t.RegisterAllPeersForType(1337); err != nil {
		panic(err)
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

	slog.Info("sleep to see more of the periodic proof-of-work")
	time.Sleep(5 * time.Second)

	slog.Info("wait for dissemination")
	ctx, cfunc := context.WithTimeout(context.Background(), time.Minute)
	defer cfunc()
	// interval is two gossip rounds long
	t.WaitUntilSilent(ctx, true, 0, 2*time.Second)
	time.Sleep(1 * time.Second)

	slog.Info("teardown test")
	t.Teardown()

	slog.Info("processing the logs -> when was which node reached")
	if data, err := t.ProcessReachedWhen(1337, true); err == nil {
		if err = data.WriteCss("reached.css"); err != nil {
			panic(err)
		}
	} else {
		panic(err)
	}

	slog.Info("processing the logs -> which distances are present how often")
	if data, err := t.ProcessGraphDistCnt(STARTNODE); err == nil {
		f, err := os.Create("dist_cnt.csv")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		w := csv.NewWriter(f)
		defer w.Flush()
		enc := csvutil.NewEncoder(w)
		if err := enc.Encode(data); err != nil {
			panic(err)
		}
	} else {
		panic(err)
	}

	slog.Info("processing the logs -> when was which distance reached")
	if data, _, err := t.ProcessReachedDistCnt(STARTNODE, 1337, true); err == nil {
		f, err := os.Create("dist_reached.csv")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		w := csv.NewWriter(f)
		defer w.Flush()
		enc := csvutil.NewEncoder(w)
		if err = enc.Encode(data); err != nil {
			panic(err)
		}
	} else {
		panic(err)
	}

	slog.Info("processing the logs -> how many packets aere sent")
	if data, err := t.ProcessSentPackets(1337, true); err == nil {
		f, err := os.Create("packets_sent.csv")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		w := csv.NewWriter(f)
		defer w.Flush()
		enc := csvutil.NewEncoder(w)
		if err = enc.Encode(data); err != nil {
			panic(err)
		}
	} else {
		panic(err)
	}
}
