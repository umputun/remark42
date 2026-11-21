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

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
)

const lastLimit = 1000

// MemData implements in-memory data store.
// User flags and details are kept per site, as the bolt engine keeps them in the site's own db.
type MemData struct {
	posts    map[string][]store.Comment          // key is siteID
	verified map[siteUser]struct{}               // verified users
	blocked  map[siteUser]time.Time              // blocked users, value is the time the block ends
	details  map[siteUser]engine.UserDetailEntry // users with at least one detail set
	readOnly map[store.Locator]struct{}          // posts made read-only by hand
	mu       sync.RWMutex
}

// siteUser is the key of a user's flags and details on a site
type siteUser struct {
	siteID string
	userID string
}

// NewMemData makes in-memory engine.
func NewMemData() *MemData {

	result := &MemData{
		posts:    map[string][]store.Comment{},
		verified: map[siteUser]struct{}{},
		blocked:  map[siteUser]time.Time{},
		details:  map[siteUser]engine.UserDetailEntry{},
		readOnly: map[store.Locator]struct{}{},
	}
	return result
}

// Create new comment
func (m *MemData) Create(comment store.Comment) (commentID string, err error) {

	// an imported comment is written whatever the post's state, as in the bolt engine
	if ro, e := m.Flag(engine.FlagRequest{Flag: engine.ReadOnly, Locator: comment.Locator}); e == nil && ro && !comment.Imported {
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

	case req.Locator.SiteID != "" && req.UserID != "": // find comments for user, deleted ones excluded as in Count
		comments = m.match(m.posts[req.Locator.SiteID], func(c store.Comment) bool {
			return c.User.ID == req.UserID && !c.Deleted
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
		info, ok := m.postInfos(req.Locator.SiteID)[req.Locator]
		if !ok {
			return nil, fmt.Errorf("not found")
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

		for _, v := range m.postInfos(req.Locator.SiteID) {
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
	return m.setFlag(req), nil
}

// ListFlags get list of flagged keys, like blocked & verified user
// works for full locator (post flags) or with userID
func (m *MemData) ListFlags(req engine.FlagRequest) (res []any, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res = []any{}

	// users are listed by id, as the bolt engine lists its bucket keys
	switch req.Flag {
	case engine.Verified:
		for _, id := range siteUserIDs(m.verified, req.Locator.SiteID) {
			res = append(res, id)
		}
		return res, nil

	case engine.Blocked:
		for _, id := range siteUserIDs(m.blocked, req.Locator.SiteID) {
			if until := m.blocked[siteUser{siteID: req.Locator.SiteID, userID: id}]; until.After(time.Now()) {
				res = append(res, store.BlockedUser{ID: id, Name: m.userName(req.Locator.SiteID, id), Until: until})
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
			return m.getUserDetail(req), nil
		}

		return m.setUserDetail(req), nil
	case engine.AllUserDetails:
		// list of all details returned in case request is a read request
		// (Update is not set) and does not have UserID or Detail set
		if req.Update == "" && req.UserID == "" { // read list of all details
			m.mu.Lock()
			defer m.mu.Unlock()
			return m.listDetails(req.Locator), nil
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
		m.deleteUserDetail(req.Locator, req.UserID, req.UserDetail)
		return nil
	case req.Locator.URL != "" && req.CommentID != "" && req.UserDetail == "": // delete comment
		return m.deleteComment(req.Locator, req.CommentID, req.DeleteMode)

	case req.Locator.SiteID != "" && req.UserID != "" && req.CommentID == "" && req.UserDetail == "": // delete user
		// already deleted comments too, a hard delete has to clear the user's name and id in them
		comments := m.match(m.posts[req.Locator.SiteID], func(c store.Comment) bool {
			return c.User.ID == req.UserID
		})
		for _, c := range comments {
			if e := m.deleteComment(c.Locator, c.ID, req.DeleteMode); e != nil {
				return e
			}
		}
		m.deleteUserDetail(req.Locator, req.UserID, engine.AllUserDetails)
		return nil

	case req.Locator.SiteID != "" && req.Locator.URL == "" && req.CommentID == "" && req.UserID == "" && req.UserDetail == "": // delete site
		if _, ok := m.posts[req.Locator.SiteID]; !ok {
			return fmt.Errorf("not found")
		}
		m.posts[req.Locator.SiteID] = []store.Comment{}
		// user details go with the site, flags stay, as in the bolt engine
		for k := range m.details {
			if k.siteID == req.Locator.SiteID {
				delete(m.details, k)
			}
		}
		return nil
	}

	return fmt.Errorf("invalid delete request %+v", req)
}

func (m *MemData) deleteComment(loc store.Locator, id string, mode store.DeleteMode) error {
	comments := m.posts[loc.SiteID]
	for i := range comments {
		if comments[i].Locator == loc && comments[i].ID == id {
			comments[i].SetDeleted(mode)
			return nil
		}
	}
	return fmt.Errorf("not found")
}

// Close store
func (m *MemData) Close() error {
	return nil
}

func (m *MemData) checkFlag(req engine.FlagRequest) (val bool) {
	key := siteUser{siteID: req.Locator.SiteID, userID: req.UserID}
	switch req.Flag {
	case engine.Blocked:
		until, ok := m.blocked[key]
		return ok && until.After(time.Now())
	case engine.Verified:
		_, ok := m.verified[key]
		return ok
	case engine.ReadOnly:
		_, ok := m.readOnly[req.Locator]
		return ok
	}
	return false
}

func (m *MemData) setFlag(req engine.FlagRequest) bool {
	status := req.Update == engine.FlagTrue
	key := siteUser{siteID: req.Locator.SiteID, userID: req.UserID}

	switch req.Flag {
	case engine.Blocked:
		if !status {
			delete(m.blocked, key)
			break
		}
		until := time.Now().AddDate(100, 0, 0) // permanent is 100years
		if req.TTL > 0 {
			until = time.Now().Add(req.TTL)
		}
		m.blocked[key] = until
	case engine.Verified:
		setMember(m.verified, key, status)
	case engine.ReadOnly:
		setMember(m.readOnly, req.Locator, status)
	}
	return status
}

// setMember adds key to the set or removes it from the set
func setMember[K comparable](set map[K]struct{}, key K, add bool) {
	if add {
		set[key] = struct{}{}
		return
	}
	delete(set, key)
}

// siteUserIDs returns the sorted ids of the site's users in the map
func siteUserIDs[V any](users map[siteUser]V, siteID string) []string {
	ids := []string{}
	for k := range users {
		if k.siteID == siteID {
			ids = append(ids, k.userID)
		}
	}
	sort.Strings(ids)
	return ids
}

// userName returns the user's name from the user's latest comment on the site, deleted ones included,
// as the bolt engine does: they are often all a blocked user has left
func (m *MemData) userName(siteID, userID string) (name string) {
	var latest time.Time
	for _, c := range m.posts[siteID] {
		if c.User.ID == userID && !c.Timestamp.Before(latest) {
			latest, name = c.Timestamp, c.User.Name
		}
	}
	return name
}

// getUserDetail returns UserDetailEntry with requested userDetail (omitting other details)
// as an only element of the slice, or nil if the user has no details.
func (m *MemData) getUserDetail(req engine.UserDetailRequest) []engine.UserDetailEntry {
	if entry, ok := m.details[siteUser{siteID: req.Locator.SiteID, userID: req.UserID}]; ok {
		switch req.Detail {
		case engine.UserEmail:
			return []engine.UserDetailEntry{{UserID: req.UserID, Email: entry.Email}}
		case engine.UserTelegram:
			return []engine.UserDetailEntry{{UserID: req.UserID, Telegram: entry.Telegram}}
		}
	}

	return nil
}

// setUserDetail sets requested userDetail, returning complete updated UserDetailEntry as an only
// element of the slice
func (m *MemData) setUserDetail(req engine.UserDetailRequest) []engine.UserDetailEntry {
	key := siteUser{siteID: req.Locator.SiteID, userID: req.UserID}
	entry := m.details[key]
	entry.UserID = req.UserID

	switch req.Detail {
	case engine.UserEmail:
		entry.Email = req.Update
	case engine.UserTelegram:
		entry.Telegram = req.Update
	}

	m.details[key] = entry
	return []engine.UserDetailEntry{entry}
}

// listDetails lists all available users details for given siteID, sorted by user id
func (m *MemData) listDetails(loc store.Locator) []engine.UserDetailEntry {
	var res []engine.UserDetailEntry
	for k, entry := range m.details {
		if k.siteID == loc.SiteID {
			res = append(res, entry)
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].UserID < res[j].UserID })
	return res
}

// deleteUserDetail deletes requested UserDetail or whole UserDetailEntry,
// deletion of the absent entry doesn't produce error.
func (m *MemData) deleteUserDetail(locator store.Locator, userID string, userDetail engine.UserDetail) {
	key := siteUser{siteID: locator.SiteID, userID: userID}
	entry, ok := m.details[key]
	if !ok {
		// absent entry means that we should not do anything
		return
	}

	switch userDetail {
	case engine.UserEmail:
		entry.Email = ""
	case engine.UserTelegram:
		entry.Telegram = ""
	case engine.AllUserDetails:
		entry = engine.UserDetailEntry{UserID: userID}
	}

	if entry == (engine.UserDetailEntry{UserID: userID}) {
		// no user details are left, drop the entry
		delete(m.details, key)
		return
	}

	m.details[key] = entry
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
		// keep the immutable fields and replace the others, as the bolt engine does
		comment.ParentID, comment.Timestamp, comment.User = c.ParentID, c.Timestamp, c.User
		comments[i] = comment
		return nil
	}
	return fmt.Errorf("not found")
}

// postInfos returns the info of every post of the site, counted as the bolt engine counts it: deleted comments
// are not counted, and first and last are the earliest and latest comments, whatever order they were created in
func (m *MemData) postInfos(siteID string) map[store.Locator]store.PostInfo {
	res := map[store.Locator]store.PostInfo{}
	for _, c := range m.posts[siteID] {
		ts := c.Timestamp.UTC()
		info, ok := res[c.Locator]
		if !ok {
			info = store.PostInfo{URL: c.Locator.URL, FirstTS: ts, LastTS: ts}
		}
		if !c.Deleted {
			info.Count++
		}
		if ts.Before(info.FirstTS) {
			info.FirstTS = ts
		}
		if ts.After(info.LastTS) {
			info.LastTS = ts
		}
		res[c.Locator] = info
	}
	return res
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
