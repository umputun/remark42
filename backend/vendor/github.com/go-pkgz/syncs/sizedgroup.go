package syncs

import "sync"

// SizedGroup has the same role as WaitingGroup but adds a limit of the amount of goroutines started concurrently.
// Uses similar Go() scheduling as errgrp.Group, thread safe.
// SizedGroup interface enforces constructor usage and doesn't allow direct creation of sizedGroup
type SizedGroup interface {
	Go(fn func())
	Wait()
}

type sizedGroup struct {
	wg   sync.WaitGroup
	sema sync.Locker
}

// NewSizedGroup makes wait group with limited size alive goroutines
func NewSizedGroup(size int) SizedGroup {
	return &sizedGroup{sema: NewSemaphore(size)}
}

// Go calls the given function in a new goroutine.
// Every call will be unblocked, but some goroutines may wait if semaphore locked.
func (g *sizedGroup) Go(fn func()) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		g.sema.Lock()
		fn()
		g.sema.Unlock()
	}()
}

// Wait blocks until the SizedGroup counter is zero.
// See sync.WaitGroup documentation for more information.
func (g *sizedGroup) Wait() {
	g.wg.Wait()
}
