package migrator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
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

	assert.Equal(t, "some text, <a href=\"http://radio-t.com\" rel=\"nofollow ugc\">link</a>", comments[0].Text)
}

func TestNative_Import(t *testing.T) {
	b, teardown := prep(t) // write 2 comments
	defer teardown()

	inp := `{"version":1,"users":[{"id":"user1","blocked":{"status":false,"until":"0001-01-01T00:00:00Z"},"verified":true},{"id":"user2","blocked":{"status":true,"until":"2100-01-01T00:00:00Z"},"verified":false}],"posts":[{"url":"https://radio-t.com","read_only":true}]}
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

func TestNative_ImportSkipsCommentsOfAnotherSite(t *testing.T) {
	// a backup restored under the wrong site ID: the site given is cleared, and a comment recorded
	// for another site must not be written there, into a site the import did not clear
	boltStore, err := engine.NewBoltDB(bolt.Options{},
		engine.BoltSite{SiteID: "radio-t", FileName: filepath.Join(t.TempDir(), "radio-t.db")},
		engine.BoltSite{SiteID: "other", FileName: filepath.Join(t.TempDir(), "other.db")})
	require.NoError(t, err)
	b := &service.DataStore{Engine: boltStore, AdminStore: admin.NewStaticStore("12345", nil, []string{}, "")}
	defer func() { require.NoError(t, b.Close()) }()

	inp := `{"version":1,"users":[],"posts":[]}
	{"id":"c1","text":"own site","user":{"name":"user name","id":"user1"},"locator":{"site":"radio-t","url":"https://radio-t.com"},"time":"2017-12-20T15:18:22-06:00"}
	{"id":"c2","text":"another site","user":{"name":"user name","id":"user1"},"locator":{"site":"other","url":"https://other.com"},"time":"2017-12-20T15:18:23-06:00"}`

	size, err := (&Native{DataStore: b}).Import(strings.NewReader(inp), "radio-t")
	require.EqualError(t, err, "failed to save 1 comments")
	assert.Equal(t, 1, size)

	comments, err := b.Last("radio-t", 10, time.Time{}, store.User{})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "c1", comments[0].ID)

	comments, err = b.Last("other", 10, time.Time{}, store.User{})
	require.NoError(t, err)
	assert.Empty(t, comments, "nothing is written into the site the import did not clear")
}

func TestNative_ImportWithMapperBadRecord(t *testing.T) {
	mapper, err := NewURLMapper(strings.NewReader("https://radio-t.com* https://rdt.c*"))
	require.NoError(t, err)
	good := `{"id":"c1","pid":"","text":"text","user":{"name":"user name","id":"user1"},"locator":{"site":"radio-t","url":"https://radio-t.com"},"time":"2017-12-20T15:18:22-06:00"}`
	tbl := []struct{ name, bad string }{
		{"bad value", `{"id":"c2","text":"text","time":"not a time"}`},
		{"truncated", `{"id":"c2","text":"te`},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			b, teardown := prep(t)
			defer teardown()
			inp := `{"version":1}` + "\n" + good + "\n" + tt.bad + "\n" + strings.ReplaceAll(good, `"c1"`, `"c3"`) + "\n"
			size, err := (&Native{DataStore: b}).Import(WithMapper(strings.NewReader(inp), mapper), "radio-t")
			require.Error(t, err, "a record the mapper can't read fails the import")
			assert.Equal(t, 1, size)
		})
	}
}

func TestNative_ImportWithMapper(t *testing.T) {
	b, teardown := prep(t) // write 2 comments
	defer teardown()

	// want to remap comments to https://rdt.c
	rules := `https://radio-t.com* https://rdt.c*`
	mapper, err := NewURLMapper(strings.NewReader(rules))
	assert.NoError(t, err)

	inp := `{"version":1,"users":[{"id":"user1","blocked":{"status":false,"until":"0001-01-01T00:00:00Z"},"verified":true},{"id":"user2","blocked":{"status":true,"until":"2100-01-01T00:00:00Z"},"verified":false}],"posts":[{"url":"https://radio-t.com","read_only":true}]}
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
	for i := range 100 {
		fmt.Fprintf(buf, goodRec, i)
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
	testDB := filepath.Join(t.TempDir(), "migrator.db")

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
	}
}

func TestNative_ImportMalformedStream(t *testing.T) {
	tbl := []struct {
		name, tail string
	}{
		{"truncated", `{"id":"c2","text":"some te`},
		{"invalid json", `{"id":"c2", text}` + "\n" + `{"id":"c3","text":"x","locator":{"site":"radio-t","url":"https://radio-t.com"}}`},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			b, teardown := prep(t)
			defer teardown()
			inp := `{"version":1,"users":[{"id":"user1","blocked":{"status":false},"verified":true}],"posts":[]}
	{"id":"c1","text":"some text","user":{"name":"user name","id":"user1"},"locator":{"site":"radio-t","url":"https://radio-t.com"},"time":"2017-12-20T15:18:22-06:00"}
	` + tt.tail

			type result struct {
				size int
				err  error
			}
			done := make(chan result, 1)
			go func() {
				r := Native{DataStore: b}
				size, err := r.Import(strings.NewReader(inp), "radio-t")
				done <- result{size, err}
			}()
			var res result
			require.Eventually(t, func() bool {
				select {
				case res = <-done:
					return true
				default:
					return false
				}
			}, 30*time.Second, 10*time.Millisecond, "import of a malformed stream must finish")

			require.Error(t, res.err)
			assert.Contains(t, res.err.Error(), "can't decode comment after 1 records")
			assert.NotContains(t, res.err.Error(), "failed to save")
			assert.Equal(t, 1, res.size)
			assert.True(t, b.IsVerified("radio-t", "user1"), "metas are restored")
		})
	}
}

func TestNative_ImportRestoresMetasWhenCommentRejected(t *testing.T) {
	b, teardown := prep(t)
	defer teardown()

	inp := `{"version":1,"users":[{"id":"user1","blocked":{"status":false},"verified":true},{"id":"user2","blocked":{"status":true,"until":"2100-01-01T00:00:00Z"}}],"posts":[{"url":"https://radio-t.com","read_only":true}]}
	{"id":"c1","text":"some text","user":{"name":"user name","id":"user1"},"locator":{"site":"radio-t","url":"https://radio-t.com"},"time":"2017-12-20T15:18:22-06:00"}
	{}`

	r := Native{DataStore: b}
	size, err := r.Import(strings.NewReader(inp), "radio-t")
	require.EqualError(t, err, "failed to save 1 comments")
	assert.Equal(t, 1, size)
	assert.True(t, b.IsVerified("radio-t", "user1"), "verified flag restored")
	assert.True(t, b.IsBlocked("radio-t", "user2"), "block restored")
	assert.True(t, b.IsReadOnly(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}), "read-only post restored")
}

func TestNative_ImportKeepsCommentsOfReadOnlyPost(t *testing.T) {
	b, teardown := prep(t)
	defer teardown()

	closed := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	require.NoError(t, b.SetReadOnly(closed, true))

	// a restore into the same store, as the remap does: the site is cleared and the post stays read-only
	buf := &bytes.Buffer{}
	_, err := (&Native{DataStore: b}).Export(buf, "radio-t")
	require.NoError(t, err)
	size, err := (&Native{DataStore: b}).Import(buf, "radio-t")
	require.NoError(t, err)
	assert.Equal(t, 2, size)

	comments, err := b.Engine.Find(engine.FindRequest{Locator: closed})
	require.NoError(t, err)
	require.Len(t, comments, 1, "the comment of the read-only post is back")
	assert.Equal(t, "efbc17f177ee1a1c0ee6e1e025749966ec071adc", comments[0].ID)
	assert.True(t, b.IsReadOnly(closed), "the post is still read-only")
}

func TestNative_ImportKeepsCommentsWithRestrictedWords(t *testing.T) {
	b, teardown := prep(t)
	defer teardown()

	buf := &bytes.Buffer{}
	_, err := (&Native{DataStore: b}).Export(buf, "radio-t")
	require.NoError(t, err)

	// both stored comments hold a word restricted after they were written
	b.RestrictedWordsMatcher = service.NewRestrictedWordsMatcher(service.StaticRestrictedWordsLister{Words: []string{"text*"}})
	size, err := (&Native{DataStore: b}).Import(buf, "radio-t")
	require.NoError(t, err)
	assert.Equal(t, 2, size)
}

func TestNative_ImportSkipsRecordWithWrongValue(t *testing.T) {
	b, teardown := prep(t)
	defer teardown()

	inp := `{"version":1,"users":[{"id":"user1","blocked":{"status":false},"verified":true}],"posts":[]}
	{"id":"c1","text":"text 1","user":{"name":"user name","id":"user1"},"locator":{"site":"radio-t","url":"https://radio-t.com"},"time":"2017-12-20T15:18:22-06:00"}
	{"id":"c2","text":"text 2","user":{"name":"user name","id":"user1"},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":"x","time":"2017-12-20T15:18:23-06:00"}
	{"id":"c3","text":"text 3","user":{"name":"user name","id":"user1"},"locator":{"site":"radio-t","url":"https://radio-t.com"},"time":"not a time"}
	{"id":"c4","text":"text 4","user":{"name":"user name","id":"user1"},"locator":{"site":"radio-t","url":"https://radio-t.com"},"time":"2017-12-20T15:18:25-06:00"}`

	r := Native{DataStore: b}
	size, err := r.Import(strings.NewReader(inp), "radio-t")
	require.EqualError(t, err, "failed to save 2 comments")
	assert.Equal(t, 2, size)

	comments, err := b.Find(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}, "time", store.User{})
	require.NoError(t, err)
	ids := make([]string, 0, len(comments))
	for _, c := range comments {
		ids = append(ids, c.ID)
	}
	assert.Equal(t, []string{"c1", "c4"}, ids, "records after a wrong value are restored")
	assert.True(t, b.IsVerified("radio-t", "user1"))
}

func TestNative_ExportKeepsStoredComments(t *testing.T) {
	b, teardown := prep(t) // write 2 comments
	defer teardown()
	b.MaxVotes = service.UnlimitedVotes
	_, err := b.Vote(service.VoteReq{Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"},
		CommentID: "efbc17f177ee1a1c0ee6e1e025749966ec071adc", UserID: "user3", UserIP: "1.2.3.4", Val: true})
	require.NoError(t, err)
	require.NoError(t, b.SetBlock("radio-t", "user1", true, time.Hour))

	buf := &bytes.Buffer{}
	r := Native{DataStore: b}
	size, err := r.Export(buf, "radio-t")
	require.NoError(t, err)
	assert.Equal(t, 2, size)

	dec := json.NewDecoder(buf)
	require.NoError(t, dec.Decode(&meta{}))
	c := store.Comment{}
	require.NoError(t, dec.Decode(&c))
	require.Equal(t, "efbc17f177ee1a1c0ee6e1e025749966ec071adc", c.ID)
	assert.Equal(t, map[string]bool{"user3": true}, c.Votes, "votes are exported")
	assert.Equal(t, 1, c.Score)
	assert.Empty(t, c.VotedIPs, "voters' IP hashes are not exported")
	assert.False(t, c.User.Blocked, "blocked status lives in the users meta, not in the comment")
}

func TestNative_ExportDropsTitleOfDeletedComment(t *testing.T) {
	b, teardown := prep(t) // write 2 comments
	defer teardown()
	const deletedID = "efbc17f177ee1a1c0ee6e1e025749966ec071adc"

	// both comments get a title, and the first is stored the way a delete used to leave it: no text, the title kept
	for _, post := range []string{"https://radio-t.com", "https://radio-t.com/2"} {
		comments, err := b.Engine.Find(engine.FindRequest{Locator: store.Locator{SiteID: "radio-t", URL: post}})
		require.NoError(t, err)
		require.Len(t, comments, 1)
		c := comments[0]
		c.PostTitle = "title sent with the comment"
		if c.ID == deletedID {
			c.Text, c.Orig, c.Deleted = "", "", true
		}
		require.NoError(t, b.Engine.Update(c))
	}

	buf := &bytes.Buffer{}
	size, err := (&Native{DataStore: b}).Export(buf, "radio-t")
	require.NoError(t, err)
	require.Equal(t, 2, size)

	dec := json.NewDecoder(buf)
	require.NoError(t, dec.Decode(&meta{}))
	titles := map[bool]string{}
	for range size {
		c := store.Comment{}
		require.NoError(t, dec.Decode(&c))
		require.Equal(t, c.ID == deletedID, c.Deleted)
		titles[c.Deleted] = c.PostTitle
	}
	assert.Equal(t, map[bool]string{true: "", false: "title sent with the comment"}, titles,
		"the title of a deleted comment is not exported, the title of another comment is")
}

func TestNative_ImportClearsBlockedMarkInComments(t *testing.T) {
	b, teardown := prep(t)
	defer teardown()

	// backups made by earlier versions carry "block":true inside comments of a user blocked at export time
	inp := `{"version":1,"users":[],"posts":[]}
	{"id":"c1","text":"some text","user":{"name":"user name","id":"user1","block":true},"locator":{"site":"radio-t","url":"https://radio-t.com"},"votes":{"user3":true},"score":1,"time":"2017-12-20T15:18:22-06:00"}`

	r := Native{DataStore: b}
	size, err := r.Import(strings.NewReader(inp), "radio-t")
	require.NoError(t, err)
	assert.Equal(t, 1, size)

	comments, err := b.Find(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}, "time", store.User{})
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.False(t, comments[0].User.Blocked, "user is not blocked after restore")
	assert.Equal(t, 1, comments[0].Score)
}

func TestNative_ImportRestoresUserDetails(t *testing.T) {
	b, teardown := prep(t)
	defer teardown()

	inp := `{"version":1,"users":[{"id":"user1","details":{"user_id":"user1","email":"user1@example.com"}},{"id":"user2","details":{"user_id":"user2","telegram":"user2_tg"}}],"posts":[]}
	{"id":"c1","text":"some text","user":{"name":"user name","id":"user1"},"locator":{"site":"radio-t","url":"https://radio-t.com"},"time":"2017-12-20T15:18:22-06:00"}`

	r := Native{DataStore: b}
	_, err := r.Import(strings.NewReader(inp), "radio-t")
	require.NoError(t, err)

	email, err := b.GetUserEmail("radio-t", "user1")
	require.NoError(t, err)
	assert.Equal(t, "user1@example.com", email)
	tg, err := b.GetUserTelegram("radio-t", "user2")
	require.NoError(t, err)
	assert.Equal(t, "user2_tg", tg, "telegram restored")
}

func TestNative_ExportImportKeepsEachUsersDetails(t *testing.T) {
	b, teardown := prep(t)
	defer teardown()

	// user1 sorts before user2 in the details bucket, so a value left over from user1's record would reach user2's
	_, err := b.SetUserTelegram("radio-t", "user1", "user1_tg")
	require.NoError(t, err)
	_, err = b.SetUserEmail("radio-t", "user2", "user2@example.com")
	require.NoError(t, err)

	r := Native{DataStore: b}
	buf := &bytes.Buffer{}
	_, err = r.Export(buf, "radio-t")
	require.NoError(t, err)
	_, err = r.Import(buf, "radio-t")
	require.NoError(t, err)

	tg, err := b.GetUserTelegram("radio-t", "user1")
	require.NoError(t, err)
	assert.Equal(t, "user1_tg", tg)
	email, err := b.GetUserEmail("radio-t", "user1")
	require.NoError(t, err)
	assert.Empty(t, email, "telegram-only user has no email after restore")

	email, err = b.GetUserEmail("radio-t", "user2")
	require.NoError(t, err)
	assert.Equal(t, "user2@example.com", email)
	tg, err = b.GetUserTelegram("radio-t", "user2")
	require.NoError(t, err)
	assert.Empty(t, tg, "email-only user has no telegram after restore")
}

// import reads a record only when a writer is free for it, so a slow store holds back the reading
// of the export instead of piling decoded records up in memory
func TestNative_ImportWaitsForFreeWriter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		st := &blockingStore{release: make(chan struct{})}
		pr, pw := io.Pipe()
		var sent atomic.Int32 // records the import has read from the pipe
		go func() {
			_, _ = io.WriteString(pw, `{"version":1}`+"\n")
			for i := range 10 {
				rec := fmt.Sprintf(`{"id":"%d","locator":{"site":"radio-t","url":"https://radio-t.com"}}`+"\n", i)
				if _, err := io.WriteString(pw, rec); err != nil {
					return
				}
				sent.Add(1)
			}
			_ = pw.Close()
		}()

		var size int
		var err error
		done := make(chan struct{})
		go func() {
			defer close(done)
			size, err = (&Native{DataStore: st, Concurrent: 2}).Import(pr, "radio-t")
		}()

		synctest.Wait()
		assert.Equal(t, int32(2), st.started.Load(), "two writes run")
		assert.Equal(t, int32(3), sent.Load(), "one more record is read, then reading waits for a free writer")

		close(st.release)
		<-done
		require.NoError(t, err)
		assert.Equal(t, 10, size)
		assert.Equal(t, int32(10), st.started.Load())
	})
}

// blockingStore is a Store whose Create waits until release is closed
type blockingStore struct {
	release chan struct{}
	started atomic.Int32
}

func (s *blockingStore) Create(c store.Comment) (string, error) {
	s.started.Add(1)
	<-s.release
	return c.ID, nil
}

func (*blockingStore) FindRaw(store.Locator) ([]store.Comment, error)  { return nil, nil }
func (*blockingStore) List(string, int, int) ([]store.PostInfo, error) { return nil, nil }
func (*blockingStore) DeleteAll(string) error                          { return nil }
func (*blockingStore) Metas(string) ([]service.UserMetaData, []service.PostMetaData, error) {
	return nil, nil, nil
}

func (*blockingStore) SetMetas(string, []service.UserMetaData, []service.PostMetaData) error {
	return nil
}
