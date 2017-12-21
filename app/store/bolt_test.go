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
	assert.Equal(t, "some text", res[0].Text)
	assert.Equal(t, "user1", res[0].User.ID)
	t.Log(res[0].ID)
}

func TestBoltDB_Delete(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	res, err := b.Find(Request{Locator: Locator{URL: "https://radio-t.com"}})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))

	err = b.Delete("https://radio-t.com", res[0].ID)
	assert.Nil(t, err)

	res, err = b.Find(Request{Locator: Locator{URL: "https://radio-t.com"}})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "some text2", res[0].Text)
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

	comment, err = b.Get(Locator{URL: "https://radio-t.com"}, 1234567)
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

// makes new boltdb, put two records
func prep(t *testing.T) *BoltDB {
	b, err := NewBoltDB(testDb)
	assert.Nil(t, err)

	comment := Comment{Text: "some text", Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator: Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, User: User{ID: "user1", Name: "user name"}}
	_, err = b.Create(comment)
	assert.Nil(t, err)

	comment = Comment{Text: "some text2", Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.Local),
		Locator: Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, User: User{ID: "user1", Name: "user name"}}
	_, err = b.Create(comment)
	assert.Nil(t, err)

	return b
}
