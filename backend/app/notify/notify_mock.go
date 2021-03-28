package notify

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/go-pkgz/lgr"
)

// MockDest is a destination mock
type MockDest struct {
	data             []Request
	verificationData []VerificationRequest
	id               int
	closed           bool
	lock             sync.Mutex
}

// Send mock
func (m *MockDest) Send(ctx context.Context, r Request) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	select {
	case <-time.After(10 * time.Millisecond):
		m.data = append(m.data, r)
		log.Printf("sent %s -> %d", r.Comment.ID, m.id)
	case <-ctx.Done():
		log.Printf("ctx closed %d", m.id)
		m.closed = true
	}
	return nil
}

// SendVerification mock
func (m *MockDest) SendVerification(ctx context.Context, v VerificationRequest) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	select {
	case <-time.After(10 * time.Millisecond):
		m.verificationData = append(m.verificationData, v)
		log.Printf("sent verification %s -> %d", v.User, m.id)
	case <-ctx.Done():
		log.Printf("verification ctx closed %d", m.id)
		m.closed = true
	}
	return nil
}

// Get mock
func (m *MockDest) Get() []Request {
	m.lock.Lock()
	defer m.lock.Unlock()
	res := make([]Request, len(m.data))
	copy(res, m.data)
	return res
}

// GetVerify mock
func (m *MockDest) GetVerify() []VerificationRequest {
	m.lock.Lock()
	defer m.lock.Unlock()
	res := make([]VerificationRequest, len(m.verificationData))
	copy(res, m.verificationData)
	return res
}

func (m *MockDest) String() string { return fmt.Sprintf("mock id=%d, closed=%v", m.id, m.closed) }
