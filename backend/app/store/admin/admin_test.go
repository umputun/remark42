package admin

import (
	"testing"

	"github.com/globalsign/mgo"
	"github.com/go-pkgz/mongo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticStore_Get(t *testing.T) {
	var ks Store = NewStaticStore("key123", []string{"123", "xyz"}, "aa@example.com")

	k, err := ks.Key("any")
	assert.NoError(t, err, "valid store")
	assert.Equal(t, "key123", k, "valid site")

	a := ks.Admins("any")
	assert.Equal(t, []string{"123", "xyz"}, a)

	email := ks.Email("blah")
	assert.Equal(t, "aa@example.com", email)

	ks = NewStaticStore("", []string{"123", "xyz"}, "aa@example.com")
	_, err = ks.Key("any")
	assert.NotNil(t, err, "invalid (empty key) store")
}

func TestMongoStore_Get(t *testing.T) {
	conn, err := mongo.MakeTestConnection(t)
	require.NoError(t, err)
	var ms Store = NewMongoStore(conn)

	recs := []mongoRec{
		{"site1", "secret1", []string{"i11", "i12"}, "e1"},
		{"site2", "secret2", []string{"i21", "i22"}, "e2"},
	}
	err = conn.WithCollection(func(coll *mgo.Collection) error {
		if e1 := coll.Insert(recs[0]); e1 != nil {
			return e1
		}
		if e2 := coll.Insert(recs[1]); e2 != nil {
			return e2
		}
		return nil
	})
	require.NoError(t, err)

	admins := ms.Admins("site1")
	assert.Equal(t, []string{"i11", "i12"}, admins)
	email := ms.Email("site1")
	assert.Equal(t, "e1", email)
	key, err := ms.Key("site1")
	assert.NoError(t, err)
	assert.Equal(t, "secret1", key)

	admins = ms.Admins("site2")
	assert.Equal(t, []string{"i21", "i22"}, admins)
	email = ms.Email("site2")
	assert.Equal(t, "e2", email)
	key, err = ms.Key("site2")
	assert.NoError(t, err)
	assert.Equal(t, "secret2", key)

	admins = ms.Admins("no-site-in-db")
	assert.Equal(t, []string{}, admins)
	email = ms.Email("no-site-in-db")
	assert.Equal(t, "", email)
	_, err = ms.Key("no-site-in-db")
	assert.Error(t, err, "can't get secret for site no-site-in-db")
}
