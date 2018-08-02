package store

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/crc64"
	"io"
	"log"
	"regexp"
)

// User holds user-related info
type User struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Picture  string `json:"picture"`
	IP       string `json:"ip,omitempty"`
	Admin    bool   `json:"admin"`
	Blocked  bool   `json:"block,omitempty"`
	Verified bool   `json:"verified,omitempty"`
}

var reValidSha = regexp.MustCompile("^[a-fA-F0-9]{40}$")

// HashIP replace IP field with hashed hmac
func (u *User) HashIP(secret string) {
	u.IP = HashValue(u.IP, secret)
}

// HashValue makes hmac with secret
func HashValue(val string, secret string) string {
	if val == "" || reValidSha.MatchString(val) {
		return val // already hashed or empty
	}
	key := []byte(secret)
	h := hmac.New(sha1.New, key)
	return hashWithFailback(h, val)
}

// EncodeID hashes id to sha1. The function intentionally left outside of User struct because in some cases
// we need hashing for parts of id, in some others hashing for non-User values.
func EncodeID(id string) string {
	if reValidSha.MatchString(id) {
		return id // already hashed or empty
	}
	h := sha1.New()
	return hashWithFailback(h, id)
}

// hashWithFailback tries to has val with hash.Hash and failback to crc if needed
func hashWithFailback(h hash.Hash, val string) string {
	if _, err := io.WriteString(h, val); err != nil {
		// fail back to crc64
		log.Printf("[WARN] can't hash id %s, %s", val, err)
		return fmt.Sprintf("%x", crc64.Checksum([]byte(val), crc64.MakeTable(crc64.ECMA)))
	}
	return hex.EncodeToString(h.Sum(nil))
}
