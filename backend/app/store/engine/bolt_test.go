package engine

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/remark42/backend/app/store"
)

var testDB = "/tmp/test-remark.db"

func TestBoltDB_CreateAndFind(t *testing.T) {
	var b, teardown = prep(t)
	defer teardown()

	var bb Interface = b
	_ = bb

	req := FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, Sort: "time"}
	res, err := b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Equal(t, `some text, <a href="http://radio-t.com">link</a>`, res[0].Text)
	assert.Equal(t, "user1", res[0].User.ID)
	t.Log(res[0].ID)

	_, err = b.Create(store.Comment{ID: res[0].ID, Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}})
	assert.Error(t, err)
	assert.Equal(t, "key id-1 already in store", err.Error())

	req = FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t-bad"}, Sort: "time"}
	_, err = b.Find(req)
	assert.EqualError(t, err, `site "radio-t-bad" not found`)

	assert.NoError(t, b.Close())
}

func TestBoltDB_CreateFailedReadOnly(t *testing.T) {
	var b, teardown = prep(t)
	defer teardown()

	comment := store.Comment{
		ID:        "id-ro",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com/ro", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}

	flagReq := FlagRequest{Locator: comment.Locator, Flag: ReadOnly, Update: FlagTrue}
	v, err := b.Flag(flagReq)
	require.NoError(t, err)
	assert.Equal(t, true, v)

	_, err = b.Create(comment)
	assert.Error(t, err)
	assert.Equal(t, "post https://radio-t.com/ro is read-only", err.Error())

	flagReq = FlagRequest{Locator: comment.Locator, Flag: ReadOnly, Update: FlagFalse}
	v, err = b.Flag(flagReq)
	require.NoError(t, err)
	assert.Equal(t, false, v)

	_, err = b.Create(comment)
	assert.NoError(t, err)
}

func TestBoltDB_Get(t *testing.T) {
	var b, teardown = prep(t)
	defer teardown()

	req := FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, Sort: "time"}
	res, err := b.Find(req)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(res), "2 records initially")

	comment, err := b.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[1].ID))
	assert.NoError(t, err)
	assert.Equal(t, "some text2", comment.Text)

	comment, err = b.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "1234567"))
	assert.Error(t, err)

	_, err = b.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "bad"}, res[1].ID))
	assert.EqualError(t, err, `site "bad" not found`)
}

func TestBoltDB_Update(t *testing.T) {
	var b, teardown = prep(t)
	defer teardown()

	req := FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, Sort: "time"}
	res, err := b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res), "2 records initially")

	comment := res[0]
	comment.Text = "abc 123"
	comment.Score = 100
	err = b.Update(comment)
	assert.NoError(t, err)

	comment, err = b.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[0].ID))
	assert.NoError(t, err)
	assert.Equal(t, "abc 123", comment.Text)
	assert.Equal(t, res[0].ID, comment.ID)
	assert.Equal(t, 100, comment.Score)

	comment.Locator.SiteID = "bad"
	err = b.Update(comment)
	assert.EqualError(t, err, `site "bad" not found`)

	comment.Locator.SiteID = "radio-t"
	comment.Locator.URL = "https://radio-t.com-bad"
	err = b.Update(comment)
	assert.EqualError(t, err, `no bucket https://radio-t.com-bad in store`)
}

func TestBoltDB_FindLast(t *testing.T) {
	var b, teardown = prep(t)
	defer teardown()

	req := FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time"}
	res, err := b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Equal(t, "some text2", res[0].Text)

	req.Limit = 1
	res, err = b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 1, len(res))
	assert.Equal(t, "some text2", res[0].Text)

	req.Locator.SiteID = "bad"
	_, err = b.Find(req)
	assert.EqualError(t, err, `site "bad" not found`)
}

func TestBoltDB_FindLastSince(t *testing.T) {
	var b, teardown = prep(t)
	defer teardown()

	ts := time.Date(2017, 12, 20, 15, 18, 21, 0, time.Local)
	req := FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time", Since: ts}
	res, err := b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Equal(t, "some text2", res[0].Text)

	req.Since = time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local)
	res, err = b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 1, len(res))
	assert.Equal(t, "some text2", res[0].Text)

	req.Since = time.Date(2017, 12, 20, 16, 18, 22, 0, time.Local)
	res, err = b.Find(req)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res))
}

func TestBoltDB_FindInPostSince(t *testing.T) {
	var b, teardown = prep(t)
	defer teardown()

	ts := time.Date(2017, 12, 20, 15, 18, 21, 0, time.Local)
	req := FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, Sort: "-time", Since: ts}
	res, err := b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Equal(t, "some text2", res[0].Text)

	req.Since = time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local)
	res, err = b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 1, len(res))
	assert.Equal(t, "some text2", res[0].Text)

	req.Since = time.Date(2017, 12, 20, 16, 18, 22, 0, time.Local)
	res, err = b.Find(req)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res))
}

func TestBoltDB_FindForUser(t *testing.T) {
	var b, teardown = prep(t)
	defer teardown()

	req := FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time", UserID: "user1", Limit: 5}
	res, err := b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Equal(t, "some text2", res[0].Text, "sorted by -time")

	req = FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time", UserID: "user1", Limit: 1}
	res, err = b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 1, len(res), "allow 1 comment")
	assert.Equal(t, "some text2", res[0].Text, "sorted by -time")

	req = FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time", UserID: "user1", Limit: 1, Skip: 1}
	res, err = b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 1, len(res), "allow 1 comment")
	assert.Equal(t, `some text, <a href="http://radio-t.com">link</a>`, res[0].Text, "second comment")

	req = FindRequest{Locator: store.Locator{SiteID: "bad"}, Sort: "-time", UserID: "user1", Limit: 1, Skip: 1}
	_, err = b.Find(req)
	assert.EqualError(t, err, `site "bad" not found`)

	req = FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time", UserID: "userZ", Limit: 1, Skip: 1}
	_, err = b.Find(req)
	assert.EqualError(t, err, `no comments for user userZ in store`)
}

func TestBoltDB_FindForUserPagination(t *testing.T) {
	_ = os.Remove(testDB)
	b, err := NewBoltDB(bolt.Options{}, BoltSite{FileName: testDB, SiteID: "radio-t"})
	require.NoError(t, err)

	defer func() {
		require.NoError(t, b.Close())
		_ = os.Remove(testDB)
	}()

	c := store.Comment{
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:    store.User{ID: "user1", Name: "user name"},
	}

	// write 200 comments
	for i := 0; i < 200; i++ {
		c.ID = fmt.Sprintf("id-%d", i)
		c.Text = fmt.Sprintf("text #%d", i)
		c.Timestamp = time.Date(2017, 12, 20, 15, 18, i, 0, time.Local)
		_, err = b.Create(c)
		require.NoError(t, err)
	}

	// get all comments
	req := FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time", UserID: "user1"}
	res, err := b.Find(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, len(res))
	assert.Equal(t, "id-199", res[0].ID)

	// seek 0, 5 comments
	req.Limit = 5
	res, err = b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 5, len(res))
	assert.Equal(t, "id-199", res[0].ID)
	assert.Equal(t, "id-195", res[4].ID)

	// seek 10, 3 comments
	req.Skip, req.Limit = 10, 3
	res, err = b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 3, len(res))
	assert.Equal(t, "id-189", res[0].ID)
	assert.Equal(t, "id-187", res[2].ID)

	// seek 195, ask 10 comments
	req.Skip, req.Limit = 195, 10
	res, err = b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 5, len(res))
	assert.Equal(t, "id-4", res[0].ID)
	assert.Equal(t, "id-0", res[4].ID)

	// seek 255, ask 10 comments
	req.Skip, req.Limit = 255, 10
	res, err = b.Find(req)
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res))
}

func TestBoltDB_CountPost(t *testing.T) {
	var b, teardown = prep(t)
	defer teardown()

	req := FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}}
	c, err := b.Count(req)
	assert.NoError(t, err)
	assert.Equal(t, 2, c)

	req = FindRequest{Locator: store.Locator{URL: "https://radio-t.com-xxx", SiteID: "radio-t"}}
	c, err = b.Count(req)
	assert.NoError(t, err)
	assert.Equal(t, 0, c)

	req = FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "bad"}}
	_, err = b.Count(req)
	assert.EqualError(t, err, `site "bad" not found`)
}

func TestBoltDB_CountUser(t *testing.T) {
	var b, teardown = prep(t)
	defer teardown()

	req := FindRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1"}
	c, err := b.Count(req)
	assert.NoError(t, err)
	assert.Equal(t, 2, c)

	req = FindRequest{Locator: store.Locator{SiteID: "bad"}, UserID: "user1"}
	_, err = b.Count(req)
	assert.EqualError(t, err, `site "bad" not found`)

	req = FindRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "userZ"}
	_, err = b.Count(req)
	assert.EqualError(t, err, `no comments for user userZ in store for radio-t site`)
}

func TestBoltDB_InfoPost(t *testing.T) {
	b, teardown := prep(t) // two comments for https://radio-t.com
	defer teardown()

	ts := func(min int) time.Time { return time.Date(2017, 12, 20, 15, 18, min, 0, time.Local) }

	// add one more for https://radio-t.com/2
	comment := store.Comment{
		ID:        "12345",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 24, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)

	req := InfoRequest{Locator: store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"}, ReadOnlyAge: 0}
	r, err := b.Info(req)
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(24), LastTS: ts(24)}}, r)

	req = InfoRequest{Locator: store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"}, ReadOnlyAge: 10}
	r, err = b.Info(req)
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(24), LastTS: ts(24),
		ReadOnly: true}}, r)

	req = InfoRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, ReadOnlyAge: 0}
	r, err = b.Info(req)
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com", Count: 2, FirstTS: ts(22), LastTS: ts(23)}}, r)

	req = InfoRequest{Locator: store.Locator{URL: "https://radio-t.com/error", SiteID: "radio-t"}, ReadOnlyAge: 0}
	_, err = b.Info(req)
	require.Error(t, err)

	req = InfoRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t-error"}, ReadOnlyAge: 0}
	_, err = b.Info(req)
	require.Error(t, err)

	fr := FlagRequest{Flag: ReadOnly, Locator: store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"}, Update: FlagTrue}
	_, err = b.Flag(fr)
	require.NoError(t, err)
	req = InfoRequest{Locator: store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"}, ReadOnlyAge: 0}
	r, err = b.Info(req)
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(24), LastTS: ts(24),
		ReadOnly: true}}, r)
}

func TestBoltDB_InfoList(t *testing.T) {
	b, teardown := prep(t) // two comments for https://radio-t.com
	defer teardown()

	// add one more for https://radio-t.com/2
	comment := store.Comment{
		ID:        "12345",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err := b.Create(comment)
	assert.NoError(t, err)

	ts := func(sec int) time.Time { return time.Date(2017, 12, 20, 15, 18, sec, 0, time.Local) }

	req := InfoRequest{Locator: store.Locator{SiteID: "radio-t"}}
	res, err := b.Info(req)
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(22), LastTS: ts(22)},
		{URL: "https://radio-t.com", Count: 2, FirstTS: ts(22), LastTS: ts(23)}}, res)

	req = InfoRequest{Locator: store.Locator{SiteID: "radio-t"}, Limit: -1, Skip: -1}
	res, err = b.Info(req)
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(22), LastTS: ts(22)},
		{URL: "https://radio-t.com", Count: 2, FirstTS: ts(22), LastTS: ts(23)}}, res)

	req = InfoRequest{Locator: store.Locator{SiteID: "radio-t"}, Limit: 1}
	res, err = b.Info(req)
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(22), LastTS: ts(22)}}, res)

	req = InfoRequest{Locator: store.Locator{SiteID: "radio-t"}, Limit: 1, Skip: 1}
	res, err = b.Info(req)
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com", Count: 2, FirstTS: ts(22), LastTS: ts(23)}}, res)

	req = InfoRequest{Locator: store.Locator{SiteID: "bad"}, Limit: 1, Skip: 1}
	_, err = b.Info(req)
	assert.EqualError(t, err, `site "bad" not found`)
}

func TestBolt_FlagBlockedUser(t *testing.T) {

	b, teardown := prep(t)
	defer teardown()

	req := FlagRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1"}
	val, err := b.Flag(req)
	assert.NoError(t, err)
	assert.False(t, val, "nothing blocked yet")

	req = FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", Update: FlagTrue}
	_, err = b.Flag(req)
	assert.NoError(t, err)
	val, err = b.Flag(FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1"})
	assert.NoError(t, err)
	assert.True(t, val, "user1 blocked")

	req = FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", Update: FlagTrue}
	_, err = b.Flag(req)
	assert.NoError(t, err)
	val, err = b.Flag(FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1"})
	assert.NoError(t, err)
	assert.True(t, val, "user1 still blocked")

	req = FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", Update: FlagFalse}
	_, err = b.Flag(req)
	assert.NoError(t, err)
	val, err = b.Flag(FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1"})
	assert.NoError(t, err)
	assert.False(t, val, "user1 unblocked")

	req = FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "bad"}, UserID: "user1", Update: FlagTrue}
	_, err = b.Flag(req)
	assert.EqualError(t, err, `site "bad" not found`)

	req = FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: "userX", Update: FlagTrue}
	_, err = b.Flag(req)
	assert.NoError(t, err, "non-existing user can't be blocked")

	req = FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "radio-t-bad"}, UserID: "user1"}
	val, err = b.Flag(req)
	assert.NoError(t, err)
	assert.False(t, val, "nothing blocked on wrong site")
}

func TestBolt_FlagReadOnlyPost(t *testing.T) {

	b, teardown := prep(t)
	defer teardown()

	req := FlagRequest{Locator: store.Locator{SiteID: "radio-t", URL: "url-1"}, Flag: ReadOnly}
	val, err := b.Flag(req)
	assert.NoError(t, err)
	assert.False(t, val, "nothing ro")

	req = FlagRequest{Locator: store.Locator{SiteID: "radio-t", URL: "url-1"}, Flag: ReadOnly, Update: FlagTrue}
	val, err = b.Flag(req)
	assert.NoError(t, err)
	assert.Equal(t, true, val)
	req = FlagRequest{Locator: store.Locator{SiteID: "radio-t", URL: "url-1"}, Flag: ReadOnly}
	val, err = b.Flag(req)
	assert.NoError(t, err)
	assert.True(t, val, "url-1 ro")

	req = FlagRequest{Locator: store.Locator{SiteID: "radio-t", URL: "url-2"}, Flag: ReadOnly}
	val, err = b.Flag(req)
	assert.NoError(t, err)
	assert.False(t, val, "url-2 still writable")

	req = FlagRequest{Locator: store.Locator{SiteID: "radio-t", URL: "url-1"}, Flag: ReadOnly, Update: FlagFalse}
	_, err = b.Flag(req)
	assert.NoError(t, err)
	req = FlagRequest{Locator: store.Locator{SiteID: "radio-t", URL: "url-1"}, Flag: ReadOnly}
	val, err = b.Flag(req)
	assert.NoError(t, err)
	assert.False(t, val, "url-1 writable")

	req = FlagRequest{Locator: store.Locator{SiteID: "bad", URL: "url-1"}, Flag: ReadOnly, Update: FlagFalse}
	_, err = b.Flag(req)
	assert.EqualError(t, err, `site "bad" not found`)

	req = FlagRequest{Locator: store.Locator{SiteID: "radio-t-bad", URL: "url-1"}, Flag: ReadOnly}
	val, err = b.Flag(req)
	assert.NoError(t, err)
	assert.False(t, val, "nothing ro on wrong site")
}

func TestBolt_FlagVerified(t *testing.T) {

	b, teardown := prep(t)
	defer teardown()

	isVerified := func(site, user string) bool {
		req := FlagRequest{Flag: Verified, Locator: store.Locator{SiteID: site}, UserID: user}
		v, err := b.Flag(req)
		require.NoError(t, err)
		return v
	}

	setVerified := func(site, user string, status FlagStatus) error {
		req := FlagRequest{Flag: Verified, Locator: store.Locator{SiteID: site}, UserID: user, Update: status}
		_, err := b.Flag(req)
		return err
	}

	assert.False(t, isVerified("radio-t", "u1"), "nothing verified")

	assert.NoError(t, setVerified("radio-t", "u1", FlagTrue))
	assert.True(t, isVerified("radio-t", "u1"), "u1 verified")

	assert.False(t, isVerified("radio-t", "u2"), "u2 still not verified")
	assert.NoError(t, setVerified("radio-t", "u1", FlagFalse))
	assert.False(t, isVerified("radio-t", "u1"), "u1 not verified anymore")

	assert.EqualError(t, setVerified("bad", "u1", FlagTrue), `site "bad" not found`)
	assert.NoError(t, setVerified("radio-t", "u1xyz", FlagFalse))

	assert.False(t, isVerified("radio-t-bad", "u1"), "nothing verified on wrong site")

	assert.NoError(t, setVerified("radio-t", "u1", FlagTrue))
	assert.NoError(t, setVerified("radio-t", "u2", FlagTrue))
	assert.NoError(t, setVerified("radio-t", "u3", FlagFalse))
}

func TestBolt_FlagListVerified(t *testing.T) {

	b, teardown := prep(t)
	defer teardown()

	toIDs := func(inp []interface{}) (res []string) {
		res = make([]string, len(inp))
		for i, v := range inp {
			vv, ok := v.(string)
			require.True(t, ok)
			res[i] = vv
		}
		return res
	}

	setVerified := func(site, user string, status FlagStatus) error {
		req := FlagRequest{Flag: Verified, Locator: store.Locator{SiteID: site}, UserID: user, Update: status}
		_, err := b.Flag(req)
		return err
	}

	ids, err := b.ListFlags(FlagRequest{Flag: Verified, Locator: store.Locator{SiteID: "radio-t"}})
	assert.NoError(t, err)
	assert.Equal(t, []string{}, toIDs(ids), "verified list empty")

	assert.NoError(t, setVerified("radio-t", "u1", FlagTrue))
	assert.NoError(t, setVerified("radio-t", "u2", FlagTrue))
	ids, err = b.ListFlags(FlagRequest{Flag: Verified, Locator: store.Locator{SiteID: "radio-t"}})
	assert.NoError(t, err)
	assert.Equal(t, []string{"u1", "u2"}, toIDs(ids), "verified 2 ids")

	_, err = b.ListFlags(FlagRequest{Flag: Verified, Locator: store.Locator{SiteID: "radio-t-bad"}})
	assert.Error(t, err, "site \"radio-t-bad\" not found", "fail on wrong site")
}

func TestBolt_FlagListBlocked(t *testing.T) {

	b, teardown := prep(t)
	defer teardown()

	setBlocked := func(site, user string, status FlagStatus, ttl time.Duration) error {
		req := FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: site}, UserID: user, Update: status, TTL: ttl}
		_, err := b.Flag(req)
		return err
	}

	toBlocked := func(inp []interface{}) (res []store.BlockedUser) {
		res = make([]store.BlockedUser, len(inp))
		for i, v := range inp {
			vv, ok := v.(store.BlockedUser)
			require.True(t, ok)
			res[i] = vv
		}
		return res
	}
	assert.NoError(t, setBlocked("radio-t", "user1", FlagTrue, 0))
	assert.NoError(t, setBlocked("radio-t", "user2", FlagTrue, 150*time.Millisecond))
	assert.NoError(t, setBlocked("radio-t", "user3", FlagFalse, 0))

	vv, err := b.ListFlags(FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "radio-t"}})
	assert.NoError(t, err)

	blockedList := toBlocked(vv)
	require.Equal(t, 2, len(blockedList))
	assert.Equal(t, "user1", blockedList[0].ID)
	assert.Equal(t, "user2", blockedList[1].ID)
	t.Logf("%+v", blockedList)

	// check block expiration
	time.Sleep(150 * time.Millisecond)
	vv, err = b.ListFlags(FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "radio-t"}})
	assert.NoError(t, err)
	blockedList = toBlocked(vv)
	require.Equal(t, 1, len(blockedList))
	assert.Equal(t, "user1", blockedList[0].ID)

	_, err = b.ListFlags(FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "bad"}})
	assert.EqualError(t, err, `site "bad" not found`)

}

func TestBoltDB_UserDetail(t *testing.T) {

	b, teardown := prep(t)
	defer teardown()

	// add two entries to DB before we start
	result, err := b.UserDetail(UserDetailRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "u1", Detail: UserEmail, Update: "test@example.com"})
	assert.NoError(t, err, "No error inserting entry expected")
	assert.ElementsMatch(t, []UserDetailEntry{{UserID: "u1", Email: "test@example.com"}}, result)
	result, err = b.UserDetail(UserDetailRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "u2", Detail: UserEmail, Update: "other@example.com"})
	assert.NoError(t, err, "No error inserting entry expected")
	assert.ElementsMatch(t, []UserDetailEntry{{UserID: "u2", Email: "other@example.com"}}, result)

	// stateless tests without changing the state we set up before
	var testData = []struct {
		req      UserDetailRequest
		error    string
		expected []UserDetailEntry
	}{
		{req: UserDetailRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "u1", Detail: UserEmail},
			expected: []UserDetailEntry{{UserID: "u1", Email: "test@example.com"}}},
		{req: UserDetailRequest{Locator: store.Locator{SiteID: "bad"}, UserID: "u1", Detail: UserEmail},
			error: `site "bad" not found`},
		{req: UserDetailRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "u1xyz", Detail: UserEmail}},
		{req: UserDetailRequest{Detail: UserEmail, Update: "new_value"},
			error: `userid cannot be empty in request for single detail`},
		{req: UserDetailRequest{Detail: UserDetail("bad")},
			error: `unsupported detail "bad"`},
		{req: UserDetailRequest{Update: "not_relevant", Detail: AllUserDetails},
			error: `unsupported request with userdetail all`},
		{req: UserDetailRequest{Locator: store.Locator{SiteID: "bad"}, Detail: AllUserDetails},
			error: `site "bad" not found`},
		{req: UserDetailRequest{Locator: store.Locator{SiteID: "radio-t"}, Detail: AllUserDetails},
			expected: []UserDetailEntry{{UserID: "u1", Email: "test@example.com"}, {UserID: "u2", Email: "other@example.com"}}},
	}

	for i, x := range testData {
		result, err := b.UserDetail(x.req)
		if x.error != "" {
			assert.EqualError(t, err, x.error, "Error should match expected for case %d", i)
		} else {
			assert.NoError(t, err, "Error is not expected expected for case %d", i)
		}
		assert.ElementsMatch(t, x.expected, result, "Result should match expected for case %d", i)
	}
}

func TestBolt_DeleteComment(t *testing.T) {

	b, teardown := prep(t)
	defer teardown()

	reqReq := FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, Sort: "time"}
	res, err := b.Find(reqReq)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(res), "initially 2 comments")

	count, err := b.Count(reqReq)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "count=2 initially")

	delReq := DeleteRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		CommentID: res[0].ID, DeleteMode: store.SoftDelete}

	err = b.Delete(delReq)
	assert.NoError(t, err)

	res, err = b.Find(reqReq)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Equal(t, "", res[0].Text)
	assert.True(t, res[0].Deleted, "marked deleted")
	assert.Equal(t, store.User{Name: "user name", ID: "user1", Picture: "", Admin: false, Blocked: false, IP: ""}, res[0].User)

	// repeated deletion should not decrease comments count
	err = b.Delete(delReq)
	assert.NoError(t, err)

	assert.Equal(t, "some text2", res[1].Text)
	assert.False(t, res[1].Deleted)

	comments, err := b.Find(FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Limit: 10})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(comments), "1 in last, 1 removed")

	count, err = b.Count(reqReq)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	delReq.CommentID = "123456"
	err = b.Delete(delReq)
	assert.Error(t, err)

	delReq.Locator.SiteID = "bad"
	delReq.CommentID = res[0].ID
	err = b.Delete(delReq)
	assert.EqualError(t, err, `site "bad" not found`)

	delReq.Locator = store.Locator{URL: "https://radio-t.com/bad", SiteID: "radio-t"}
	err = b.Delete(delReq)
	assert.EqualError(t, err, `no bucket https://radio-t.com/bad in store`)
}

func TestBolt_DeleteHard(t *testing.T) {

	b, teardown := prep(t)
	defer teardown()

	reqReq := FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, Sort: "time"}
	res, err := b.Find(reqReq)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res), "initially 2 comments")

	delReq := DeleteRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		CommentID: res[0].ID, DeleteMode: store.HardDelete}
	err = b.Delete(delReq)
	assert.NoError(t, err)

	res, err = b.Find(reqReq)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Equal(t, "", res[0].Text)
	assert.True(t, res[0].Deleted, "marked deleted")
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, res[0].User)
}

func TestBolt_DeleteAll(t *testing.T) {

	b, teardown := prep(t)
	defer teardown()

	delReq := DeleteRequest{Locator: store.Locator{SiteID: "radio-t"}}
	err := b.Delete(delReq)
	assert.NoError(t, err)

	comments, err := b.Find(FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Limit: 10})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(comments), "nothing left")

	delReq = DeleteRequest{Locator: store.Locator{SiteID: "bad"}}
	err = b.Delete(delReq)
	assert.EqualError(t, err, `site "bad" not found`)
}

func TestBolt_DeleteUserDetail(t *testing.T) {
	var (
		createUser = UserDetailRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", Detail: UserEmail, Update: "value1"}
		readUser   = UserDetailRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", Detail: UserEmail}
		emailSet   = []UserDetailEntry{{UserID: "user1", Email: "value1"}}
	)

	b, teardown := prep(t)
	defer teardown()

	var testData = []struct {
		delReq    DeleteRequest
		detailReq UserDetailRequest
		expected  []UserDetailEntry
		err       string
	}{
		{delReq: DeleteRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", UserDetail: UserEmail},
			detailReq: createUser, expected: emailSet},
		{delReq: DeleteRequest{Locator: store.Locator{SiteID: "bad"}, UserID: "user1", UserDetail: UserEmail},
			detailReq: readUser, expected: emailSet, err: `site "bad" not found`},
		{delReq: DeleteRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", UserDetail: UserEmail},
			detailReq: readUser},
		{delReq: DeleteRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", UserDetail: AllUserDetails},
			detailReq: createUser, expected: emailSet},
		{delReq: DeleteRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", UserDetail: AllUserDetails},
			detailReq: readUser},
	}

	for i, x := range testData {
		err := b.Delete(x.delReq)
		if x.err == "" {
			require.NoError(t, err, "delete request #%d error", i)
		} else {
			require.EqualError(t, err, x.err, "delete request #%d error", i)
		}

		val, err := b.UserDetail(x.detailReq)
		require.NoError(t, err, "user request #%d error", i)
		require.Equal(t, x.expected, val, "user request #%d result", i)
	}
}

func TestBoltAdmin_DeleteUserHard(t *testing.T) {

	b, teardown := prep(t)
	defer teardown()

	comments, err := b.Find(FindRequest{Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}, Sort: "time"})
	assert.NoError(t, err)

	// soft delete one comment
	delReq := DeleteRequest{
		Locator:    store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		CommentID:  comments[0].ID,
		DeleteMode: store.SoftDelete,
	}
	err = b.Delete(delReq)
	assert.NoError(t, err)

	err = b.Delete(DeleteRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", DeleteMode: store.HardDelete})
	require.NoError(t, err)

	comments, err = b.Find(FindRequest{Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}, Sort: "time"})
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments), "2 comments with deleted info")
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, comments[0].User)
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, comments[1].User)

	c, err := b.Count(FindRequest{Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}})
	assert.NoError(t, err)
	assert.Equal(t, 0, c, "0 count")

	_, err = b.Find(FindRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", Limit: 5})
	assert.EqualError(t, err, "no comments for user user1 in store")

	comments, err = b.Find(FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "time"})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(comments), "nothing left")

	err = b.Delete(DeleteRequest{Locator: store.Locator{SiteID: "radio-t-bad"}, UserID: "user1"})
	assert.EqualError(t, err, `site "radio-t-bad" not found`)
}

func TestBoltAdmin_DeleteUserSoft(t *testing.T) {

	b, teardown := prep(t)
	defer teardown()

	comments, err := b.Find(FindRequest{Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}, Sort: "time"})
	assert.NoError(t, err)

	// soft delete one comment
	delReq := DeleteRequest{
		Locator:    store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		CommentID:  comments[0].ID,
		DeleteMode: store.SoftDelete,
	}
	err = b.Delete(delReq)
	assert.NoError(t, err)

	err = b.Delete(DeleteRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", DeleteMode: store.SoftDelete})
	require.NoError(t, err)

	comments, err = b.Find(FindRequest{Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}, Sort: "time"})
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments), "2 comments with deleted info")
	assert.Equal(t, store.User{Name: "user name", ID: "user1", Picture: "", Admin: false, Blocked: false, IP: ""}, comments[0].User)
	assert.Equal(t, store.User{Name: "user name", ID: "user1", Picture: "", Admin: false, Blocked: false, IP: ""}, comments[1].User)

	c, err := b.Count(FindRequest{Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}})
	assert.NoError(t, err)
	assert.Equal(t, 0, c, "0 count")

	comments, err = b.Find(FindRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", Limit: 5})
	assert.NoError(t, err, "no comments for user user1 in store")
	require.Equal(t, 2, len(comments), "2 comments with deleted info")
	assert.True(t, comments[0].Deleted)
	assert.True(t, comments[1].Deleted)
	assert.Equal(t, "", comments[0].Text)
	assert.Equal(t, "", comments[1].Text)

	comments, err = b.Find(FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "time"})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(comments), "nothing left")

	err = b.Delete(DeleteRequest{Locator: store.Locator{SiteID: "radio-t-bad"}, UserID: "user1"})
	assert.EqualError(t, err, `site "radio-t-bad" not found`)
}

func TestBoltDB_ref(t *testing.T) {
	b := BoltDB{}
	comment := store.Comment{
		ID:        "12345",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	res := b.makeRef(comment)
	assert.Equal(t, "https://radio-t.com/2!!12345", string(res))

	url, id, err := b.parseRef([]byte("https://radio-t.com/2!!12345"))
	assert.NoError(t, err)
	assert.Equal(t, "https://radio-t.com/2", url)
	assert.Equal(t, "12345", id)

	_, _, err = b.parseRef([]byte("https://radio-t.com/2"))
	assert.Error(t, err)
}

func TestBoltDB_NewFailed(t *testing.T) {
	_, err := NewBoltDB(bolt.Options{}, BoltSite{FileName: "/tmp/no-such-place/tmp.db", SiteID: "radio-t"})
	assert.EqualError(t, err, "failed to make boltdb for /tmp/no-such-place/tmp.db: open /tmp/no-such-place/tmp.db: no such file or directory")
}

// makes new boltdb, put two records
func prep(t *testing.T) (b *BoltDB, teardown func()) {
	_ = os.Remove(testDB)

	boltStore, err := NewBoltDB(bolt.Options{}, BoltSite{FileName: testDB, SiteID: "radio-t"})
	assert.NoError(t, err)
	b = boltStore

	comment := store.Comment{
		ID:        "id-1",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = b.Create(comment)
	assert.NoError(t, err)

	comment = store.Comment{
		ID:        "id-2",
		Text:      "some text2",
		Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = b.Create(comment)
	assert.NoError(t, err)

	teardown = func() {
		require.NoError(t, b.Close())
		_ = os.Remove(testDB)
	}
	return b, teardown
}

func getReq(locator store.Locator, commentID string) GetRequest {
	return GetRequest{
		Locator:   locator,
		CommentID: commentID,
	}
}
