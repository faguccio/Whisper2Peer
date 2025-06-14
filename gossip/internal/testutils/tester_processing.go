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

package testutils

import (
	"errors"
	"gossip/common"
	"gossip/internal/testutils/data"
	"slices"
	"time"
)

func (t *Tester) ProcessReachedWhen(gtype common.GossipType, any bool) (data.ReachedWhenAll, error) {
	ret := make(data.ReachedWhenAll)
	if t.state != TestStateProcessing {
		return ret, errors.New("cannot do processing if tester is not in processing state")
	}

	for _, e := range t.Events {
		if !(e.Msg == "received" || e.Msg == "announce") {
			continue
		}
		if !(any || gtype == e.MsgType) {
			continue
		}
		// calculate when the node received the message, as percantage of the test duration
		tAbs := float64(e.Time.UnixMilli()-t.tmin.UnixMilli()) / 1000
		nodeIdx := t.PeersLut[e.Id]

		if _, ok := ret[nodeIdx]; !ok {
			ret[nodeIdx] = data.ReachedWhen{
				TimeUnixSec: tAbs,
			}
		}
	}

	mi := 1000.0
	ma := 0.0
	for _, v := range ret {
		if v.TimeUnixSec < mi {
			mi = v.TimeUnixSec
		}
		if v.TimeUnixSec > ma {
			ma = v.TimeUnixSec
		}
	}
	dur := ma - mi

	for k, v := range ret {
		v.TimeUnixSec -= mi
		v.TimePercent = v.TimeUnixSec / dur * 100
		ret[k] = v
	}

	return ret, nil
}

func (t *Tester) ProcessReachedDistCnt(startNode uint, gtype common.GossipType, all bool) (data.ReachedDistCntAll, map[uint]uint, error) {
	ret := make(data.ReachedDistCntAll, 0)
	if t.state != TestStateProcessing {
		return ret, nil, errors.New("cannot do processing if tester is not in processing state")
	}

	// setup for distances
	distCnt := t.distanceBook.processingSetupForDistance(func(u uint) map[uint]uint { return t.G.CalcDistances(u) }, startNode)

	for _, d := range t.distanceBook.distOrd {
		ret = append(ret, data.ReachedDistCnt{
			TimeUnixSec:            0,
			Distance:               d,
			CntReachedSameDistance: 0,
		})
	}

	for _, e := range t.Events {
		if !(e.Msg == "received" || e.Msg == "announce") {
			continue
		}
		if !(all || gtype == e.MsgType) {
			continue
		}
		// calculate when the node received the message, as percantage of the test duration
		tAbs := float64(e.Time.UnixMilli()-t.tmin.UnixMilli()) / 1000
		nodeIdx := t.PeersLut[e.Id]
		dist := t.distanceBook.nodeToDist[nodeIdx]
		distCnt[dist] += 1

		ret = append(ret, data.ReachedDistCnt{
			TimeUnixSec:            tAbs,
			Distance:               dist,
			CntReachedSameDistance: distCnt[dist],
		})
	}

	for _, d := range t.distanceBook.distOrd {
		ret = append(ret, data.ReachedDistCnt{
			TimeUnixSec:            t.durSec,
			Distance:               d,
			CntReachedSameDistance: distCnt[d],
		})
	}

	return ret, distCnt, nil
}

// get how many nodes exist with a specific distance
func (t *Tester) ProcessGraphDistCnt(startNode uint) (data.CntDistancesAll, error) {
	ret := make(data.CntDistancesAll, 0)
	if t.state != TestStateProcessing {
		return ret, errors.New("cannot do processing if tester is not in processing state")
	}

	// setup for distances
	t.distanceBook.processingSetupForDistance(func(u uint) map[uint]uint { return t.G.CalcDistances(u) }, startNode)

	for d, c := range t.distanceBook.distMaxCnt {
		ret = append(ret, data.CntDistances{
			Distance: d,
			Cnt:      c,
		})
	}

	return ret, nil
}

// get timeseries with amount of packets sent over time
func (t *Tester) ProcessSentPackets(gtype common.GossipType, all bool) (data.SentPacketsCntAll, error) {
	ret := make(data.SentPacketsCntAll, 0)
	if t.state != TestStateProcessing {
		return ret, errors.New("cannot do processing if tester is not in processing state")
	}

	sentEvents := make([]Event, 0, len(t.Events))
	for _, e := range t.Events {
		if !(e.Msg == "hz packet sent") {
			continue
		}
		if !(all || gtype == e.MsgType) {
			continue
		}
		sentEvents = append(sentEvents, e)
	}
	slices.SortFunc(sentEvents, func(a Event, b Event) int { return a.TimeBucket.Compare(b.TimeBucket) })

	var currTime time.Time
	var cnt uint
	for _, e := range sentEvents {
		if currTime.Equal(e.TimeBucket) {
			cnt += e.Cnt
			continue
		}
		if !currTime.IsZero() {
			ret = append(ret, data.SentPacketsCnt{
				TimeUnixSec: float64(currTime.UnixMilli()-t.tmin.UnixMilli()) / 1000,
				Cnt:         cnt,
			})
		}
		currTime = e.TimeBucket
		cnt = e.Cnt
	}
	if !currTime.IsZero() {
		ret = append(ret, data.SentPacketsCnt{
			TimeUnixSec: float64(currTime.UnixMilli()-t.tmin.UnixMilli()) / 1000,
			Cnt:         cnt,
		})
	}

	return ret, nil
}
