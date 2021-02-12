package store

import (
	"crypto/hmac"
	"crypto/sha1" // nolint
	"encoding/hex"
	"fmt"
	"hash"
	"hash/crc64"
	"io"
	"regexp"

	log "github.com/go-pkgz/lgr"
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
}

var reValidSha = regexp.MustCompile("^[a-fA-F0-9]{40}$")
var reValidCrc64 = regexp.MustCompile("^[a-fA-F0-9]{16}$")

// HashIP replace IP field with hashed hmac
func (u *User) HashIP(secret string) {
	u.IP = HashValue(u.IP, secret)
}

// HashValue makes hmac with secret
func HashValue(val, secret string) string {
	key := []byte(secret)
	return hashWithFallback(hmac.New(sha1.New, key), val)
}

// EncodeID hashes id to sha1. The function intentionally left outside of User struct because in some cases
// we need hashing for parts of id, in some others hashing for non-User values.
func EncodeID(id string) string {
	return hashWithFallback(sha1.New(), id) // nolint
}

// hashWithFallback tries to has val with hash.Hash and fallback to crc if needed
func hashWithFallback(h hash.Hash, val string) string {

	if reValidSha.MatchString(val) {
		return val // already hashed
	}

	if _, err := io.WriteString(h, val); err != nil {
		// fail back to crc64
		log.Printf("[WARN] can't hash id %s, %s", val, err)
		if reValidCrc64.MatchString(val) {
			return val // already crced
		}
		return fmt.Sprintf("%x", crc64.Checksum([]byte(val), crc64.MakeTable(crc64.ECMA)))
	}
	return hex.EncodeToString(h.Sum(nil))
}
