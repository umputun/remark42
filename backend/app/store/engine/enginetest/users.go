package enginetest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
)

func (s suite) userDetail(t *testing.T) {
	e := s.prep(t)

	// add two entries before we start
	result, err := e.UserDetail(engine.UserDetailRequest{Locator: site, UserID: "u1", Detail: engine.UserEmail, Update: "test@example.com"})
	require.NoError(t, err, "No error inserting entry expected")
	assert.ElementsMatch(t, []engine.UserDetailEntry{{UserID: "u1", Email: "test@example.com"}}, result)
	result, err = e.UserDetail(engine.UserDetailRequest{Locator: site, UserID: "u2", Detail: engine.UserEmail, Update: "other@example.com"})
	require.NoError(t, err, "No error inserting entry expected")
	assert.ElementsMatch(t, []engine.UserDetailEntry{{UserID: "u2", Email: "other@example.com"}}, result)

	// stateless tests without changing the state we set up before
	var testData = []struct {
		req      engine.UserDetailRequest
		error    string
		expected []engine.UserDetailEntry
	}{
		{req: engine.UserDetailRequest{Locator: site, UserID: "u1", Detail: engine.UserEmail},
			expected: []engine.UserDetailEntry{{UserID: "u1", Email: "test@example.com"}}},
		{req: engine.UserDetailRequest{Locator: site, Detail: engine.UserEmail},
			error: "userid cannot be empty in request for single detail"},
		{req: engine.UserDetailRequest{Locator: site, Detail: engine.AllUserDetails, UserID: "u1"},
			error: "unsupported request with userdetail all"},
		{req: engine.UserDetailRequest{Locator: site, UserID: "u1xyz", Detail: engine.UserEmail}},
		{req: engine.UserDetailRequest{Detail: engine.UserEmail, Update: "new_value"},
			error: `userid cannot be empty in request for single detail`},
		{req: engine.UserDetailRequest{Detail: engine.UserDetail("bad")},
			error: `unsupported detail "bad"`},
		{req: engine.UserDetailRequest{Update: "not_relevant", Detail: engine.AllUserDetails},
			error: `unsupported request with userdetail all`},
		{req: engine.UserDetailRequest{Locator: site, Detail: engine.AllUserDetails},
			expected: []engine.UserDetailEntry{{UserID: "u1", Email: "test@example.com"}, {UserID: "u2", Email: "other@example.com"}}},
	}

	for i, x := range testData {
		result, err := e.UserDetail(x.req)
		if x.error != "" {
			assert.EqualError(t, err, x.error, "Error should match expected for case %d", i)
			assert.Nil(t, result, "no result with an error for case %d", i)
			continue
		}
		assert.NoError(t, err, "Error is not expected expected for case %d", i)
		assert.ElementsMatch(t, x.expected, result, "Result should match expected for case %d", i)
	}
}

func (s suite) userDetailAllEmpty(t *testing.T) {
	e := s.prep(t)
	val, err := e.UserDetail(engine.UserDetailRequest{Locator: site, Detail: engine.AllUserDetails})
	require.NoError(t, err)
	assert.Nil(t, val)
}

func (s suite) setUserDetailReturnsEntry(t *testing.T) {
	e := s.prep(t)
	_, err := e.UserDetail(engine.UserDetailRequest{Detail: engine.UserEmail, Locator: site, UserID: "u1", Update: "u1@example.com"})
	require.NoError(t, err)
	res, err := e.UserDetail(engine.UserDetailRequest{Detail: engine.UserTelegram, Locator: site, UserID: "u1", Update: "u1_tg"})
	require.NoError(t, err)
	assert.Equal(t, []engine.UserDetailEntry{{UserID: "u1", Email: "u1@example.com", Telegram: "u1_tg"}}, res,
		"a set returns the complete entry")
}

func (s suite) listDetailsKeepsEntriesSeparate(t *testing.T) {
	e := s.prep(t)
	_, err := e.UserDetail(engine.UserDetailRequest{Locator: site, UserID: "u1", Detail: engine.UserEmail, Update: "u1@example.com"})
	require.NoError(t, err)
	_, err = e.UserDetail(engine.UserDetailRequest{Locator: site, UserID: "u2", Detail: engine.UserTelegram, Update: "u2_telegram"})
	require.NoError(t, err)

	// u2 has no email, so listing must not carry u1's email over into u2's entry
	res, err := e.UserDetail(engine.UserDetailRequest{Locator: site, Detail: engine.AllUserDetails})
	require.NoError(t, err)
	assert.Equal(t, []engine.UserDetailEntry{{UserID: "u1", Email: "u1@example.com"}, {UserID: "u2", Telegram: "u2_telegram"}}, res)
}

func (s suite) userDetailsListOnlyStored(t *testing.T) {
	e := s.prep(t)

	_, err := e.Flag(engine.FlagRequest{Flag: engine.Verified, Locator: site, UserID: "u1", Update: engine.FlagTrue})
	require.NoError(t, err)
	_, err = e.UserDetail(engine.UserDetailRequest{Detail: engine.UserEmail, Locator: site, UserID: "u2", Update: "u2@example.com"})
	require.NoError(t, err)
	require.NoError(t, e.Delete(engine.DeleteRequest{Locator: site, UserID: "u2", UserDetail: engine.UserEmail}))

	res, err := e.UserDetail(engine.UserDetailRequest{Detail: engine.AllUserDetails, Locator: site})
	require.NoError(t, err)
	assert.Empty(t, res, "a user with only a flag, or with every detail deleted, has no details entry")
	res, err = e.UserDetail(engine.UserDetailRequest{Detail: engine.UserEmail, Locator: site, UserID: "u2"})
	require.NoError(t, err)
	assert.Empty(t, res, "a deleted detail leaves no entry")
}

func (s suite) deleteUserDetail(t *testing.T) {
	var (
		otherSite          = store.Locator{SiteID: OtherSiteID}
		createEmailUser    = engine.UserDetailRequest{Locator: site, UserID: "user1", Detail: engine.UserEmail, Update: "value1"}
		readEmailUser      = engine.UserDetailRequest{Locator: site, UserID: "user1", Detail: engine.UserEmail}
		createTelegramUser = engine.UserDetailRequest{Locator: site, UserID: "user1", Detail: engine.UserTelegram, Update: "value1"}
		readTelegramUser   = engine.UserDetailRequest{Locator: site, UserID: "user1", Detail: engine.UserTelegram}
		emailSet           = []engine.UserDetailEntry{{UserID: "user1", Email: "value1"}}
		telegramSet        = []engine.UserDetailEntry{{UserID: "user1", Telegram: "value1"}}
	)

	e := s.prep(t)

	// each row deletes, then reads or sets a detail and checks the result; nil means no entry
	var testData = []struct {
		delReq    engine.DeleteRequest
		detailReq engine.UserDetailRequest
		expected  []engine.UserDetailEntry
	}{
		{delReq: engine.DeleteRequest{Locator: site, UserID: "user1", UserDetail: engine.UserEmail},
			detailReq: createEmailUser, expected: emailSet},
		{delReq: engine.DeleteRequest{Locator: otherSite, UserID: "user1", UserDetail: engine.UserEmail},
			detailReq: readEmailUser, expected: emailSet},
		{delReq: engine.DeleteRequest{Locator: site, UserID: "user1", UserDetail: engine.UserEmail},
			detailReq: readEmailUser},
		{delReq: engine.DeleteRequest{Locator: site, UserID: "user1", UserDetail: engine.UserTelegram},
			detailReq: createTelegramUser, expected: telegramSet},
		{delReq: engine.DeleteRequest{Locator: otherSite, UserID: "user1", UserDetail: engine.UserTelegram},
			detailReq: readTelegramUser, expected: telegramSet},
		{delReq: engine.DeleteRequest{Locator: site, UserID: "user1", UserDetail: engine.UserTelegram},
			detailReq: readTelegramUser},
		{delReq: engine.DeleteRequest{Locator: site, UserID: "user1", UserDetail: engine.AllUserDetails},
			detailReq: createEmailUser, expected: emailSet},
		{delReq: engine.DeleteRequest{Locator: site, UserID: "user1", UserDetail: engine.AllUserDetails},
			detailReq: readEmailUser},
	}

	for i, x := range testData {
		require.NoError(t, e.Delete(x.delReq), "delete request #%d error", i)
		val, err := e.UserDetail(x.detailReq)
		require.NoError(t, err, "user request #%d error", i)
		require.Equal(t, x.expected, val, "user request #%d result", i)
	}
}

func (s suite) deleteComment(t *testing.T) {
	e := s.prep(t)
	findReq := engine.FindRequest{Locator: loc(postURL), Sort: "time"}

	res, err := e.Find(findReq)
	require.NoError(t, err)
	require.Len(t, res, 2, "initially 2 comments")

	count, err := e.Count(findReq)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "count=2 initially")

	delReq := engine.DeleteRequest{Locator: loc(postURL), CommentID: res[0].ID, DeleteMode: store.SoftDelete}
	require.NoError(t, e.Delete(delReq))

	res, err = e.Find(findReq)
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Empty(t, res[0].Text)
	assert.True(t, res[0].Deleted, "marked deleted")
	assert.Equal(t, store.User{Name: "user name", ID: "user1"}, res[0].User)
	assert.Equal(t, "some text2", res[1].Text)
	assert.False(t, res[1].Deleted)

	// repeated deletion should not decrease comments count
	require.NoError(t, e.Delete(delReq))

	comments, err := e.Find(engine.FindRequest{Locator: site, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, comments, 1, "1 in last, 1 removed")

	count, err = e.Count(findReq)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	delReq.CommentID = "123456"
	assert.Error(t, e.Delete(delReq), "no such comment")

	delReq.Locator, delReq.CommentID = loc("https://radio-t.com/bad"), res[0].ID
	assert.Error(t, e.Delete(delReq), "no such post")
}

func (s suite) deleteHard(t *testing.T) {
	e := s.prep(t)
	findReq := engine.FindRequest{Locator: loc(postURL), Sort: "time"}

	res, err := e.Find(findReq)
	require.NoError(t, err)
	require.Len(t, res, 2, "initially 2 comments")

	require.NoError(t, e.Delete(engine.DeleteRequest{Locator: loc(postURL), CommentID: res[0].ID, DeleteMode: store.HardDelete}))

	res, err = e.Find(findReq)
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Empty(t, res[0].Text)
	assert.True(t, res[0].Deleted, "marked deleted")
	assert.Equal(t, store.User{Name: "deleted", ID: "deleted"}, res[0].User)
}

func (s suite) deleteAll(t *testing.T) {
	e := s.prep(t)
	require.NoError(t, e.Delete(engine.DeleteRequest{Locator: site}))

	comments, err := e.Find(engine.FindRequest{Locator: site, Limit: 10})
	require.NoError(t, err)
	assert.Empty(t, comments, "nothing left")

	// a backup restore clears the site first, also when nothing was ever written to it
	assert.NoError(t, e.Delete(engine.DeleteRequest{Locator: store.Locator{SiteID: OtherSiteID}}), "a site with no comments")
}

func (s suite) deleteAllDropsUserDetails(t *testing.T) {
	e := s.prep(t)
	_, err := e.UserDetail(engine.UserDetailRequest{Detail: engine.UserEmail, Locator: site, UserID: "user1", Update: "u1@example.com"})
	require.NoError(t, err)
	_, err = e.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: site, UserID: "user1", Update: engine.FlagTrue})
	require.NoError(t, err)

	require.NoError(t, e.Delete(engine.DeleteRequest{Locator: site}))

	res, err := e.UserDetail(engine.UserDetailRequest{Detail: engine.AllUserDetails, Locator: site})
	require.NoError(t, err)
	assert.Empty(t, res, "deleting a site removes its user details")
	blocked, err := e.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: site, UserID: "user1"})
	require.NoError(t, err)
	assert.True(t, blocked, "deleting a site keeps the blocks")
}

// deleteUser soft-deletes one of the two comments of user1, then deletes user1 in the given mode
// and checks what is left of both comments
func (s suite) deleteUser(t *testing.T, mode store.DeleteMode, wantUser store.User) engine.Interface {
	e := s.prep(t)

	comments, err := e.Find(engine.FindRequest{Locator: loc(postURL), Sort: "time"})
	require.NoError(t, err)
	require.Len(t, comments, 2)

	// soft delete one comment first, deleting the user clears it as well
	require.NoError(t, e.Delete(engine.DeleteRequest{Locator: loc(postURL), CommentID: comments[0].ID, DeleteMode: store.SoftDelete}))

	require.NoError(t, e.Delete(engine.DeleteRequest{Locator: site, UserID: "user1", DeleteMode: mode}))

	comments, err = e.Find(engine.FindRequest{Locator: loc(postURL), Sort: "time"})
	require.NoError(t, err)
	require.Len(t, comments, 2, "2 comments with deleted info")
	assert.Equal(t, wantUser, comments[0].User)
	assert.Equal(t, wantUser, comments[1].User)

	c, err := e.Count(engine.FindRequest{Locator: loc(postURL)})
	require.NoError(t, err)
	assert.Equal(t, 0, c, "0 count")

	comments, err = e.Find(engine.FindRequest{Locator: site, Sort: "time"})
	require.NoError(t, err)
	assert.Empty(t, comments, "nothing left")
	return e
}

func (s suite) deleteUserHard(t *testing.T) {
	s.deleteUser(t, store.HardDelete, store.User{Name: "deleted", ID: "deleted"})
}

func (s suite) deleteUserSoft(t *testing.T) {
	e := s.deleteUser(t, store.SoftDelete, store.User{Name: "user name", ID: "user1"})

	comments, err := e.Find(engine.FindRequest{Locator: site, UserID: "user1", Limit: 5})
	require.NoError(t, err)
	assert.Empty(t, comments, "deleted comments are not listed for the user")
}

// deleteUserHardNoComments covers hard-deleting a user who has no comments, e.g. one who only logged in.
// This must succeed and must still remove any stored user details.
func (s suite) deleteUserHardNoComments(t *testing.T) {
	s.deleteUserNoComments(t, store.HardDelete)
}

// deleteUserSoftNoComments covers soft-deleting a user with no comments. As with the hard path it cleans
// stored user details and is a no-op for an unknown user.
func (s suite) deleteUserSoftNoComments(t *testing.T) {
	s.deleteUserNoComments(t, store.SoftDelete)
}

func (s suite) deleteUserNoComments(t *testing.T, mode store.DeleteMode) {
	e := s.prep(t)

	t.Run("login-only user with a detail but no comments", func(t *testing.T) {
		const userID = "login-only-user"

		// user logged in and has a stored detail, but never commented
		_, err := e.UserDetail(engine.UserDetailRequest{Locator: site, UserID: userID, Detail: engine.UserEmail, Update: "user@example.com"})
		require.NoError(t, err)

		require.NoError(t, e.Delete(engine.DeleteRequest{Locator: site, UserID: userID, DeleteMode: mode}),
			"deleting a user without comments must not fail")

		details, err := e.UserDetail(engine.UserDetailRequest{Locator: site, UserID: userID, Detail: engine.UserEmail})
		require.NoError(t, err)
		assert.Empty(t, details, "stored user detail must be removed")
	})

	t.Run("unknown user is a no-op", func(t *testing.T) {
		err := e.Delete(engine.DeleteRequest{Locator: site, UserID: "never-seen-user", DeleteMode: mode})
		assert.NoError(t, err, "deleting an unknown user must not error")
	})
}

func (s suite) close(t *testing.T) {
	e := s.prep(t)
	require.NoError(t, e.Close())
	assert.NoError(t, e.Close(), "second call should not result in panic or errors")
}
