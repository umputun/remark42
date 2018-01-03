package store

import (
	"time"

	"github.com/pkg/errors"
)

// Service wraps store.Interface with additional methods
type Service struct {
	Interface
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

	if _, voted := comment.Votes[userID]; voted {
		return comment, errors.Errorf("user %s already voted for %s", userID, commentID)
	}
	// update votes and score
	comment.Votes[userID] = val

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
	if !comment.Edit.Timestamp.IsZero() {
		return comment, errors.Errorf("comment %s already edited at %s", commentID, comment.Edit.Timestamp)
	}

	comment.Text = text
	comment.Edit = edit
	comment.Edit.Timestamp = time.Now()
	comment = sanitizeComment(comment)
	err = s.Put(locator, comment)
	return comment, err
}
