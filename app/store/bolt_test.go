package store

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testDb = "/tmp/test-remark.db"

func TestBoltDB_CreateAndFind(t *testing.T) {
	defer os.Remove(testDb)

	b, err := NewBoltDB(testDb)
	assert.Nil(t, err)

	comment := Comment{Text: "some text", Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator: Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, User: User{ID: "user1", Name: "user name"}}
	err = b.Create(comment)
	assert.Nil(t, err)

	comment = Comment{Text: "some text2", Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.Local),
		Locator: Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, User: User{ID: "user1", Name: "user name"}}
	err = b.Create(comment)
	assert.Nil(t, err)

	res, err := b.Find(Request{Locator: Locator{URL: "https://radio-t.com"}})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "some text", res[0].Text)
	assert.Equal(t, "user1", res[0].User.ID)
}

func TestBoltDB_Delete(t *testing.T) {
	defer os.Remove(testDb)

	b, err := NewBoltDB(testDb)
	assert.Nil(t, err)

	comment := Comment{Text: "some text", Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator: Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, User: User{ID: "user1", Name: "user name"}}
	err = b.Create(comment)
	assert.Nil(t, err)

	comment = Comment{Text: "some text2", Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.Local),
		Locator: Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, User: User{ID: "user1", Name: "user name"}}
	err = b.Create(comment)
	assert.Nil(t, err)

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
