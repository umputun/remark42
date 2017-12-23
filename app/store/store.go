package store

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"log"
	"time"
)

// Comment represents a single comment with optinal reference to its parent
type Comment struct {
	ID        string          `json:"id"`
	ParentID  string          `json:"pid"`
	Text      string          `json:"text"`
	User      User            `json:"user"`
	Locator   Locator         `json:"locator"`
	Score     int             `json:"score"`
	Votes     map[string]bool `json:"votes"`
	Timestamp time.Time       `json:"time"`
}

// Locator keeps site and url of the post
type Locator struct {
	SiteID string `json:"site,omitempty"`
	URL    string `json:"url"`
}

// User holds user-related info
type User struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	Picture string `json:"picture"`
	Profile string `json:"profile"`
	Admin   bool   `json:"admin"`
	IP      string `json:"-"`
}

// Request is a container for all finds
type Request struct {
	Locator Locator `json:"locator"`
	Sort    string  `json:"sort"`
	Offset  int     `json:"offset"`
	Limit   int     `json:"limit"`
}

// Interface defines basic CRUD for comments
type Interface interface {
	Create(comment Comment) (commentID string, err error)
	Delete(locator Locator, commentID string) error
	Find(request Request) ([]Comment, error)
	Last(locator Locator, max int) ([]Comment, error)
	Get(locator Locator, commentID string) (Comment, error)
	Vote(locator Locator, commentID string, userID string, val bool) (Comment, error)
	Count(locator Locator) (int, error)

	SetBlock(locator Locator, userID string, status bool) error
	IsBlocked(locator Locator, userID string) bool
}

func makeCommentID() string {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("[ERROR] can't get randoms, %s", err)
	}
	s := sha1.New()
	if _, err := s.Write(b); err != nil {
		log.Fatalf("[ERROR] can't make sha1 for random, %s", err)
	}
	return fmt.Sprintf("%x", s.Sum(nil))
}
