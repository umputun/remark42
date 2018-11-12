// Package notify provides notification functionality.
package notify

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/go-pkgz/repeater"
	"github.com/go-pkgz/repeater/strategy"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/service"
)

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

	closed uint32 // non-zero means closed. uses uint instead of bool for atomic
	ctx    context.Context
	cancel context.CancelFunc
}

type request struct {
	comment store.Comment
	parent  store.Comment
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
	log.Printf("[INFO] create notifier service, queue size=%d, destinations=%d", size, len(destinations))
	return &res
}

// Submit comment to internal channel if not busy, drop if can't send
func (s *Service) Submit(comment store.Comment) {
	if len(s.destinations) == 0 || atomic.LoadUint32(&s.closed) != 0 {
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
	if s.queue != nil {
		log.Print("[DEBUG] close notifier")
		close(s.queue)
		s.cancel()
		<-s.ctx.Done()
	}
	atomic.StoreUint32(&s.closed, 1)
}

func (s *Service) do() {
	for c := range s.queue {
		var wg sync.WaitGroup
		for _, dest := range s.destinations {
			wg.Add(1)
			rpt := repeater.New(strategy.NewBackoff(5, 1.5, true))
			go func(d Destination) {
				err := rpt.Do(func() error {
					return d.Send(s.ctx, c)
				})
				if err != nil {
					log.Printf("[WARN] failed to send to %s, %s", d, err)
				}
				wg.Done()
			}(dest)
		}
		wg.Wait()
	}
	log.Print("[WARN] terminated notifier")
}

// NopService is do-nothing notifier, without destinations
var NopService = &Service{}
