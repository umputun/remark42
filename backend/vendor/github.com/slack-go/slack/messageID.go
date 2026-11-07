package slack

import "sync/atomic"

// IDGenerator provides an interface for generating integer ID values.
type IDGenerator interface {
	Next() int
}

// NewSafeID returns a new instance of an IDGenerator which is safe for
// concurrent use by multiple goroutines.
func NewSafeID(startID int) IDGenerator {
	s := &safeID{}
	s.nextID.Store(int64(startID))
	return s
}

type safeID struct {
	nextID atomic.Int64
}

// make sure safeID implements the IDGenerator interface.
var _ IDGenerator = (*safeID)(nil)

// Next implements IDGenerator.Next.
func (s *safeID) Next() int {
	return int(s.nextID.Add(1))
}
