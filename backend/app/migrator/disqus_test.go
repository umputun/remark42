package migrator

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/admin"
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/service"
)

func TestDisqus_Import(t *testing.T) {
	defer os.Remove("/tmp/remark-test.db")
	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: "/tmp/remark-test.db", SiteID: "test"})
	require.NoError(t, err, "create store")
	dataStore := service.DataStore{Engine: b, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
	defer dataStore.Close()
	d := Disqus{DataStore: &dataStore}
	fh, err := os.Open("testdata/disqus1.xml")
	require.NoError(t, err)
	size, err := d.Import(fh, "test")
	assert.NoError(t, err)
	assert.Equal(t, 4, size)

	last, err := dataStore.Last("test", 10, time.Time{}, adminUser)
	assert.NoError(t, err)
	require.Equal(t, 4, len(last), "4 comments imported")

	c := last[len(last)-1] // last reverses, get first one
	assert.True(t, strings.HasPrefix(c.Text, "<p>The quick brown fox"))
	assert.Equal(t, "299619020", c.ID)
	assert.Equal(t, "", c.ParentID)
	assert.Equal(t, store.Locator{SiteID: "test", URL: "https://radio-t.com/p/2011/03/05/podcast-229/"}, c.Locator)
	assert.Equal(t, "Alexander Blah", c.User.Name)
	assert.Equal(t, "disqus_328c8b68974aef73785f6b38c3d3fedfdf941434", c.User.ID)
	assert.Equal(t, "2ba6b71dbf9750ae3356cce14cac6c1b1962747c", c.User.IP)
	assert.True(t, c.Imported)

	c = last[1] // get comment with empty username
	assert.Equal(t, "No Username", c.User.Name)
	assert.Equal(t, "disqus_62e24ea213756cda0339e1074819f15e25214361", c.User.ID)

	posts, err := dataStore.List("test", 0, 0)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(posts), "2 posts")

	count, err := dataStore.Count(store.Locator{SiteID: "test", URL: "https://radio-t.com/p/2011/03/05/podcast-229/"})
	assert.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestDisqus_Convert(t *testing.T) {
	d := Disqus{}
	fh, err := os.Open("testdata/disqus1.xml")
	require.NoError(t, err)
	ch := d.convert(fh, "test")

	res := []store.Comment{}
	for comment := range ch {
		res = append(res, comment)
	}
	require.Equal(t, 4, len(res), "4 comments total, 1 spam excluded, 1 bad excluded")

	exp0 := store.Comment{
		ID: "299619020",
		Locator: store.Locator{
			SiteID: "test",
			URL:    "https://radio-t.com/p/2011/03/05/podcast-229/",
		},
		Text: `<p>The quick brown fox jumps over the lazy dog.</p><p><a href="https://https://radio-t.com" rel="nofollow noopener" title="radio-t">some link</a></p>`,
		User: store.User{
			Name: "Alexander Blah",
			ID:   "disqus_328c8b68974aef73785f6b38c3d3fedfdf941434",
			IP:   "178.178.178.178",
		},
		Imported: true,
	}
	exp0.Timestamp, _ = time.Parse("2006-01-02T15:04:05Z", "2011-08-31T15:16:29Z")
	assert.Equal(t, exp0, res[0])
}
