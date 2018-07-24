package strategy

import (
	"context"
	"time"
)

// FixedDelay implements strategy.Interface for fixed intervals up to max repeats
type FixedDelay struct {
	repeats int
	delay   time.Duration
}

// NewFixedDelay makes a Interface
func NewFixedDelay(repeats int, delay time.Duration) Interface {
	if repeats == 0 {
		repeats = 1
	}
	result := FixedDelay{repeats: repeats, delay: delay}
	return &result
}

// Start returns channel, similar to time.Timer
// then publishing signals to channel ch for retries attempt.
// can be terminated (canceled) via context.
func (s *FixedDelay) Start(ctx context.Context) (ch chan struct{}) {
	ch = make(chan struct{})
	go func() {
		defer close(ch)
		for i := 0; i < s.repeats; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				ch <- struct{}{}
				time.Sleep(s.delay)
			}
		}
	}()
	return ch
}
