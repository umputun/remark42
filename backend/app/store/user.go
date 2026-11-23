package store

import (
	"crypto/hmac"
	"crypto/sha1" // nolint
	"encoding/hex"
	"errors"
	"hash"
	"regexp"
	"strings"
	"unicode/utf8"
)

// User holds user-related info
type User struct {
	Name              string `json:"name"`
	ID                string `json:"id"`
	Picture           string `json:"picture"`
	IP                string `json:"ip,omitempty"`
	Admin             bool   `json:"admin"`
	Blocked           bool   `json:"block,omitempty"`
	Verified          bool   `json:"verified,omitempty"`
	EmailSubscription bool   `json:"email_subscription,omitempty"`
	SiteID            string `json:"site_id,omitempty"`
	PaidSub           bool   `json:"paid_sub,omitempty"`
}

var reValidSha = regexp.MustCompile("^[a-fA-F0-9]{40}$")

// reUserName matches a user name made of letters, ASCII digits, underscores, dots, hyphens and ordinary spaces.
// The widget's sign-in form checks the same characters (the pattern of the username input in auth.tsx)
var reUserName = regexp.MustCompile(`^[\p{L}\d_. -]+$`)

// CheckUserName checks a name typed at anonymous or email sign-in: 3 to 64 characters, each a letter,
// an ASCII digit, an underscore, a dot, a hyphen or an ordinary space, and no space at either end.
// The error says which part of the rule the name breaks, without repeating the name
func CheckUserName(name string) error {
	if strings.HasPrefix(name, " ") || strings.HasSuffix(name, " ") {
		return errors.New("name starts or ends with a space")
	}
	if n := utf8.RuneCountInString(name); n < 3 || n > 64 {
		return errors.New("name should be 3 to 64 characters long")
	}
	if !reUserName.MatchString(name) {
		return errors.New("name should have letters, digits, underscores, dots, hyphens and spaces only")
	}
	return nil
}

// HashIP replace IP field with hashed hmac
func (u *User) HashIP(secret string) {
	u.IP = HashValue(u.IP, secret)
}

// HashValue makes hmac with secret
func HashValue(val, secret string) string {
	key := []byte(secret)
	return hashHex(hmac.New(sha1.New, key), val)
}

// EncodeID hashes id to sha1. The function intentionally left outside of User struct because in some cases
// we need hashing for parts of id, in some others hashing for non-User values.
func EncodeID(id string) string {
	return hashHex(sha1.New(), id) // nolint
}

// hashHex returns the hex digest of val made with h, or val itself when it is already a sha1 hex digest
func hashHex(h hash.Hash, val string) string {
	if reValidSha.MatchString(val) {
		return val // already hashed
	}
	h.Write([]byte(val)) // hash.Hash.Write never returns an error
	return hex.EncodeToString(h.Sum(nil))
}
