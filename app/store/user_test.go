package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_EncodeID(t *testing.T) {
	tbl := []struct {
		id   string
		hash string
	}{
		{"myid", "6e34471f84557e1713012d64a7477c71bfdac631"},
		{"", "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
		{"blah blah", "135a1e01bae742c4a576b20fd41a683f6483ca43"},
	}

	for i, tt := range tbl {
		assert.Equal(t, tt.hash, EncodeID(tt.id), "case #%d", i)
	}
}

func TestUser_HashIP(t *testing.T) {
	tbl := []struct {
		ip           string
		hash1, hash2 string
	}{
		{"127.0.0.1", "ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741", "dbc7c999343f003f189f70aaf52cc04443f90790"},
		{"8.8.8.8", "8cee77c27e32a2b5aec95c29888ac9946618d9a2", "70a46afce9633f010b06e129b8ad08243a1c4da9"},
		{"8cee77c27e32a2b5aec95c29888ac9946618d9a2", "8cee77c27e32a2b5aec95c29888ac9946618d9a2", "8cee77c27e32a2b5aec95c29888ac9946618d9a2"},
		{"", "", ""},
	}

	for i, tt := range tbl {
		u := User{IP: tt.ip}
		u.HashIP("")
		assert.Equal(t, tt.hash1, u.IP, "case #%d", i)

		u = User{IP: tt.ip}
		u.HashIP("123456")
		assert.Equal(t, tt.hash2, u.IP, "case #%d", i)
	}
}
