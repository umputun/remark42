// +build mongo

package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/globalsign/mgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/engine/mongo"
)

func TestMongo_CreateAndFind(t *testing.T) {
	var m Interface
	m = prepMongo(t, true) // adds two comments

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
	m := prepMongo(t, true) // adds two comments

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
	m := prepMongo(t, true) // adds two comments

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
	m := prepMongo(t, true) // adds two comments

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
	m := prepMongo(t, true) // adds two comments

	c, err := m.Count(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"})
	assert.Nil(t, err)
	assert.Equal(t, 2, c)

	c, err = m.Count(store.Locator{URL: "https://radio-t.com-xxx", SiteID: "radio-t"})
	assert.Nil(t, err)
	assert.Equal(t, 0, c)
}

func TestMongo_List(t *testing.T) {
	m := prepMongo(t, true) // adds two comments

	// add one more for https://radio-t.com/2
	comment := store.Comment{
		ID:        "12345",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err := m.Create(comment)
	assert.Nil(t, err)

	ts := func(sec int) time.Time { return time.Date(2017, 12, 20, 15, 18, sec, 0, time.Local).In(time.UTC) }

	res, err := m.List("radio-t", 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(22), LastTS: ts(22)},
		{URL: "https://radio-t.com", Count: 2, FirstTS: ts(22), LastTS: ts(23)}},
		res)

	res, err = m.List("radio-t", -1, -1)
	assert.Nil(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(22), LastTS: ts(22)},
		{URL: "https://radio-t.com", Count: 2, FirstTS: ts(22), LastTS: ts(23)}}, res)

	res, err = m.List("radio-t", 1, 0)
	assert.Nil(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(22), LastTS: ts(22)}}, res)

	res, err = m.List("radio-t", 1, 1)
	assert.Nil(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com", Count: 2, FirstTS: ts(22), LastTS: ts(23)}}, res)

	res, err = m.List("bad", 1, 1)
	assert.Nil(t, err)
	assert.Equal(t, []store.PostInfo{}, res)
}

func TestMongo_Info(t *testing.T) {
	m := prepMongo(t, true) // adds two comments

	ts := func(min int) time.Time { return time.Date(2017, 12, 20, 15, 18, min, 0, time.Local).In(time.UTC) }

	// add one more for https://radio-t.com/2
	comment := store.Comment{
		ID:        "12345",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 24, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err := m.Create(comment)
	assert.Nil(t, err)

	r, err := m.Info(store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"}, 0)
	require.Nil(t, err)
	assert.Equal(t, store.PostInfo{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(24), LastTS: ts(24)}, r)

	r, err = m.Info(store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"}, 10)
	require.Nil(t, err)
	assert.Equal(t, store.PostInfo{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(24), LastTS: ts(24), ReadOnly: true}, r)

	r, err = m.Info(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, 0)
	require.Nil(t, err)
	assert.Equal(t, store.PostInfo{URL: "https://radio-t.com", Count: 2, FirstTS: ts(22), LastTS: ts(23)}, r)

	_, err = m.Info(store.Locator{URL: "https://radio-t.com/error", SiteID: "radio-t"}, 0)
	require.NotNil(t, err)

	_, err = m.Info(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t-error"}, 0)
	require.NotNil(t, err)

	err = m.SetReadOnly(store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"}, true)
	require.Nil(t, err)
	r, err = m.Info(store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"}, 0)
	require.Nil(t, err)
	assert.Equal(t, store.PostInfo{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(24), LastTS: ts(24), ReadOnly: true}, r)
}

func TestMongo_ReadOnly(t *testing.T) {
	m := prepMongo(t, true) // adds two comments

	assert.False(t, m.IsReadOnly(store.Locator{SiteID: "radio-t", URL: "url-1"}), "nothing ro")

	assert.NoError(t, m.SetReadOnly(store.Locator{SiteID: "radio-t", URL: "url-1"}, true))
	assert.True(t, m.IsReadOnly(store.Locator{SiteID: "radio-t", URL: "url-1"}), "url-1 ro")

	assert.False(t, m.IsReadOnly(store.Locator{SiteID: "radio-t", URL: "url-2"}), "url-2 still writable")

	assert.NoError(t, m.SetReadOnly(store.Locator{SiteID: "radio-t", URL: "url-1"}, false))
	assert.False(t, m.IsReadOnly(store.Locator{SiteID: "radio-t", URL: "url-1"}), "url-1 writable")

	assert.NotNil(t, m.SetReadOnly(store.Locator{SiteID: "bad", URL: "url-1"}, true), "nos site \"bad\"")
	assert.NoError(t, m.SetReadOnly(store.Locator{SiteID: "radio-t", URL: "url-1xyz"}, false))

	assert.False(t, m.IsReadOnly(store.Locator{SiteID: "radio-t-bad", URL: "url-1"}), "nothing blocked on wrong site")
}

func TestMongo_Verified(t *testing.T) {
	m := prepMongo(t, true) // adds two comments

	assert.False(t, m.IsVerified("radio-t", "u1"), "nothing verified")

	assert.NoError(t, m.SetVerified("radio-t", "u1", true))
	assert.True(t, m.IsVerified("radio-t", "u1"), "u1 verified")

	assert.False(t, m.IsVerified("radio-t", "u2"), "u2 still not verified")
	assert.NoError(t, m.SetVerified("radio-t", "u1", false))
	assert.False(t, m.IsVerified("radio-t", "u1"), "u1 not verified anymore")

	assert.NotNil(t, m.SetVerified("bad", "u1", true), `site "bad" not found`)
	assert.NoError(t, m.SetVerified("radio-t", "u1xyz", false))

	assert.False(t, m.IsVerified("radio-t-bad", "u1"), "nothing verified on wrong site")
}

func TestMongo_GetForUser(t *testing.T) {
	m := prepMongo(t, true) // adds two comments

	res, err := m.User("radio-t", "user1", 5, 0)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "some text2", res[0].Text, "sorted by -time")

	res, err = m.User("radio-t", "user1", 1, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(res), "allow 1 comment")
	assert.Equal(t, "some text2", res[0].Text, "sorted by -time")

	res, err = m.User("radio-t", "user1", 1, 1)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(res), "allow 1 comment")
	assert.Equal(t, `some text, <a href="http://radio-t.com">link</a>`, res[0].Text, "second comment")

	res, err = m.User("bad", "user1", 1, 0)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(res))
}

func TestMongo_GetForUserPagination(t *testing.T) {
	m := prepMongo(t, false)

	c := store.Comment{
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:    store.User{ID: "user1", Name: "user name"},
	}

	// write 200 comments
	for i := 0; i < 200; i++ {
		c.ID = fmt.Sprintf("id-%d", i)
		c.Text = fmt.Sprintf("text #%d", i)
		c.Timestamp = time.Date(2017, 12, 20, 15, 18, i, 0, time.Local)
		_, err := m.Create(c)
		require.Nil(t, err, c.ID)
	}

	// get all comments
	res, err := m.User("radio-t", "user1", 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 200, len(res))
	assert.Equal(t, "id-199", res[0].ID)

	// seek 0, 5 comments
	res, err = m.User("radio-t", "user1", 5, 0)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(res))
	assert.Equal(t, "id-199", res[0].ID)
	assert.Equal(t, "id-195", res[4].ID)

	// seek 10, 3 comments
	res, err = m.User("radio-t", "user1", 3, 10)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(res))
	assert.Equal(t, "id-189", res[0].ID)
	assert.Equal(t, "id-187", res[2].ID)

	// seek 195, ask 10 comments
	res, err = m.User("radio-t", "user1", 10, 195)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(res))
	assert.Equal(t, "id-4", res[0].ID)
	assert.Equal(t, "id-0", res[4].ID)

	// seek 255, ask 10 comments
	res, err = m.User("radio-t", "user1", 10, 255)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(res))
}

func TestMongo_BlockUser(t *testing.T) {
	m := prepMongo(t, true) // adds two comments

	assert.False(t, m.IsBlocked("radio-t", "user1"), "nothing blocked")

	assert.NoError(t, m.SetBlock("radio-t", "user1", true, 0))
	assert.True(t, m.IsBlocked("radio-t", "user1"), "user1 blocked")

	assert.False(t, m.IsBlocked("radio-t", "user2"), "user2 still unblocked")

	assert.NoError(t, m.SetBlock("radio-t", "user1", false, 0))
	assert.False(t, m.IsBlocked("radio-t", "user1"), "user1 unblocked")

	assert.NotNil(t, m.SetBlock("bad", "user1", true, 0), `site "bad" not found`)
	assert.NoError(t, m.SetBlock("radio-t", "userX", false, 0))

	assert.False(t, m.IsBlocked("radio-t-bad", "user1"), "nothing blocked on wrong site")
}

func TestMongo_BlockUserWithTTL(t *testing.T) {
	m := prepMongo(t, true) // adds two comments

	assert.False(t, m.IsBlocked("radio-t", "user1"), "nothing blocked")
	assert.NoError(t, m.SetBlock("radio-t", "user1", true, 50*time.Millisecond))
	assert.True(t, m.IsBlocked("radio-t", "user1"), "user1 blocked")
	time.Sleep(50 * time.Millisecond)
	assert.False(t, m.IsBlocked("radio-t", "user1"), "user1 un-blocked automatically")
}

func TestMongo_GetForUserCounter(t *testing.T) {
	m := prepMongo(t, true) // adds two comments

	count, err := m.UserCount("radio-t", "user1")
	assert.Nil(t, err)
	assert.Equal(t, 2, count)

	count, err = m.UserCount("bad", "user1")
	assert.Nil(t, err)
	assert.Equal(t, 0, count)
}

func TestMongo_BlockList(t *testing.T) {
	m := prepMongo(t, true) // adds two comments

	assert.NoError(t, m.SetBlock("radio-t", "user1", true, 0))
	assert.NoError(t, m.SetBlock("radio-t", "user2", true, 50*time.Millisecond))
	assert.NoError(t, m.SetBlock("radio-t", "user3", false, 0))

	ids, err := m.Blocked("radio-t")
	assert.NoError(t, err)

	assert.Equal(t, 2, len(ids))
	assert.Equal(t, "user1", ids[0].ID)
	assert.Equal(t, "user2", ids[1].ID)
	t.Logf("%+v", ids)

	time.Sleep(50 * time.Millisecond)
	ids, err = m.Blocked("radio-t")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ids))
	assert.Equal(t, "user1", ids[0].ID)

	ids, err = m.Blocked("bad")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(ids))
}

func TestMongo_Delete(t *testing.T) {
	m := prepMongo(t, true) // adds two comments

	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	res, err := m.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res), "initially 2 comments")

	err = m.Delete(loc, res[0].ID, store.SoftDelete)
	assert.Nil(t, err)

	res, err = m.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "", res[0].Text)
	assert.True(t, res[0].Deleted, "marked deleted")
	assert.Equal(t, store.User{Name: "user name", ID: "user1", Picture: "", Admin: false, Blocked: false, IP: ""}, res[0].User)

	assert.Equal(t, "some text2", res[1].Text)
	assert.False(t, res[1].Deleted)

	comments, err := m.Last("radio-t", 10)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(comments), "1 in last, 1 removed")

	err = m.Delete(loc, "123456", store.SoftDelete)
	assert.NotNil(t, err)

	loc.SiteID = "bad"
	err = m.Delete(loc, res[0].ID, store.SoftDelete)
	assert.EqualError(t, err, `can't delete id-1: not found`)

	loc = store.Locator{URL: "https://radio-t.com/bad", SiteID: "radio-t"}
	err = m.Delete(loc, res[0].ID, store.SoftDelete)
	assert.EqualError(t, err, `can't delete id-1: not found`)
}

func TestMongo_DeleteHard(t *testing.T) {
	m := prepMongo(t, true) // adds two comments

	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	res, err := m.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res), "initially 2 comments")

	err = m.Delete(loc, res[0].ID, store.HardDelete)
	assert.Nil(t, err)

	res, err = m.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "", res[0].Text)
	assert.True(t, res[0].Deleted, "marked deleted")
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, res[0].User)
}

func TestMongo_DeleteAll(t *testing.T) {
	m := prepMongo(t, true) // adds two comments

	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	res, err := m.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res), "initially 2 comments")

	err = m.DeleteAll("radio-t")
	assert.Nil(t, err)

	comments, err := m.Last("radio-t", 10)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(comments), "nothing left")

	c, err := m.Count(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"})
	assert.Nil(t, err)
	assert.Equal(t, 0, c, "0 count")
}

func TestMongo_DeleteUser(t *testing.T) {
	m := prepMongo(t, true) // adds two comments

	err := m.DeleteUser("radio-t", "user1")
	require.NoError(t, err)

	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	res, err := m.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res), "2 comments with deleted info")
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, res[0].User)
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, res[1].User)

	c, err := m.Count(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"})
	assert.Nil(t, err)
	assert.Equal(t, 0, c, "0 count")

	cc, err := m.User("radio-t", "user1", 5, 0)
	assert.Nil(t, err, "no comments for user user1 in store")
	assert.Equal(t, 0, len(cc), "no comments for user user1 in store")

	comments, err := m.Last("radio-t", 10)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(comments), "nothing left")
}

func prepMongo(t *testing.T, writeRecs bool) *Mongo {
	mg := mongo.NewTesting(mongoPosts)
	mg.DropCollection()
	conn, err := mg.Get()
	require.Nil(t, err)
	m, err := NewMongo(conn, 1, 0*time.Microsecond)
	require.Nil(t, err)
	_ = m.conn.WithCustomCollection(mongoMetaPosts, func(coll *mgo.Collection) error {
		return coll.DropCollection()
	})
	_ = m.conn.WithCustomCollection(mongoMetaUsers, func(coll *mgo.Collection) error {
		return coll.DropCollection()
	})

	comment := store.Comment{
		ID:        "id-1",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	if writeRecs {
		_, err = m.Create(comment)
		assert.Nil(t, err)
	}

	comment = store.Comment{
		ID:        "id-2",
		Text:      "some text2",
		Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	if writeRecs {
		_, err = m.Create(comment)
		assert.Nil(t, err)
	}

	return m
}
