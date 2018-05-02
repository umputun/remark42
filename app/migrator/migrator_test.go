package migrator

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/coreos/bbolt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/app/store"
)

func TestMigrator_ImportDisqus(t *testing.T) {
	defer func() {
		os.Remove("/tmp/remark-test.db")
		os.Remove("/tmp/disqus-test.xml")
	}()

	err := ioutil.WriteFile("/tmp/disqus-test.xml", []byte(xmlTest), 0600)
	require.Nil(t, err)

	dataStore, err := store.NewBoltDB(bolt.Options{}, store.BoltSite{FileName: "/tmp/remark-test.db", SiteID: "test"})
	require.Nil(t, err, "create store")

	size, err := ImportComments(ImportParams{
		CommentCreator: dataStore,
		InputFile:      "/tmp/disqus-test.xml",
		SiteID:         "test",
		Provider:       "disqus",
	})
	assert.Nil(t, err)
	assert.Equal(t, 3, size)

	last, err := dataStore.Last("test", 10)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(last), "3 comments imported")
}

func TestMigrator_ImportRemark(t *testing.T) {
	defer func() {
		os.Remove("/tmp/remark-test.db")
		os.Remove("/tmp/disqus-test.r42")
	}()

	data := `{"id":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"","text":"some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","profile":"","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:22-06:00"}` + "\n" +
		`{"id":"afbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","text":"some text2, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","profile":"","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:23-06:00"}` + "\n"

	err := ioutil.WriteFile("/tmp/disqus-test.r42", []byte(data), 0600)
	require.Nil(t, err)

	dataStore, err := store.NewBoltDB(bolt.Options{}, store.BoltSite{FileName: "/tmp/remark-test.db", SiteID: "radio-t"})
	require.Nil(t, err, "create store")

	size, err := ImportComments(ImportParams{
		CommentCreator: dataStore,
		InputFile:      "/tmp/disqus-test.r42",
		SiteID:         "radio-t",
		Provider:       "native",
	})
	assert.Nil(t, err)
	assert.Equal(t, 2, size)

	last, err := dataStore.Last("radio-t", 10)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(last), "2 comments imported")
}
