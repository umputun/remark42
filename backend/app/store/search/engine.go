package search

import (
	"github.com/umputun/remark42/backend/app/store"
)

// Engine provides core functionality for searching used by Service
type Engine interface {

	// Index adds or updates a document in the index
	Index(comments []store.Comment) error

	// Search performs search request
	Search(req *Request) (*Result, error)

	// Size returns number of documents in the index
	Size() (uint64, error)

	// Close closes the index
	Close() error
}
