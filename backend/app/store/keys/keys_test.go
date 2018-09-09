package keys

import (
	"testing"

	"github.com/globalsign/mgo"
	"github.com/go-pkgz/mongo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestMongoStore_Get(t *testing.T) {
	conn, err := mongo.MakeTestConnection(t)
	require.NoError(t, err)
	var ms Store = NewMongoStore(conn)

	recs := []struct {
		SiteID    string `bson:"site"`
		SecretKey string `bson:"secret"`
	}{
		{"site1", "secret1"},
		{"site2", "secret2"},
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

	r, err := ms.Get("site1")
	assert.NoError(t, err)
	assert.Equal(t, "secret1", r)

	r, err = ms.Get("site2")
	assert.NoError(t, err)
	assert.Equal(t, "secret2", r)

	r, err = ms.Get("no-site-in-db")
	assert.Error(t, err, "can't get secret for site no-site-in-db")
}
