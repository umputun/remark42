package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticStore_StoreWithoutSites(t *testing.T) {
	var ks Store = NewStaticKeyStore("key123")
	enabled, err := ks.Enabled("any")
	assert.NoError(t, err)
	assert.True(t, enabled, "on empty store all sites are enabled")
	assert.NoError(t, ks.OnEvent("test", EvCreate), "static store does nothing OnEvent")

	// empty key
	ks = NewStaticKeyStore("")
	key, err := ks.Key("any")
	assert.Error(t, err, "empty key")
	assert.Empty(t, key)
}

func TestStaticStore_Get(t *testing.T) {
	var ks Store = NewStaticStore("key123", []string{"s1", "s2", "s3"},
		[]string{"123", "xyz"}, "aa@example.com")

	k, err := ks.Key("any")
	assert.NoError(t, err, "valid store")
	assert.Equal(t, "key123", k, "valid site")

	a, err := ks.Admins("s1")
	assert.NoError(t, err)
	assert.Equal(t, []string{"123", "xyz"}, a)

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
