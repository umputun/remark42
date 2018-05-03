package migrator

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"

	"github.com/pkg/errors"

	"github.com/umputun/remark/app/store"
)

// Remark implements exporter and importer for internal store format
type Remark struct {
	DataStore store.Interface
}

// Export all comments to writer as json strings. Each comment is one string, separated by "\n"
func (r *Remark) Export(w io.Writer, siteID string) (size int, err error) {
	topics, err := r.DataStore.List(siteID, 0, 0)
	if err != nil {
		return 0, err
	}
	log.Printf("[DEBUG] exporting %d topics", len(topics))

	commentsCount := 0
	for i := len(topics) - 1; i >= 0; i-- { // topics from List sorted in opposite direction
		topic := topics[i]
		comments, err := r.DataStore.Find(store.Locator{SiteID: siteID, URL: topic.URL}, "time")
		if err != nil {
			return commentsCount, err
		}

		for _, comment := range comments {

			buf := &bytes.Buffer{}
			enc := json.NewEncoder(buf)
			enc.SetEscapeHTML(false)

			if err := enc.Encode(comment); err != nil {
				return commentsCount, errors.Wrapf(err, "can't marshal %v", comments)
			}
			data := buf.Bytes()
			if _, err := w.Write(data); err != nil {
				return commentsCount, errors.Wrap(err, "can't write comment data")
			}
			commentsCount++
		}
	}
	log.Printf("[DEBUG] exported %d comments", commentsCount)
	return commentsCount, nil
}

// Import comments from json strings produced by Remark.Export
func (r *Remark) Import(reader io.Reader, siteID string) (size int, err error) {

	if err := r.DataStore.DeleteAll(siteID); err != nil {
		return 0, err
	}

	failed := 0
	total, comments := 0, 0
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		rec := scanner.Bytes()
		if len(rec) < 2 {
			continue
		}
		total++
		comment := store.Comment{}
		if err := json.Unmarshal(rec, &comment); err != nil {
			failed++
			log.Printf("[WARN] unmarshal failed for %s, %s", string(rec), err)
			continue
		}
		if _, err := r.DataStore.Create(comment); err != nil {
			failed++
			log.Printf("[WARN] can't write %+v to store, %s", comment, err)
			continue
		}
		comments++
		if comments%1000 == 0 {
			log.Printf("[DEBUG] imported %d comments", comments)
		}
	}
	if scanner.Err() != nil {
		return comments, errors.Wrap(scanner.Err(), "error in scan")
	}
	if failed > 0 {
		return comments, errors.Errorf("failed to save %d comments", failed)
	}
	log.Printf("[INFO] imported %d comments from %d records", comments, total)
	return comments, nil
}
