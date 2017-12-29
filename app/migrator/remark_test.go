package migrator

import (
	"bytes"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark/app/store"
)

var testDb = "/tmp/test-remark.db"

func TestRemark_Export(t *testing.T) {
	b := prep(t)
	r := Remark{DataStore: b}

	buf := &bytes.Buffer{}
	err := r.Export(buf, "radio-t")
	assert.Nil(t, err)

	c1, err := buf.ReadString('\n')
	assert.Nil(t, err)
	log.Print(c1)
	exp := `{"id":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"","text":"some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","profile":"","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:22-06:00"}` + "\n"
	assert.Equal(t, exp, c1)
}

func TestRemark_Import(t *testing.T) {
	r1 := `{"id":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"","text":"some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","profile":"","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:22-06:00"}` + "\n"

	r2 := `{"id":"afbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","text":"some text2, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","profile":"","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:23-06:00"}` + "\n"

	buf := &bytes.Buffer{}
	buf.WriteString(r1)
	buf.WriteString(r2)

	os.Remove(testDb)
	b, err := store.NewBoltDB(store.BoltSite{SiteID: "radio-t", FileName: testDb})
	assert.Nil(t, err)
	r := Remark{DataStore: b}
	err = r.Import(buf, "radio-t")
	assert.Nil(t, err)

	comments, err := b.Find(store.Request{Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments))
	assert.Equal(t, "efbc17f177ee1a1c0ee6e1e025749966ec071adc", comments[0].ID)
	assert.Equal(t, "afbc17f177ee1a1c0ee6e1e025749966ec071adc", comments[1].ID)
	assert.Equal(t, "efbc17f177ee1a1c0ee6e1e025749966ec071adc", comments[1].ParentID)
}

// makes new boltdb, put two records
func prep(t *testing.T) *store.BoltDB {
	os.Remove(testDb)

	b, err := store.NewBoltDB(store.BoltSite{SiteID: "radio-t", FileName: testDb})
	assert.Nil(t, err)

	comment := store.Comment{
		ID:        "efbc17f177ee1a1c0ee6e1e025749966ec071adc",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = b.Create(comment)
	assert.Nil(t, err)

	comment = store.Comment{
		Text: "some text2", Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.Local),
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:    store.User{ID: "user1", Name: "user name"},
	}
	_, err = b.Create(comment)
	assert.Nil(t, err)

	return b
}
