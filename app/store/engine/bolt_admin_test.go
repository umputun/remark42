package engine

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark/app/store"
)

func TestBoltDB_Delete(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	res, err := b.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res), "initially 2 comments")

	err = b.Delete(loc, res[0].ID, store.SoftDelete)
	assert.Nil(t, err)

	res, err = b.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "", res[0].Text)
	assert.True(t, res[0].Deleted, "marked deleted")
	assert.Equal(t, store.User{Name: "user name", ID: "user1", Picture: "", Admin: false, Blocked: false, IP: ""}, res[0].User)

	assert.Equal(t, "some text2", res[1].Text)
	assert.False(t, res[1].Deleted)

	comments, err := b.Last("radio-t", 10)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(comments), "1 in last, 1 removed")

	err = b.Delete(loc, "123456", store.SoftDelete)
	assert.NotNil(t, err)

	loc.SiteID = "bad"
	err = b.Delete(loc, res[0].ID, store.SoftDelete)
	assert.EqualError(t, err, `site "bad" not found`)
}

func TestBoltDB_DeleteHard(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	res, err := b.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res), "initially 2 comments")

	err = b.Delete(loc, res[0].ID, store.HardDelete)
	assert.Nil(t, err)

	res, err = b.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "", res[0].Text)
	assert.True(t, res[0].Deleted, "marked deleted")
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, res[0].User)
}

func TestBoltDB_DeleteAll(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	res, err := b.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res), "initially 2 comments")

	err = b.DeleteAll("radio-t")
	assert.Nil(t, err)

	comments, err := b.Last("radio-t", 10)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(comments), "nothing left")

	c, err := b.Count(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"})
	assert.Nil(t, err)
	assert.Equal(t, 0, c, "0 count")

	err = b.DeleteAll("bad")
	assert.EqualError(t, err, `site "bad" not found`)
}

func TestBoltDB_DeleteUser(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)
	err := b.DeleteUser("radio-t", "user1")
	require.NoError(t, err)

	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	res, err := b.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res), "2 comments with deleted info")
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, res[0].User)
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, res[1].User)

	c, err := b.Count(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"})
	assert.Nil(t, err)
	assert.Equal(t, 0, c, "0 count")

	_, _, err = b.User("radio-t", "user1", 5)
	assert.EqualError(t, err, "no comments for user user1 in store")

	comments, err := b.Last("radio-t", 10)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(comments), "nothing left")
}

func TestBoltDB_BlockUser(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	assert.False(t, b.IsBlocked("radio-t", "user1"), "nothing blocked")

	assert.NoError(t, b.SetBlock("radio-t", "user1", true))
	assert.True(t, b.IsBlocked("radio-t", "user1"), "user1 blocked")

	assert.False(t, b.IsBlocked("radio-t", "user2"), "user2 still unblocked")

	assert.NoError(t, b.SetBlock("radio-t", "user1", false))
	assert.False(t, b.IsBlocked("radio-t", "user1"), "user1 unblocked")

	assert.EqualError(t, b.SetBlock("bad", "user1", true), `site "bad" not found`)
	assert.NoError(t, b.SetBlock("radio-t", "userX", false))
}

func TestBoltDB_BlockList(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	assert.NoError(t, b.SetBlock("radio-t", "user1", true))
	assert.NoError(t, b.SetBlock("radio-t", "user2", true))
	assert.NoError(t, b.SetBlock("radio-t", "user3", false))

	ids, err := b.Blocked("radio-t")
	assert.NoError(t, err)

	assert.Equal(t, 2, len(ids))
	assert.Equal(t, "user1", ids[0].ID)
	assert.Equal(t, "user2", ids[1].ID)
	t.Logf("%+v", ids)

	_, err = b.Blocked("bad")
	assert.EqualError(t, err, `site "bad" not found`)
}
