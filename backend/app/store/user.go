package store

import (
	"crypto/hmac"
	"crypto/sha1" // nolint
	"encoding/hex"
	"hash"
	"regexp"
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
