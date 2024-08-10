package testutils

import (
	"errors"
	"fmt"
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
		tAbs := float64(e.Time.UnixMilli()-t.tmin.UnixMilli())/1000
		tRel := (tAbs) / t.durSec
		nodeIdx := t.PeersLut[e.Id]

		if _,ok := ret[nodeIdx]; !ok {
			ret[nodeIdx] = data.ReachedWhen{
				TimeUnixSec: tAbs,
				TimePercent: tRel*100,
			}
		}
	}

	return ret, nil
}

func (t *Tester) ProcessReachedDistCnt(startNode uint, gtype common.GossipType, all bool) (data.ReachedDistCntAll, error) {
	ret := make(data.ReachedDistCntAll, 0)
	if t.state != TestStateProcessing {
		return ret, errors.New("cannot do processing if tester is not in processing state")
	}

	// setup for distances
	distCnt := t.distanceBook.processingSetupForDistance(func(u uint) map[uint]uint {return t.G.CalcDistances(u)}, startNode)

	for _,d := range t.distanceBook.distOrd {
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
		tAbs := float64(e.Time.UnixMilli()-t.tmin.UnixMilli())/1000
		nodeIdx := t.PeersLut[e.Id]
		dist := t.distanceBook.nodeToDist[nodeIdx]
		distCnt[dist] += 1

		ret = append(ret, data.ReachedDistCnt{
			TimeUnixSec:            tAbs,
			Distance:               dist,
			CntReachedSameDistance: distCnt[dist],
		})
	}

	for _,d := range t.distanceBook.distOrd {
		ret = append(ret, data.ReachedDistCnt{
			TimeUnixSec:            t.durSec,
			Distance:               d,
			CntReachedSameDistance: distCnt[d],
		})
	}


	return ret, nil
}

// get how many nodes exist with a specific distance
func (t *Tester) ProcessGraphDistCnt(startNode uint) (data.CntDistancesAll,error) {
	ret := make(data.CntDistancesAll, 0)
	if t.state != TestStateProcessing {
		return ret, errors.New("cannot do processing if tester is not in processing state")
	}

	// setup for distances
	t.distanceBook.processingSetupForDistance(func(u uint) map[uint]uint {return t.G.CalcDistances(u)}, startNode)

	fmt.Printf("%+v\n", t.distanceBook.distMaxCnt)

	for d,c := range t.distanceBook.distMaxCnt {
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
				TimeUnixSec: float64(currTime.UnixMilli()-t.tmin.UnixMilli())/1000,
				Cnt:         cnt,
			})
		}
		currTime = e.TimeBucket
		cnt = e.Cnt
	}
	if !currTime.IsZero() {
		ret = append(ret, data.SentPacketsCnt{
			TimeUnixSec: float64(currTime.UnixMilli()-t.tmin.UnixMilli())/1000,
			Cnt:         cnt,
		})
	}

	return ret, nil
}
