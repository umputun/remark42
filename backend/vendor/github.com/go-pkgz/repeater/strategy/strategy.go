// Package strategy defines repeater's strategy and implements some.
// Strategy result is a channel acting like time.Timer ot time.Tick
package strategy

import (
	"context"
	"time"
)

// Interface for repeater strategy. Returns channel with ticks
type Interface interface {
	Start(ctx context.Context) <-chan struct{}
}

// Once strategy eliminate repeats and makes a single try only
type Once struct{}

// Start returns closed channel with a single element to prevent any repeats
func (s *Once) Start(_ context.Context) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		ch <- struct{}{}
		close(ch)
	}()
	return ch
}

func sleep(ctx context.Context, duration time.Duration) {
	select {
	case <-time.After(duration):
		return
	case <-ctx.Done():
		return
	}
}
