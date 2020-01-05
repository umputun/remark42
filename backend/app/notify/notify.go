// Package notify provides notification functionality.
package notify

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/remark/backend/app/store"
)

// Service delivers notifications to multiple destinations
type Service struct {
	dataService  Store
	destinations []Destination
	queue        chan Request

	closed uint32 // non-zero means closed. uses uint instead of bool for atomic
	ctx    context.Context
	cancel context.CancelFunc
}

// Destination defines interface for a given destination service, like telegram, email and so on
type Destination interface {
	fmt.Stringer
	Send(ctx context.Context, req Request) error
}

// Store defines the minimal interface accessing stored comments used by notifier
type Store interface {
	Get(locator store.Locator, id string, user store.User) (store.Comment, error)
	GetUserEmail(siteID string, userID string) (string, error)
}

// Request notification either about comment or about particular user verification
type Request struct {
	Comment      store.Comment        // if set sent notifications about new comment
	parent       store.Comment        // fetched only in case Comment is set
	Email        string               // if set (also) send email
	Verification VerificationMetadata // if set sent verification notification
}

// VerificationMetadata required to send notify method verification message
type VerificationMetadata struct {
	SiteID string
	User   string
	Token  string
}

const defaultQueueSize = 100
const uiNav = "#remark42__comment-"

// NewService makes notification service routing comments to all destinations.
func NewService(dataService Store, size int, destinations ...Destination) *Service {
	if size <= 0 {
		size = defaultQueueSize
	}
	ctx, cancel := context.WithCancel(context.Background())
	res := Service{
		dataService:  dataService,
		queue:        make(chan Request, size),
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

// Submit Request to internal channel if not busy, drop if can't send
func (s *Service) Submit(req Request) {
	if len(s.destinations) == 0 || atomic.LoadUint32(&s.closed) != 0 {
		return
	}
	// parent comment is fetched only if comment is present in the Request
	if s.dataService != nil && req.Comment.ParentID != "" {
		if p, err := s.dataService.Get(req.Comment.Locator, req.Comment.ParentID, store.User{}); err == nil {
			req.parent = p
			req.Email, err = s.dataService.GetUserEmail(req.Comment.Locator.SiteID, p.User.ID)
			if err != nil {
				log.Printf("[WARN] can't read email for %s, %v", p.User.ID, err)
			}
		}
	}
	select {
	case s.queue <- req:
	default:
		log.Printf("[WARN] can't send notification to queue, %+v", req.Comment)
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
		wg.Add(len(s.destinations))
		for _, dest := range s.destinations {
			go func(d Destination) {
				if err := d.Send(s.ctx, c); err != nil {
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
