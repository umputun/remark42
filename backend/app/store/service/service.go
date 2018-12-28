package service

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/admin"
	"github.com/umputun/remark/backend/app/store/engine"
)

// DataStore wraps store.Interface with additional methods
type DataStore struct {
	engine.Interface
	EditDuration   time.Duration
	AdminStore     admin.Store
	MaxCommentSize int
	MaxVotes       int

	// granular locks
	scopedLocks struct {
		sync.Mutex
		sync.Once
		locks map[string]sync.Locker
	}
}

// UserMetaData keeps info about user flags
type UserMetaData struct {
	ID      string `json:"id"`
	Blocked struct {
		Status bool      `json:"status"`
		Until  time.Time `json:"until"`
	} `json:"blocked"`
	Verified bool `json:"verified"`
}

// PostMetaData keeps info about post flags
type PostMetaData struct {
	URL      string `json:"url"`
	ReadOnly bool   `json:"read_only"`
}

const defaultCommentMaxSize = 2000

// UnlimitedVotes doesn't restrict MaxVotes
const UnlimitedVotes = -1

// Create prepares comment and forward to Interface.Create
func (s *DataStore) Create(comment store.Comment) (commentID string, err error) {

	if comment, err = s.prepareNewComment(comment); err != nil {
		return "", errors.Wrap(err, "failed to prepare comment")
	}

	return s.Interface.Create(comment)
}

// prepareNewComment sets new comment fields, hashing and sanitizing data
func (s *DataStore) prepareNewComment(comment store.Comment) (store.Comment, error) {
	// fill ID and time if empty
	if comment.ID == "" {
		comment.ID = uuid.New().String()
	}
	if comment.Timestamp.IsZero() {
		comment.Timestamp = time.Now()
	}
	// reset votes if nothing
	if comment.Votes == nil {
		comment.Votes = make(map[string]bool)
	}
	comment.Sanitize() // clear potentially dangerous js from all parts of comment

	secret, err := s.AdminStore.Key(comment.Locator.SiteID)
	if err != nil {
		return store.Comment{}, errors.Wrapf(err, "can't get secret for site %s", comment.Locator.SiteID)
	}
	comment.User.HashIP(secret) // replace ip by hash
	return comment, nil
}

// SetPin pin/un-pin comment as special
func (s *DataStore) SetPin(locator store.Locator, commentID string, status bool) error {
	comment, err := s.Get(locator, commentID)
	if err != nil {
		return err
	}
	comment.Pin = status
	return s.Put(locator, comment)
}

// Vote for comment by id and locator
func (s *DataStore) Vote(locator store.Locator, commentID string, userID string, val bool) (comment store.Comment, err error) {

	cLock := s.getsScopedLocks(locator.URL) // get lock for URL scope
	cLock.Lock()                            // prevents race on voting
	defer cLock.Unlock()

	comment, err = s.Get(locator, commentID)
	if err != nil {
		return comment, err
	}

	if comment.User.ID == userID && userID != "dev" {
		return comment, errors.Errorf("user %s can not vote for his own comment %s", userID, commentID)
	}

	if comment.Votes == nil {
		comment.Votes = make(map[string]bool)
	}
	v, voted := comment.Votes[userID]

	if voted && v == val {
		return comment, errors.Errorf("user %s already voted for %s", userID, commentID)
	}

	maxVotes := s.MaxVotes // 0 value allowed and treated as "no comments allowed"
	if s.MaxVotes < 0 {    // any negative value reset max votes to unlimited
		maxVotes = UnlimitedVotes
	}

	if maxVotes >= 0 && len(comment.Votes) >= maxVotes {
		return comment, errors.Errorf("maximum number of votes exceeded for comment %s", commentID)
	}

	// reset vote if user changed to opposite
	if voted && v != val {
		delete(comment.Votes, userID)
	}

	// add to voted map if first vote
	if !voted {
		comment.Votes[userID] = val
	}

	// update score
	if val {
		comment.Score++
	} else {
		comment.Score--
	}

	return comment, s.Put(locator, comment)
}

// EditRequest contains fields needed for comment update
type EditRequest struct {
	Text    string
	Orig    string
	Summary string
	Delete  bool
}

// EditComment to edit text and update Edit info
func (s *DataStore) EditComment(locator store.Locator, commentID string, req EditRequest) (comment store.Comment, err error) {
	comment, err = s.Get(locator, commentID)
	if err != nil {
		return comment, err
	}

	// edit allowed in editDuration window only
	if s.EditDuration > 0 && time.Now().After(comment.Timestamp.Add(s.EditDuration)) {
		return comment, errors.Errorf("too late to edit %s", commentID)
	}

	if req.Delete { // delete request
		comment.Deleted = true
		return comment, s.Delete(locator, commentID, store.SoftDelete)
	}

	comment.Text = req.Text
	comment.Orig = req.Orig
	comment.Edit = &store.Edit{
		Timestamp: time.Now(),
		Summary:   req.Summary,
	}

	comment.Sanitize()
	err = s.Put(locator, comment)
	return comment, err
}

// Counts returns postID+count list for given comments
func (s *DataStore) Counts(siteID string, postIDs []string) ([]store.PostInfo, error) {
	res := []store.PostInfo{}
	for _, p := range postIDs {
		if c, err := s.Count(store.Locator{SiteID: siteID, URL: p}); err == nil {
			res = append(res, store.PostInfo{URL: p, Count: c})
		}
	}
	return res, nil
}

// ValidateComment checks if comment size below max and user fields set
func (s *DataStore) ValidateComment(c *store.Comment) error {
	maxSize := s.MaxCommentSize
	if s.MaxCommentSize <= 0 {
		maxSize = defaultCommentMaxSize
	}
	if c.Orig == "" {
		return errors.New("empty comment text")
	}
	if len([]rune(c.Orig)) > maxSize {
		return errors.Errorf("comment text exceeded max allowed size %d (%d)", maxSize, len([]rune(c.Orig)))
	}
	if c.User.ID == "" || c.User.Name == "" {
		return errors.Errorf("empty user info")
	}
	return nil
}

// IsAdmin checks if usesID in the list of admins
func (s *DataStore) IsAdmin(siteID string, userID string) bool {
	for _, a := range s.AdminStore.Admins(siteID) {
		if a == userID {
			return true
		}
	}
	return false
}

// Metas returns metadata for users and posts
func (s *DataStore) Metas(siteID string) (umetas []UserMetaData, pmetas []PostMetaData, err error) {
	umetas = []UserMetaData{}
	pmetas = []PostMetaData{}
	// set posts meta
	posts, err := s.List(siteID, 0, 0)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "can't get list of posts for %s", siteID)
	}
	for _, p := range posts {
		if s.IsReadOnly(store.Locator{SiteID: siteID, URL: p.URL}) {
			pmetas = append(pmetas, PostMetaData{URL: p.URL, ReadOnly: true})
		}

	}

	// set users meta
	m := map[string]UserMetaData{}

	// process blocked users
	blocked, err := s.Blocked(siteID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "can't get list of blocked users for %s", siteID)
	}
	for _, b := range blocked {
		val, ok := m[b.ID]
		if !ok {
			val = UserMetaData{ID: b.ID}
		}
		val.Blocked.Status = true
		val.Blocked.Until = b.Until
		m[b.ID] = val
	}

	// process verified users
	verified, err := s.Verified(siteID)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "can't get list of verified users for %s", siteID)
	}
	for _, v := range verified {
		val, ok := m[v]
		if !ok {
			val = UserMetaData{ID: v}
		}
		val.Verified = true
		m[v] = val
	}

	for _, u := range m {
		umetas = append(umetas, u)
	}
	sort.Slice(umetas, func(i, j int) bool { return umetas[i].ID < umetas[j].ID })

	return umetas, pmetas, nil
}

// SetMetas saves metadata for users and posts
func (s *DataStore) SetMetas(siteID string, umetas []UserMetaData, pmetas []PostMetaData) (err error) {
	errs := new(multierror.Error)

	// save posts metas
	for _, pm := range pmetas {
		if pm.ReadOnly {
			errs = multierror.Append(errs, s.SetReadOnly(store.Locator{SiteID: siteID, URL: pm.URL}, true))
		}
	}

	// save users metas
	for _, um := range umetas {
		if um.Blocked.Status {
			errs = multierror.Append(errs, s.SetBlock(siteID, um.ID, true, time.Until(um.Blocked.Until)))
		}
		if um.Verified {
			errs = multierror.Append(errs, s.SetVerified(siteID, um.ID, true))
		}
	}

	return errs.ErrorOrNil()
}

// getsScopedLocks pull lock from the map if found or create a new one
func (s *DataStore) getsScopedLocks(id string) (lock sync.Locker) {
	s.scopedLocks.Do(func() { s.scopedLocks.locks = map[string]sync.Locker{} })

	s.scopedLocks.Lock()
	lock, ok := s.scopedLocks.locks[id]
	if !ok {
		lock = &sync.Mutex{}
		s.scopedLocks.locks[id] = lock
	}
	s.scopedLocks.Unlock()

	return lock
}
