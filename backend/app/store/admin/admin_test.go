package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticStore_Get(t *testing.T) {
	var ks Store = NewStaticStore("key123", []string{"s1", "s2", "s3"},
		[]string{"123", "xyz"}, []string{"name1", "name2"}, "aa@example.com")

	k, err := ks.Key()
	assert.NoError(t, err, "valid store")
	assert.Equal(t, "key123", k, "valid site")

	a, err := ks.Admins("s1")
	assert.NoError(t, err)
	assert.Equal(t, []string{"123", "xyz"}, a)

	a, err = ks.Names("s1")
	assert.NoError(t, err)
	assert.Equal(t, []string{"name1", "name2"}, a)

	email, err := ks.Email("s2")
	assert.NoError(t, err)
	assert.Equal(t, "aa@example.com", email)

	enabled, err := ks.Enabled("s3")
	assert.NoError(t, err)
	assert.Equal(t, true, enabled)

	enabled, err = ks.Enabled("serr")
	assert.NoError(t, err)
	assert.Equal(t, false, enabled)
}
