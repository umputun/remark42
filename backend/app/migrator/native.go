package migrator

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sync/atomic"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/syncs"
	"github.com/pkg/errors"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/service"
)

const nativeVersion = 1
const defaultConcurrent = 8

// Native implements exporter and importer for internal store format
// {"version": 1, comments:[{...}\n,{}], meta: {meta}}
// each comments starts from the new line
type Native struct {
	DataStore  Store
	Concurrent int
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
		comments, e := n.DataStore.Find(store.Locator{SiteID: siteID, URL: topic.URL}, "time", adminUser)
		if e != nil {
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
	m := meta{Version: nativeVersion}
	m.Users, m.Posts, err = n.DataStore.Metas(siteID)
	if err != nil {
		return errors.Wrap(err, "can't get meta")
	}

	if err = json.NewEncoder(w).Encode(m); err != nil {
		return errors.Wrap(err, "can't encode meta")
	}
	return nil
}

// WithMapper wraps reader with url-mapper.
func WithMapper(reader io.Reader, mapper Mapper) io.Reader {
	r, w := io.Pipe()
	go func() {
		var err error
		defer func() {
			log.Printf("[DEBUG] finish write to pipe with %+v", err)
			if e := w.Close(); e != nil {
				log.Printf("[WARN] failed close pipe writer with %+v", e)
			}
		}()

		// decode from reader and encode to pipe writer
		dec, enc := json.NewDecoder(reader), json.NewEncoder(w)

		m := meta{}
		if err = dec.Decode(&m); err != nil {
			return
		}
		for i := range m.Posts {
			m.Posts[i].URL = mapper.URL(m.Posts[i].URL)
		}
		if err = enc.Encode(m); err != nil {
			return
		}

		for {
			comment := store.Comment{}
			if err = dec.Decode(&comment); err != nil {
				return
			}
			comment.Locator.URL = mapper.URL(comment.Locator.URL)
			if err = enc.Encode(comment); err != nil {
				return
			}
		}
	}()

	return r
}

// Import comments from json strings produced by Remark.Export
func (n *Native) Import(reader io.Reader, siteID string) (size int, err error) {
	m := meta{}
	dec := json.NewDecoder(reader)
	if err = dec.Decode(&m); err != nil {
		return 0, errors.Wrapf(err, "failed to import meta for site %s", siteID)
	}

	if m.Version != nativeVersion && m.Version != 0 { // this version allows back compatibility with 0 version
		return 0, errors.Errorf("unexpected import file version %d", m.Version)
	}

	if e := n.DataStore.DeleteAll(siteID); e != nil {
		return 0, e
	}

	var failed, total, comments int64

	concurrent := defaultConcurrent
	if n.Concurrent > 0 {
		concurrent = n.Concurrent
	}
	grp := syncs.NewSizedGroup(concurrent, syncs.Preemptive)

	for {
		comment := store.Comment{}
		err = dec.Decode(&comment)
		comment.Imported = true
		if err == io.EOF {
			break
		}

		total++

		if err != nil {
			atomic.AddInt64(&failed, 1)
			failed++
			continue
		}

		// write comments in parallel
		grp.Go(func(context.Context) {
			if _, e := n.DataStore.Create(comment); e != nil {
				atomic.AddInt64(&failed, 1)
				log.Printf("[WARN] can't write %+v to store, %s", comment, e)
				return
			}
			num := atomic.AddInt64(&comments, 1)
			if num%1000 == 0 {
				log.Printf("[DEBUG] imported %d comments", num)
			}
		})

	}

	grp.Wait()

	if failed > 0 {
		return int(comments), errors.Errorf("failed to save %d comments", failed)
	}
	log.Printf("[INFO] imported %d comments from %d records", comments, total)

	err = n.DataStore.SetMetas(siteID, m.Users, m.Posts)

	return int(comments), err
}
