package search

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
)

// IndexSite rebuilds the search index for all topics within site from scratch
func IndexSite(ctx context.Context, siteID string, maxBatchSize int, s *Service, e engine.Interface) error {
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

	req := engine.InfoRequest{Locator: store.Locator{SiteID: siteID}}
	topics, err := e.Info(req)

	if err != nil {
		return fmt.Errorf("failed to get topics for site %q: %w", siteID, err)
	}

	for i := len(topics) - 1; i >= 0; i-- {
		locator := store.Locator{SiteID: siteID, URL: topics[i].URL}
		req := engine.FindRequest{Locator: locator, Since: time.Time{}}
		comments, findErr := e.Find(req)
		for i := 0; i < len(comments); i += maxBatchSize {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if findErr != nil {
				return fmt.Errorf("failed to fetch comments: %w", findErr)
			}

			next := i + maxBatchSize
			if next > len(comments) {
				next = len(comments)
			}
			indexErr := s.indexBatch(comments[i:next])
			if indexErr != nil {
				return fmt.Errorf("failed to index comments for search: %w", indexErr)
			}

			log.Printf("[INFO] %d documents indexed from topic %v", next-i, locator)
		}
	}
	return nil
}
