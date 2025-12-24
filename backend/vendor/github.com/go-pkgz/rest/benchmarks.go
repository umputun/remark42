package rest

import (
	"container/list"
	"net/http"
	"sync"
	"time"
)

var maxTimeRangeDefault = time.Duration(15) * time.Minute

// Benchmarks is a basic benchmarking middleware collecting and reporting performance metrics
// It keeps track of the requests speeds and counts in 1s benchData buckets ,limiting the number of buckets
// to maxTimeRange. User can request the benchmark for any time duration. This is intended to be used
// for retrieving the benchmark data for the last minute, 5 minutes and up to maxTimeRange.
type Benchmarks struct {
	st           time.Time
	data         *list.List
	lock         sync.RWMutex
	maxTimeRange time.Duration

	nowFn func() time.Time // for testing only
}

type benchData struct {
	// 1s aggregates
	requests    int
	respTime    time.Duration
	minRespTime time.Duration
	maxRespTime time.Duration
	ts          time.Time
}

// BenchmarkStats holds the stats for a given interval
type BenchmarkStats struct {
	Requests        int     `json:"requests"`
	RequestsSec     float64 `json:"requests_sec"`
	AverageRespTime int64   `json:"average_resp_time"`
	MinRespTime     int64   `json:"min_resp_time"`
	MaxRespTime     int64   `json:"max_resp_time"`
}

// NewBenchmarks creates a new benchmark middleware
func NewBenchmarks() *Benchmarks {
	res := &Benchmarks{
		st:           time.Now(),
		data:         list.New(),
		nowFn:        time.Now,
		maxTimeRange: maxTimeRangeDefault,
	}
	return res
}

// WithTimeRange sets the maximum time range for the benchmark to keep data for.
// Default is 15 minutes. The increase of this range will change memory utilization as each second of the range
// kept as benchData aggregate. The default means 15*60 = 900 seconds of data aggregate.
// Larger range allows for longer time periods to be benchmarked.
func (b *Benchmarks) WithTimeRange(maximum time.Duration) *Benchmarks {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.maxTimeRange = maximum
	return b
}

// Handler calculates 1/5/10m request per second and allows to access those values
func (b *Benchmarks) Handler(next http.Handler) http.Handler {

	fn := func(w http.ResponseWriter, r *http.Request) {
		st := b.nowFn()
		defer func() {
			b.update(time.Since(st))
		}()
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func (b *Benchmarks) update(reqDuration time.Duration) {
	now := b.nowFn().Truncate(time.Second)

	b.lock.Lock()
	defer b.lock.Unlock()

	// keep maxTimeRange in the list, drop the rest
	for e := b.data.Front(); e != nil; e = e.Next() {
		if b.data.Front().Value.(benchData).ts.After(b.nowFn().Add(-b.maxTimeRange)) {
			break
		}
		b.data.Remove(b.data.Front())
	}

	last := b.data.Back()
	if last == nil || last.Value.(benchData).ts.Before(now) {
		b.data.PushBack(benchData{requests: 1, respTime: reqDuration, ts: now,
			minRespTime: reqDuration, maxRespTime: reqDuration})
		return
	}

	bd := last.Value.(benchData)
	bd.requests++
	bd.respTime += reqDuration

	if bd.minRespTime == 0 || reqDuration < bd.minRespTime {
		bd.minRespTime = reqDuration
	}
	if bd.maxRespTime == 0 || reqDuration > bd.maxRespTime {
		bd.maxRespTime = reqDuration
	}

	last.Value = bd
}

// Stats returns the current benchmark stats for the given duration
func (b *Benchmarks) Stats(interval time.Duration) BenchmarkStats {
	if interval < time.Second { // minimum interval is 1s due to the bucket size
		return BenchmarkStats{}
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	var (
		requests int
		respTime time.Duration
	)

	now := b.nowFn().Truncate(time.Second)
	cutoff := now.Add(-interval)
	stInterval, fnInterval := time.Time{}, time.Time{}
	var minRespTime, maxRespTime time.Duration
	count := 0

	for e := b.data.Back(); e != nil && count < int(interval.Seconds()); e = e.Prev() { // reverse order
		bd := e.Value.(benchData)
		if bd.ts.Before(cutoff) {
			break
		}

		if minRespTime == 0 || bd.minRespTime < minRespTime {
			minRespTime = bd.minRespTime
		}
		if maxRespTime == 0 || bd.maxRespTime > maxRespTime {
			maxRespTime = bd.maxRespTime
		}
		requests += bd.requests
		respTime += bd.respTime
		if fnInterval.IsZero() {
			fnInterval = bd.ts.Add(time.Second)
		}
		stInterval = bd.ts
		count++
	}

	if requests == 0 {
		return BenchmarkStats{}
	}

	// ensure we calculate rate based on actual interval
	actualInterval := fnInterval.Sub(stInterval)
	if actualInterval < time.Second {
		actualInterval = time.Second
	}

	return BenchmarkStats{
		Requests:        requests,
		RequestsSec:     float64(requests) / actualInterval.Seconds(),
		AverageRespTime: respTime.Microseconds() / int64(requests),
		MinRespTime:     minRespTime.Microseconds(),
		MaxRespTime:     maxRespTime.Microseconds(),
	}
}
