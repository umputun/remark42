package store

import "time"

// Comment represents a single comment with reference to its parent
type Comment struct {
	ID        int64           `json:"id"`
	ParentID  int64           `json:"pid"`
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
	Create(comment Comment) (int64, error)
	Delete(locator Locator, id int64) error
	Find(request Request) ([]Comment, error)
	Last(locator Locator, max int) ([]Comment, error)
	Get(locator Locator, id int64) (Comment, error)
	Vote(locator Locator, commentID int64, userID string, val bool) (Comment, error)
	Count(locator Locator) (int, error)

	SetBlock(locator Locator, userID string, status bool) error
	IsBlocked(locator Locator, userID string) bool
}
