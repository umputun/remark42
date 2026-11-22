package engine

import (
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/remark42/backend/app/store"
)

// What the engine does is checked by the shared set in enginetest, run by TestBoltDB_Engine.
// The tests here cover what is particular to bolt: its error texts, its errors for an unknown site
// or user where the memory engine returns nothing, and its storage format.

func TestBoltDB_ErrorTexts(t *testing.T) {
	b, teardown := prep(t)
	defer teardown()

	_, err := b.Create(store.Comment{ID: "id-1", Locator: store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}})
	assert.EqualError(t, err, "key id-1 already in store")

	err = b.Update(store.Comment{ID: "id-1", Locator: store.Locator{URL: "https://radio-t.com-bad", SiteID: "radio-t"}})
	assert.EqualError(t, err, `no bucket https://radio-t.com-bad in store`)

	err = b.Delete(DeleteRequest{Locator: store.Locator{URL: "https://radio-t.com/bad", SiteID: "radio-t"}, CommentID: "id-1"})
	assert.EqualError(t, err, `no bucket https://radio-t.com/bad in store`)
}

// TestBoltDB_UnknownSite checks that every request for a site the engine was not opened with fails,
// and that a failed delete leaves the known site alone
func TestBoltDB_UnknownSite(t *testing.T) {
	b, teardown := prep(t)
	defer teardown()

	_, err := b.UserDetail(UserDetailRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", Detail: UserEmail, Update: "value1"})
	require.NoError(t, err)

	bad := store.Locator{URL: "https://radio-t.com", SiteID: "bad"}
	badSite := store.Locator{SiteID: "bad"}
	tbl := []struct {
		name string
		call func() error
	}{
		{"find post", func() error { _, e := b.Find(FindRequest{Locator: bad, Sort: "time"}); return e }},
		{"find last", func() error { _, e := b.Find(FindRequest{Locator: badSite, Sort: "-time"}); return e }},
		{"find for user", func() error {
			_, e := b.Find(FindRequest{Locator: badSite, Sort: "-time", UserID: "user1", Limit: 1, Skip: 1})
			return e
		}},
		{"get", func() error { _, e := b.Get(getReq(bad, "id-1")); return e }},
		{"update", func() error { return b.Update(store.Comment{ID: "id-1", Locator: bad}) }},
		{"count post", func() error { _, e := b.Count(FindRequest{Locator: bad}); return e }},
		{"count user", func() error { _, e := b.Count(FindRequest{Locator: badSite, UserID: "user1"}); return e }},
		{"info list", func() error { _, e := b.Info(InfoRequest{Locator: badSite, Limit: 1, Skip: 1}); return e }},
		{"block", func() error {
			_, e := b.Flag(FlagRequest{Flag: Blocked, Locator: badSite, UserID: "user1", Update: FlagTrue})
			return e
		}},
		{"read-only", func() error {
			_, e := b.Flag(FlagRequest{Flag: ReadOnly, Locator: store.Locator{SiteID: "bad", URL: "url-1"}, Update: FlagFalse})
			return e
		}},
		{"verify", func() error {
			_, e := b.Flag(FlagRequest{Flag: Verified, Locator: badSite, UserID: "u1", Update: FlagTrue})
			return e
		}},
		{"list verified", func() error { _, e := b.ListFlags(FlagRequest{Flag: Verified, Locator: badSite}); return e }},
		{"list blocked", func() error { _, e := b.ListFlags(FlagRequest{Flag: Blocked, Locator: badSite}); return e }},
		{"user detail", func() error {
			res, e := b.UserDetail(UserDetailRequest{Locator: badSite, UserID: "u1", Detail: UserEmail})
			assert.Nil(t, res)
			return e
		}},
		{"all user details", func() error {
			res, e := b.UserDetail(UserDetailRequest{Locator: badSite, Detail: AllUserDetails})
			assert.Nil(t, res)
			return e
		}},
		{"delete comment", func() error {
			return b.Delete(DeleteRequest{Locator: bad, CommentID: "id-1", DeleteMode: store.SoftDelete})
		}},
		{"delete user detail", func() error {
			return b.Delete(DeleteRequest{Locator: badSite, UserID: "user1", UserDetail: UserEmail})
		}},
		{"delete user", func() error { return b.Delete(DeleteRequest{Locator: badSite, UserID: "user1"}) }},
		{"delete site", func() error { return b.Delete(DeleteRequest{Locator: badSite}) }},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualError(t, tt.call(), `site "bad" not found`)
		})
	}

	val, err := b.UserDetail(UserDetailRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", Detail: UserEmail})
	require.NoError(t, err)
	assert.Equal(t, []UserDetailEntry{{UserID: "user1", Email: "value1"}}, val, "the known site's detail is kept")
}

// TestBoltDB_UnknownUser checks that a user with no comment reference in the site fails a user's find and count
func TestBoltDB_UnknownUser(t *testing.T) {
	b, teardown := prep(t)
	defer teardown()

	_, err := b.Find(FindRequest{Locator: store.Locator{SiteID: "radio-t"}, Sort: "-time", UserID: "userZ", Limit: 1, Skip: 1})
	assert.EqualError(t, err, `no comments for user userZ in store`)

	_, err = b.Count(FindRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "userZ"})
	assert.EqualError(t, err, `no comments for user userZ in store for radio-t site`)

	// a hard delete drops the user's references
	require.NoError(t, b.Delete(DeleteRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", DeleteMode: store.HardDelete}))
	_, err = b.Find(FindRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1", Limit: 5})
	assert.EqualError(t, err, "no comments for user user1 in store")
}

func TestBoltDB_CountUserUnparsableRef(t *testing.T) {
	b, teardown := prep(t)
	defer teardown()

	// a page url with the reference delimiter in it makes an unparsable reference, which is passed over
	_, err := b.Create(store.Comment{ID: "id-4", Text: "some text4", Timestamp: time.Date(2017, 12, 20, 15, 18, 25, 0, time.UTC),
		Locator: store.Locator{URL: "https://radio-t.com/a!!b", SiteID: "radio-t"}, User: store.User{ID: "user1", Name: "user name"}})
	require.NoError(t, err)
	req := FindRequest{Locator: store.Locator{SiteID: "radio-t"}, UserID: "user1"}
	c, err := b.Count(req)
	require.NoError(t, err)
	assert.Equal(t, 2, c, "unparsable reference is not counted")
	req.Sort = "-time"
	comments, err := b.Find(req)
	require.NoError(t, err)
	assert.Equal(t, []string{"id-2", "id-1"}, commentIDs(comments))
}

// TestBoltDB_FlagListBlockedExpires checks that a block ends once its TTL has passed. It stays per engine
// because it moves the clock, which a synctest bubble can do here and the shared set can't do for an engine
// behind a network.
func TestBoltDB_FlagListBlockedExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b, teardown := prep(t)
		defer teardown()

		for user, ttl := range map[string]time.Duration{"user1": 0, "user2": 150 * time.Millisecond} {
			_, err := b.Flag(FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: user, Update: FlagTrue, TTL: ttl})
			require.NoError(t, err)
		}
		blocked := func() []any {
			res, err := b.ListFlags(FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "radio-t"}})
			require.NoError(t, err)
			return res
		}
		require.Len(t, blocked(), 2)

		time.Sleep(150 * time.Millisecond)
		res := blocked()
		require.Len(t, res, 1)
		assert.Equal(t, "user1", res[0].(store.BlockedUser).ID)

		stillBlocked, err := b.Flag(FlagRequest{Flag: Blocked, Locator: store.Locator{SiteID: "radio-t"}, UserID: "user2"})
		require.NoError(t, err)
		assert.False(t, stillBlocked, "a block that ran out does not block")
	})
}

func commentIDs(comments []store.Comment) []string {
	ids := make([]string, 0, len(comments))
	for _, c := range comments {
		ids = append(ids, c.ID)
	}
	return ids
}

func TestBoltDB_ref(t *testing.T) {
	b := BoltDB{}
	comment := store.Comment{
		ID:        "12345",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
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
	boltStore, err := NewBoltDB(bolt.Options{}, BoltSite{FileName: filepath.Join(t.TempDir(), "test-remark.db"), SiteID: "radio-t"})
	assert.NoError(t, err)
	b = boltStore

	comment := store.Comment{
		ID:        "id-1",
		Text:      `some text, <a href="http://radio-t.com">link</a>`,
		Timestamp: time.Date(2017, 12, 20, 15, 18, 22, 0, time.UTC),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = b.Create(comment)
	assert.NoError(t, err)

	comment = store.Comment{
		ID:        "id-2",
		Text:      "some text2",
		Timestamp: time.Date(2017, 12, 20, 15, 18, 23, 0, time.UTC),
		Locator:   store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"},
		User:      store.User{ID: "user1", Name: "user name"},
	}
	_, err = b.Create(comment)
	assert.NoError(t, err)

	teardown = func() {
		require.NoError(t, b.Close())
	}
	return b, teardown
}

func getReq(locator store.Locator, commentID string) GetRequest {
	return GetRequest{
		Locator:   locator,
		CommentID: commentID,
	}
}
