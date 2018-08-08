package keys

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticStore_Get(t *testing.T) {
	var ks Store = NewStaticStore("key123")

	k, err := ks.Get("any")
	assert.NoError(t, err, "valid store")
	assert.Equal(t, "key123", k, "valid site")

	ks = NewStaticStore("")

	_, err = ks.Get("any")
	assert.NotNil(t, err, "invalid (empty key) store")
}
