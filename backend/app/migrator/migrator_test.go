package migrator

import (
	"os"
	"path/filepath"
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

func TestMigrator_ImportDisqus(t *testing.T) {
	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: filepath.Join(t.TempDir(), "remark-test.db"), SiteID: "test"})
	require.NoError(t, err, "create store")
	dataStore := &service.DataStore{Engine: b, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
	defer dataStore.Close()
	size, err := ImportComments(ImportParams{
		DataStore: dataStore,
		InputFile: "testdata/disqus.xml",
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
	tmp := t.TempDir()
	inputFile := filepath.Join(tmp, "wordpress-test.xml")
	err := os.WriteFile(inputFile, []byte(xmlTestWP), 0o600)
	require.NoError(t, err)

	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: filepath.Join(tmp, "remark-test.db"), SiteID: "test"})
	require.NoError(t, err, "create store")
	dataStore := &service.DataStore{Engine: b, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
	defer dataStore.Close()
	size, err := ImportComments(ImportParams{
		DataStore: dataStore,
		InputFile: inputFile,
		SiteID:    "test",
		Provider:  "wordpress",
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, size)

	last, err := dataStore.Last("test", 10, time.Time{}, store.User{})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(last), "2 comments imported, the pingback skipped")
}

func TestMigrator_ImportCommento(t *testing.T) {
	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: filepath.Join(t.TempDir(), "remark-test.db"), SiteID: "test"})
	require.NoError(t, err, "create store")
	dataStore := &service.DataStore{Engine: b, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
	defer dataStore.Close()
	size, err := ImportComments(ImportParams{
		DataStore: dataStore,
		InputFile: "testdata/commento.json",
		SiteID:    "test",
		Provider:  "commento",
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, size)

	last, err := dataStore.Last("test", 10, time.Time{}, store.User{})
	assert.NoError(t, err)
	assert.Equal(t, 3, len(last), "3 comments imported")
}

func TestMigrator_ImportNative(t *testing.T) {
	tmp := t.TempDir()
	inputFile := filepath.Join(tmp, "disqus-test.r42")

	data := `{"version":1} {"id":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"","text":"some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","profile":"","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:22-06:00"}` + "\n" +
		`{"id":"afbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","text":"some text2, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","profile":"","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:23-06:00"}` + "\n"

	err := os.WriteFile(inputFile, []byte(data), 0o600)
	require.NoError(t, err)

	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: filepath.Join(tmp, "remark-test.db"), SiteID: "radio-t"})
	require.NoError(t, err, "create store")
	dataStore := &service.DataStore{Engine: b, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
	defer dataStore.Close()

	size, err := ImportComments(ImportParams{
		DataStore: dataStore,
		InputFile: inputFile,
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
	missing := filepath.Join(t.TempDir(), "missing.xml")
	b, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{FileName: filepath.Join(t.TempDir(), "remark-test.db"), SiteID: "test"})
	require.NoError(t, err, "create store")
	dataStore := &service.DataStore{Engine: b}
	defer dataStore.Close()
	_, err = ImportComments(ImportParams{
		DataStore: dataStore,
		InputFile: "testdata/disqus.xml",
		SiteID:    "test",
		Provider:  "bad",
	})

	assert.EqualError(t, err, "unsupported import provider bad")

	_, err = ImportComments(ImportParams{
		DataStore: dataStore,
		InputFile: missing,
		SiteID:    "test",
		Provider:  "native",
	})
	assert.EqualError(t, err, "can't open import file "+missing+": open "+missing+": no such file or directory")
}
