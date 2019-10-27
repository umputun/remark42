package strategy

import (
	"context"
	"time"
)

// FixedDelay implements strategy.Interface for fixed intervals up to max repeats
type FixedDelay struct {
	Repeats int
	Delay   time.Duration
}

// Start returns channel, similar to time.Timer
// then publishing signals to channel ch for retries attempt.
// can be terminated (canceled) via context.
func (s *FixedDelay) Start(ctx context.Context) <-chan struct{} {
	if s.Repeats == 0 {
		s.Repeats = 1
	}
	ch := make(chan struct{})
	go func() {
		defer func() {
			close(ch)
		}()
		for i := 0; i < s.Repeats; i++ {
			select {
			case <-ctx.Done():
				return
			case ch <- struct{}{}:
			}
			sleep(ctx, s.Delay)
		}
	}()
	return ch
}
