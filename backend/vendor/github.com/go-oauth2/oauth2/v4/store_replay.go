package oauth2

import (
	"context"
	stderrors "errors"
	"sync"
	"time"
)

type AssertionReplayStore interface {
	StoreAssertionID(ctx context.Context, jti string, exp time.Time) error
}

type memoryAssertionReplayStore struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

func NewMemoryAssertionReplayStore() AssertionReplayStore {
	return &memoryAssertionReplayStore{entries: make(map[string]time.Time)}
}

func (s *memoryAssertionReplayStore) StoreAssertionID(ctx context.Context, jti string, exp time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, e := range s.entries {
		if now.After(e) {
			delete(s.entries, k)
		}
	}
	if _, ok := s.entries[jti]; ok {
		return stderrors.New("jti already used")
	}
	s.entries[jti] = exp
	return nil
}
