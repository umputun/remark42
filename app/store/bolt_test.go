package store

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testDb = "/tmp/test-remark.db"

func TestBoltDB_CreateAndFind(t *testing.T) {
	var b Interface
	defer os.Remove(testDb)
	b = prep(t)

	res, err := b.Find(Request{Locator: Locator{URL: "https://radio-t.com"}})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "some text, <a href=\"http://radio-t.com\" rel=\"nofollow\">link</a>", res[0].Text)
	assert.Equal(t, "user1", res[0].User.ID)
	t.Log(res[0].ID)
}

func TestBoltDB_Delete(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	loc := Locator{URL: "https://radio-t.com"}
	res, err := b.Find(Request{Locator: loc})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))

	err = b.Delete(loc, res[0].ID)
	assert.Nil(t, err)

	res, err = b.Find(Request{Locator: loc})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "some text2", res[0].Text)

	comments, err := b.Last(loc, 10)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(comments), "only 1 left in last")
}

func TestBoltDB_Get(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	res, err := b.Find(Request{Locator: Locator{URL: "https://radio-t.com"}})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))

	comment, err := b.Get(Locator{URL: "https://radio-t.com"}, res[1].ID)
	assert.Nil(t, err)
	assert.Equal(t, "some text2", comment.Text)

	comment, err = b.Get(Locator{URL: "https://radio-t.com"}, "1234567")
	assert.NotNil(t, err)
}

func TestBoltDB_Last(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	res, err := b.Last(Locator{URL: "https://radio-t.com"}, 0)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "some text2", res[0].Text)

	res, err = b.Last(Locator{URL: "https://radio-t.com"}, 1)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "some text2", res[0].Text)
}

func TestBoltDB_Vote(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	res, err := b.Last(Locator{URL: "https://radio-t.com"}, 0)
	t.Logf("%+v", res[0])
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, 0, res[0].Score)
	assert.Equal(t, map[string]bool{}, res[0].Votes)

	c, err := b.Vote(Locator{URL: "https://radio-t.com"}, res[0].ID, "user1", true)
	assert.Nil(t, err)
	assert.Equal(t, 1, c.Score)
	assert.Equal(t, map[string]bool{"user1": true}, c.Votes)

	_, err = b.Vote(Locator{URL: "https://radio-t.com"}, res[0].ID, "user1", true)
	assert.NotNil(t, err, "double-voting rejected")

	res, err = b.Last(Locator{URL: "https://radio-t.com"}, 0)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, 1, res[0].Score)
}

func TestBoltDB_Count(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	c, err := b.Count(Locator{URL: "https://radio-t.com"})
	assert.Nil(t, err)
	assert.Equal(t, 2, c)
}

func TestBoltDB_BlockUser(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	assert.False(t, b.IsBlocked(Locator{SiteID: "site1"}, "user1"), "nothing blocked")

	assert.NoError(t, b.SetBlock(Locator{SiteID: "site1"}, "user1", true))
	assert.True(t, b.IsBlocked(Locator{SiteID: "site1"}, "user1"), "user1 blocked")

	assert.False(t, b.IsBlocked(Locator{SiteID: "site1"}, "user2"), "user2 still unblocked")

	assert.NoError(t, b.SetBlock(Locator{SiteID: "site1"}, "user1", false))
	assert.False(t, b.IsBlocked(Locator{SiteID: "site1"}, "user1"), "user1 unblocked")

}

// makes new boltdb, put two records
func prep(t *testing.T) *BoltDB {
	os.Remove(testDb)

	b, err := NewBoltDB(testDb)
	assert.Nil(t, err)

	comment := Comment{Text: `some text, <a href="http://radio-t.com">link</a>`, Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator: Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, User: User{ID: "user1", Name: "user name"}}
	_, err = b.Create(comment)
	assert.Nil(t, err)

	comment = Comment{Text: "some text2", Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.Local),
		Locator: Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, User: User{ID: "user1", Name: "user name"}}
	_, err = b.Create(comment)
	assert.Nil(t, err)

	return b
}
