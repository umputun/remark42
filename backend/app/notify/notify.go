// Package notify provides notification functionality.
package notify

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/remark42/backend/app/store"
)

// Service delivers notifications to multiple destinations
type Service struct {
	dataService       Store
	destinations      []Destination
	queue             chan Request
	verificationQueue chan VerificationRequest

	closed       atomic.Uint32 // non-zero means closed. uses uint instead of bool for atomic
	queueMu      sync.RWMutex  // held for reading while sending to the queues, for writing while closing them
	done         chan struct{} // closed when the delivery goroutine exits
	closeTimeout time.Duration // how long Close waits for queued notifications before aborting them
	ctx          context.Context
	cancel       context.CancelFunc
}

// Destination defines interface for a given destination service, like telegram, email and so on
type Destination interface {
	fmt.Stringer
	Send(context.Context, Request) error
	SendVerification(context.Context, VerificationRequest) error
}

// Store defines the minimal interface accessing stored comments used by notifier
type Store interface {
	Get(locator store.Locator, id string, user store.User) (store.Comment, error)
	GetUserEmail(siteID, userID string) (string, error)
	GetUserTelegram(siteID, userID string) (string, error)
}

// used for email and telegram retrieval from user details
type getUserDetail func(string, string) (string, error)

// Request notification for a Comment
type Request struct {
	Comment   store.Comment
	parent    store.Comment
	Emails    []string
	Telegrams []string
}

// VerificationRequest notification for user
type VerificationRequest struct {
	SiteID string
	User   string
	Email  string // if set, send email only
	Token  string
}

const defaultQueueSize = 100

// defaultCloseTimeout bounds how long Close waits for queued notifications to be delivered
const defaultCloseTimeout = 10 * time.Second

// closeGrace bounds how long Close waits for destinations to stop after aborting them
const closeGrace = time.Second
const uiNav = "#remark42__comment-"

// NewService makes notification service routing comments to all destinations.
func NewService(dataService Store, size int, destinations ...Destination) *Service {
	if size <= 0 {
		size = defaultQueueSize
	}
	ctx, cancel := context.WithCancel(context.Background())
	res := Service{
		dataService:       dataService,
		queue:             make(chan Request, size),
		verificationQueue: make(chan VerificationRequest, size),
		destinations:      destinations,
		done:              make(chan struct{}),
		closeTimeout:      defaultCloseTimeout,
		ctx:               ctx,
		cancel:            cancel,
	}
	if len(destinations) > 0 {
		go res.do()
	} else {
		close(res.done)
	}
	log.Printf("[INFO] create notifier service, queue size=%d, destinations=%d", size, len(destinations))
	return &res
}

// Submit Request to internal channel if not busy, drop if can't send
func (s *Service) Submit(req Request) {
	if len(s.destinations) == 0 || s.closed.Load() != 0 {
		return
	}
	if s.dataService != nil && req.Comment.ParentID != "" {
		if p, err := s.dataService.Get(req.Comment.Locator, req.Comment.ParentID, store.User{}); err == nil {
			req.parent = p
			req.Emails = s.getNotificationTargets(req, p, s.dataService.GetUserEmail)
			req.Telegrams = s.getNotificationTargets(req, p, s.dataService.GetUserTelegram)
		}
	}
	s.queueMu.RLock()
	defer s.queueMu.RUnlock()
	if s.closed.Load() != 0 {
		return
	}
	select {
	case s.queue <- req:
	default:
		log.Printf("[WARN] can't send notification to queue, %+v", req.Comment)
	}
}

// getNotificationTargets returns list of notification targets (like email or telegram username) for users
// interested in notifications for provided comment.
// Targets are not added to the returned list in case the original message
// is from the same user as the notification receiver.
// Results are deduplicated.
func (s *Service) getNotificationTargets(
	req Request,
	notifyComment store.Comment,
	getUserDetail getUserDetail,
) (result []string) {
	// add current user email only if the user is not the one who wrote the original comment
	if notifyComment.User.ID != req.Comment.User.ID {
		detail, err := getUserDetail(req.Comment.Locator.SiteID, notifyComment.User.ID)
		if err != nil {
			log.Printf("[WARN] can't read notification detail for %s, %v", notifyComment.User.ID, err)
		}
		if detail != "" {
			result = append(result, detail)
		}
	}
	if notifyComment.ParentID != "" {
		if p, err := s.dataService.Get(req.Comment.Locator, notifyComment.ParentID, store.User{}); err == nil {
			result = append(result, s.getNotificationTargets(req, p, getUserDetail)...)
		}
	}
	return deduplicateStrings(result)
}

// SubmitVerification to internal channel if not busy, drop if can't send
func (s *Service) SubmitVerification(req VerificationRequest) {
	if len(s.destinations) == 0 || s.closed.Load() != 0 {
		return
	}
	s.queueMu.RLock()
	defer s.queueMu.RUnlock()
	if s.closed.Load() != 0 {
		return
	}
	select {
	case s.verificationQueue <- req:
	default:
		log.Printf("[WARN] can't send verification to queue, %s for %s", req.User, req.Email)
	}
}

// Close stops accepting notifications and waits for the queued ones to be delivered, up to closeTimeout.
// Notifications still undelivered after that are aborted, and Close waits at most closeGrace more for the
// destinations to stop, since some of them may ignore the context. Repeated calls do nothing.
func (s *Service) Close() {
	s.queueMu.Lock()
	if s.closed.Load() != 0 {
		s.queueMu.Unlock()
		return
	}
	s.closed.Store(1)
	if s.queue == nil { // zero-value service, such as NopService
		s.queueMu.Unlock()
		return
	}
	log.Print("[DEBUG] close notifier")
	close(s.queue)
	close(s.verificationQueue)
	s.queueMu.Unlock()

	select {
	case <-s.done:
	case <-time.After(s.closeTimeout):
		log.Printf("[WARN] notifications not delivered in %v, aborting the rest", s.closeTimeout)
	}
	s.cancel()
	select {
	case <-s.done:
	case <-time.After(closeGrace):
		log.Printf("[WARN] notification destinations did not stop in %v after abort, not waiting for them", closeGrace)
	}
}

// do delivers queued notifications until both queues are closed and drained, or the context is canceled
func (s *Service) do() {
	defer close(s.done)
	defer log.Print("[DEBUG] terminated notifier")
	queue, verificationQueue := s.queue, s.verificationQueue
	for queue != nil || verificationQueue != nil {
		select {
		case c, ok := <-queue:
			if !ok {
				queue = nil
				continue
			}
			s.deliver(func(d Destination) error { return d.Send(s.ctx, c) })
		case v, ok := <-verificationQueue:
			if !ok {
				verificationQueue = nil
				continue
			}
			s.deliver(func(d Destination) error { return d.SendVerification(s.ctx, v) })
		case <-s.ctx.Done():
			return
		}
	}
}

// deliver calls send for every destination in parallel and waits for all of them
func (s *Service) deliver(send func(Destination) error) {
	var wg sync.WaitGroup
	for _, dest := range s.destinations {
		wg.Go(func() {
			if err := send(dest); err != nil {
				log.Printf("[WARN] failed to send to %s, %s", dest, err)
			}
		})
	}
	wg.Wait()
}

// NopService is do-nothing notifier, without destinations
var NopService = &Service{}

// deduplicateStrings returns provided slice of strings will all duplicates removed.
// Resulting slice is not sorted.
func deduplicateStrings(source []string) []string {
	set := make(map[string]struct{})

	for _, k := range source {
		set[k] = struct{}{}
	}

	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
	}

	return result
}
