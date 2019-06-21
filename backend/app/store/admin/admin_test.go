package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticStore_Get(t *testing.T) {
	var ks Store = NewStaticStore("key123", []string{"123", "xyz"}, "aa@example.com")

	k, err := ks.Key()
	assert.NoError(t, err, "valid store")
	assert.Equal(t, "key123", k, "valid site")

	a, err := ks.Admins("any")
	assert.NoError(t, err)
	assert.Equal(t, []string{"123", "xyz"}, a)

	email, err := ks.Email("blah")
	assert.NoError(t, err)
	assert.Equal(t, "aa@example.com", email)
}
