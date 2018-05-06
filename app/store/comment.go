package store

import (
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"html/template"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
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
	Admin   bool   `json:"admin"`
	Blocked bool   `json:"block,omitempty"`
	IP      string `json:"ip,omitempty"`
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

// BlockedUser holds id and ts for blocked user
type BlockedUser struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Timestamp time.Time `json:"time"`
}

// PrepareUntrusted preprocess comment received from untrusted source by clearing all
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

// SetDeleted clears comment info, reset to deleted state
func (c *Comment) SetDeleted() {
	c.Text = ""
	c.Score = 0
	c.Votes = map[string]bool{}
	c.Edit = nil
	c.Deleted = true
}

// Sanitize clean dangerous html/js from the comment
func (c *Comment) Sanitize() {
	p := bluemonday.UGCPolicy()
	c.Text = p.Sanitize(c.Text)
	c.User.ID = template.HTMLEscapeString(c.User.ID)
	c.User.Name = template.HTMLEscapeString(c.User.Name)
	c.User.Picture = p.Sanitize(c.User.Picture)

	c.Text = strings.Replace(c.Text, "\n", "", -1)
	c.Text = strings.Replace(c.Text, "\t", "", -1)
}

// hashIP replace sensitive fields with hashes
func (u *User) hashIP(secret string) {

	hashVal := func(val string) string {
		if _, err := strconv.ParseUint(val, 16, 64); err == nil || val == "" {
			return val // already hashed
		}
		key := []byte(secret)
		h := hmac.New(sha1.New, key)
		if _, err := h.Write([]byte(val)); err != nil {
			log.Printf("[WARN] can't hash ip, %s", err)
		}
		return fmt.Sprintf("%x", h.Sum(nil))
	}

	u.IP = hashVal(u.IP)
}
