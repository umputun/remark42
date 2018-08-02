package strategy

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// Backoff implements strategy.Interface for exponential-backoff
// it starts from 100ms and goes in steps with last * math.Pow(factor, attempt)
// optional jitter randomize intervals a little bit.
type Backoff struct {
	repeats int
	factor  float64
	jitter  bool
}

// NewBackoff makes Backoff strategy with given factor and optional jitter
func NewBackoff(repeats int, factor float64, jitter bool) Interface {
	if repeats == 0 {
		repeats = 1
	}
	if factor <= 0 {
		factor = 1
	}
	result := Backoff{repeats: repeats, factor: factor, jitter: jitter}
	return &result
}

// Start returns channel, similar to time.Timer
// then publishing signals to channel ch for retries attempt. Closed ch indicates "done" event
// consumer (repeater) should stop it explicitly after completion
func (b *Backoff) Start(ctx context.Context) (ch chan struct{}) {
	ch = make(chan struct{})
	go func() {
		defer close(ch)
		rnd := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
		minDelay := 100 * time.Millisecond // starts 100ms
		for i := 0; i < b.repeats; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				ch <- struct{}{}
				delay := float64(minDelay) * math.Pow(b.factor, float64(i))
				if b.jitter {
					delay = rnd.Float64()*(float64(2*minDelay)) + (delay - float64(minDelay))
				}
				time.Sleep(time.Duration(delay))
			}
		}
	}()
	return ch
}
