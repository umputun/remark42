// Package notify provides notification functionality.
package notify

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/service"
)

type request struct {
	comment store.Comment
	parent  store.Comment
}

// Destination defines interface for a given destination service, like telegram, email and so on
type Destination interface {
	fmt.Stringer
	Send(ctx context.Context, req request) error
}

// Service delivers notifications to multiple destinations
type Service struct {
	dataService  *service.DataStore
	destinations []Destination
	queue        chan request

	closed bool
	ctx    context.Context
	cancel context.CancelFunc
}

const defaultQueueSize = 100
const uiNav = "#remark42__comment-"

// NewService makes notification service routing comments to all destinations.
func NewService(dataService *service.DataStore, size int, destinations ...Destination) *Service {
	if size <= 0 {
		size = defaultQueueSize
	}
	ctx, cancel := context.WithCancel(context.Background())
	res := Service{
		dataService:  dataService,
		queue:        make(chan request, size),
		destinations: destinations,
		ctx:          ctx,
		cancel:       cancel,
	}
	if len(destinations) > 0 {
		go res.do()
	}
	log.Print("[INFO] create notifier service, queue size=%d", size)
	return &res
}

// Submit comment to internal channel if not busy, drop if can't send
func (s *Service) Submit(comment store.Comment) {
	if len(s.destinations) == 0 || s.closed {
		return
	}
	parentComment := store.Comment{}
	if s.dataService != nil {
		if p, err := s.dataService.Get(comment.Locator, comment.ParentID); err == nil {
			parentComment = p
		}
	}
	select {
	case s.queue <- request{comment: comment, parent: parentComment}:
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
				if err := d.Send(s.ctx, c); err != nil {
					log.Printf("[WARN] failed to send to %s", d)
				}
				wg.Done()
			}(dest)
		}
		wg.Wait()
	}
}
