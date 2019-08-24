package syncs

import (
	"context"
	"sync"
)

// SizedGroup has the same role as WaitingGroup but adds a limit of the amount of goroutines started concurrently.
// Uses similar Go() scheduling as errgrp.Group, thread safe.
// SizedGroup interface enforces constructor usage and doesn't allow direct creation of sizedGroup
type SizedGroup struct {
	options
	wg   sync.WaitGroup
	sema sync.Locker
}

// NewSizedGroup makes wait group with limited size alive goroutines
func NewSizedGroup(size int, opts ...GroupOption) *SizedGroup {
	res := SizedGroup{sema: NewSemaphore(size)}
	res.options.ctx = context.Background()
	for _, opt := range opts {
		opt(&res.options)
	}
	return &res
}

// Go calls the given function in a new goroutine.
// Every call will be unblocked, but some goroutines may wait if semaphore locked.
func (g *SizedGroup) Go(fn func(ctx context.Context)) {

	canceled := func() bool {
		select {
		case <-g.ctx.Done():
			return true
		default:
			return false
		}
	}

	if canceled() {
		return
	}

	g.wg.Add(1)

	if g.preLock {
		g.sema.Lock()
	}

	go func() {
		defer g.wg.Done()

		if canceled() {
			return
		}

		if !g.preLock {
			g.sema.Lock()
		}

		fn(g.ctx)
		g.sema.Unlock()
	}()
}

// Wait blocks until the SizedGroup counter is zero.
// See sync.WaitGroup documentation for more information.
func (g *SizedGroup) Wait() {
	g.wg.Wait()
}
