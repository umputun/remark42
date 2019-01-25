package engine

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store"
)

func TestBoltAdmin_Delete(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}
	res, err := b.Find(loc, "time")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(res), "initially 2 comments")

	count, err := b.Count(loc)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "count=2 initially")

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

	count, err = b.Count(loc)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	err = b.Delete(loc, "123456", store.SoftDelete)
	assert.NotNil(t, err)

	loc.SiteID = "bad"
	err = b.Delete(loc, res[0].ID, store.SoftDelete)
	assert.EqualError(t, err, `site "bad" not found`)

	loc = store.Locator{URL: "https://radio-t.com/bad", SiteID: "radio-t"}
	err = b.Delete(loc, res[0].ID, store.SoftDelete)
	assert.EqualError(t, err, `no bucket https://radio-t.com/bad in store`)
}

func TestBoltAdmin_DeleteHard(t *testing.T) {
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

func TestBoltAdmin_DeleteAll(t *testing.T) {
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

func TestBoltAdmin_DeleteUser(t *testing.T) {
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

	_, err = b.User("radio-t", "user1", 5, 0)
	assert.EqualError(t, err, "no comments for user user1 in store")

	comments, err := b.Last("radio-t", 10)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(comments), "nothing left")

	err = b.DeleteUser("radio-t-bad", "user1")
	assert.EqualError(t, err, `site "radio-t-bad" not found`)
}

func TestBoltAdmin_BlockUser(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	assert.False(t, b.IsBlocked("radio-t", "user1"), "nothing blocked")

	assert.NoError(t, b.SetBlock("radio-t", "user1", true, 0))
	assert.True(t, b.IsBlocked("radio-t", "user1"), "user1 blocked")

	assert.False(t, b.IsBlocked("radio-t", "user2"), "user2 still unblocked")

	assert.NoError(t, b.SetBlock("radio-t", "user1", false, 0))
	assert.False(t, b.IsBlocked("radio-t", "user1"), "user1 unblocked")

	assert.EqualError(t, b.SetBlock("bad", "user1", true, 0), `site "bad" not found`)
	assert.NoError(t, b.SetBlock("radio-t", "userX", false, 0))

	assert.False(t, b.IsBlocked("radio-t-bad", "user1"), "nothing blocked on wrong site")
}

func TestBoltAdmin_BlockUserWithTTL(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)
	assert.False(t, b.IsBlocked("radio-t", "user1"), "nothing blocked")
	assert.NoError(t, b.SetBlock("radio-t", "user1", true, 50*time.Millisecond))
	assert.True(t, b.IsBlocked("radio-t", "user1"), "user1 blocked")
	time.Sleep(50 * time.Millisecond)
	assert.False(t, b.IsBlocked("radio-t", "user1"), "user1 un-blocked automatically")
}

func TestBoltAdmin_BlockList(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	assert.NoError(t, b.SetBlock("radio-t", "user1", true, 0))
	assert.NoError(t, b.SetBlock("radio-t", "user2", true, 50*time.Millisecond))
	assert.NoError(t, b.SetBlock("radio-t", "user3", false, 0))

	ids, err := b.Blocked("radio-t")
	assert.NoError(t, err)

	assert.Equal(t, 2, len(ids))
	assert.Equal(t, "user1", ids[0].ID)
	assert.Equal(t, "user2", ids[1].ID)
	t.Logf("%+v", ids)

	time.Sleep(50 * time.Millisecond)
	ids, err = b.Blocked("radio-t")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ids))
	assert.Equal(t, "user1", ids[0].ID)

	_, err = b.Blocked("bad")
	assert.EqualError(t, err, `site "bad" not found`)
}

func TestBoltAdmin_ReadOnly(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	assert.False(t, b.IsReadOnly(store.Locator{SiteID: "radio-t", URL: "url-1"}), "nothing ro")

	assert.NoError(t, b.SetReadOnly(store.Locator{SiteID: "radio-t", URL: "url-1"}, true))
	assert.True(t, b.IsReadOnly(store.Locator{SiteID: "radio-t", URL: "url-1"}), "url-1 ro")

	assert.False(t, b.IsReadOnly(store.Locator{SiteID: "radio-t", URL: "url-2"}), "url-2 still writable")

	assert.NoError(t, b.SetReadOnly(store.Locator{SiteID: "radio-t", URL: "url-1"}, false))
	assert.False(t, b.IsReadOnly(store.Locator{SiteID: "radio-t", URL: "url-1"}), "url-1 writable")

	assert.EqualError(t, b.SetReadOnly(store.Locator{SiteID: "bad", URL: "url-1"}, true), `site "bad" not found`)
	assert.NoError(t, b.SetReadOnly(store.Locator{SiteID: "radio-t", URL: "url-1xyz"}, false))

	assert.False(t, b.IsReadOnly(store.Locator{SiteID: "radio-t-bad", URL: "url-1"}), "nothing blocked on wrong site")
}

func TestBoltAdmin_Verified(t *testing.T) {
	defer os.Remove(testDb)
	b := prep(t)

	assert.False(t, b.IsVerified("radio-t", "u1"), "nothing verified")

	assert.NoError(t, b.SetVerified("radio-t", "u1", true))
	assert.True(t, b.IsVerified("radio-t", "u1"), "u1 verified")

	assert.False(t, b.IsVerified("radio-t", "u2"), "u2 still not verified")
	assert.NoError(t, b.SetVerified("radio-t", "u1", false))
	assert.False(t, b.IsVerified("radio-t", "u1"), "u1 not verified anymore")

	assert.EqualError(t, b.SetVerified("bad", "u1", true), `site "bad" not found`)
	assert.NoError(t, b.SetVerified("radio-t", "u1xyz", false))

	assert.False(t, b.IsVerified("radio-t-bad", "u1"), "nothing verified on wrong site")

	assert.NoError(t, b.SetVerified("radio-t", "u1", true))
	assert.NoError(t, b.SetVerified("radio-t", "u2", true))
	assert.NoError(t, b.SetVerified("radio-t", "u3", false))

	ids, err := b.Verified("radio-t")
	assert.NoError(t, err)
	assert.Equal(t, []string{"u1", "u2"}, ids, "verified 2 ids")

	_, err = b.Verified("radio-t-bad")
	assert.Error(t, err, "site \"radio-t-bad\" not found", "fail on wrong site")
}
