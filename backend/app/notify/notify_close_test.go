package notify

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

func TestService_CloseDeliversQueued(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := &slowDest{delay: 100 * time.Millisecond}
		s := NewService(nil, 10, d)
		for _, id := range []string{"1", "2", "3"} {
			s.Submit(Request{Comment: store.Comment{ID: id}})
		}
		synctest.Wait() // the consumer holds the first notification in Send, the other two stay queued
		require.Len(t, s.queue, 2, "two notifications still queued when Close is called")
		s.Close()
		assert.Equal(t, int32(3), d.delivered.Load(), "queued notifications delivered before Close returns")
		assert.Equal(t, int32(0), d.aborted.Load())
	})
}

func TestService_CloseDeliversBothQueues(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := &slowDest{delay: 100 * time.Millisecond}
		s := NewService(nil, 10, d)
		for _, id := range []string{"1", "2", "3"} {
			s.Submit(Request{Comment: store.Comment{ID: id}})
			s.SubmitVerification(VerificationRequest{SiteID: "site", User: "u" + id, Email: id + "@example.com", Token: "t"})
		}
		synctest.Wait() // the consumer holds one request in a send, the rest stay queued in both queues
		require.NotEmpty(t, s.queue, "notifications still queued when Close is called")
		require.NotEmpty(t, s.verificationQueue, "verifications still queued when Close is called")
		s.Close()
		assert.Equal(t, int32(3), d.delivered.Load(), "queued notifications delivered before Close returns")
		assert.Equal(t, int32(3), d.verified.Load(), "queued verifications delivered before Close returns")
		assert.Equal(t, int32(0), d.aborted.Load())
	})
}

func TestService_CloseCancelsAfterTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := &slowDest{delay: time.Hour}
		s := NewService(nil, 10, d)
		s.Submit(Request{Comment: store.Comment{ID: "1"}})
		synctest.Wait() // the notification is in Send

		start := time.Now()
		s.Close()
		assert.Equal(t, defaultCloseTimeout, time.Since(start), "Close waits for closeTimeout, then cancels")
		assert.Equal(t, int32(1), d.aborted.Load(), "the stuck send sees its context end")
		assert.Equal(t, int32(0), d.delivered.Load())
	})
}

func TestService_CloseWithDestinationIgnoringContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		d := &MockDest{id: 1, block: release} // blocks in Send until released, whatever its context does
		s := NewService(nil, 10, d)
		s.Submit(Request{Comment: store.Comment{ID: "1"}})
		synctest.Wait() // the notification is in Send

		start := time.Now()
		s.Close()
		assert.Equal(t, defaultCloseTimeout+closeGrace, time.Since(start), "Close returns after the abort grace period")
		close(release) // let the stuck destination finish so the bubble can exit
		synctest.Wait()
	})
}

func TestService_SubmitRacingClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st := &blockingStore{entered: make(chan struct{}), release: make(chan struct{})}
		s := NewService(st, 10, &MockDest{id: 1})

		res := make(chan any, 1)
		go func() {
			defer func() { res <- recover() }()
			// passes the closed check, then blocks in the parent lookup until Close has run
			s.Submit(Request{Comment: store.Comment{ID: "2", ParentID: "1"}})
		}()
		<-st.entered
		s.Close()
		close(st.release)
		assert.Nil(t, <-res, "Submit finishing after Close must not panic")
	})
}

func TestService_CloseTwiceWithDestinations(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := &MockDest{id: 1}
		s := NewService(nil, 10, d)
		s.Submit(Request{Comment: store.Comment{ID: "1"}})
		s.Close()
		s.Close()
		s.Submit(Request{Comment: store.Comment{ID: "2"}})
		s.SubmitVerification(VerificationRequest{User: "u"})
		synctest.Wait()
		require.Len(t, d.Get(), 1)
		assert.Empty(t, d.GetVerify())
	})
}

// slowDest takes delay to deliver each notification, or gives up when its context ends
type slowDest struct {
	delay     time.Duration
	delivered atomic.Int32
	verified  atomic.Int32
	aborted   atomic.Int32
}

func (d *slowDest) String() string { return "slow" }

func (d *slowDest) Send(ctx context.Context, _ Request) error {
	select {
	case <-time.After(d.delay):
		d.delivered.Add(1)
	case <-ctx.Done():
		d.aborted.Add(1)
	}
	return nil
}

func (d *slowDest) SendVerification(ctx context.Context, _ VerificationRequest) error {
	select {
	case <-time.After(d.delay):
		d.verified.Add(1)
	case <-ctx.Done():
		d.aborted.Add(1)
	}
	return nil
}

// blockingStore blocks the parent lookup in Submit until released
type blockingStore struct {
	entered chan struct{}
	release chan struct{}
}

func (b *blockingStore) Get(store.Locator, string, store.User) (store.Comment, error) {
	close(b.entered)
	<-b.release
	return store.Comment{ID: "1"}, nil
}

func (b *blockingStore) GetUserEmail(string, string) (string, error)    { return "", nil }
func (b *blockingStore) GetUserTelegram(string, string) (string, error) { return "", nil }
