package enginetest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
)

func (s suite) flagBlockedUser(t *testing.T) {
	e := s.prep(t)
	isBlocked := func(siteID string) bool {
		v, err := e.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: store.Locator{SiteID: siteID}, UserID: "user1"})
		require.NoError(t, err)
		return v
	}
	setBlocked := func(user string, status engine.FlagStatus) {
		_, err := e.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: site, UserID: user, Update: status})
		require.NoError(t, err)
	}

	val, err := e.Flag(engine.FlagRequest{Locator: site, UserID: "user1"})
	require.NoError(t, err)
	assert.False(t, val, "nothing blocked yet")

	setBlocked("user1", engine.FlagTrue)
	assert.True(t, isBlocked(SiteID), "user1 blocked")

	setBlocked("user1", engine.FlagTrue)
	assert.True(t, isBlocked(SiteID), "user1 still blocked")

	setBlocked("user1", engine.FlagFalse)
	assert.False(t, isBlocked(SiteID), "user1 unblocked")

	setBlocked("userX", engine.FlagTrue) // a user with no comments can be blocked

	setBlocked("user1", engine.FlagTrue)
	assert.False(t, isBlocked("radio-t-bad"), "nothing blocked on wrong site")
}

func (s suite) flagReadOnlyPost(t *testing.T) {
	e := s.prep(t)
	isReadOnly := func(locator store.Locator) bool {
		v, err := e.Flag(engine.FlagRequest{Locator: locator, Flag: engine.ReadOnly})
		require.NoError(t, err)
		return v
	}

	assert.False(t, isReadOnly(loc("url-1")), "nothing ro")

	val, err := e.Flag(engine.FlagRequest{Locator: loc("url-1"), Flag: engine.ReadOnly, Update: engine.FlagTrue})
	require.NoError(t, err)
	assert.True(t, val)
	assert.True(t, isReadOnly(loc("url-1")), "url-1 ro")
	assert.False(t, isReadOnly(loc("url-2")), "url-2 still writable")
	assert.False(t, isReadOnly(store.Locator{SiteID: "radio-t-bad", URL: "url-1"}), "nothing ro on wrong site")

	_, err = e.Flag(engine.FlagRequest{Locator: loc("url-1"), Flag: engine.ReadOnly, Update: engine.FlagFalse})
	require.NoError(t, err)
	assert.False(t, isReadOnly(loc("url-1")), "url-1 writable")
}

func (s suite) flagVerified(t *testing.T) {
	e := s.prep(t)
	isVerified := func(siteID, user string) bool {
		v, err := e.Flag(engine.FlagRequest{Flag: engine.Verified, Locator: store.Locator{SiteID: siteID}, UserID: user})
		require.NoError(t, err)
		return v
	}
	setVerified := func(user string, status engine.FlagStatus) error {
		_, err := e.Flag(engine.FlagRequest{Flag: engine.Verified, Locator: site, UserID: user, Update: status})
		return err
	}

	assert.False(t, isVerified(SiteID, "u1"), "nothing verified")

	require.NoError(t, setVerified("u1", engine.FlagTrue))
	assert.True(t, isVerified(SiteID, "u1"), "u1 verified")

	assert.False(t, isVerified(SiteID, "u2"), "u2 still not verified")
	require.NoError(t, setVerified("u1", engine.FlagFalse))
	assert.False(t, isVerified(SiteID, "u1"), "u1 not verified anymore")

	assert.NoError(t, setVerified("u1xyz", engine.FlagFalse), "unverifying a user who was never verified")

	require.NoError(t, setVerified("u1", engine.FlagTrue))
	assert.False(t, isVerified("radio-t-bad", "u1"), "nothing verified on wrong site")
	assert.NoError(t, setVerified("u2", engine.FlagTrue))
	assert.NoError(t, setVerified("u3", engine.FlagFalse))
}

func (s suite) flagListVerified(t *testing.T) {
	e := s.prep(t)
	setVerified := func(user string, status engine.FlagStatus) {
		_, err := e.Flag(engine.FlagRequest{Flag: engine.Verified, Locator: site, UserID: user, Update: status})
		require.NoError(t, err)
	}

	ids, err := e.ListFlags(engine.FlagRequest{Flag: engine.Verified, Locator: site})
	require.NoError(t, err)
	assert.Equal(t, []any{}, ids, "verified list empty")

	setVerified("u1", engine.FlagTrue)
	setVerified("u2", engine.FlagTrue)
	ids, err = e.ListFlags(engine.FlagRequest{Flag: engine.Verified, Locator: site})
	require.NoError(t, err)
	assert.Equal(t, []any{"u1", "u2"}, ids, "verified 2 ids")

	_, err = e.ListFlags(engine.FlagRequest{Flag: engine.ReadOnly, Locator: site})
	assert.Error(t, err, "read-only is not a listable flag")
}

func (s suite) flagListBlocked(t *testing.T) {
	e := s.prep(t)
	setBlocked := func(user string, status engine.FlagStatus, ttl time.Duration) {
		_, err := e.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: site, UserID: user, Update: status, TTL: ttl})
		require.NoError(t, err)
	}
	blockedIDs := func(siteID string) []string {
		vv, err := e.ListFlags(engine.FlagRequest{Flag: engine.Blocked, Locator: store.Locator{SiteID: siteID}})
		require.NoError(t, err)
		res := make([]string, 0, len(vv))
		for _, v := range vv {
			u, ok := v.(store.BlockedUser)
			require.True(t, ok)
			res = append(res, u.ID)
		}
		return res
	}

	// a block that runs out is left to each engine's own tests: checking it needs time to pass, and the set
	// can't move the clock of an engine behind a network
	setBlocked("user1", engine.FlagTrue, 0)
	setBlocked("user2", engine.FlagTrue, time.Hour)
	setBlocked("user3", engine.FlagFalse, 0)

	assert.Equal(t, []string{"user1", "user2"}, blockedIDs(SiteID), "a permanent block and one that has not run out")
	assert.Empty(t, blockedIDs(OtherSiteID), "blocks are listed for their own site only")
}

func (s suite) flagListsSortedWithNames(t *testing.T) {
	e := s.prep(t)
	ids := []string{"u5", "u3", "u9", "u1", "u7", "u2", "u8", "u4", "u6", "user1"}
	for _, id := range ids {
		_, err := e.Flag(engine.FlagRequest{Flag: engine.Verified, Locator: site, UserID: id, Update: engine.FlagTrue})
		require.NoError(t, err)
		_, err = e.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: site, UserID: id, Update: engine.FlagTrue})
		require.NoError(t, err)
	}
	sorted := []string{"u1", "u2", "u3", "u4", "u5", "u6", "u7", "u8", "u9", "user1"}

	verified, err := e.ListFlags(engine.FlagRequest{Flag: engine.Verified, Locator: site})
	require.NoError(t, err)
	gotVerified := make([]string, 0, len(verified))
	for _, v := range verified {
		gotVerified = append(gotVerified, v.(string))
	}
	assert.Equal(t, sorted, gotVerified, "verified users are listed by id")

	blocked, err := e.ListFlags(engine.FlagRequest{Flag: engine.Blocked, Locator: site})
	require.NoError(t, err)
	gotBlocked, names := make([]string, 0, len(blocked)), map[string]string{}
	for _, v := range blocked {
		u := v.(store.BlockedUser)
		gotBlocked = append(gotBlocked, u.ID)
		names[u.ID] = u.Name
	}
	assert.Equal(t, sorted, gotBlocked, "blocked users are listed by id")
	assert.Equal(t, "user name", names["user1"], "a blocked user's name comes from the user's comments")
	assert.Empty(t, names["u1"], "a blocked user without comments has no name")
}

func (s suite) flagsKeepUserMeta(t *testing.T) {
	e := s.prep(t)

	_, err := e.UserDetail(engine.UserDetailRequest{Detail: engine.UserEmail, Locator: site, UserID: "u1", Update: "u1@example.com"})
	require.NoError(t, err)
	_, err = e.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: site, UserID: "u1", Update: engine.FlagTrue})
	require.NoError(t, err)

	verified, err := e.ListFlags(engine.FlagRequest{Flag: engine.Verified, Locator: site})
	require.NoError(t, err)
	assert.Empty(t, verified, "a blocked user with an email is not listed as verified")

	_, err = e.Flag(engine.FlagRequest{Flag: engine.Verified, Locator: site, UserID: "u1", Update: engine.FlagTrue})
	require.NoError(t, err)
	blocked, err := e.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: site, UserID: "u1"})
	require.NoError(t, err)
	assert.True(t, blocked, "verifying a user keeps the block")
	email, err := e.UserDetail(engine.UserDetailRequest{Detail: engine.UserEmail, Locator: site, UserID: "u1"})
	require.NoError(t, err)
	assert.Equal(t, []engine.UserDetailEntry{{UserID: "u1", Email: "u1@example.com"}}, email, "flag changes keep the email")
}

func (s suite) userMetaPerSite(t *testing.T) {
	e := s.newEngine(t)
	siteA, siteB := store.Locator{SiteID: SiteID}, store.Locator{SiteID: OtherSiteID}

	_, err := e.Flag(engine.FlagRequest{Flag: engine.Verified, Locator: siteA, UserID: "u1", Update: engine.FlagTrue})
	require.NoError(t, err)
	_, err = e.UserDetail(engine.UserDetailRequest{Detail: engine.UserEmail, Locator: siteA, UserID: "u1", Update: "u1@example.com"})
	require.NoError(t, err)

	_, err = e.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: siteB, UserID: "u1", Update: engine.FlagTrue})
	require.NoError(t, err)

	verified, err := e.Flag(engine.FlagRequest{Flag: engine.Verified, Locator: siteA, UserID: "u1"})
	require.NoError(t, err)
	assert.True(t, verified, "a flag on site b keeps the verified mark on site a")
	email, err := e.UserDetail(engine.UserDetailRequest{Detail: engine.UserEmail, Locator: siteA, UserID: "u1"})
	require.NoError(t, err)
	assert.Equal(t, []engine.UserDetailEntry{{UserID: "u1", Email: "u1@example.com"}}, email, "a flag on site b keeps the email on site a")

	_, err = e.UserDetail(engine.UserDetailRequest{Detail: engine.UserTelegram, Locator: siteA, UserID: "u1", Update: "u1_tg"})
	require.NoError(t, err)
	tg, err := e.UserDetail(engine.UserDetailRequest{Detail: engine.UserTelegram, Locator: siteA, UserID: "u1"})
	require.NoError(t, err)
	assert.Equal(t, []engine.UserDetailEntry{{UserID: "u1", Telegram: "u1_tg"}}, tg, "a set on site a after a flag on site b stores the value")
	blocked, err := e.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: siteB, UserID: "u1"})
	require.NoError(t, err)
	assert.True(t, blocked, "a detail on site a keeps the block on site b")
	blocked, err = e.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: siteA, UserID: "u1"})
	require.NoError(t, err)
	assert.False(t, blocked, "a block on site b does not block on site a")
}
