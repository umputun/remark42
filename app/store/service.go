package store

import "github.com/pkg/errors"

// Service wraps store.Interface with additional methods
type Service struct {
	Interface
}

// SetPin pin/un-pin comment as special
func (s *Service) SetPin(locator Locator, commentID string, status bool) error {
	comment, err := s.GetComment(locator, commentID)
	if err != nil {
		return err
	}
	comment.Pin = status
	return s.PutComment(locator, comment)
}

// Vote for comment by id and locator
func (s *Service) Vote(locator Locator, commentID string, userID string, val bool) (comment Comment, err error) {

	comment, err = s.GetComment(locator, commentID)
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

	return comment, s.PutComment(locator, comment)
}
