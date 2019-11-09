/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package accessor

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/engine"
)

func TestMemData_CreateAndFind(t *testing.T) {
	m := prepMem(t) // adds two comments

	req := engine.FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, Sort: "time"}
	res, err := m.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Equal(t, `some text, <a href="http://radio-t.com">link</a>`, res[0].Text)
	assert.Equal(t, "user1", res[0].User.ID)

	_, err = m.Create(store.Comment{ID: res[0].ID, Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}})
	require.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "dup key"), err.Error())

	id, err := m.Create(store.Comment{ID: "id-3", Locator: store.Locator{URL: "https://radio-t2.com", SiteID: "radio-t2"}})
	require.NoError(t, err)
	assert.Equal(t, "id-3", id)
	req = engine.FindRequest{Locator: store.Locator{URL: "https://radio-t2.com", SiteID: "radio-t2"}, Sort: "time"}
	res, err = m.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 1, len(res))
}

func TestMemData_CreateFailedReadOnly(t *testing.T) {
	b := prepMem(t)
	comment := store.Comment{
		ID:        "id-ro",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com/ro", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}

	flagReq := engine.FlagRequest{Locator: comment.Locator, Flag: engine.ReadOnly, Update: engine.FlagTrue}
	v, err := b.Flag(flagReq)
	require.NoError(t, err)
	assert.Equal(t, true, v)

	_, err = b.Create(comment)
	assert.NotNil(t, err)
	assert.Equal(t, "post https://radio-t.com/ro is read-only", err.Error())

	flagReq = engine.FlagRequest{Locator: comment.Locator, Flag: engine.ReadOnly, Update: engine.FlagFalse}
	v, err = b.Flag(flagReq)
	require.NoError(t, err)
	assert.Equal(t, false, v)

	_, err = b.Create(comment)
	assert.NoError(t, err)
}

func TestMemData_Get(t *testing.T) {
	b := prepMem(t)
	req := engine.FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, Sort: "time"}
	res, err := b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res), "2 records initially")

	comment, err := b.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, res[1].ID))
	assert.NoError(t, err)
	assert.Equal(t, "some text2", comment.Text)

	_, err = b.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, "1234567"))
	assert.EqualError(t, err, `not found`)

	_, err = b.Get(getReq(store.Locator{URL: "https://radio-t.com", SiteID: "bad"}, res[1].ID))
	assert.EqualError(t, err, `not found`)
}

func TestMemData_Update(t *testing.T) {
	b := prepMem(t)
	req := engine.FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, Sort: "time"}
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
	assert.EqualError(t, err, `not found`)

	comment.Locator.SiteID = "https://radio-t.com"
	comment.Locator.URL = "https://radio-t.com-bad"
	err = b.Update(comment)
	assert.EqualError(t, err, `not found`)
}

func TestMemData_FindLast(t *testing.T) {
	b := prepMem(t)
	req := engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time"}
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
	res, err = b.Find(req)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res))
}

func TestMemData_FindLastSince(t *testing.T) {
	b := prepMem(t)
	ts := time.Date(2017, 12, 20, 15, 18, 21, 0, time.Local)
	req := engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time", Since: ts}
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

func TestMemData_FindForUser(t *testing.T) {
	b := prepMem(t)
	req := engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time", UserID: "user1", Limit: 5}
	res, err := b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Equal(t, "some text2", res[0].Text, "sorted by -time")

	req = engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time", UserID: "user1", Limit: 1}
	res, err = b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 1, len(res), "allow 1 comment")
	assert.Equal(t, "some text2", res[0].Text, "sorted by -time")

	req = engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time", UserID: "user1", Limit: 1, Skip: 1}
	res, err = b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 1, len(res), "allow 1 comment")
	assert.Equal(t, `some text, <a href="http://radio-t.com">link</a>`, res[0].Text, "second comment")

	req = engine.FindRequest{Locator: store.Locator{SiteID: "bad"}, Sort: "-time", UserID: "user1", Limit: 1, Skip: 1}
	res, err = b.Find(req)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res), "no comments")

	req = engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time", UserID: "userZ", Limit: 1, Skip: 1}
	res, err = b.Find(req)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res), "no comments")
}

func TestMemData_FindForUserPagination(t *testing.T) {
	b := NewMemData()

	c := store.Comment{
		Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:    store.User{ID: "user1", Name: "user name"},
	}

	// write 200 comments
	for i := 0; i < 200; i++ {
		c.ID = fmt.Sprintf("idd-%d", i)
		c.Text = fmt.Sprintf("text #%d", i)
		c.Timestamp = time.Date(2017, 12, 20, 15, 18, i, 0, time.Local)
		_, err := b.Create(c)
		require.Nil(t, err)
	}

	// get all comments
	req := engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time", UserID: "user1"}
	res, err := b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 200, len(res))
	assert.Equal(t, "idd-199", res[0].ID)

	// seek 0, 5 comments
	req.Limit = 5
	res, err = b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 5, len(res))
	assert.Equal(t, "idd-199", res[0].ID)
	assert.Equal(t, "idd-195", res[4].ID)

	// seek 10, 3 comments
	req.Skip, req.Limit = 10, 3
	res, err = b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 3, len(res))
	assert.Equal(t, "idd-189", res[0].ID)
	assert.Equal(t, "idd-187", res[2].ID)

	// seek 195, ask 10 comments
	req.Skip, req.Limit = 195, 10
	res, err = b.Find(req)
	assert.NoError(t, err)
	require.Equal(t, 5, len(res))
	assert.Equal(t, "idd-4", res[0].ID)
	assert.Equal(t, "idd-0", res[4].ID)

	// seek 255, ask 10 comments
	req.Skip, req.Limit = 255, 10
	res, err = b.Find(req)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res))
}

func TestMemData_CountPost(t *testing.T) {
	b := prepMem(t)
	req := engine.FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}}
	c, err := b.Count(req)
	assert.NoError(t, err)
	require.Equal(t, 2, c)

	req = engine.FindRequest{Locator: store.Locator{URL: "https://radio-t.com-xxx", SiteID: "radio-t"}}
	c, err = b.Count(req)
	assert.NoError(t, err)
	assert.Equal(t, 0, c)

	req = engine.FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "bad"}}
	c, err = b.Count(req)
	assert.NoError(t, err)
	assert.Equal(t, 0, c)

	c, err = b.Count(engine.FindRequest{})
	assert.Error(t, err)
	assert.Equal(t, 0, c)
}

func TestMemData_CountUser(t *testing.T) {
	b := prepMem(t)
	req := engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1"}
	c, err := b.Count(req)
	assert.NoError(t, err)
	require.Equal(t, 2, c)

	req = engine.FindRequest{Locator: store.Locator{SiteID: "bad"}, UserID: "user1"}
	c, err = b.Count(req)
	assert.NoError(t, err)
	assert.Equal(t, 0, c)

	req = engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "userZ"}
	c, err = b.Count(req)
	assert.NoError(t, err)
	assert.Equal(t, 0, c)
}

func TestMemData_InfoPost(t *testing.T) {
	b := prepMem(t)
	ts := func(min int) time.Time { return time.Date(2017, 12, 20, 15, 18, min, 0, time.Local).In(time.UTC) }

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

	req := engine.InfoRequest{Locator: store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"}, ReadOnlyAge: 0}
	r, err := b.Info(req)
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(24), LastTS: ts(24)}}, r)

	req = engine.InfoRequest{Locator: store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"}, ReadOnlyAge: 10}
	r, err = b.Info(req)
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1,
		FirstTS: ts(24), LastTS: ts(24), ReadOnly: true}}, r)

	req = engine.InfoRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, ReadOnlyAge: 0}
	r, err = b.Info(req)
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com", Count: 2, FirstTS: ts(22), LastTS: ts(23)}}, r)

	req = engine.InfoRequest{Locator: store.Locator{URL: "https://radio-t.com/error", SiteID: "radio-t"}, ReadOnlyAge: 0}
	_, err = b.Info(req)
	require.NotNil(t, err)

	req = engine.InfoRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t-error"}, ReadOnlyAge: 0}
	_, err = b.Info(req)
	require.NotNil(t, err)

	_, err = b.Info(engine.InfoRequest{})
	require.Error(t, err)

	fr := engine.FlagRequest{Flag: engine.ReadOnly,
		Locator: store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"}, Update: engine.FlagTrue}
	_, err = b.Flag(fr)
	require.NoError(t, err)
	req = engine.InfoRequest{Locator: store.Locator{URL: "https://radio-t.com/2", SiteID: "radio-t"}, ReadOnlyAge: 0}
	r, err = b.Info(req)
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(24), LastTS: ts(24),
		ReadOnly: true}}, r)
}

func TestMemData_InfoList(t *testing.T) {
	b := prepMem(t)
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

	ts := func(sec int) time.Time { return time.Date(2017, 12, 20, 15, 18, sec, 0, time.Local).In(time.UTC) }

	req := engine.InfoRequest{Locator: store.Locator{SiteID: "radio-t"}}
	res, err := b.Info(req)
	assert.NoError(t, err)
	assert.EqualValues(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(22), LastTS: ts(22)},
		{URL: "https://radio-t.com", Count: 2, FirstTS: ts(22), LastTS: ts(23)}}, res)

	req = engine.InfoRequest{Locator: store.Locator{SiteID: "radio-t"}, Limit: -1, Skip: -1}
	res, err = b.Info(req)
	assert.NoError(t, err)
	assert.EqualValues(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(22), LastTS: ts(22)},
		{URL: "https://radio-t.com", Count: 2, FirstTS: ts(22), LastTS: ts(23)}}, res)

	req = engine.InfoRequest{Locator: store.Locator{SiteID: "radio-t"}, Limit: 1}
	res, err = b.Info(req)
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(22), LastTS: ts(22)}}, res)

	req = engine.InfoRequest{Locator: store.Locator{SiteID: "radio-t"}, Limit: 1, Skip: 1}
	res, err = b.Info(req)
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com", Count: 2, FirstTS: ts(22), LastTS: ts(23)}}, res)

	req = engine.InfoRequest{Locator: store.Locator{SiteID: "bad"}, Limit: 1, Skip: 1}
	res, err = b.Info(req)
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{}, res)
}

func TestMemData_FlagBlockedUser(t *testing.T) {

	b := prepMem(t)
	req := engine.FlagRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1"}
	val, err := b.Flag(req)
	assert.NoError(t, err)
	assert.False(t, val, "nothing blocked yet")

	req = engine.FlagRequest{Flag: engine.Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1",
		Update: engine.FlagTrue}
	_, err = b.Flag(req)
	assert.NoError(t, err)
	val, err = b.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1"})
	assert.NoError(t, err)
	assert.True(t, val, "user1 blocked")

	req = engine.FlagRequest{Flag: engine.Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1",
		Update: engine.FlagTrue}
	_, err = b.Flag(req)
	assert.NoError(t, err)
	val, err = b.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1"})
	assert.NoError(t, err)
	assert.True(t, val, "user1 still blocked")

	req = engine.FlagRequest{Flag: engine.Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1",
		Update: engine.FlagFalse}
	_, err = b.Flag(req)
	assert.NoError(t, err)
	val, err = b.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1"})
	assert.NoError(t, err)
	assert.False(t, val, "user1 unblocked")
}

func TestMemData_FlagReadOnlyPost(t *testing.T) {

	b := prepMem(t)
	req := engine.FlagRequest{Locator: store.Locator{SiteID: "radio-t", URL: "url-1"}, Flag: engine.ReadOnly}
	val, err := b.Flag(req)
	assert.NoError(t, err)
	assert.False(t, val, "nothing ro")

	req = engine.FlagRequest{Locator: store.Locator{SiteID: "radio-t", URL: "url-1"}, Flag: engine.ReadOnly,
		Update: engine.FlagTrue}
	val, err = b.Flag(req)
	assert.NoError(t, err)
	req = engine.FlagRequest{Locator: store.Locator{SiteID: "radio-t", URL: "url-1"}, Flag: engine.ReadOnly}
	val, err = b.Flag(req)
	assert.NoError(t, err)
	assert.True(t, val, "url-1 ro")

	req = engine.FlagRequest{Locator: store.Locator{SiteID: "radio-t", URL: "url-2"}, Flag: engine.ReadOnly}
	val, err = b.Flag(req)
	assert.NoError(t, err)
	assert.False(t, val, "url-2 still writable")

	req = engine.FlagRequest{Locator: store.Locator{SiteID: "radio-t", URL: "url-1"}, Flag: engine.ReadOnly,
		Update: engine.FlagFalse}
	_, err = b.Flag(req)
	assert.NoError(t, err)
	req = engine.FlagRequest{Locator: store.Locator{SiteID: "radio-t", URL: "url-1"}, Flag: engine.ReadOnly}
	val, err = b.Flag(req)
	assert.NoError(t, err)
	assert.False(t, val, "url-1 writable")
}

func TestMemData_FlagVerified(t *testing.T) {

	b := prepMem(t)
	isVerified := func(site, user string) bool {
		req := engine.FlagRequest{Flag: engine.Verified, Locator: store.Locator{SiteID: site}, UserID: user}
		v, err := b.Flag(req)
		require.NoError(t, err)
		return v
	}

	setVerified := func(site, user string, status engine.FlagStatus) error {
		req := engine.FlagRequest{Flag: engine.Verified, Locator: store.Locator{SiteID: site}, UserID: user, Update: status}
		_, err := b.Flag(req)
		return err
	}

	assert.False(t, isVerified("radio-t", "u1"), "nothing verified")

	assert.NoError(t, setVerified("radio-t", "u1", engine.FlagTrue))
	assert.True(t, isVerified("radio-t", "u1"), "u1 verified")

	assert.False(t, isVerified("radio-t", "u2"), "u2 still not verified")
	assert.NoError(t, setVerified("radio-t", "u1", engine.FlagFalse))
	assert.False(t, isVerified("radio-t", "u1"), "u1 not verified anymore")

	assert.NoError(t, setVerified("bad", "u1", engine.FlagTrue))
	assert.NoError(t, setVerified("radio-t", "u1xyz", engine.FlagFalse))

	assert.False(t, isVerified("radio-t-bad", "u1"), "nothing verified on wrong site")

	assert.NoError(t, setVerified("radio-t", "u1", engine.FlagTrue))
	assert.NoError(t, setVerified("radio-t", "u2", engine.FlagTrue))
	assert.NoError(t, setVerified("radio-t", "u3", engine.FlagFalse))
}

func TestMemData_FlagListVerified(t *testing.T) {

	b := prepMem(t)
	toIDs := func(inp []interface{}) (res []string) {
		res = make([]string, len(inp))
		for i, v := range inp {
			vv, ok := v.(string)
			require.True(t, ok)
			res[i] = vv
		}
		sort.Strings(res)
		return res
	}

	setVerified := func(site, user string, status engine.FlagStatus) error {
		req := engine.FlagRequest{Flag: engine.Verified, Locator: store.Locator{SiteID: site}, UserID: user, Update: status}
		_, err := b.Flag(req)
		return err
	}

	ids, err := b.ListFlags(engine.FlagRequest{Flag: engine.Verified, Locator: store.Locator{SiteID: "radio-t"}})
	assert.NoError(t, err)
	assert.Equal(t, []string{}, toIDs(ids), "verified list empty")

	assert.NoError(t, setVerified("radio-t", "u1", engine.FlagTrue))
	assert.NoError(t, setVerified("radio-t", "u2", engine.FlagTrue))
	ids, err = b.ListFlags(engine.FlagRequest{Flag: engine.Verified, Locator: store.Locator{SiteID: "radio-t"}})
	assert.NoError(t, err)
	assert.EqualValues(t, []string{"u1", "u2"}, toIDs(ids), "verified 2 ids")

	ids, err = b.ListFlags(engine.FlagRequest{Flag: engine.Verified, Locator: store.Locator{SiteID: "radio-t-bad"}})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(ids))

	ids, err = b.ListFlags(engine.FlagRequest{})
	assert.Error(t, err)
	assert.Equal(t, 0, len(ids))
}

func TestMemData_FlagListBlocked(t *testing.T) {

	b := prepMem(t)
	setBlocked := func(site, user string, status engine.FlagStatus, ttl time.Duration) error {
		req := engine.FlagRequest{Flag: engine.Blocked, Locator: store.Locator{SiteID: site}, UserID: user, Update: status,
			TTL: ttl}
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
	assert.NoError(t, setBlocked("radio-t", "user1", engine.FlagTrue, 0))
	assert.NoError(t, setBlocked("radio-t", "user2", engine.FlagTrue, 50*time.Millisecond))
	assert.NoError(t, setBlocked("radio-t", "user3", engine.FlagFalse, 0))

	vv, err := b.ListFlags(engine.FlagRequest{Flag: engine.Blocked, Locator: store.Locator{SiteID: "radio-t"}})
	assert.NoError(t, err)

	blockedList := toBlocked(vv)
	require.Equal(t, 2, len(blockedList), b.metaUsers)
	assert.Equal(t, "user1", blockedList[0].ID)
	assert.Equal(t, "user2", blockedList[1].ID)
	t.Logf("%+v", blockedList)

	// check block expiration
	time.Sleep(50 * time.Millisecond)
	vv, err = b.ListFlags(engine.FlagRequest{Flag: engine.Blocked, Locator: store.Locator{SiteID: "radio-t"}})
	assert.NoError(t, err)
	blockedList = toBlocked(vv)
	require.Equal(t, 1, len(blockedList))
	assert.Equal(t, "user1", blockedList[0].ID)

	vv, err = b.ListFlags(engine.FlagRequest{Flag: engine.Blocked, Locator: store.Locator{SiteID: "bad"}})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(vv))
}

func TestMemData_UserDetail(t *testing.T) {

	b := prepMem(t)

	var testData = []struct {
		site     string
		user     string
		update   string
		delete   bool
		expected string
		error    string
		detail   engine.UserDetail
	}{
		{site: "radio-t", user: "u1"},
		{site: "radio-t", user: "u1", update: "value1", expected: "value1"},
		{site: "radio-t", user: "u1", expected: "value1"},
		{site: "bad", user: "u1", update: "value1", expected: ""},
		{site: "bad", user: "u1", expected: ""},
		{site: "radio-t", user: "u1", delete: true},
		{site: "radio-t", user: "u1"},
		{site: "radio-t", user: "u1xyz", delete: true},
		{site: "radio-t", user: "u1", update: "value3", expected: "value3"},
		{site: "radio-t", user: "u2", update: "value4", delete: true, error: `both delete and update fields are set, pick one`},
		{update: "new_value", error: `userid cannot be empty`},
		{site: "radio-t", error: `userid cannot be empty`},
		{site: "radio-t", user: "u1", delete: true, detail: "bad", error: `unsupported detail bad`},
		{site: "radio-t", user: "u1", detail: "bad", error: `unsupported detail bad`},
	}

	for i, x := range testData {
		if x.detail == engine.UserDetail("") {
			x.detail = engine.Email
		}
		req := engine.UserDetailRequest{
			Detail:  x.detail,
			Locator: store.Locator{SiteID: x.site},
			UserID:  x.user,
			Update:  x.update,
			Delete:  x.delete}
		result, err := b.UserDetail(req)
		if x.error != "" {
			assert.EqualError(t, err, x.error, i)
		} else {
			assert.NoError(t, err, i)
		}
		assert.Equal(t, x.expected, result, i)
	}
}

func TestMemData_ListDetails(t *testing.T) {

	b := prepMem(t)

	req := engine.UserDetailRequest{
		Detail:  engine.Email,
		Locator: store.Locator{SiteID: "radio-t"},
		UserID:  "u1",
		Update:  "test@example.com"}
	_, err := b.UserDetail(req)
	assert.NoError(t, err)
	req.UserID = "u2"
	req.Update = "other@example.com"
	_, err = b.UserDetail(req)
	assert.NoError(t, err)

	var testData = []struct {
		site     string
		expected map[string]engine.UserDetailEntry
		error    string
	}{
		{site: "radio-t", expected: map[string]engine.UserDetailEntry{"u1": {Email: "test@example.com"}, "u2": {Email: "other@example.com"}}},
		{site: "bad", expected: map[string]engine.UserDetailEntry{}},
	}

	for i, x := range testData {
		result, err := b.ListDetails(store.Locator{SiteID: x.site})
		if x.error != "" {
			assert.EqualError(t, err, x.error, i)
		} else {
			assert.NoError(t, err, i)
		}
		assert.Equal(t, x.expected, result, i)
	}
}

func TestMemData_DeleteComment(t *testing.T) {

	b := prepMem(t)
	reqReq := engine.FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, Sort: "time"}
	res, err := b.Find(reqReq)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res), "initially 2 comments")

	count, err := b.Count(reqReq)
	require.NoError(t, err)
	require.Equal(t, 2, count, "count=2 initially")

	delReq := engine.DeleteRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		CommentID: res[0].ID, DeleteMode: store.SoftDelete}

	err = b.Delete(delReq)
	assert.NoError(t, err)

	res, err = b.Find(reqReq)
	assert.NoError(t, err)
	require.Equal(t, 2, len(res))
	assert.Equal(t, "", res[0].Text)
	assert.True(t, res[0].Deleted, "marked deleted")
	assert.Equal(t, store.User{Name: "user name", ID: "user1", Picture: "", Admin: false, Blocked: false, IP: ""}, res[0].User)

	assert.Equal(t, "some text2", res[1].Text)
	assert.False(t, res[1].Deleted)

	comments, err := b.Find(engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Limit: 10})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(comments), "1 in last, 1 removed")

	count, err = b.Count(reqReq)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	delReq.CommentID = "123456"
	err = b.Delete(delReq)
	assert.NotNil(t, err)

	delReq.Locator.SiteID = "bad"
	delReq.CommentID = res[0].ID
	err = b.Delete(delReq)
	assert.EqualError(t, err, `not found`)

	delReq.Locator = store.Locator{URL: "https://radio-t.com/bad", SiteID: "radio-t"}
	err = b.Delete(delReq)
	assert.EqualError(t, err, `not found`)

	err = b.Delete(engine.DeleteRequest{Locator: store.Locator{SiteID: "bad"}})
	assert.Error(t, err)
}

func TestMemData_Close(t *testing.T) {
	b := prepMem(t)
	assert.NoError(t, b.Close())
}

func TestMemData_DeleteHard(t *testing.T) {

	b := prepMem(t)
	reqReq := engine.FindRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}, Sort: "time"}
	res, err := b.Find(reqReq)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(res), "initially 2 comments")

	delReq := engine.DeleteRequest{Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
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

func TestMemData_DeleteAll(t *testing.T) {
	b := prepMem(t)
	delReq := engine.DeleteRequest{Locator: store.Locator{SiteID: "radio-t"}}
	err := b.Delete(delReq)
	assert.NoError(t, err)

	comments, err := b.Find(engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Limit: 10})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(comments), "nothing left")
}

func TestMemAdmin_DeleteUserHard(t *testing.T) {
	b := prepMem(t)
	err := b.Delete(engine.DeleteRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1",
		DeleteMode: store.HardDelete})
	require.NoError(t, err)

	comments, err := b.Find(engine.FindRequest{Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"},
		Sort: "time"})
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments), "2 comments with deleted info")
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, comments[0].User)
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, comments[1].User)

	c, err := b.Count(engine.FindRequest{Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}})
	assert.NoError(t, err)
	assert.Equal(t, 0, c, "0 count")

	_, err = b.Find(engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", Limit: 5})
	assert.NoError(t, err, "no comments for user user1 in store")

	comments, err = b.Find(engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "time"})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(comments), "nothing left")
}

func TestMemAdmin_DeleteUserSoft(t *testing.T) {

	b := prepMem(t)
	err := b.Delete(engine.DeleteRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1",
		DeleteMode: store.SoftDelete})
	require.NoError(t, err)

	comments, err := b.Find(engine.FindRequest{Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"},
		Sort: "time"})
	assert.NoError(t, err)
	require.Equal(t, 2, len(comments), "2 comments with deleted info")
	assert.Equal(t, store.User{Name: "user name", ID: "user1", Picture: "", Admin: false, Blocked: false, IP: ""}, comments[0].User)
	assert.Equal(t, store.User{Name: "user name", ID: "user1", Picture: "", Admin: false, Blocked: false, IP: ""}, comments[1].User)

	c, err := b.Count(engine.FindRequest{Locator: store.Locator{SiteID: "radio-t", URL: "https://radio-t.com"}})
	assert.NoError(t, err)
	assert.Equal(t, 0, c, "0 count")

	comments, err = b.Find(engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", Limit: 5})
	assert.NoError(t, err, "no comments for user user1 in store")
	require.Equal(t, 2, len(comments), "2 comments with deleted info")
	assert.True(t, comments[0].Deleted)
	assert.True(t, comments[1].Deleted)
	assert.Equal(t, "", comments[0].Text)
	assert.Equal(t, "", comments[1].Text)

	comments, err = b.Find(engine.FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "time"})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(comments), "nothing left")
}

func prepMem(t *testing.T) *MemData {

	m := NewMemData()

	comment := store.Comment{
		ID:        "id-1",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err := m.Create(comment)
	require.NoError(t, err)

	comment = store.Comment{
		ID:        "id-2",
		Text:      "some text2",
		Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.Local),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = m.Create(comment)
	require.NoError(t, err)
	return m
}

func getReq(locator store.Locator, commentID string) engine.GetRequest {
	return engine.GetRequest{
		Locator:   locator,
		CommentID: commentID,
	}
}
