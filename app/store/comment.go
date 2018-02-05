package store

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"strings"
	"time"

	"github.com/alecthomas/template"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pkg/errors"
)

// Comment represents a single comment with optional reference to its parent
type Comment struct {
	ID        string          `json:"id"`
	ParentID  string          `json:"pid"`
	Text      string          `json:"text"`
	User      User            `json:"user"`
	Locator   Locator         `json:"locator"`
	Score     int             `json:"score"`
	Votes     map[string]bool `json:"votes"`
	Timestamp time.Time       `json:"time"`
	Pin       bool            `json:"pin,omitempty"`
	Edit      *Edit           `json:"edit,omitempty"`
	Deleted   bool            `json:"delete,omitempty"`
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
	Blocked bool   `json:"block,omitempty"`
	IP      string `json:"-"`
}

// Edit indication
type Edit struct {
	Timestamp time.Time `json:"time"`
	Summary   string    `json:"summary"`
}

// PostInfo holds summary for given post url
type PostInfo struct {
	URL   string `json:"url"`
	Count int    `json:"count"`
}

// GenID generates sha1(random) string
func (c *Comment) GenID() error {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return errors.Wrap(err, "can't get random")
	}
	s := sha1.New()
	if _, err := s.Write(b); err != nil {
		return errors.Wrap(err, "can't make sha1 for random")
	}
	c.ID = fmt.Sprintf("%x", s.Sum(nil))
	return nil
}

// Sanitize clean dangerous html/js from the comment
func (c *Comment) Sanitize() {
	p := bluemonday.UGCPolicy()
	c.Text = p.Sanitize(c.Text)
	c.User.ID = template.HTMLEscapeString(c.User.ID)
	c.User.Name = template.HTMLEscapeString(c.User.Name)
	c.User.Picture = p.Sanitize(c.User.Picture)
	c.User.Profile = p.Sanitize(c.User.Profile)

	c.Text = strings.Replace(c.Text, "\n", "", -1)
	c.Text = strings.Replace(c.Text, "\t", "", -1)
}

// Mask clears comment info, reset to "Deleted/Blocked"
func (c *Comment) Mask() {
	c.Text = "this comment was deleted"
	c.Score = 0
	c.Votes = map[string]bool{}
	c.Edit = nil
}
