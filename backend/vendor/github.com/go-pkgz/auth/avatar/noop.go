package avatar

import (
	"bytes"
	"io"
)

// NoOp is an empty (no-op) implementation of Store interface
type NoOp struct{}

// NewNoOp provides an empty (no-op) implementation of Store interface
func NewNoOp() *NoOp { return &NoOp{} }

// String is a NoOp implementation
func (s *NoOp) String() string { return "" }

// Put is a NoOp implementation
func (s *NoOp) Put(userID string, reader io.Reader) (avatarID string, err error) { return "", nil }

// Get is a NoOp implementation
func (s *NoOp) Get(avatarID string) (reader io.ReadCloser, size int, err error) {
	return io.NopCloser(bytes.NewBuffer([]byte(""))), 0, nil
}

// ID is a NoOp implementation
func (s *NoOp) ID(avatarID string) (id string) { return "" }

// Remove is a NoOp implementation
func (s *NoOp) Remove(avatarID string) error { return nil }

// List is a NoOp implementation
func (s *NoOp) List() (ids []string, err error) { return nil, nil }

// Close is a NoOp implementation
func (s *NoOp) Close() error { return nil }
