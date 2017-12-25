package migrator

import (
	"bytes"
	"encoding/json"
	"io"
	"log"

	"github.com/pkg/errors"

	"github.com/umputun/remark/app/store"
)

// Remark implements exporter and importer for internal store
type Remark struct {
	DataStore store.Interface
}

// Export all comments to writer as json
func (r *Remark) Export(w io.Writer, siteID string) error {
	topics, err := r.DataStore.List(store.Locator{SiteID: siteID})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] exporting %d topics", len(topics))

	commentsCount := 0
	for _, topic := range topics {
		comments, err := r.DataStore.Find(store.Request{Locator: store.Locator{SiteID: siteID, URL: topic}})
		if err != nil {
			return err
		}

		for _, comment := range comments {

			buf := &bytes.Buffer{}
			enc := json.NewEncoder(buf)
			enc.SetEscapeHTML(false)

			if err := enc.Encode(comment); err != nil {
				return errors.Wrapf(err, "can't marshal %v", comments)
			}
			data := buf.Bytes()
			data = append(data, '\n')
			if _, err := w.Write(data); err != nil {
				return errors.Wrap(err, "can't write comment data")
			}
			commentsCount++
		}
	}
	log.Printf("[DEBUG] exported %d comments", commentsCount)
	return nil
}
