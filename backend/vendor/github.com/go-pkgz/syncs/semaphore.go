package syncs

import "sync"

// Semaphore implementation, counted lock only. Implements sync.Locker interface, thread safe.
type semaphore struct {
	sync.Locker
	ch chan struct{}
}

// NewSemaphore makes Semaphore with given capacity
func NewSemaphore(capacity int) sync.Locker {
	if capacity <= 0 {
		capacity = 1
	}
	return &semaphore{ch: make(chan struct{}, capacity)}
}

// Lock acquires semaphore, can block if out of capacity.
func (s *semaphore) Lock() {
	s.ch <- struct{}{}
}

// Unlock releases semaphore, can block if nothing acquired before.
func (s *semaphore) Unlock() {
	<-s.ch
}
