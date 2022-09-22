package search

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/go-pkgz/syncs"
	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
)

// IndexSite rebuilds search index for the site
// Run indexing of each topic in parallel in a sized group
func IndexSite(ctx context.Context, siteID string, s *Service, e engine.Interface, grp *syncs.ErrSizedGroup) error {
	siteIdx, isIndexed := s.sitesEngines[siteID]
	if !isIndexed {
		log.Printf("[INFO] skipping indexing site %q", siteID)
		return nil
	}
	indexSize, err := siteIdx.Size()
	if err != nil {
		log.Printf("[WARN] failed to get index size, %s", err)
		return nil
	}

	// index only it's a first run with enabled search on top of existing comments
	if indexSize > 0 {
		log.Printf("[INFO] index for site %q already exists, size %d", siteID, indexSize)
		return nil
	}

	log.Printf("[INFO] indexing site %s", siteID)
	startTime := time.Now()

	req := engine.InfoRequest{Locator: store.Locator{SiteID: siteID}}
	topics, err := e.Info(req)

	if err != nil {
		return fmt.Errorf("failed to get topics for site %q: %w", siteID, err)
	}

	var indexedCnt uint64
	worker := func(ctx context.Context, url string) error {
		locator := store.Locator{SiteID: siteID, URL: url}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req := engine.FindRequest{Locator: locator, Since: time.Time{}}
		comments, findErr := e.Find(req)
		if findErr != nil {
			return fmt.Errorf("failed to fetch comments: %w", findErr)
		}

		indexErr := s.indexBatch(comments)
		log.Printf("[INFO] %d documents indexed from site %v", len(comments), locator)

		if indexErr != nil {
			return fmt.Errorf("failed to index comments for search: %w", indexErr)
		}

		atomic.AddUint64(&indexedCnt, uint64(len(comments)))

		return nil
	}

	for i := len(topics) - 1; i >= 0; i-- {
		url := topics[i].URL
		grp.Go(func() error { return worker(ctx, url) })
	}

	log.Printf("[INFO] total %d documents indexed for site %q in %v", indexedCnt, siteID, time.Since(startTime))
	return nil
}
