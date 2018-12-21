package migrator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	bolt "github.com/coreos/bbolt"
	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/admin"
	"github.com/umputun/remark/backend/app/store/engine"
	"github.com/umputun/remark/backend/app/store/service"
)

var testDb = "/tmp/test-remark.db"

func TestRemark_Export(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t) // write 2 comments
	b.SetReadOnly(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, true)
	b.SetVerified("radio-t", "user1", true)
	b.SetBlock("radio-t", "user2", true, time.Hour)
	r := Remark{DataStore: b}

	buf := &bytes.Buffer{}
	size, err := r.Export(buf, "radio-t")
	assert.Nil(t, err)
	assert.Equal(t, 2, size)

	c1 := buf.String()
	log.Print(c1)

	res := struct {
		Version  int             `json:"version"`
		Comments []store.Comment `json:"comments"`
		Meta     struct {
			Users []service.UserMetaData `json:"users"`
			Posts []service.PostMetaData `json:"posts"`
		} `json:"meta"`
	}{}

	err = json.Unmarshal([]byte(c1), &res)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(res.Comments))
	assert.Equal(t, "some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>", res.Comments[0].Text)

	assert.Equal(t, 2, len(res.Meta.Users))
	assert.Equal(t, "user1", res.Meta.Users[0].ID)
	assert.Equal(t, false, res.Meta.Users[0].Blocked.Status)
	assert.Equal(t, true, res.Meta.Users[0].Verified)
	assert.Equal(t, "user2", res.Meta.Users[1].ID)
	assert.Equal(t, true, res.Meta.Users[1].Blocked.Status)
	assert.Equal(t, false, res.Meta.Users[1].Verified)

	assert.Equal(t, 1, len(res.Meta.Posts))
	assert.Equal(t, "https://radio-t.com", res.Meta.Posts[0].URL)
	assert.Equal(t, true, res.Meta.Posts[0].ReadOnly)
}

func TestRemark_Import(t *testing.T) {
	defer os.Remove(testDb)

	r1 := `{"id":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"","text":"some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","profile":"","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:22-06:00"}` + "\n"

	r2 := `{"id":"afbc17f177ee1a1c0ee6e1e025749966ec071adc","pid":"efbc17f177ee1a1c0ee6e1e025749966ec071adc","text":"some text2, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","profile":"","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:23-06:00"}` + "\n"

	buf := &bytes.Buffer{}
	buf.WriteString(r1)
	buf.WriteString(r2)
	buf.WriteString("{}")

	b := prep(t) // write some recs
	r := Remark{DataStore: &service.DataStore{Interface: b, AdminStore: admin.NewStaticStore("12345", []string{}, "")}}
	size, err := r.Import(buf, "radio-t")
	assert.Nil(t, err)
	assert.Equal(t, 2, size)

	comments, err := b.Find(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(comments))
	assert.Equal(t, "efbc17f177ee1a1c0ee6e1e025749966ec071adc", comments[0].ID)
	assert.Equal(t, "afbc17f177ee1a1c0ee6e1e025749966ec071adc", comments[1].ID)
	assert.Equal(t, "efbc17f177ee1a1c0ee6e1e025749966ec071adc", comments[1].ParentID)

	// try import again
	buf.WriteString(r1)
	buf.WriteString(r2)
	buf.WriteString("{}")
	size, err = r.Import(buf, "radio-t")
	assert.Nil(t, err)
	assert.Equal(t, 2, size)
}

func TestRemark_ImportManyWithError(t *testing.T) {
	defer os.Remove(testDb)

	goodRec := `{"id":"%d","pid":"","text":"some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>","user":{"name":"user name","id":"user1","picture":"","profile":"","admin":false},"locator":{"site":"radio-t","url":"https://radio-t.com"},"score":0,"votes":{},"time":"2017-12-20T15:18:22-06:00"}` + "\n"

	buf := &bytes.Buffer{}
	for i := 0; i < 1200; i++ {
		buf.WriteString(fmt.Sprintf(goodRec, i))
	}
	buf.WriteString("bad1\n")
	buf.WriteString("bad2\n")

	b := prep(t) // write some recs
	r := Remark{DataStore: &service.DataStore{Interface: b, AdminStore: admin.NewStaticStore("12345", []string{}, "")}}
	n, err := r.Import(buf, "radio-t")
	assert.EqualError(t, err, "failed to save 2 comments")
	assert.Equal(t, 1200, n)
	comments, err := b.Find(store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}, "time")
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
