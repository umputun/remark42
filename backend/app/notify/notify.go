// Package notify provides notification functionality.
package notify

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/remark42/backend/app/store"
)

// Service delivers notifications to multiple destinations
type Service struct {
	dataService       Store
	destinations      []Destination
	queue             chan Request
	verificationQueue chan VerificationRequest

	closed uint32 // non-zero means closed. uses uint instead of bool for atomic
	ctx    context.Context
	cancel context.CancelFunc
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
	GetUserEmail(siteID string, userID string) (string, error)
}

// Request notification for a Comment
type Request struct {
	Comment     store.Comment
	parent      store.Comment
	Emails      []string
}

// VerificationRequest notification for user
type VerificationRequest struct {
	SiteID string
	User   string
	Email  string // if set, send email only
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
		dataService:       dataService,
		queue:             make(chan Request, size),
		verificationQueue: make(chan VerificationRequest, size),
		destinations:      destinations,
		ctx:               ctx,
		cancel:            cancel,
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
	if s.dataService != nil && req.Comment.ParentID != "" {
		if p, err := s.dataService.Get(req.Comment.Locator, req.Comment.ParentID, store.User{}); err == nil {
			req.parent = p
			req.Emails = deduplicateStrings(s.getNotificationEmails(req, p))
		}
	}
	select {
	case s.queue <- req:
	default:
		log.Printf("[WARN] can't send notification to queue, %+v", req.Comment)
	}
}

// getNotificationEmails returns list of emails for notifications for provided comment.
// Emails is not added to the returned list in case original message is from the same user as the notification receiver.
func (s *Service) getNotificationEmails(req Request, notifyComment store.Comment) (result []string) {
	// add current user email only if the user is not the one who wrote the original comment
	if notifyComment.User.ID != req.Comment.User.ID {
		email, err := s.dataService.GetUserEmail(req.Comment.Locator.SiteID, notifyComment.User.ID)
		if err != nil {
			log.Printf("[WARN] can't read email for %s, %v", notifyComment.User.ID, err)
		}
		if email != "" {
			result = append(result, email)
		}
	}
	if notifyComment.ParentID != "" {
		if p, err := s.dataService.Get(req.Comment.Locator, notifyComment.ParentID, store.User{}); err == nil {
			result = append(result, s.getNotificationEmails(req, p)...)
		}
	}
	return result
}

// SubmitVerification to internal channel if not busy, drop if can't send
func (s *Service) SubmitVerification(req VerificationRequest) {
	if len(s.destinations) == 0 || atomic.LoadUint32(&s.closed) != 0 {
		return
	}
	select {
	case s.verificationQueue <- req:
	default:
		log.Printf("[WARN] can't send verification to queue, %s for %s", req.User, req.Email)
	}
}

// Close queue channel and wait for completion
func (s *Service) Close() {
	if s.queue != nil {
		log.Print("[DEBUG] close notifier")
		close(s.queue)
		close(s.verificationQueue)
		s.cancel()
		<-s.ctx.Done()
	}
	atomic.StoreUint32(&s.closed, 1)
}

func (s *Service) do() {
	defer log.Print("[WARN] terminated notifier")
	var wg sync.WaitGroup
	for {
		select {
		case c, ok := <-s.queue:
			if !ok {
				return
			}
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
		case v, ok := <-s.verificationQueue:
			if !ok {
				return
			}
			wg.Add(len(s.destinations))
			for _, dest := range s.destinations {
				go func(d Destination) {
					if err := d.SendVerification(s.ctx, v); err != nil {
						log.Printf("[WARN] failed to send to %s, %s", d, err)
					}
					wg.Done()
				}(dest)
			}
			wg.Wait()
		case <-s.ctx.Done():
			return
		}
	}
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
