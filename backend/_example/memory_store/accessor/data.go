/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package accessor

import (
	"fmt"
	"sort"
	"sync"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
)

const lastLimit = 1000

// MemData implements in-memory data store
type MemData struct {
	posts     map[string][]store.Comment // key is siteID
	metaUsers map[string]metaUser        // key is userID
	metaPosts map[store.Locator]metaPost // key is post's locator
	mu        sync.RWMutex
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
		return "", fmt.Errorf("post %s is read-only", comment.Locator.URL)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	comments := m.posts[comment.Locator.SiteID]
	for _, c := range comments { // don't allow duplicated IDs
		if c.ID == comment.ID {
			return "", fmt.Errorf("dup key")
		}
	}
	comments = append(comments, comment)
	m.posts[comment.Locator.SiteID] = comments
	return comment.ID, nil
}

// Find returns all comments for post and sorts results
func (m *MemData) Find(req engine.FindRequest) (comments []store.Comment, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

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
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.get(req.Locator, req.CommentID)
}

// Update updates comment for locator.URL with mutable part of comment
func (m *MemData) Update(comment store.Comment) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateComment(comment)
}

// Count returns number of comments for post or user
func (m *MemData) Count(req engine.FindRequest) (count int, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

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
		return 0, fmt.Errorf("invalid count request %+v", req)
	}
}

// Info get post(s) meta info
func (m *MemData) Info(req engine.InfoRequest) (res []store.PostInfo, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res = []store.PostInfo{}

	if req.Locator.URL != "" { // post info
		comments := m.match(m.posts[req.Locator.SiteID], func(c store.Comment) bool {
			return c.Locator == req.Locator
		})
		if len(comments) == 0 {
			return nil, fmt.Errorf("not found")
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

	return nil, fmt.Errorf("invalid info request %+v", req)
}

// Flag sets and gets flag values
func (m *MemData) Flag(req engine.FlagRequest) (val bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if req.Update == engine.FlagNonSet { // read flag value, no update requested
		return m.checkFlag(req), nil
	}
	// write flag value
	return m.setFlag(req)
}

// ListFlags get list of flagged keys, like blocked & verified user
// works for full locator (post flags) or with userID
func (m *MemData) ListFlags(req engine.FlagRequest) (res []interface{}, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

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
		log.Printf("[INFO] metaUsers: %+v", m.metaUsers)
		for _, u := range m.metaUsers {
			if u.SiteID == req.Locator.SiteID && u.Blocked && u.BlockedUntil.After(time.Now()) {
				res = append(res, store.BlockedUser{ID: u.UserID, Until: u.BlockedUntil})
			}
		}
		return res, nil
	}

	return nil, fmt.Errorf("flag %s not listable", req.Flag)
}

// UserDetail sets or gets single detail value, or gets all details fo§r requested site.
// UserDetail returns list even for single entry request is a compromise in order to have both single detail getting and setting
// and all site's details listing under the same function (and not to extend engine interface by two separate functions).
func (m *MemData) UserDetail(req engine.UserDetailRequest) ([]engine.UserDetailEntry, error) {
	switch req.Detail {
	case engine.UserEmail, engine.UserTelegram:
		if req.UserID == "" {
			return nil, fmt.Errorf("userid cannot be empty in request for single detail")
		}

		m.mu.Lock()
		defer m.mu.Unlock()

		if req.Update == "" { // read detail value, no update requested
			return m.getUserDetail(req)
		}

		return m.setUserDetail(req)
	case engine.AllUserDetails:
		// list of all details returned in case request is a read request
		// (Update is not set) and does not have UserID or Detail set
		if req.Update == "" && req.UserID == "" { // read list of all details
			m.mu.Lock()
			defer m.mu.Unlock()
			return m.listDetails(req.Locator)
		}
		return nil, fmt.Errorf("unsupported request with userdetail all")
	default:
		return nil, fmt.Errorf("unsupported detail %q", req.Detail)
	}
}

// Delete post(s), user, comment, user details, or everything
func (m *MemData) Delete(req engine.DeleteRequest) error {

	m.mu.Lock()
	defer m.mu.Unlock()

	switch {
	case req.UserDetail != "": // delete user detail
		return m.deleteUserDetail(req.Locator, req.UserID, req.UserDetail)
	case req.Locator.URL != "" && req.CommentID != "" && req.UserDetail == "": // delete comment
		return m.deleteComment(req.Locator, req.CommentID, req.DeleteMode)

	case req.Locator.SiteID != "" && req.UserID != "" && req.CommentID == "" && req.UserDetail == "": // delete user
		comments := m.match(m.posts[req.Locator.SiteID], func(c store.Comment) bool {
			return c.User.ID == req.UserID && !c.Deleted
		})
		for _, c := range comments {
			if e := m.deleteComment(c.Locator, c.ID, req.DeleteMode); e != nil {
				return e
			}
		}
		return m.deleteUserDetail(req.Locator, req.UserID, engine.AllUserDetails)

	case req.Locator.SiteID != "" && req.Locator.URL == "" && req.CommentID == "" && req.UserID == "" && req.UserDetail == "": // delete site
		if _, ok := m.posts[req.Locator.SiteID]; !ok {
			return fmt.Errorf("not found")
		}
		m.posts[req.Locator.SiteID] = []store.Comment{}
		return nil
	}

	return fmt.Errorf("invalid delete request %+v", req)
}

func (m *MemData) deleteComment(loc store.Locator, id string, mode store.DeleteMode) error {

	comments := m.match(m.posts[loc.SiteID], func(c store.Comment) bool {
		return c.Locator == loc && c.ID == id
	})
	if len(comments) == 0 {
		return fmt.Errorf("not found")
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
		meta := metaUser{
			UserID:       req.UserID,
			SiteID:       req.Locator.SiteID,
			Blocked:      status,
			BlockedUntil: until,
		}
		m.metaUsers[req.UserID] = meta

	case engine.Verified:
		meta := metaUser{
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
	if err != nil {
		return false, fmt.Errorf("failed to set flag %+v: %w", req, err)
	}
	return status, nil
}

// getUserDetail returns UserDetailEntry with requested userDetail (omitting other details)
// as an only element of the slice.
func (m *MemData) getUserDetail(req engine.UserDetailRequest) ([]engine.UserDetailEntry, error) {
	if meta, ok := m.metaUsers[req.UserID]; ok {
		if meta.SiteID != req.Locator.SiteID {
			return []engine.UserDetailEntry{}, nil
		}
		switch req.Detail {
		case engine.UserEmail:
			return []engine.UserDetailEntry{{UserID: req.UserID, Email: meta.Details.Email}}, nil
		case engine.UserTelegram:
			return []engine.UserDetailEntry{{UserID: req.UserID, Telegram: meta.Details.Telegram}}, nil
		}
	}

	return []engine.UserDetailEntry{}, nil
}

// setUserDetail sets requested userDetail, returning complete updated UserDetailEntry as an onlyIps
// element of the slice in case of success
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
	case engine.UserEmail:
		entry.Details.Email = req.Update
		m.metaUsers[req.UserID] = entry
		return []engine.UserDetailEntry{{UserID: req.UserID, Email: req.Update}}, nil
	case engine.UserTelegram:
		entry.Details.Telegram = req.Update
		m.metaUsers[req.UserID] = entry
		return []engine.UserDetailEntry{{UserID: req.UserID, Telegram: req.Update}}, nil
	}

	return []engine.UserDetailEntry{}, nil
}

// listDetails lists all available users details for given siteID
func (m *MemData) listDetails(loc store.Locator) ([]engine.UserDetailEntry, error) {
	var res []engine.UserDetailEntry
	for _, u := range m.metaUsers {
		if u.SiteID == loc.SiteID {
			res = append(res, u.Details)
		}
	}
	return res, nil
}

// deleteUserDetail deletes requested UserDetail or whole UserDetailEntry,
// deletion of the absent entry doesn't produce error.
// Trying to delete user with wrong siteID doesn't to anything and doesn't produce error.
func (m *MemData) deleteUserDetail(locator store.Locator, userID string, userDetail engine.UserDetail) error {
	var entry metaUser
	if meta, ok := m.metaUsers[userID]; ok {
		if meta.SiteID != locator.SiteID {
			return nil
		}
		entry = meta
	}

	if entry == (metaUser{}) || entry.Details == (engine.UserDetailEntry{}) {
		// absent entry means that we should not do anything
		return nil
	}

	switch userDetail {
	case engine.UserEmail:
		entry.Details.Email = ""
	case engine.UserTelegram:
		entry.Details.Telegram = ""
	case engine.AllUserDetails:
		entry.Details = engine.UserDetailEntry{UserID: userID}
	}

	if entry.Details == (engine.UserDetailEntry{UserID: userID}) {
		// no user details are stored, empty details entry altogether
		entry.Details = engine.UserDetailEntry{}
	}

	m.metaUsers[userID] = entry
	return nil
}

func (m *MemData) get(loc store.Locator, commentID string) (store.Comment, error) {
	comments := m.match(m.posts[loc.SiteID], func(c store.Comment) bool {
		return c.Locator == loc && c.ID == commentID
	})
	if len(comments) == 0 {
		return store.Comment{}, fmt.Errorf("not found")
	}
	return comments[0], nil
}

func (m *MemData) updateComment(comment store.Comment) error {
	comments := m.posts[comment.Locator.SiteID]
	for i, c := range comments {
		if c.ID != comment.ID || c.Locator != comment.Locator {
			continue
		}
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
	return fmt.Errorf("not found")
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
