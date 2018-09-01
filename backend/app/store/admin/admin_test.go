package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticStore_Get(t *testing.T) {
	var ks Store = NewStaticStore([]string{"123", "xyz"}, "aa@example.com")

	a := ks.Admins("any")
	assert.Equal(t, []string{"123", "xyz"}, a)

	email := ks.Email("blah")
	assert.Equal(t, "aa@example.com", email)
}
