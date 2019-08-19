package store

import (
	"html/template"
	"regexp"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
)

// Comment represents a single comment with optional reference to its parent
type Comment struct {
	ID          string                 `json:"id" bson:"_id"`
	ParentID    string                 `json:"pid"`
	Text        string                 `json:"text"`
	Orig        string                 `json:"orig,omitempty"`
	User        User                   `json:"user"`
	Locator     Locator                `json:"locator"`
	Score       int                    `json:"score"`
	Votes       map[string]bool        `json:"votes,omitempty"`
	VotedIPs    map[string]VotedIPInfo `json:"voted_ips,omitempty"` // voted ips (hashes) with TS
	Vote        int                    `json:"vote"`                // vote for the current user, -1/1/0.
	Controversy float64                `json:"controversy,omitempty"`
	Timestamp   time.Time              `json:"time" bson:"time"`
	Edit        *Edit                  `json:"edit,omitempty" bson:"edit,omitempty"` // pointer to have empty default in json response
	Pin         bool                   `json:"pin,omitempty" bson:"pin,omitempty"`
	Deleted     bool                   `json:"delete,omitempty" bson:"delete"`
	PostTitle   string                 `json:"title,omitempty" bson:"title"`
}

// Locator keeps site and url of the post
type Locator struct {
	SiteID string `json:"site,omitempty" bson:"site"`
	URL    string `json:"url"`
}

// Edit indication
type Edit struct {
	Timestamp time.Time `json:"time" bson:"time"`
	Summary   string    `json:"summary"`
}

// PostInfo holds summary for given post url
type PostInfo struct {
	URL      string    `json:"url"`
	Count    int       `json:"count"`
	ReadOnly bool      `json:"read_only,omitempty" bson:"read_only,omitempty"`
	FirstTS  time.Time `json:"first_time,omitempty" bson:"first_time,omitempty"`
	LastTS   time.Time `json:"last_time,omitempty" bson:"last_time,omitempty"`
}

// BlockedUser holds id and ts for blocked user
type BlockedUser struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Until time.Time `json:"time"`
}

// VotedIPInfo keeps timestamp and voting value (direction). Used as VotedIPs value
type VotedIPInfo struct {
	Timestamp time.Time
	Value     bool
}

// DeleteMode defines how much comment info will be erased
type DeleteMode int

// DeleteMode enum
const (
	SoftDelete DeleteMode = 0
	HardDelete DeleteMode = 1
)

// Maximum length for URL text shortening.
const shortURLLen = 48
const snippetLen = 200

// PrepareUntrusted pre-processes a comment received from untrusted source by clearing all
// autogen fields and reset everything users not supposed to provide
func (c *Comment) PrepareUntrusted() {
	c.ID = ""                 // don't allow user to define ID, force auto-gen
	c.Timestamp = time.Time{} // reset time, force auto-gen
	c.Votes = make(map[string]bool)
	c.Score = 0
	c.Edit = nil
	c.Pin = false
	c.Deleted = false
}

// SetDeleted clears comment info, reset to deleted state. hard flag will clear all user info as well
func (c *Comment) SetDeleted(mode DeleteMode) {
	c.Text = ""
	c.Orig = ""
	c.Score = 0
	c.Votes = map[string]bool{}
	c.Edit = nil
	c.Deleted = true
	c.Pin = false

	if mode == HardDelete {
		c.User.Name = "deleted"
		c.User.ID = "deleted"
		c.User.Picture = ""
		c.User.IP = ""
	}
}

// Sanitize clean dangerous html/js from the comment
func (c *Comment) Sanitize() {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").Matching(regexp.MustCompile("^language-[a-zA-Z0-9]+$")).OnElements("code")
	c.Text = p.Sanitize(c.Text)
	c.Orig = p.Sanitize(c.Orig)
	c.User.ID = template.HTMLEscapeString(c.User.ID)
	c.User.Name = template.HTMLEscapeString(c.User.Name)
	c.User.Picture = p.Sanitize(c.User.Picture)
}

// Snippet from comment's text
func (c *Comment) Snippet(limit int) string {
	if limit <= 0 {
		limit = snippetLen
	}
	cleanText := strings.Replace(c.Text, "\n", " ", -1)
	size := len([]rune(cleanText))
	if size < limit {
		return cleanText
	}
	snippet := []rune(cleanText)[:size]
	// go back in snippet and found the first space
	for i := len(snippet) - 1; i >= 0; i-- {
		if snippet[i] == ' ' {
			snippet = snippet[:i]
			break
		}
	}
	return string(snippet) + " ..."
}
