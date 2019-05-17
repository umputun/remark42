package migrator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	bolt "github.com/coreos/bbolt"
	log "github.com/go-pkgz/lgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/admin"
	"github.com/umputun/remark/backend/app/store/engine"
	"github.com/umputun/remark/backend/app/store/service"
)

var testDb = "/tmp/test-remark.db"

func TestNative_Export(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t) // write 2 comments
	assert.NoError(t, b.SetReadOnly(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, true))
	assert.NoError(t, b.SetVerified("radio-t", "user1", true))
	assert.NoError(t, b.SetBlock("radio-t", "user2", true, time.Hour))
	r := Native{DataStore: b}

	buf := &bytes.Buffer{}
	size, err := r.Export(buf, "radio-t")
	assert.Nil(t, err)
	assert.Equal(t, 2, size)

	c1 := buf.String()
	log.Print(c1)

	dec := json.NewDecoder(strings.NewReader(c1))

	meta := struct {
		Version int                    `json:"version"`
		Users   []service.UserMetaData `json:"users"`
		Posts   []service.PostMetaData `json:"posts"`
	}{}

	require.NoError(t, dec.Decode(&meta), "decode meta")

	assert.Equal(t, 2, len(meta.Users))
	assert.Equal(t, "user1", meta.Users[0].ID)
	assert.Equal(t, false, meta.Users[0].Blocked.Status)
	assert.Equal(t, true, meta.Users[0].Verified)
	assert.Equal(t, "user2", meta.Users[1].ID)
	assert.Equal(t, true, meta.Users[1].Blocked.Status)
	assert.Equal(t, false, meta.Users[1].Verified)

	assert.Equal(t, 1, len(meta.Posts))
	assert.Equal(t, "https://radio-t.com", meta.Posts[0].URL)
	assert.Equal(t, true, meta.Posts[0].ReadOnly)

	comments := [3]store.Comment{}

	assert.NoError(t, dec.Decode(&comments[0]), "decode comment 0")
	assert.NoError(t, dec.Decode(&comments[1]), "decode comment 0")
	assert.Error(t, dec.Decode(&comments[2]), "EOF")

	assert.Equal(t, "some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>", comments[0].Text)
}

func TestNative_Import(t *testing.T) {
	defer os.Remove(testDb)

	inp := `{"version":1,"users":[{"id":"user1","blocked":{"status":false,"until":"0001-01-01T00:00:00Z"},"verified":true},{"id":"user2","blocked":{"status":true,"until":"2018-12-23T02:55:22.472041-06:00"},"verified":false}],"posts":[{"url":"https://radio-t.com","read_only":true}]}
	{"id":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"","text":"some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","ip":"293ec5b0cf154855258824ec7fac5dc63d176915","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:22-06:00"}
	{"id":"f863bd79-fec6-4a75-b308-61fe5dd02aa1","pid":"1234","text":"some text2","user":{"name":"user name","id":"user2","picture":"","ip":"293ec5b0cf154855258824ec7fac5dc63d176915","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com/2"},"score":0,"votes":{},"time":"2017-12-20T15:18:23-06:00"}`

	b := prep(t) // write some recs
	b.AdminStore = admin.NewStaticStore("12345", []string{}, "")
	r := Native{DataStore: b}
	size, err := r.Import(strings.NewReader(inp), "radio-t")
	assert.Nil(t, err)
	assert.Equal(t, 2, size)

	comments, err := b.Last("radio-t", 10, time.Time{}, store.User{})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments))
	assert.Equal(t, "f863bd79-fec6-4a75-b308-61fe5dd02aa1", comments[0].ID)
	assert.Equal(t, "1234", comments[0].ParentID)
	assert.Equal(t, false, b.IsReadOnly(comments[0].Locator))

	assert.Equal(t, "efbc17f177ee1a1c0ee6e1e025749966ec071adc", comments[1].ID)
	assert.Equal(t, "https://radio-t.com", comments[1].Locator.URL)
	assert.Equal(t, true, b.IsReadOnly(comments[1].Locator))

	assert.Equal(t, false, b.IsBlocked("radio-t", "user1"))
	assert.Equal(t, true, b.IsVerified("radio-t", "user1"))

	assert.Equal(t, true, b.IsBlocked("radio-t", "user2"))
	assert.Equal(t, false, b.IsVerified("radio-t", "user2"))
}

func TestNative_ImportWrongVersion(t *testing.T) {
	inp := `{"version":2,"users":[{"id":"user1","blocked":{"status":false,"until":"0001-01-01T00:00:00Z"},"verified":true},{"id":"user2","blocked":{"status":true,"until":"2018-12-23T02:55:22.472041-06:00"},"verified":false}],"posts":[{"url":"https://radio-t.com","read_only":true}]}
	{"id":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"","text":"some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","ip":"293ec5b0cf154855258824ec7fac5dc63d176915","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:22-06:00"}
	{"id":"f863bd79-fec6-4a75-b308-61fe5dd02aa1","pid":"1234","text":"some text2","user":{"name":"user name","id":"user2","picture":"","ip":"293ec5b0cf154855258824ec7fac5dc63d176915","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com/2"},"score":0,"votes":{},"time":"2017-12-20T15:18:23-06:00"}`

	b := prep(t) // write some recs
	b.AdminStore = admin.NewStaticStore("12345", []string{}, "")
	r := Native{DataStore: b}
	size, err := r.Import(strings.NewReader(inp), "radio-t")
	assert.EqualError(t, err, "unexpected import file version 2")
	assert.Equal(t, 0, size)

}
func TestNative_ImportManyWithError(t *testing.T) {
	defer os.Remove(testDb)

	goodRec := `{"id":"%d","pid":"","text":"some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","profile":"","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:22-06:00"}` + "\n"

	buf := &bytes.Buffer{}
	buf.WriteString(`{"version":1, "users":[], "posts":[]}` + "\n")
	for i := 0; i < 1200; i++ {
		buf.WriteString(fmt.Sprintf(goodRec, i))
	}
	buf.WriteString("{}\n")
	buf.WriteString("{}\n")

	b := prep(t) // write some recs
	b.AdminStore = admin.NewStaticStore("12345", []string{}, "")
	r := Native{DataStore: b}
	n, err := r.Import(buf, "radio-t")
	assert.EqualError(t, err, "failed to save 2 comments")
	assert.Equal(t, 1200, n)
	comments, err := b.Find(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}, "time", store.User{})
	assert.Nil(t, err)
	assert.Equal(t, 1200, len(comments))
}

// makes new boltdb, put two records
func prep(t *testing.T) *service.DataStore {
	os.Remove(testDb)

	boltStore, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{SiteID: "radio-t", FileName: testDb})
	assert.Nil(t, err)

	b := &service.DataStore{Interface: boltStore, AdminStore: admin.NewStaticStore("12345", []string{}, "")}

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
		Locator: store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:    store.User{ID: "user2", Name: "user name"},
	}
	_, err = b.Create(comment)
	assert.Nil(t, err)

	return b
}
