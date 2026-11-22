package enginetest

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
)

func (s suite) createAndFind(t *testing.T) {
	e := s.prep(t)

	res, err := e.Find(engine.FindRequest{Locator: loc(postURL), Sort: "time"})
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, `some text, <a href="http://radio-t.com">link</a>`, res[0].Text)
	assert.Equal(t, "user1", res[0].User.ID)

	dup := res[0]
	dup.Text = "another text"
	_, err = e.Create(dup)
	assert.Error(t, err, "a comment id is stored once")
	stored, err := e.Get(getReq(loc(postURL), res[0].ID))
	require.NoError(t, err)
	assert.Equal(t, res[0], stored, "the stored comment is unchanged")

	other := store.Locator{URL: "https://radio-t2.com", SiteID: OtherSiteID}
	id, err := e.Create(store.Comment{ID: "id-3", Locator: other, Timestamp: ts(24), User: store.User{ID: "user1", Name: "user name"}})
	require.NoError(t, err)
	assert.Equal(t, "id-3", id)
	res, err = e.Find(engine.FindRequest{Locator: other, Sort: "time"})
	require.NoError(t, err)
	assert.Equal(t, []string{"id-3"}, commentIDs(res))
}

func (s suite) createFailedReadOnly(t *testing.T) {
	e := s.prep(t)
	comment := store.Comment{
		ID:        "id-ro",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: ts(22),
		Locator:   loc("https://radio-t.com/ro"),
		User:      store.User{ID: "user1", Name: "user name"},
	}

	v, err := e.Flag(engine.FlagRequest{Locator: comment.Locator, Flag: engine.ReadOnly, Update: engine.FlagTrue})
	require.NoError(t, err)
	assert.True(t, v)

	_, err = e.Create(comment)
	require.Error(t, err)
	assert.Equal(t, "post https://radio-t.com/ro is read-only", err.Error())

	imported := comment
	imported.ID, imported.Imported = "id-imported", true
	_, err = e.Create(imported)
	require.NoError(t, err, "an import writes to a read-only post")
	res, err := e.Find(engine.FindRequest{Locator: comment.Locator})
	require.NoError(t, err)
	assert.Equal(t, []string{"id-imported"}, commentIDs(res))

	v, err = e.Flag(engine.FlagRequest{Locator: comment.Locator, Flag: engine.ReadOnly, Update: engine.FlagFalse})
	require.NoError(t, err)
	assert.False(t, v)

	_, err = e.Create(comment)
	assert.NoError(t, err)
}

func (s suite) get(t *testing.T) {
	e := s.prep(t)

	res, err := e.Find(engine.FindRequest{Locator: loc(postURL), Sort: "time"})
	require.NoError(t, err)
	require.Len(t, res, 2, "2 records initially")

	comment, err := e.Get(getReq(loc(postURL), res[1].ID))
	require.NoError(t, err)
	assert.Equal(t, "some text2", comment.Text)

	_, err = e.Get(getReq(loc(postURL), "1234567"))
	assert.Error(t, err)
}

func (s suite) update(t *testing.T) {
	e := s.prep(t)

	res, err := e.Find(engine.FindRequest{Locator: loc(postURL), Sort: "time"})
	require.NoError(t, err)
	require.Len(t, res, 2, "2 records initially")

	comment := res[0]
	comment.Text = "abc 123"
	comment.Score = 100
	require.NoError(t, e.Update(comment))

	comment, err = e.Get(getReq(loc(postURL), res[0].ID))
	require.NoError(t, err)
	assert.Equal(t, "abc 123", comment.Text)
	assert.Equal(t, res[0].ID, comment.ID)
	assert.Equal(t, 100, comment.Score)

	comment.Locator.URL = "https://radio-t.com-bad"
	assert.Error(t, e.Update(comment), "no such post")
}

func (s suite) updateKeepsFields(t *testing.T) {
	e := s.prep(t)
	orig, err := e.Get(getReq(loc(postURL), "id-1"))
	require.NoError(t, err)

	upd := orig
	upd.Text = "edited"
	upd.Edit = &store.Edit{Timestamp: time.Date(2017, 12, 21, 15, 18, 22, 0, time.UTC), Summary: "typo"}
	upd.Controversy = 1.5
	upd.VotedIPs = map[string]store.VotedIPInfo{"ip-hash": {Timestamp: time.Date(2017, 12, 21, 15, 18, 22, 0, time.UTC), Value: true}}
	upd.PostTitle = "new title"
	upd.ParentID = "other-parent"
	upd.Timestamp = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	upd.User = store.User{ID: "other-user", Name: "other"}
	require.NoError(t, e.Update(upd))

	res, err := e.Get(getReq(loc(postURL), "id-1"))
	require.NoError(t, err)
	assert.Equal(t, "edited", res.Text)
	assert.Equal(t, upd.Edit, res.Edit, "edit mark is stored")
	assert.InDelta(t, 1.5, res.Controversy, 0.001, "controversy is stored")
	assert.Equal(t, upd.VotedIPs, res.VotedIPs, "voted IPs are stored")
	assert.Equal(t, "new title", res.PostTitle, "post title is stored")
	assert.Equal(t, orig.ParentID, res.ParentID, "parent id is immutable")
	assert.Equal(t, orig.Timestamp, res.Timestamp, "timestamp is immutable")
	assert.Equal(t, orig.User, res.User, "user is immutable")
}

func (s suite) findLast(t *testing.T) {
	e := s.prep(t)

	req := engine.FindRequest{Locator: site, Sort: "-time"}
	res, err := e.Find(req)
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, "some text2", res[0].Text)

	req.Limit = 1
	res, err = e.Find(req)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "some text2", res[0].Text)
}

func (s suite) findLastSince(t *testing.T) {
	s.findSince(t, site)
}

func (s suite) findInPostSince(t *testing.T) {
	s.findSince(t, loc(postURL))
}

// findSince checks Since for the site's last comments or the post's comments, depending on the locator
func (s suite) findSince(t *testing.T, locator store.Locator) {
	e := s.prep(t)

	req := engine.FindRequest{Locator: locator, Sort: "-time", Since: ts(21)}
	res, err := e.Find(req)
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, "some text2", res[0].Text)

	req.Since = ts(22)
	res, err = e.Find(req)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "some text2", res[0].Text)

	req.Since = time.Date(2017, 12, 20, 16, 18, 22, 0, time.UTC)
	res, err = e.Find(req)
	require.NoError(t, err)
	assert.Empty(t, res)
}

func (s suite) findForUser(t *testing.T) {
	e := s.prep(t)

	req := engine.FindRequest{Locator: site, Sort: "-time", UserID: "user1", Limit: 5}
	res, err := e.Find(req)
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, "some text2", res[0].Text, "sorted by -time")

	req = engine.FindRequest{Locator: site, Sort: "-time", UserID: "user1", Limit: 1}
	res, err = e.Find(req)
	require.NoError(t, err)
	require.Len(t, res, 1, "allow 1 comment")
	assert.Equal(t, "some text2", res[0].Text, "sorted by -time")

	req = engine.FindRequest{Locator: site, Sort: "-time", UserID: "user1", Limit: 1, Skip: 1}
	res, err = e.Find(req)
	require.NoError(t, err)
	require.Len(t, res, 1, "allow 1 comment")
	assert.Equal(t, `some text, <a href="http://radio-t.com">link</a>`, res[0].Text, "second comment")
}

func (s suite) findForUserPagination(t *testing.T) {
	e := s.newEngine(t)
	c := store.Comment{Locator: loc(postURL), User: store.User{ID: "user1", Name: "user name"}}

	// write 200 comments
	for i := range 200 {
		c.ID = fmt.Sprintf("id-%d", i)
		c.Text = fmt.Sprintf("text #%d", i)
		c.Timestamp = time.Date(2017, 12, 20, 15, 18, i, 0, time.UTC)
		_, err := e.Create(c)
		require.NoError(t, err)
	}

	// get all comments
	req := engine.FindRequest{Locator: site, Sort: "-time", UserID: "user1"}
	res, err := e.Find(req)
	require.NoError(t, err)
	require.Len(t, res, 200)
	assert.Equal(t, "id-199", res[0].ID)

	// seek 0, 5 comments
	req.Limit = 5
	res, err = e.Find(req)
	require.NoError(t, err)
	require.Len(t, res, 5)
	assert.Equal(t, "id-199", res[0].ID)
	assert.Equal(t, "id-195", res[4].ID)

	// seek 10, 3 comments
	req.Skip, req.Limit = 10, 3
	res, err = e.Find(req)
	require.NoError(t, err)
	require.Len(t, res, 3)
	assert.Equal(t, "id-189", res[0].ID)
	assert.Equal(t, "id-187", res[2].ID)

	// seek 195, ask 10 comments
	req.Skip, req.Limit = 195, 10
	res, err = e.Find(req)
	require.NoError(t, err)
	require.Len(t, res, 5)
	assert.Equal(t, "id-4", res[0].ID)
	assert.Equal(t, "id-0", res[4].ID)

	// seek 255, ask 10 comments
	req.Skip, req.Limit = 255, 10
	res, err = e.Find(req)
	require.NoError(t, err)
	assert.Empty(t, res)

	// deleted comments take no place in a page and are not counted by skip
	for _, id := range []string{"id-199", "id-197", "id-190"} {
		require.NoError(t, e.Delete(engine.DeleteRequest{Locator: c.Locator, CommentID: id, DeleteMode: store.SoftDelete}))
	}
	req.Skip, req.Limit = 0, 5
	res, err = e.Find(req)
	require.NoError(t, err)
	assert.Equal(t, []string{"id-198", "id-196", "id-195", "id-194", "id-193"}, commentIDs(res))
	req.Skip, req.Limit = 5, 3
	res, err = e.Find(req)
	require.NoError(t, err)
	assert.Equal(t, []string{"id-192", "id-191", "id-189"}, commentIDs(res))
	count, err := e.Count(engine.FindRequest{Locator: site, UserID: "user1"})
	require.NoError(t, err)
	assert.Equal(t, 197, count)
}

func (s suite) countPost(t *testing.T) {
	e := s.prep(t)

	c, err := e.Count(engine.FindRequest{Locator: loc(postURL)})
	require.NoError(t, err)
	assert.Equal(t, 2, c)

	c, err = e.Count(engine.FindRequest{Locator: loc("https://radio-t.com-xxx")})
	require.NoError(t, err)
	assert.Equal(t, 0, c)

	_, err = e.Count(engine.FindRequest{Locator: site})
	assert.Error(t, err, "neither post nor user")
}

func (s suite) countUser(t *testing.T) {
	e := s.prep(t)
	req := engine.FindRequest{Locator: site, UserID: "user1"}

	c, err := e.Count(req)
	require.NoError(t, err)
	assert.Equal(t, 2, c)

	// a comment on another post counts, a deleted one does not
	_, err = e.Create(store.Comment{ID: "id-3", Text: "some text3", Timestamp: ts(24),
		Locator: loc("https://radio-t.com/2"), User: store.User{ID: "user1", Name: "user name"}})
	require.NoError(t, err)
	c, err = e.Count(req)
	require.NoError(t, err)
	assert.Equal(t, 3, c)
	require.NoError(t, e.Delete(engine.DeleteRequest{Locator: loc(postURL), CommentID: "id-1", DeleteMode: store.SoftDelete}))
	c, err = e.Count(req)
	require.NoError(t, err)
	assert.Equal(t, 2, c, "soft-deleted comment is not counted")
}

func (s suite) infoPost(t *testing.T) {
	e := s.prep(t) // two comments for postURL

	// add one more for https://radio-t.com/2
	_, err := e.Create(store.Comment{ID: "12345", Text: `some text, <a href="http://radio-t.com">link</a>`, Timestamp: ts(24),
		Locator: loc("https://radio-t.com/2"), User: store.User{ID: "user1", Name: "user name"}})
	require.NoError(t, err)

	r, err := e.Info(engine.InfoRequest{Locator: loc("https://radio-t.com/2"), ReadOnlyAge: 0})
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(24), LastTS: ts(24)}}, r)

	r, err = e.Info(engine.InfoRequest{Locator: loc("https://radio-t.com/2"), ReadOnlyAge: 10})
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(24), LastTS: ts(24),
		ReadOnly: true}}, r)

	r, err = e.Info(engine.InfoRequest{Locator: loc(postURL), ReadOnlyAge: 0})
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: postURL, Count: 2, FirstTS: ts(22), LastTS: ts(23)}}, r)

	_, err = e.Info(engine.InfoRequest{Locator: loc("https://radio-t.com/error"), ReadOnlyAge: 0})
	require.Error(t, err, "no such post")

	_, err = e.Info(engine.InfoRequest{Locator: store.Locator{URL: postURL, SiteID: "radio-t-error"}, ReadOnlyAge: 0})
	require.Error(t, err, "no such site")

	_, err = e.Info(engine.InfoRequest{})
	require.Error(t, err, "empty site id is an unknown site")

	_, err = e.Flag(engine.FlagRequest{Flag: engine.ReadOnly, Locator: loc("https://radio-t.com/2"), Update: engine.FlagTrue})
	require.NoError(t, err)
	r, err = e.Info(engine.InfoRequest{Locator: loc("https://radio-t.com/2"), ReadOnlyAge: 0})
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(24), LastTS: ts(24),
		ReadOnly: true}}, r)
}

func (s suite) infoPostCreatedOutOfOrder(t *testing.T) {
	// imports and backup restores create comments in any time order, deleted ones included
	e := s.newEngine(t)

	tm := func(minute int) time.Time { return time.Date(2017, 12, 20, 15, minute, 0, 0, time.UTC) }
	mk := func(id, url string, minute int, deleted bool) {
		c := store.Comment{ID: id, Text: "text", Timestamp: tm(minute), Locator: loc(url),
			User: store.User{ID: "user1", Name: "user name"}}
		if deleted {
			c.SetDeleted(store.SoftDelete)
		}
		_, err := e.Create(c)
		require.NoError(t, err)
	}
	mk("c1", "https://radio-t.com/1", 20, false)
	mk("c2", "https://radio-t.com/1", 5, true)
	mk("c3", "https://radio-t.com/1", 10, false)
	mk("c4", "https://radio-t.com/1", 50, true)
	mk("c5", "https://radio-t.com/1", 30, false)
	mk("c6", "https://radio-t.com/2", 40, true)

	r, err := e.Info(engine.InfoRequest{Locator: loc("https://radio-t.com/1")})
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/1", Count: 3, FirstTS: tm(5), LastTS: tm(50)}}, r,
		"deleted comments are not counted, first and last are the earliest and latest comments, as on the site the backup came from")

	r, err = e.Info(engine.InfoRequest{Locator: loc("https://radio-t.com/2"), ReadOnlyAge: 10})
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 0, FirstTS: tm(40), LastTS: tm(40), ReadOnly: true}}, r,
		"a post with only a deleted comment has no comments and keeps its age")

	r, err = e.Info(engine.InfoRequest{Locator: site})
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/2", Count: 0, FirstTS: tm(40), LastTS: tm(40)},
		{URL: "https://radio-t.com/1", Count: 3, FirstTS: tm(5), LastTS: tm(50)}}, r, "the site list counts the same way")
}

func (s suite) infoList(t *testing.T) {
	e := s.prep(t) // two comments for postURL

	// add one more for https://radio-t.com/2
	_, err := e.Create(store.Comment{ID: "12345", Text: `some text, <a href="http://radio-t.com">link</a>`, Timestamp: ts(22),
		Locator: loc("https://radio-t.com/2"), User: store.User{ID: "user1", Name: "user name"}})
	require.NoError(t, err)

	post2 := store.PostInfo{URL: "https://radio-t.com/2", Count: 1, FirstTS: ts(22), LastTS: ts(22)}
	post1 := store.PostInfo{URL: postURL, Count: 2, FirstTS: ts(22), LastTS: ts(23)}

	res, err := e.Info(engine.InfoRequest{Locator: site})
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{post2, post1}, res)

	res, err = e.Info(engine.InfoRequest{Locator: site, Limit: -1, Skip: -1})
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{post2, post1}, res)

	res, err = e.Info(engine.InfoRequest{Locator: site, Limit: 1})
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{post2}, res)

	res, err = e.Info(engine.InfoRequest{Locator: site, Limit: 1, Skip: 1})
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{post1}, res)
}
