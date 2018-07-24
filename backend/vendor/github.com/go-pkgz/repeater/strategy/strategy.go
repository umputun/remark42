// Package strategy defines repeater's strategy and implements some.
// Strategy result is a channel acting like time.Timer ot time.Tick
package strategy

import "context"

// Interface for repeater strategy. Returns channel with ticks
type Interface interface {
	Start(ctx context.Context) chan struct{}
}

// Once strategy eliminate repeats and makes a single try only
type Once struct{}

// NewOnce makes no-repeat strategy
func NewOnce() Interface {
	return &Once{}
}

// Start returns closed channel with a single element to prevent any repeats
func (s *Once) Start(ctx context.Context) (ch chan struct{}) {
	ch = make(chan struct{})
	go func() {
		ch <- struct{}{}
		close(ch)
	}()
	return ch
}
