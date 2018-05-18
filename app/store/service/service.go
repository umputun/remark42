package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/umputun/remark/app/store"
	"github.com/umputun/remark/app/store/engine"
)

// DataStore wraps store.Interface with additional methods
type DataStore struct {
	engine.Interface
	EditDuration   time.Duration
	Secret         string
	MaxCommentSize int
}

const defaultCommentMaxSize = 2000

// Create prepares comment and forward to Interface.Create
func (s *DataStore) Create(comment store.Comment) (commentID string, err error) {
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

	comment.Sanitize()            // clear potentially dangerous js from all parts of comment
	comment.User.HashIP(s.Secret) // replace ip by hash

	return s.Interface.Create(comment)
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
