package store

import (
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Service wraps store.Interface with additional methods
type Service struct {
	Interface
	EditDuration time.Duration
}

// Create prepares comment and forward to Interface.Create
func (s *Service) Create(comment Comment) (commentID string, err error) {
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

	comment.sanitize()    // clear potentially dangerous js from all parts of comment
	comment.User.hashIP() // replace ip by hash

	return s.Interface.Create(comment)
}

// SetPin pin/un-pin comment as special
func (s *Service) SetPin(locator Locator, commentID string, status bool) error {
	comment, err := s.Get(locator, commentID)
	if err != nil {
		return err
	}
	comment.Pin = status
	return s.Put(locator, comment)
}

// Vote for comment by id and locator
func (s *Service) Vote(locator Locator, commentID string, userID string, val bool) (comment Comment, err error) {

	comment, err = s.Get(locator, commentID)
	if err != nil {
		return comment, err
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

// EditComment to edit text and update Edit info
func (s *Service) EditComment(locator Locator, commentID string, text string, edit Edit) (comment Comment, err error) {
	comment, err = s.Get(locator, commentID)
	if err != nil {
		return comment, err
	}
	// edit allowed only once
	if comment.Edit != nil {
		return comment, errors.Errorf("comment %s already edited at %s", commentID, comment.Edit.Timestamp)
	}

	// edit allowed in editDuration window only
	if s.EditDuration > 0 && time.Now().After(comment.Timestamp.Add(s.EditDuration)) {
		return comment, errors.Errorf("too late to edit %s", commentID)
	}

	comment.Text = text
	comment.Edit = &edit
	comment.Edit.Timestamp = time.Now()
	comment.sanitize()
	err = s.Put(locator, comment)
	return comment, err
}

// Counts returns postID+count list for given comments
func (s *Service) Counts(siteID string, postIDs []string) ([]PostInfo, error) {
	res := []PostInfo{}
	for _, p := range postIDs {
		if c, err := s.Count(Locator{SiteID: siteID, URL: p}); err == nil {
			res = append(res, PostInfo{URL: p, Count: c})
		}
	}
	return res, nil
}
