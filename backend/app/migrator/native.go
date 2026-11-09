package migrator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync/atomic"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/syncs"

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
		return 0, fmt.Errorf("failed to export meta for site %s: %w", siteID, err)
	}

	topics, err := n.DataStore.List(siteID, 0, 0)
	if err != nil {
		return 0, err
	}

	log.Printf("[DEBUG] exporting %d topics", len(topics))
	commentsCount := 0
	for _, topic := range slices.Backward(topics) { // topics from List sorted in opposite direction

		comments, e := n.DataStore.FindRaw(store.Locator{SiteID: siteID, URL: topic.URL})
		if e != nil {
			return commentsCount, e
		}

		for _, comment := range comments {
			comment.VotedIPs = nil // voters' IP hashes stay out of backups, as before
			buf := &bytes.Buffer{}
			enc := json.NewEncoder(buf)
			enc.SetEscapeHTML(false)

			if err = enc.Encode(comment); err != nil {
				return commentsCount, fmt.Errorf("can't marshal %v: %w", comments, err)
			}
			if _, err = w.Write(buf.Bytes()); err != nil {
				return commentsCount, fmt.Errorf("can't write comment data: %w", err)
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
		return fmt.Errorf("can't get meta: %w", err)
	}

	if err = json.NewEncoder(w).Encode(m); err != nil {
		return fmt.Errorf("can't encode meta: %w", err)
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
		return 0, fmt.Errorf("failed to import meta for site %s: %w", siteID, err)
	}

	if m.Version != nativeVersion && m.Version != 0 { // this version allows back compatibility with 0 version
		return 0, fmt.Errorf("unexpected import file version %d", m.Version)
	}

	if e := n.DataStore.DeleteAll(siteID); e != nil {
		return 0, e
	}

	var failed, total, comments int64
	var decodeErr error

	concurrent := defaultConcurrent
	if n.Concurrent > 0 {
		concurrent = n.Concurrent
	}
	grp := syncs.NewSizedGroup(concurrent, syncs.Preemptive)

	for {
		// read each record as raw JSON first: the decoder can't move past a syntax error or a truncated record
		// and returns it again on every call, while a well-formed record with a wrong value is skipped below
		var rec json.RawMessage
		if err = dec.Decode(&rec); err != nil {
			if err != io.EOF {
				decodeErr = fmt.Errorf("can't decode comment after %d records: %w", total, err)
			}
			break
		}
		total++
		comment := store.Comment{}
		if err = json.Unmarshal(rec, &comment); err != nil {
			atomic.AddInt64(&failed, 1)
			log.Printf("[WARN] can't decode comment record %d, %s", total, err)
			continue
		}
		comment.Imported = true
		// exports made by earlier versions stored the blocked status of the time inside comments;
		// it is kept in the users metadata, and leaving it in the comment marks the user blocked forever
		comment.User.Blocked = false

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
	log.Printf("[INFO] imported %d comments from %d records", comments, total)

	// metas are restored even when some comments failed, so blocks, verified users, read-only posts
	// and user details are not lost because of one bad record
	var errs []error
	if decodeErr != nil {
		errs = append(errs, decodeErr)
	}
	if failed > 0 {
		errs = append(errs, fmt.Errorf("failed to save %d comments", failed))
	}
	errs = append(errs, n.DataStore.SetMetas(siteID, m.Users, m.Posts))
	return int(comments), errors.Join(errs...)
}
