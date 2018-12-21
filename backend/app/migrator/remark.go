package migrator

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/service"
)

const (
	header     = `{"version":1, "comments":[`
	metaHeader = "],\n\"meta\":"
	footer     = `}`
)

// Remark implements exporter and importer for internal store format
// {"version": 1, comments:[{...}\n,{}], meta: {meta}}
type Remark struct {
	DataStore Store
}

// Export all comments to writer as json strings. Each comment is one string, separated by "\n"
func (r *Remark) Export(w io.Writer, siteID string) (size int, err error) {

	if _, err := fmt.Fprintf(w, "%s\n", header); err != nil {
		return 0, err
	}

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

		for n, comment := range comments {

			buf := &bytes.Buffer{}
			enc := json.NewEncoder(buf)
			enc.SetEscapeHTML(false)

			if err := enc.Encode(comment); err != nil {
				return commentsCount, errors.Wrapf(err, "can't marshal %v", comments)
			}
			data := buf.Bytes()
			data = bytes.TrimSuffix(data, []byte("\n"))
			if _, err := w.Write(data); err != nil {
				return commentsCount, errors.Wrap(err, "can't write comment data")
			}
			if n < len(comments)-1 || i != 0 { // don't add , on last comment
				w.Write([]byte(","))
			}
			w.Write([]byte("\n"))

			commentsCount++
		}
	}
	log.Printf("[DEBUG] exported %d comments", commentsCount)

	if _, err := fmt.Fprintf(w, "%s", metaHeader); err != nil {
		return 0, err
	}

	meta := struct {
		Users []service.UserMetaData `json:"users"`
		Posts []service.PostMetaData `json:"posts"`
	}{}

	meta.Users, meta.Posts, err = r.DataStore.Metas(siteID)
	if err != nil {
		return 0, err
	}

	if err := json.NewEncoder(w).Encode(meta); err != nil {
		return 0, err
	}

	if _, err := fmt.Fprintf(w, "%s\n", footer); err != nil {
		return 0, err
	}

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
		if len(rec) < 3 {
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
