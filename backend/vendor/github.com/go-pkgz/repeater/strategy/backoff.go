package strategy

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Backoff implements strategy.Interface for exponential-backoff
// it starts from 100ms (by default, if no Duration set) and goes in steps with last * math.Pow(factor, attempt)
// optional jitter randomize intervals a little bit.
type Backoff struct {
	Duration time.Duration
	Repeats  int
	Factor   float64
	Jitter   bool

	once sync.Once
}

// Start returns channel, similar to time.Timer
// then publishing signals to channel ch for retries attempt. Closed ch indicates "done" event
// consumer (repeater) should stop it explicitly after completion
func (b *Backoff) Start(ctx context.Context) <-chan struct{} {

	b.once.Do(func() {
		if b.Duration == 0 {
			b.Duration = 100 * time.Millisecond
		}
		if b.Repeats == 0 {
			b.Repeats = 1
		}
		if b.Factor <= 0 {
			b.Factor = 1
		}
	})

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		rnd := rand.New(rand.NewSource(int64(time.Now().Nanosecond()))) //nolint:gosec
		for i := 0; i < b.Repeats; i++ {
			select {
			case <-ctx.Done():
				return
			case ch <- struct{}{}:
			}

			delay := float64(b.Duration) * math.Pow(b.Factor, float64(i))
			if b.Jitter {
				delay = rnd.Float64()*(float64(2*b.Duration)) + (delay - float64(b.Duration))
			}
			sleep(ctx, time.Duration(delay))
		}
	}()
	return ch
}
