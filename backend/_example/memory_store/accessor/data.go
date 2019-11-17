/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package accessor

import (
	"log"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store/engine"

	"github.com/umputun/remark/backend/app/store"
)

const lastLimit = 1000

// MemData implements in-memory data store
type MemData struct {
	posts     map[string][]store.Comment // key is siteID
	metaUsers map[string]metaUser        // key is userID
	metaPosts map[store.Locator]metaPost // key is post's locator
	sync.RWMutex
}

type metaPost struct {
	PostURL  string
	SiteID   string
	ReadOnly bool
}

type metaUser struct {
	UserID       string
	SiteID       string
	Verified     bool
	Blocked      bool
	BlockedUntil time.Time
	Details      engine.UserDetailEntry
}

// NewMemData makes in-memory engine.
func NewMemData() *MemData {

	result := &MemData{
		posts:     map[string][]store.Comment{},
		metaUsers: map[string]metaUser{},
		metaPosts: map[store.Locator]metaPost{},
	}
	return result
}

// Create new comment
func (m *MemData) Create(comment store.Comment) (commentID string, err error) {

	if ro, e := m.Flag(engine.FlagRequest{Flag: engine.ReadOnly, Locator: comment.Locator}); e == nil && ro {
		return "", errors.Errorf("post %s is read-only", comment.Locator.URL)
	}

	m.Lock()
	defer m.Unlock()
	comments := m.posts[comment.Locator.SiteID]
	for _, c := range comments { // don't allow duplicated IDs
		if c.ID == comment.ID {
			return "", errors.New("dup key")
		}
	}
	comments = append(comments, comment)
	m.posts[comment.Locator.SiteID] = comments
	return comment.ID, nil
}

// Find returns all comments for post and sorts results
func (m *MemData) Find(req engine.FindRequest) (comments []store.Comment, err error) {
	m.RLock()
	defer m.RUnlock()

	comments = []store.Comment{}

	if req.Sort == "" {
		req.Sort = "time"
	}

	switch {

	case req.Locator.SiteID != "" && req.Locator.URL != "": // find comments for site and url
		comments = m.match(m.posts[req.Locator.SiteID], func(c store.Comment) bool {
			return c.Locator == req.Locator && (req.Since.IsZero() || c.Timestamp.After(req.Since))
		})

	case req.Locator.SiteID != "" && req.Locator.URL == "" && req.UserID == "": // find last comments for site
		if req.Limit > lastLimit || req.Limit == 0 {
			req.Limit = lastLimit
		}
		if req.Since.IsZero() {
			req.Since = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		}

		comments = m.match(m.posts[req.Locator.SiteID], func(c store.Comment) bool {
			return !c.Deleted && c.Timestamp.After(req.Since)
		})
		comments = engine.SortComments(comments, "-time")
		if len(comments) > req.Limit {
			comments = comments[:req.Limit]
		}
		return comments, nil

	case req.Locator.SiteID != "" && req.UserID != "": // find comments for user
		comments = m.match(m.posts[req.Locator.SiteID], func(c store.Comment) bool {
			return c.User.ID == req.UserID
		})
	}

	comments = engine.SortComments(comments, req.Sort)
	if req.Skip > 0 && req.Skip > len(comments) {
		return []store.Comment{}, nil
	}
	if req.Skip > 0 && req.Skip < len(comments) {
		comments = comments[req.Skip:]
	}

	if req.Limit > 0 && req.Limit < len(comments) {
		comments = comments[:req.Limit]
	}

	return comments, err
}

// Get returns comment for locator.URL and commentID string
func (m *MemData) Get(req engine.GetRequest) (comment store.Comment, err error) {
	m.RLock()
	defer m.RUnlock()
	return m.get(req.Locator, req.CommentID)
}

// Update updates comment for locator.URL with mutable part of comment
func (m *MemData) Update(comment store.Comment) error {
	m.Lock()
	defer m.Unlock()
	return m.updateComment(comment)
}

// Count returns number of comments for post or user
func (m *MemData) Count(req engine.FindRequest) (count int, err error) {
	m.RLock()
	defer m.RUnlock()

	switch {
	case req.Locator.URL != "": // comment's count for post
		comments := m.match(m.posts[req.Locator.SiteID], func(c store.Comment) bool {
			return c.Locator == req.Locator && !c.Deleted
		})
		return len(comments), nil
	case req.UserID != "":
		comments := m.match(m.posts[req.Locator.SiteID], func(c store.Comment) bool {
			return c.User.ID == req.UserID && !c.Deleted
		})
		return len(comments), nil
	default:
		return 0, errors.Errorf("invalid count request %+v", req)
	}
}

// Info get post(s) meta info
func (m *MemData) Info(req engine.InfoRequest) (res []store.PostInfo, err error) {
	m.RLock()
	defer m.RUnlock()
	res = []store.PostInfo{}

	if req.Locator.URL != "" { // post info
		comments := m.match(m.posts[req.Locator.SiteID], func(c store.Comment) bool {
			return c.Locator == req.Locator
		})
		if len(comments) == 0 {
			return nil, errors.New("not found")
		}
		info := store.PostInfo{
			URL:      req.Locator.URL,
			Count:    len(comments),
			ReadOnly: false,
			FirstTS:  comments[0].Timestamp.UTC(),
			LastTS:   comments[len(comments)-1].Timestamp.UTC(),
		}
		// set read-only from age and manual bucket
		info.ReadOnly = req.ReadOnlyAge > 0 && !info.FirstTS.IsZero() &&
			info.FirstTS.AddDate(0, 0, req.ReadOnlyAge).Before(time.Now())
		if !info.ReadOnly {
			v := m.checkFlag(engine.FlagRequest{Flag: engine.ReadOnly, Locator: req.Locator})
			info.ReadOnly = v
		}
		return []store.PostInfo{info}, nil
	}

	if req.Locator.URL == "" && req.Locator.SiteID != "" { // site info (list)
		if req.Limit <= 0 {
			req.Limit = 1000
		}
		if req.Skip < 0 {
			req.Skip = 0
		}

		infoAll := map[store.Locator]store.PostInfo{}
		for _, c := range m.posts[req.Locator.SiteID] {
			var info store.PostInfo
			var ok bool
			if info, ok = infoAll[c.Locator]; !ok {
				info = store.PostInfo{URL: c.Locator.URL, FirstTS: c.Timestamp.UTC()}
			}
			info.Count++
			info.LastTS = c.Timestamp.UTC()
			infoAll[c.Locator] = info
		}

		for _, v := range infoAll {
			res = append(res, v)
		}
		sort.Slice(res, func(i, j int) bool {
			return res[i].URL > res[j].URL
		})

		if req.Skip > 0 {
			if req.Skip >= len(res) {
				return []store.PostInfo{}, nil
			}
			res = res[req.Skip:]
		}

		if req.Limit > 0 && req.Limit < len(res) {
			res = res[:req.Limit]
		}
		return res, nil
	}

	return nil, errors.Errorf("invalid info request %+v", req)
}

// Flag sets and gets flag values
func (m *MemData) Flag(req engine.FlagRequest) (val bool, err error) {
	m.Lock()
	defer m.Unlock()

	if req.Update == engine.FlagNonSet { // read flag value, no update requested
		return m.checkFlag(req), nil
	}
	// write flag value
	return m.setFlag(req)
}

// ListFlags get list of flagged keys, like blocked & verified user
// works for full locator (post flags) or with userID
func (m *MemData) ListFlags(req engine.FlagRequest) (res []interface{}, err error) {
	m.RLock()
	defer m.RUnlock()

	res = []interface{}{}

	switch req.Flag {
	case engine.Verified:
		for _, u := range m.metaUsers {
			if u.SiteID == req.Locator.SiteID {
				res = append(res, u.UserID)
			}
		}
		return res, nil

	case engine.Blocked:
		log.Printf("%+v", m.metaUsers)
		for _, u := range m.metaUsers {
			if u.SiteID == req.Locator.SiteID && u.Blocked && u.BlockedUntil.After(time.Now()) {
				res = append(res, store.BlockedUser{ID: u.UserID, Until: u.BlockedUntil})
			}
		}
		return res, nil
	}

	return nil, errors.Errorf("flag %s not listable", req.Flag)
}

// UserDetail sets and gets detail values
func (m *MemData) UserDetail(req engine.UserDetailRequest) ([]engine.UserDetailEntry, error) {
	switch req.Detail {
	case engine.Email:
		if req.UserID == "" {
			return nil, errors.New("userid cannot be empty in request for single detail")
		}

		m.Lock()
		defer m.Unlock()

		if req.Update == "" { // read detail value, no update requested
			return m.getUserDetail(req)
		}

		return m.setUserDetail(req)
	case engine.All:
		// list of all details returned in case request is a read request
		// (Update is not set) and does not have UserID or Detail set
		if req.Update == "" && req.UserID == "" { // read list of all details
			m.Lock()
			defer m.Unlock()
			return m.listDetails(req.Locator)
		}
		return nil, errors.New("unsupported request with userdetail all")
	default:
		return nil, errors.Errorf("unsupported detail %q", req.Detail)
	}
}

// Delete post(s) by id or by userID
func (m *MemData) Delete(req engine.DeleteRequest) error {

	m.Lock()
	defer m.Unlock()

	switch {
	case req.Locator.URL != "" && req.CommentID != "": // delete comment
		return m.deleteComment(req.Locator, req.CommentID, req.DeleteMode)

	case req.Locator.SiteID != "" && req.UserID != "" && req.CommentID == "": // delete user
		comments := m.match(m.posts[req.Locator.SiteID], func(c store.Comment) bool {
			return c.User.ID == req.UserID && !c.Deleted
		})
		for _, c := range comments {
			if e := m.deleteComment(c.Locator, c.ID, req.DeleteMode); e != nil {
				return e
			}
		}
		return nil

	case req.Locator.SiteID != "" && req.Locator.URL == "" && req.CommentID == "" && req.UserID == "": // delete site
		if _, ok := m.posts[req.Locator.SiteID]; !ok {
			return errors.New("not found")
		}
		m.posts[req.Locator.SiteID] = []store.Comment{}
		return nil
	}

	return errors.Errorf("invalid delete request %+v", req)
}

func (m *MemData) deleteComment(loc store.Locator, id string, mode store.DeleteMode) error {

	comments := m.match(m.posts[loc.SiteID], func(c store.Comment) bool {
		return c.Locator == loc && c.ID == id
	})
	if len(comments) == 0 {
		return errors.New("not found")
	}

	comments[0].SetDeleted(mode)
	return m.updateComment(comments[0])
}

// Close store
func (m *MemData) Close() error {
	return nil
}

func (m *MemData) checkFlag(req engine.FlagRequest) (val bool) {
	switch req.Flag {
	case engine.Blocked:
		if meta, ok := m.metaUsers[req.UserID]; ok {
			if meta.SiteID != req.Locator.SiteID {
				return false
			}
			return meta.Blocked && meta.BlockedUntil.After(time.Now())
		}
	case engine.Verified:
		if meta, ok := m.metaUsers[req.UserID]; ok {
			if meta.SiteID != req.Locator.SiteID {
				return false
			}
			return meta.Verified
		}
	case engine.ReadOnly:
		if meta, ok := m.metaPosts[req.Locator]; ok {
			return meta.ReadOnly
		}
	}
	return false
}

func (m *MemData) setFlag(req engine.FlagRequest) (res bool, err error) {

	status := false
	if req.Update == engine.FlagTrue {
		status = true
	}

	switch req.Flag {

	case engine.Blocked:
		until := time.Time{}
		if status {
			until = time.Now().AddDate(100, 0, 0) // permanent is 100years
			if req.TTL > 0 {
				until = time.Now().Add(req.TTL)
			}
		}
		meta := m.metaUsers[req.UserID]
		meta = metaUser{
			UserID:       req.UserID,
			SiteID:       req.Locator.SiteID,
			Blocked:      status,
			BlockedUntil: until,
		}
		m.metaUsers[req.UserID] = meta

	case engine.Verified:
		meta := m.metaUsers[req.UserID]
		meta = metaUser{
			UserID:   req.UserID,
			SiteID:   req.Locator.SiteID,
			Verified: status,
		}
		m.metaUsers[req.UserID] = meta

	case engine.ReadOnly:
		info, ok := m.metaPosts[req.Locator]
		if !ok {
			info.SiteID = req.Locator.SiteID
			info.PostURL = req.Locator.URL
		}
		info.ReadOnly = status
		m.metaPosts[req.Locator] = info
	}
	return status, errors.Wrapf(err, "failed to set flag %+v", req)
}

// getUserDetail returns requested userDetail value
func (m *MemData) getUserDetail(req engine.UserDetailRequest) ([]engine.UserDetailEntry, error) {
	if meta, ok := m.metaUsers[req.UserID]; ok {
		if meta.SiteID != req.Locator.SiteID {
			return []engine.UserDetailEntry{}, nil
		}
		switch req.Detail {
		case engine.Email:
			return []engine.UserDetailEntry{{UserID: req.UserID, Email: meta.Details.Email}}, nil
		}
	}

	return []engine.UserDetailEntry{}, nil
}

// setUserDetail sets requested userDetail, deletion of the absent entry doesn't produce error
func (m *MemData) setUserDetail(req engine.UserDetailRequest) ([]engine.UserDetailEntry, error) {
	var entry metaUser
	if meta, ok := m.metaUsers[req.UserID]; ok {
		if meta.SiteID != req.Locator.SiteID {
			return []engine.UserDetailEntry{}, nil
		}
		entry = meta
	}

	if entry == (metaUser{}) {
		entry = metaUser{
			UserID:  req.UserID,
			SiteID:  req.Locator.SiteID,
			Details: engine.UserDetailEntry{UserID: req.UserID},
		}
	}

	switch req.Detail {
	case engine.Email:
		entry.Details.Email = req.Update
		m.metaUsers[req.UserID] = entry
		return []engine.UserDetailEntry{{UserID: req.UserID, Email: req.Update}}, nil
	}

	return []engine.UserDetailEntry{}, nil
}

// listDetails lists all available users details. Map key is userID.
func (m *MemData) listDetails(loc store.Locator) ([]engine.UserDetailEntry, error) {
	var res []engine.UserDetailEntry
	for _, u := range m.metaUsers {
		if u.SiteID == loc.SiteID {
			res = append(res, u.Details)
		}
	}
	return res, nil
}

func (m *MemData) get(loc store.Locator, commentID string) (store.Comment, error) {
	comments := m.match(m.posts[loc.SiteID], func(c store.Comment) bool {
		return c.Locator == loc && c.ID == commentID
	})
	if len(comments) == 0 {
		return store.Comment{}, errors.New("not found")
	}
	return comments[0], nil
}

func (m *MemData) updateComment(comment store.Comment) error {
	comments := m.posts[comment.Locator.SiteID]
	for i, c := range comments {
		if c.ID == comment.ID && c.Locator == comment.Locator {
			c.Text = comment.Text
			c.Orig = comment.Orig
			c.Score = comment.Score
			c.Votes = comment.Votes
			c.Pin = comment.Pin
			c.Deleted = comment.Deleted
			c.User = comment.User
			comments[i] = c
			m.posts[comment.Locator.SiteID] = comments
			return nil
		}
	}
	return errors.New("not found")
}

func (m *MemData) match(comments []store.Comment, fn func(c store.Comment) bool) (res []store.Comment) {
	res = []store.Comment{}
	for _, c := range comments {
		if fn(c) {
			res = append(res, c)
		}
	}
	return res
}
