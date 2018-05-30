package store

import (
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"hash/crc64"
	"log"
	"regexp"
)

// User holds user-related info
type User struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Picture  string `json:"picture"`
	Admin    bool   `json:"admin"`
	Blocked  bool   `json:"block,omitempty"`
	IP       string `json:"ip,omitempty"`
	Verified bool   `json:"verified,omitempty"`
}

var reValidSha = regexp.MustCompile("^[a-fA-F0-9]{40}$")

// HashIP replace IP field with hashed hmac
func (u *User) HashIP(secret string) {

	hashVal := func(val string) string {
		if val == "" || reValidSha.Match([]byte(val)) {
			return val // already hashed or empty
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

// EncodeID hashes id to sha1. The function intentionally left outside of User struct because in some cases
// we need hashing for parts of id, in some others hashing for non-User values.
func EncodeID(id string) string {
	h := sha1.New()
	if _, err := h.Write([]byte(id)); err != nil {
		// fail back to crc64
		log.Printf("[WARN] can't hash id %s, %s", id, err)
		return fmt.Sprintf("%x", crc64.Checksum([]byte(id), crc64.MakeTable(crc64.ECMA)))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
