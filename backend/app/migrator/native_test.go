package migrator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
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

func TestNative_Export(t *testing.T) {
	b, teardown := prep(t) // write 2 comments
	defer teardown()
	assert.NoError(t, b.SetReadOnly(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, true))
	assert.NoError(t, b.SetVerified("radio-t", "user1", true))
	assert.NoError(t, b.SetBlock("radio-t", "user2", true, time.Hour))
	r := Native{DataStore: b}

	buf := &bytes.Buffer{}
	size, err := r.Export(buf, "radio-t")
	assert.NoError(t, err)
	assert.Equal(t, 2, size)

	c1 := buf.String()
	t.Log(c1)

	dec := json.NewDecoder(strings.NewReader(c1))

	m := struct {
		Version int                    `json:"version"`
		Users   []service.UserMetaData `json:"users"`
		Posts   []service.PostMetaData `json:"posts"`
	}{}

	require.NoError(t, dec.Decode(&m), "decode meta")

	require.Equal(t, 2, len(m.Users))
	assert.Equal(t, "user1", m.Users[0].ID)
	assert.Equal(t, false, m.Users[0].Blocked.Status)
	assert.Equal(t, true, m.Users[0].Verified)
	assert.Equal(t, "user2", m.Users[1].ID)
	assert.Equal(t, true, m.Users[1].Blocked.Status)
	assert.Equal(t, false, m.Users[1].Verified)

	require.Equal(t, 1, len(m.Posts))
	assert.Equal(t, "https://radio-t.com", m.Posts[0].URL)
	assert.Equal(t, true, m.Posts[0].ReadOnly)

	comments := [3]store.Comment{}

	assert.NoError(t, dec.Decode(&comments[0]), "decode comment 0")
	assert.NoError(t, dec.Decode(&comments[1]), "decode comment 0")
	assert.Error(t, dec.Decode(&comments[2]), "EOF")

	assert.Equal(t, "some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>", comments[0].Text)
}

func TestNative_Import(t *testing.T) {
	b, teardown := prep(t) // write 2 comments
	defer teardown()

	inp := `{"version":1,"users":[{"id":"user1","blocked":{"status":false,"until":"0001-01-01T00:00:00Z"},"verified":true},{"id":"user2","blocked":{"status":true,"until":"2018-12-23T02:55:22.472041-06:00"},"verified":false}],"posts":[{"url":"https://radio-t.com","read_only":true}]}
	{"id":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"","text":"some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","ip":"293ec5b0cf154855258824ec7fac5dc63d176915","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:22-06:00"}
	{"id":"f863bd79-fec6-4a75-b308-61fe5dd02aa1","pid":"1234","text":"some text2","user":{"name":"user name","id":"user2","picture":"","ip":"293ec5b0cf154855258824ec7fac5dc63d176915","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com/2"},"score":0,"votes":{},"time":"2017-12-20T15:18:23-06:00","imported":false}`

	b.AdminStore = admin.NewStaticStore("12345", nil, []string{}, "")
	r := Native{DataStore: b}
	size, err := r.Import(strings.NewReader(inp), "radio-t")
	assert.NoError(t, err)
	assert.Equal(t, 2, size)

	comments, err := b.Last("radio-t", 10, time.Time{}, store.User{})
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments))
	assert.Equal(t, "f863bd79-fec6-4a75-b308-61fe5dd02aa1", comments[0].ID)
	assert.Equal(t, "1234", comments[0].ParentID)
	assert.Equal(t, false, b.IsReadOnly(comments[0].Locator))
	assert.True(t, comments[0].Imported)

	assert.Equal(t, "efbc17f177ee1a1c0ee6e1e025749966ec071adc", comments[1].ID)
	assert.Equal(t, "https://radio-t.com", comments[1].Locator.URL)
	assert.Equal(t, true, b.IsReadOnly(comments[1].Locator))
	assert.True(t, comments[1].Imported)

	assert.Equal(t, false, b.IsBlocked("radio-t", "user1"))
	assert.Equal(t, true, b.IsVerified("radio-t", "user1"))

	assert.Equal(t, true, b.IsBlocked("radio-t", "user2"))
	assert.Equal(t, false, b.IsVerified("radio-t", "user2"))
}

func TestNative_ImportWithMapper(t *testing.T) {
	b, teardown := prep(t) // write 2 comments
	defer teardown()

	// want to remap comments to https://rdt.c
	rules := `https://radio-t.com* https://rdt.c*`
	mapper, err := NewURLMapper(strings.NewReader(rules))
	assert.NoError(t, err)

	inp := `{"version":1,"users":[{"id":"user1","blocked":{"status":false,"until":"0001-01-01T00:00:00Z"},"verified":true},{"id":"user2","blocked":{"status":true,"until":"2018-12-23T02:55:22.472041-06:00"},"verified":false}],"posts":[{"url":"https://radio-t.com","read_only":true}]}
	{"id":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"","text":"some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","ip":"293ec5b0cf154855258824ec7fac5dc63d176915","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:22-06:00"}
	{"id":"f863bd79-fec6-4a75-b308-61fe5dd02aa1","pid":"1234","text":"some text2","user":{"name":"user name","id":"user2","picture":"","ip":"293ec5b0cf154855258824ec7fac5dc63d176915","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com/2"},"score":0,"votes":{},"time":"2017-12-20T15:18:23-06:00"}`
	mappedReader := WithMapper(strings.NewReader(inp), mapper)

	b.AdminStore = admin.NewStaticStore("12345", nil, []string{}, "")
	r := Native{DataStore: b}
	size, err := r.Import(mappedReader, "radio-t")
	assert.NoError(t, err)
	assert.Equal(t, 2, size)

	comments, err := b.Last("radio-t", 10, time.Time{}, store.User{})
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments))
	assert.Equal(t, "f863bd79-fec6-4a75-b308-61fe5dd02aa1", comments[0].ID)
	assert.Equal(t, "1234", comments[0].ParentID)
	assert.Equal(t, false, b.IsReadOnly(comments[0].Locator))
	assert.Equal(t, "https://rdt.c/2", comments[0].Locator.URL)

	assert.Equal(t, "efbc17f177ee1a1c0ee6e1e025749966ec071adc", comments[1].ID)
	assert.Equal(t, true, b.IsReadOnly(comments[1].Locator))
	assert.Equal(t, "https://rdt.c", comments[1].Locator.URL)

	assert.Equal(t, false, b.IsBlocked("radio-t", "user1"))
	assert.Equal(t, true, b.IsVerified("radio-t", "user1"))

	assert.Equal(t, true, b.IsBlocked("radio-t", "user2"))
	assert.Equal(t, false, b.IsVerified("radio-t", "user2"))
}

func TestNative_ImportWrongVersion(t *testing.T) {
	b, teardown := prep(t) // write 2 comments
	defer teardown()

	inp := `{"version":2,"users":[{"id":"user1","blocked":{"status":false,"until":"0001-01-01T00:00:00Z"},"verified":true},{"id":"user2","blocked":{"status":true,"until":"2018-12-23T02:55:22.472041-06:00"},"verified":false}],"posts":[{"url":"https://radio-t.com","read_only":true}]}
	{"id":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"","text":"some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","ip":"293ec5b0cf154855258824ec7fac5dc63d176915","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:22-06:00"}
	{"id":"f863bd79-fec6-4a75-b308-61fe5dd02aa1","pid":"1234","text":"some text2","user":{"name":"user name","id":"user2","picture":"","ip":"293ec5b0cf154855258824ec7fac5dc63d176915","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com/2"},"score":0,"votes":{},"time":"2017-12-20T15:18:23-06:00"}`

	b.AdminStore = admin.NewStaticStore("12345", nil, []string{}, "")
	r := Native{DataStore: b}
	size, err := r.Import(strings.NewReader(inp), "radio-t")
	assert.EqualError(t, err, "unexpected import file version 2")
	assert.Equal(t, 0, size)

}
func TestNative_ImportManyWithError(t *testing.T) {
	b, teardown := prep(t) // write 2 comments
	defer teardown()

	goodRec := `{"id":"%d","pid":"","text":"some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","profile":"","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:22-06:00"}` + "\n"

	buf := &bytes.Buffer{}
	buf.WriteString(`{"version":1, "users":[], "posts":[]}` + "\n")
	for i := 0; i < 100; i++ {
		buf.WriteString(fmt.Sprintf(goodRec, i))
	}
	buf.WriteString("{}\n")
	buf.WriteString("{}\n")

	b.AdminStore = admin.NewStaticStore("12345", nil, []string{}, "")
	r := Native{DataStore: b}
	n, err := r.Import(buf, "radio-t")
	assert.EqualError(t, err, "failed to save 2 comments")
	assert.Equal(t, 100, n)
	comments, err := b.Find(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}, "time", store.User{})
	assert.NoError(t, err)
	assert.Equal(t, 100, len(comments))
}

// makes new boltdb, put two records
func prep(t *testing.T) (ds *service.DataStore, teardown func()) {

	testDB := fmt.Sprintf("/tmp/migrator-%d.db", rand.Intn(999999999))

	boltStore, err := engine.NewBoltDB(bolt.Options{}, engine.BoltSite{SiteID: "radio-t", FileName: testDB})
	assert.NoError(t, err)

	b := &service.DataStore{Engine: boltStore, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}

	comment := store.Comment{
		ID:        "efbc17f177ee1a1c0ee6e1e025749966ec071adc",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = b.Create(comment)
	assert.NoError(t, err)

	comment = store.Comment{
		Text: "some text2", Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.Local),
		Locator: store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:    store.User{ID: "user2", Name: "user name"},
	}
	_, err = b.Create(comment)
	assert.NoError(t, err)

	return b, func() {
		require.NoError(t, b.Close())
		_ = os.Remove(testDB)
	}
}
