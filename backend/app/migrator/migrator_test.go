package migrator

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/admin"
	"github.com/umputun/remark/backend/app/store/engine"
	"github.com/umputun/remark/backend/app/store/service"
)

func TestMigrator_ImportDisqus(t *testing.T) {
	defer func() {
		os.Remove("/tmp/remark-test.db")
		os.Remove("/tmp/disqus-test.xml")
	}()

	err := ioutil.WriteFile("/tmp/disqus-test.xml", []byte(xmlTestDisqus), 0600)
	require.NoError(t, err)

	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: "/tmp/remark-test.db", SiteID: "test"})
	require.NoError(t, err, "create store")
	dataStore := &service.DataStore{Engine: b, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
	defer dataStore.Close()
	size, err := ImportComments(ImportParams{
		DataStore: dataStore,
		InputFile: "/tmp/disqus-test.xml",
		SiteID:    "test",
		Provider:  "disqus",
	})
	assert.NoError(t, err)
	assert.Equal(t, 4, size)

	last, err := dataStore.Last("test", 10, time.Time{}, store.User{})
	assert.NoError(t, err)
	assert.Equal(t, 4, len(last), "4 comments imported")
}

func TestMigrator_ImportWordPress(t *testing.T) {
	defer func() {
		os.Remove("/tmp/remark-test.db")
		os.Remove("/tmp/wordpress-test.xml")
	}()

	err := ioutil.WriteFile("/tmp/wordpress-test.xml", []byte(xmlTestWP), 0600)
	require.NoError(t, err)

	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: "/tmp/remark-test.db", SiteID: "test"})
	require.NoError(t, err, "create store")
	dataStore := &service.DataStore{Engine: b, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
	defer dataStore.Close()
	size, err := ImportComments(ImportParams{
		DataStore: dataStore,
		InputFile: "/tmp/wordpress-test.xml",
		SiteID:    "test",
		Provider:  "wordpress",
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, size)

	last, err := dataStore.Last("test", 10, time.Time{}, store.User{})
	assert.NoError(t, err)
	assert.Equal(t, 3, len(last), "3 comments imported")
}

func TestMigrator_ImportNative(t *testing.T) {
	defer func() {
		os.Remove("/tmp/remark-test.db")
		os.Remove("/tmp/disqus-test.r42")
	}()

	data := `{"version":1} {"id":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"","text":"some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","profile":"","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:22-06:00"}` + "\n" +
		`{"id":"afbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","text":"some text2, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","profile":"","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:23-06:00"}` + "\n"

	err := ioutil.WriteFile("/tmp/disqus-test.r42", []byte(data), 0600)
	require.NoError(t, err)

	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: "/tmp/remark-test.db", SiteID: "radio-t"})
	require.NoError(t, err, "create store")
	dataStore := &service.DataStore{Engine: b, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
	defer dataStore.Close()

	size, err := ImportComments(ImportParams{
		DataStore: dataStore,
		InputFile: "/tmp/disqus-test.r42",
		SiteID:    "radio-t",
		Provider:  "native",
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, size)

	last, err := dataStore.Last("radio-t", 10, time.Time{}, store.User{})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(last), "2 comments imported")
}

func TestMigrator_ImportFailed(t *testing.T) {
	defer os.Remove("/tmp/remark-test.db")
	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: "/tmp/remark-test.db", SiteID: "test"})
	require.NoError(t, err, "create store")
	dataStore := &service.DataStore{Engine: b}
	defer dataStore.Close()
	_, err = ImportComments(ImportParams{
		DataStore: dataStore,
		InputFile: "/tmp/disqus-test.xml",
		SiteID:    "test",
		Provider:  "bad",
	})

	assert.EqualError(t, err, "unsupported import provider bad")

	_, err = ImportComments(ImportParams{
		DataStore: dataStore,
		InputFile: "/tmp/disqus-test-bad.xml",
		SiteID:    "test",
		Provider:  "native",
	})
	assert.EqualError(t, err, "can't open import file /tmp/disqus-test-bad.xml: open /tmp/disqus-test-bad.xml: no such file or directory")
}
