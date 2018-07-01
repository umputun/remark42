package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/engine/mongo"
)

func TestMongo_CreateAndFind(t *testing.T) {
	m := prepMongo(t) // adds two comments

	res, err := m.Find(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "time")
	assert.Nil(t, err)
	require.Equal(t, 2, len(res))
	assert.Equal(t, `some text, <a href="http://radio-t.com">link</a>`, res[0].Text)
	assert.Equal(t, "user1", res[0].User.ID)
	t.Log(res[0].ID)

	_, err = m.Create(store.Comment{ID: res[0].ID, Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}})
	assert.NotNil(t, err, "reject dup")

	id, err := m.Create(store.Comment{ID: "id-3", Locator: store.Locator{URL: "https://radio-t2.com", SiteID: "radio-t2"}})
	assert.Nil(t, err)
	assert.Equal(t, "id-3", id)
	res, err = m.Find(store.Locator{URL: "https://radio-t2.com", SiteID: "radio-t2"}, "time")
	assert.Nil(t, err)
	require.Equal(t, 1, len(res))
}

func TestMongo_Get(t *testing.T) {
	m := prepMongo(t) // adds two comments

	res, err := m.Find(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))

	comment, err := m.Get(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[1].ID)
	assert.Nil(t, err)
	assert.Equal(t, "some text2", comment.Text)

	comment, err = m.Get(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "1234567")
	assert.NotNil(t, err, "not found")
}

func TestMongo_Put(t *testing.T) {
	m := prepMongo(t) // adds two comments

	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	res, err := m.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))

	comment := res[0]
	comment.Text = "abc 123"
	comment.Score = 100
	err = m.Put(loc, comment)
	assert.Nil(t, err)

	comment, err = m.Get(loc, res[0].ID)
	assert.Nil(t, err)
	assert.Equal(t, "abc 123", comment.Text)
	assert.Equal(t, res[0].ID, comment.ID)
	assert.Equal(t, 100, comment.Score)

	err = m.Put(store.Locator{URL: "https://radio-t.com", SiteID: "bad"}, comment)
	assert.EqualError(t, err, `not found`)

	err = m.Put(store.Locator{URL: "https://radio-t.com-bad", SiteID: "radio-t"}, comment)
	assert.EqualError(t, err, `not found`)
}

func TestMongo_Last(t *testing.T) {
	m := prepMongo(t) // adds two comments

	res, err := m.Last("radio-t", 0)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "some text2", res[0].Text)

	res, err = m.Last("radio-t", 1)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "some text2", res[0].Text)
}

func TestMongo_Count(t *testing.T) {
	m := prepMongo(t) // adds two comments

	c, err := m.Count(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"})
	assert.Nil(t, err)
	assert.Equal(t, 2, c)

	c, err = m.Count(store.Locator{URL: "https://radio-t.com-xxx", SiteID: "radio-t"})
	assert.Nil(t, err)
	assert.Equal(t, 0, c)
}

func prepMongo(t *testing.T) *Mongo {
	mg := mongo.NewTesting("remark42")
	mg.DropCollection()
	conn, err := mg.Get()
	require.Nil(t, err)
	m := &Mongo{Connection: conn}
	comment := store.Comment{
		ID:        "id-1",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = m.Create(comment)
	assert.Nil(t, err)

	comment = store.Comment{
		ID:        "id-2",
		Text:      "some text2",
		Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = m.Create(comment)
	assert.Nil(t, err)

	return m
}
