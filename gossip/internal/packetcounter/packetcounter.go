package packetcounter

import "time"

type Counter struct {
	t   time.Time
	cnt uint
	// gets called every time t is updated
	do func(t time.Time, cnt uint)
	// duration of the "buckets" to form
	granularity time.Duration
}

func NewCounter(do func(time.Time, uint), granularity time.Duration) *Counter {
	return &Counter{
		t:           time.Time{},
		cnt:         0,
		do:          do,
		granularity: granularity,
	}
}

func (counter *Counter) Add(i uint) {
	now := time.Now().Truncate(counter.granularity)
	if now == counter.t {
		counter.cnt += i
		return
	}

	if !counter.t.IsZero() {
		counter.do(counter.t, counter.cnt)
	}

	counter.cnt = i
	counter.t = now
}

func (counter *Counter) Finalize() {
	if !counter.t.IsZero() {
		counter.do(counter.t, counter.cnt)
	}
}
