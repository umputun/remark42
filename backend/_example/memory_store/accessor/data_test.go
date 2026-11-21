/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package accessor

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/engine/enginetest"
)

// TestMemData_Engine runs the shared engine test set, the same one the bolt engine runs
func TestMemData_Engine(t *testing.T) {
	enginetest.Run(t, func(*testing.T) engine.Interface { return NewMemData() })
}

// The tests below cover what is particular to this engine: its error texts, and an unknown site or user,
// for which it returns nothing where bolt returns an error, as it serves any site id it is given.

func TestMemData_ErrorTexts(t *testing.T) {
	b := prepMem(t)
	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}

	_, err := b.Create(store.Comment{ID: "id-1", Locator: loc})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dup key")

	_, err = b.Get(getReq(loc, "1234567"))
	assert.EqualError(t, err, `not found`)

	err = b.Update(store.Comment{ID: "id-1", Locator: store.Locator{URL: "https://radio-t.com-bad", SiteID: "radio-t"}})
	assert.EqualError(t, err, `not found`)

	err = b.Delete(engine.DeleteRequest{Locator: store.Locator{URL: "https://radio-t.com/bad", SiteID: "radio-t"}, CommentID: "id-1"})
	assert.EqualError(t, err, `not found`)
}

func TestMemData_UnknownSite(t *testing.T) {
	b := prepMem(t)
	bad := store.Locator{URL: "https://radio-t.com", SiteID: "bad"}
	badSite := store.Locator{SiteID: "bad"}

	res, err := b.Find(engine.FindRequest{Locator: badSite, Sort: "-time"})
	assert.NoError(t, err)
	assert.Empty(t, res, "find last")

	res, err = b.Find(engine.FindRequest{Locator: badSite, Sort: "-time", UserID: "user1", Limit: 1, Skip: 1})
	assert.NoError(t, err)
	assert.Empty(t, res, "find for user")

	_, err = b.Get(getReq(bad, "id-1"))
	assert.EqualError(t, err, `not found`)

	err = b.Update(store.Comment{ID: "id-1", Locator: bad})
	assert.EqualError(t, err, `not found`)

	c, err := b.Count(engine.FindRequest{Locator: bad})
	assert.NoError(t, err)
	assert.Equal(t, 0, c, "count post")

	c, err = b.Count(engine.FindRequest{Locator: badSite, UserID: "user1"})
	assert.NoError(t, err)
	assert.Equal(t, 0, c, "count user")

	info, err := b.Info(engine.InfoRequest{Locator: badSite, Limit: 1, Skip: 1})
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{}, info, "info list")

	_, err = b.Flag(engine.FlagRequest{Flag: engine.Verified, Locator: badSite, UserID: "u1", Update: engine.FlagTrue})
	assert.NoError(t, err, "a flag is set on any site")

	flags, err := b.ListFlags(engine.FlagRequest{Flag: engine.Verified, Locator: store.Locator{SiteID: "radio-t-bad"}})
	assert.NoError(t, err)
	assert.Empty(t, flags, "list verified")

	flags, err = b.ListFlags(engine.FlagRequest{Flag: engine.Blocked, Locator: badSite})
	assert.NoError(t, err)
	assert.Empty(t, flags, "list blocked")

	err = b.Delete(engine.DeleteRequest{Locator: bad, CommentID: "id-1", DeleteMode: store.SoftDelete})
	assert.EqualError(t, err, `not found`)

}

func TestMemData_UnknownUser(t *testing.T) {
	b := prepMem(t)
	site := store.Locator{SiteID: "radio-t"}

	res, err := b.Find(engine.FindRequest{Locator: site, Sort: "-time", UserID: "userZ", Limit: 1, Skip: 1})
	assert.NoError(t, err)
	assert.Empty(t, res)

	c, err := b.Count(engine.FindRequest{Locator: site, UserID: "userZ"})
	assert.NoError(t, err)
	assert.Equal(t, 0, c)

	require.NoError(t, b.Delete(engine.DeleteRequest{Locator: site, UserID: "user1", DeleteMode: store.HardDelete}))
	res, err = b.Find(engine.FindRequest{Locator: site, UserID: "user1", Limit: 5})
	assert.NoError(t, err)
	assert.Empty(t, res)
}

// TestMemData_FlagListBlockedExpires checks that a block ends once its TTL has passed. It stays per engine
// because it moves the clock, which a synctest bubble can do here and the shared set can't do for an engine
// behind a network.
func TestMemData_FlagListBlockedExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b := prepMem(t)
		site := store.Locator{SiteID: "radio-t"}
		for user, ttl := range map[string]time.Duration{"user1": 0, "user2": 50 * time.Millisecond} {
			_, err := b.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: site, UserID: user, Update: engine.FlagTrue, TTL: ttl})
			require.NoError(t, err)
		}
		blocked := func() []any {
			res, err := b.ListFlags(engine.FlagRequest{Flag: engine.Blocked, Locator: site})
			require.NoError(t, err)
			return res
		}
		require.Len(t, blocked(), 2)

		time.Sleep(50 * time.Millisecond)
		res := blocked()
		require.Len(t, res, 1)
		assert.Equal(t, "user1", res[0].(store.BlockedUser).ID)

		stillBlocked, err := b.Flag(engine.FlagRequest{Flag: engine.Blocked, Locator: site, UserID: "user2"})
		require.NoError(t, err)
		assert.False(t, stillBlocked, "a block that ran out does not block")
	})
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
