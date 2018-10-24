// Package migrator provides notification functionality.
package notify

import (
	"context"
	"log"
	"sync"

	"github.com/umputun/remark/backend/app/store"
)

// Destination defines interface for a given destination service, like telegram, email and so on
type Destination interface {
	Send(ctx context.Context, comment store.Comment)
}

// Service delivers notifications to multiple destinations
type Service struct {
	destinations []Destination
	queue        chan store.Comment

	closed bool
	ctx    context.Context
	cancel context.CancelFunc
}

const defaultQueueSize = 100

// NewService makes notification service routing comments to all destinations.
func NewService(size int, destinations ...Destination) *Service {
	if size <= 0 {
		size = defaultQueueSize
	}
	ctx, cancel := context.WithCancel(context.Background())
	res := Service{queue: make(chan store.Comment, size), destinations: destinations, ctx: ctx, cancel: cancel}
	if len(destinations) > 0 {
		go res.do()
	}
	return &res
}

// Submit comment to internal channel if not busy, drop if can't send
func (s *Service) Submit(comment store.Comment) {
	if len(s.destinations) == 0 || s.closed {
		return
	}
	select {
	case s.queue <- comment:
	default:
		log.Printf("[WARN] can't send comment notification to queue, %+v", comment)
	}
}

// Close queue channel and wait for completion
func (s *Service) Close() {
	close(s.queue)
	s.cancel()
	<-s.ctx.Done()
	s.closed = true
}

func (s *Service) do() {
	for c := range s.queue {
		var wg sync.WaitGroup
		for _, dest := range s.destinations {
			wg.Add(1)
			go func(d Destination) {
				d.Send(s.ctx, c)
				wg.Done()
			}(dest)
		}
		wg.Wait()
	}
}
