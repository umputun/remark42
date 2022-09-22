package search

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"log"
	"path"

	"github.com/hashicorp/go-multierror"
	"github.com/microcosm-cc/bluemonday"
	"github.com/umputun/remark42/backend/app/store"
)

// Service provides high-level API for the search functionality the end consumer (i.e. REST) needs.
// Caller should add documents to the index using Index() and perform search among them using Search().
// Since there's no Delete() method, it's up to the caller not to show them in the search results (the store should not return their content).
//
// It uses some Engine implementation under the hood.
// Separate engine is used for each site.
// Service is thread-safe (if the engine is so) because after creating sitesEngines is not modified.
type Service struct {
	sitesEngines map[string]Engine
}

// Request for search
type Request struct {
	// Request should be sent for a specific site
	SiteID string

	// Query contains user input.
	// It's a phrase containing some words with additional search operators,
	// e.g. search for comments of specific 'user' or with specified 'time'.
	// For example: `hello world user:umputun time:>=2019-01-01`.
	// It's engine specific to parse and handle it.
	Query string

	// Sort by specified field, e.g. "time", "-time"
	// Prefix "-" means descending order
	SortBy string

	// Pagination
	Skip  int
	Limit int
}

// DocumentKey is a unique key for a document
type DocumentKey struct {
	Locator store.Locator `json:"locator"`
	ID      string        `json:"id"`
}

// Result is a search result
// Contains only keys of the documents without its content
// Caller should retrieve comment's content from the storage
type Result struct {
	// Total is a total number of documents in the index that match the query
	Total uint64        `json:"total"`
	Keys  []DocumentKey `json:"keys"`
}

// ServiceParams contains parameters for creating a search service
type ServiceParams struct {
	IndexPath string
	Analyzer  string
}

// NewService creates new search service
func NewService(sites []string, params ServiceParams) (*Service, error) {
	s := &Service{
		sitesEngines: map[string]Engine{},
	}

	var err error
	for _, site := range sites {
		// encode site name to make it safe to use as a file name
		idxPath := path.Join(params.IndexPath, hex.EncodeToString(fnv.New32().Sum([]byte(site))))

		s.sitesEngines[site], err = newBleveEngine(idxPath, params.Analyzer)
		if err != nil {
			return nil, fmt.Errorf("failed to create search engine: %w", err)
		}
	}
	return s, nil
}

// Index single document
// It should be called after each comment is created or updated.
func (s *Service) Index(doc store.Comment) error {
	return s.indexBatch([]store.Comment{doc})
}

// Search performs search query
func (s *Service) Search(req *Request) (*Result, error) {
	eng, ok := s.sitesEngines[req.SiteID]
	if !ok {
		return nil, fmt.Errorf("no search engine for site %q", req.SiteID)
	}
	return eng.Search(req)
}

// Close search service
func (s *Service) Close() error {
	if s.sitesEngines == nil {
		return nil
	}

	log.Print("[INFO] closing search service...")
	errs := new(multierror.Error)

	for siteID, searcher := range s.sitesEngines {
		if err := searcher.Close(); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("cannot close searcher for %q: %w", siteID, err))
		}
	}

	log.Print("[INFO] search service closed")

	// reset sitesEngines to make it safe to call Close() multiple times
	s.sitesEngines = nil

	return errs.ErrorOrNil()
}

// indexBatch indexes batch of document
func (s *Service) indexBatch(docs []store.Comment) error {
	if len(docs) == 0 {
		return nil
	}
	siteID := docs[0].Locator.SiteID
	if eng, has := s.sitesEngines[siteID]; has {
		for i := range docs {
			// remove all html tags from the text, because we want to search only in the text
			p := bluemonday.StrictPolicy()
			docs[i].Text = p.Sanitize(docs[i].Text)

			// check that all documents from same site
			if docs[i].Locator.SiteID != siteID {
				return fmt.Errorf("different sites in batch")
			}
		}

		return eng.Index(docs)
	}
	return fmt.Errorf("site %q not found", siteID)
}
