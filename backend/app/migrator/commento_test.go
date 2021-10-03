package migrator

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/admin"
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/service"
	bolt "go.etcd.io/bbolt"
)

func TestCommento_Import(t *testing.T) {
	defer os.Remove("/tmp/remark-test.db")
	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: "/tmp/remark-test.db", SiteID: "test"})
	require.NoError(t, err, "create store")
	dataStore := service.DataStore{Engine: b, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
	defer dataStore.Close()

	d := Commento{DataStore: &dataStore}
	fh, err := os.Open("testdata/commento.json")
	require.NoError(t, err)
	size, err := d.Import(fh, "test")
	assert.NoError(t, err)
	assert.Equal(t, 2, size)

	last, err := dataStore.Last("test", 10, time.Time{}, adminUser)
	assert.NoError(t, err)
	require.Equal(t, 2, len(last), "2 comments imported")

	t.Log(last[0])

	c := last[0] // last reverses, get first one
	assert.Equal(t, "Great reply!", c.Text)
	assert.Equal(t, "ea5f7bcd6ac9bb7b657f7d0569831104e1bcf9c253d03c1e16bf9654c49a5ce9", c.ID)
	assert.Equal(t, "7d77e39fcd813241d6281478cc8f21ab5f807d043c750bc1a936bc23b34fb854", c.ParentID)
	assert.Equal(t, store.Locator{SiteID: "test", URL: "https://example.com/blog/post/1"}, c.Locator)
	assert.Equal(t, "Saturnin Uf", c.User.Name)
	assert.Equal(t, "commento_35369aeb6ac5255de30410a0f86dc71eb9c6d0ca", c.User.ID)
	assert.True(t, c.Imported)

	posts, err := dataStore.List("test", 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(posts), "1 post")

	count, err := dataStore.Count(store.Locator{SiteID: "test", URL: "https://example.com/blog/post/1"})
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}
