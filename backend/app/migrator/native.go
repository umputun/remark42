package migrator

import (
	"bytes"
	"encoding/json"
	"io"
	"log"

	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/service"
)

// Native implements exporter and importer for internal store format
// {"version": 1, comments:[{...}\n,{}], meta: {meta}}
// each comments starts from the new line
type Native struct {
	DataStore Store
}

type meta struct {
	Version int                    `json:"version"`
	Users   []service.UserMetaData `json:"users"`
	Posts   []service.PostMetaData `json:"posts"`
}

// Export all comments to writer as json strings. Each comment is one string, separated by "\n"
// The final file is a valid json
func (n *Native) Export(w io.Writer, siteID string) (size int, err error) {

	if err = n.exportMeta(siteID, w); err != nil {
		return 0, errors.Wrapf(err, "failed to export meta for site %s", siteID)
	}

	topics, err := n.DataStore.List(siteID, 0, 0)
	if err != nil {
		return 0, err
	}

	log.Printf("[DEBUG] exporting %d topics", len(topics))
	commentsCount := 0
	for i := len(topics) - 1; i >= 0; i-- { // topics from List sorted in opposite direction
		topic := topics[i]
		comments, e := n.DataStore.Find(store.Locator{SiteID: siteID, URL: topic.URL}, "time")
		if err != nil {
			return commentsCount, e
		}

		for _, comment := range comments {

			buf := &bytes.Buffer{}
			enc := json.NewEncoder(buf)
			enc.SetEscapeHTML(false)

			if err = enc.Encode(comment); err != nil {
				return commentsCount, errors.Wrapf(err, "can't marshal %v", comments)
			}
			if _, err = w.Write(buf.Bytes()); err != nil {
				return commentsCount, errors.Wrap(err, "can't write comment data")
			}
			commentsCount++
		}
	}
	log.Printf("[DEBUG] exported %d comments", commentsCount)
	return commentsCount, nil
}

// exportMeta appends user and post metas to exported stream
func (n *Native) exportMeta(siteID string, w io.Writer) (err error) {
	m := meta{Version: 1}
	m.Users, m.Posts, err = n.DataStore.Metas(siteID)
	if err != nil {
		return errors.Wrap(err, "can't get meta")
	}

	if err := json.NewEncoder(w).Encode(m); err != nil {
		return errors.Wrap(err, "can't encode meta")
	}
	return nil
}

// Import comments from json strings produced by Remark.Export
func (n *Native) Import(reader io.Reader, siteID string) (size int, err error) {

	m := meta{}
	dec := json.NewDecoder(reader)
	if err = dec.Decode(&m); err != nil {
		return 0, errors.Wrapf(err, "failed to import meta for site %s", siteID)
	}

	if err := n.DataStore.DeleteAll(siteID); err != nil {
		return 0, err
	}

	failed := 0
	total, comments := 0, 0

	for {
		comment := store.Comment{}
		err = dec.Decode(&comment)
		if err == io.EOF {
			break
		}

		total++

		if err != nil {
			failed++
			continue
		}

		if _, err := n.DataStore.Create(comment); err != nil {
			failed++
			log.Printf("[WARN] can't write %+v to store, %s", comment, err)
			continue
		}
		comments++
		if comments%1000 == 0 {
			log.Printf("[DEBUG] imported %d comments", comments)
		}
	}

	if failed > 0 {
		return comments, errors.Errorf("failed to save %d comments", failed)
	}
	log.Printf("[INFO] imported %d comments from %d records", comments, total)

	err = n.DataStore.SetMetas(siteID, m.Users, m.Posts)

	return comments, err
}
