package api

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/engine"
)

// mockPremoderationStore implements PremoderationStore for testing.
type mockPremoderationStore struct {
	isVerified     bool
	userComments   []store.Comment
	userErr        error
	approveErr     error
	approved       store.Comment
	approvePrev    bool
	approveStatus  bool
	approveLocator store.Locator
	approveID      string
}

func (m *mockPremoderationStore) IsVerified(siteID, userID string) bool {
	return m.isVerified
}

func (m *mockPremoderationStore) User(siteID, userID string, limit, skip int, user store.User) ([]store.Comment, error) {
	if m.userErr != nil {
		return nil, m.userErr
	}
	return m.userComments, nil
}

func (m *mockPremoderationStore) ApproveComments(locator store.Locator, id string, approveStatus, approvePrevious bool) (store.Comment, error) {
	m.approveLocator = locator
	m.approveID = id
	m.approveStatus = approveStatus
	m.approvePrev = approvePrevious
	if m.approveErr != nil {
		return store.Comment{}, m.approveErr
	}
	return m.approved, nil
}

func TestNewPremoderationStrategy(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Premoderation
		wantErr  bool
	}{
		{name: "links", input: "links", expected: PremoderationLinks},
		{name: "empty defaults to links", input: "", expected: PremoderationLinks},
		{name: "first", input: "first", expected: PremoderationFirst},
		{name: "all", input: "all", expected: PremoderationAll},
		{name: "none", input: "none", expected: PremoderationNone},
		{name: "case insensitive", input: "LINKS", expected: PremoderationLinks},
		{name: "trimmed", input: "  first  ", expected: PremoderationFirst},
		{name: "unknown", input: "bogus", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newPremoderationStrategy(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNewPremoderator(t *testing.T) {
	t.Run("valid strategy", func(t *testing.T) {
		p, err := NewPremoderator("links", &mockPremoderationStore{})
		require.NoError(t, err)
		assert.Equal(t, PremoderationLinks, p.strategy)
	})

	t.Run("invalid strategy", func(t *testing.T) {
		_, err := NewPremoderator("bogus", &mockPremoderationStore{})
		require.Error(t, err)
	})
}

func TestMaskComment(t *testing.T) {
	c := store.Comment{Text: "hello", Orig: "world"}
	MaskComment(&c)
	assert.Empty(t, c.Text)
	assert.Empty(t, c.Orig)
}

func TestPremoderateComment(t *testing.T) {
	adminUser := store.User{ID: "admin", Admin: true}
	regularUser := store.User{ID: "user1", SiteID: "site1"}
	verifiedUser := store.User{ID: "user2", SiteID: "site1"}

	t.Run("admin always approved", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationAll, store: &mockPremoderationStore{}}
		c := store.Comment{PendingApproval: true, User: adminUser}
		got, err := p.premoderateComment(c, adminUser)
		require.NoError(t, err)
		assert.False(t, got.PendingApproval)
	})

	t.Run("links strategy: no link", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationLinks, store: &mockPremoderationStore{}}
		c := store.Comment{Text: "just text", User: regularUser, Locator: store.Locator{SiteID: "site1"}}
		got, err := p.premoderateComment(c, regularUser)
		require.NoError(t, err)
		assert.False(t, got.PendingApproval)
	})

	t.Run("links strategy: with link", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationLinks, store: &mockPremoderationStore{}}
		c := store.Comment{Text: `<a href="http://example.com">link</a>`, User: regularUser, Locator: store.Locator{SiteID: "site1"}}
		got, err := p.premoderateComment(c, regularUser)
		require.NoError(t, err)
		assert.True(t, got.PendingApproval)
	})

	t.Run("links strategy: verified user approved", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationLinks, store: &mockPremoderationStore{isVerified: true}}
		c := store.Comment{Text: `<a href="http://example.com">link</a>`, User: verifiedUser, Locator: store.Locator{SiteID: "site1"}}
		got, err := p.premoderateComment(c, verifiedUser)
		require.NoError(t, err)
		assert.False(t, got.PendingApproval)
	})

	t.Run("links strategy: admin in comment approved", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationLinks, store: &mockPremoderationStore{}}
		c := store.Comment{Text: `<a href="http://example.com">link</a>`, User: adminUser, Locator: store.Locator{SiteID: "site1"}}
		got, err := p.premoderateComment(c, adminUser)
		require.NoError(t, err)
		assert.False(t, got.PendingApproval)
	})

	t.Run("first strategy: no previous comments", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationFirst, store: &mockPremoderationStore{userErr: engine.ErrNoUserComments}}
		c := store.Comment{User: regularUser, Locator: store.Locator{SiteID: "site1"}}
		got, err := p.premoderateComment(c, regularUser)
		require.NoError(t, err)
		assert.True(t, got.PendingApproval)
	})

	t.Run("first strategy: previously approved comment exists", func(t *testing.T) {
		prev := []store.Comment{{PendingApproval: false, Deleted: false}}
		p := Premoderator{strategy: PremoderationFirst, store: &mockPremoderationStore{userComments: prev}}
		c := store.Comment{User: regularUser, Locator: store.Locator{SiteID: "site1"}}
		got, err := p.premoderateComment(c, regularUser)
		require.NoError(t, err)
		assert.False(t, got.PendingApproval)
	})

	t.Run("first strategy: only pending/deleted previous comments", func(t *testing.T) {
		prev := []store.Comment{
			{PendingApproval: true, Deleted: false},
			{PendingApproval: false, Deleted: true},
		}
		p := Premoderator{strategy: PremoderationFirst, store: &mockPremoderationStore{userComments: prev}}
		c := store.Comment{PendingApproval: true, User: regularUser, Locator: store.Locator{SiteID: "site1"}}
		got, err := p.premoderateComment(c, regularUser)
		require.NoError(t, err)
		assert.True(t, got.PendingApproval)
	})

	t.Run("first strategy: store error", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationFirst, store: &mockPremoderationStore{userErr: errors.New("db down")}}
		c := store.Comment{User: regularUser, Locator: store.Locator{SiteID: "site1"}}
		_, err := p.premoderateComment(c, regularUser)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db down")
	})

	t.Run("all strategy: always pending", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationAll, store: &mockPremoderationStore{}}
		c := store.Comment{User: regularUser, Locator: store.Locator{SiteID: "site1"}}
		got, err := p.premoderateComment(c, regularUser)
		require.NoError(t, err)
		assert.True(t, got.PendingApproval)
	})

	t.Run("none strategy: always approved", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationNone, store: &mockPremoderationStore{}}
		c := store.Comment{User: regularUser, Locator: store.Locator{SiteID: "site1"}}
		got, err := p.premoderateComment(c, regularUser)
		require.NoError(t, err)
		assert.False(t, got.PendingApproval)
	})
}

func TestPremoderatorActive(t *testing.T) {
	t.Run("none is not active", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationNone}
		assert.False(t, p.Active())
	})
	t.Run("links is active", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationLinks}
		assert.True(t, p.Active())
	})
	t.Run("first is active", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationFirst}
		assert.True(t, p.Active())
	})
	t.Run("all is active", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationAll}
		assert.True(t, p.Active())
	})
}

func TestPremoderatorVisibleTo(t *testing.T) {
	admin := store.User{ID: "admin", Admin: true}
	author := store.User{ID: "user1"}
	stranger := store.User{ID: "user2"}

	t.Run("approved comment visible to all", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationLinks}
		c := store.Comment{PendingApproval: false, User: author}
		assert.True(t, p.VisibleTo(c, stranger))
	})

	t.Run("inactive strategy: pending visible to all", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationNone}
		c := store.Comment{PendingApproval: true, User: author}
		assert.True(t, p.VisibleTo(c, stranger))
	})

	t.Run("pending visible to admin", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationLinks}
		c := store.Comment{PendingApproval: true, User: author}
		assert.True(t, p.VisibleTo(c, admin))
	})

	t.Run("pending visible to author", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationLinks}
		c := store.Comment{PendingApproval: true, User: author}
		assert.True(t, p.VisibleTo(c, author))
	})

	t.Run("pending not visible to stranger", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationLinks}
		c := store.Comment{PendingApproval: true, User: author}
		assert.False(t, p.VisibleTo(c, stranger))
	})
}

func TestPremoderatorFilterVisibleTo(t *testing.T) {
	admin := store.User{ID: "admin", Admin: true}
	author := store.User{ID: "user1"}
	stranger := store.User{ID: "user2"}

	approved := store.Comment{ID: "1", PendingApproval: false, User: author}
	pending := store.Comment{ID: "2", PendingApproval: true, User: author}

	t.Run("inactive returns all", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationNone}
		got := p.FilterVisibleTo([]store.Comment{approved, pending}, stranger)
		assert.Len(t, got, 2)
	})

	t.Run("admin sees all", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationLinks}
		got := p.FilterVisibleTo([]store.Comment{approved, pending}, admin)
		assert.Len(t, got, 2)
	})

	t.Run("author sees own pending", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationLinks}
		got := p.FilterVisibleTo([]store.Comment{approved, pending}, author)
		assert.Len(t, got, 2)
	})

	t.Run("stranger sees only approved", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationLinks}
		got := p.FilterVisibleTo([]store.Comment{approved, pending}, stranger)
		assert.Len(t, got, 1)
		assert.Equal(t, "1", got[0].ID)
	})

	t.Run("does not mutate input", func(t *testing.T) {
		p := Premoderator{strategy: PremoderationLinks}
		original := []store.Comment{approved, pending}
		_ = p.FilterVisibleTo(original, stranger)
		assert.Len(t, original, 2)
	})
}

func TestPremoderatorApproveComment(t *testing.T) {
	loc := store.Locator{SiteID: "site1", URL: "http://example.com/post"}
	approved := store.Comment{ID: "c1", PendingApproval: false}

	t.Run("first strategy passes approvePrevious=true", func(t *testing.T) {
		m := &mockPremoderationStore{approved: approved}
		p := Premoderator{strategy: PremoderationFirst, store: m}
		got, err := p.ApproveComment(loc, "c1", true)
		require.NoError(t, err)
		assert.Equal(t, approved, got)
		assert.True(t, m.approvePrev)
		assert.True(t, m.approveStatus)
		assert.Equal(t, loc, m.approveLocator)
		assert.Equal(t, "c1", m.approveID)
	})

	t.Run("links strategy passes approvePrevious=false", func(t *testing.T) {
		m := &mockPremoderationStore{approved: approved}
		p := Premoderator{strategy: PremoderationLinks, store: m}
		_, err := p.ApproveComment(loc, "c1", true)
		require.NoError(t, err)
		assert.False(t, m.approvePrev)
	})

	t.Run("all strategy passes approvePrevious=false", func(t *testing.T) {
		m := &mockPremoderationStore{approved: approved}
		p := Premoderator{strategy: PremoderationAll, store: m}
		_, err := p.ApproveComment(loc, "c1", false)
		require.NoError(t, err)
		assert.False(t, m.approvePrev)
		assert.False(t, m.approveStatus)
	})

	t.Run("store error propagates", func(t *testing.T) {
		m := &mockPremoderationStore{approveErr: errors.New("boom")}
		p := Premoderator{strategy: PremoderationLinks, store: m}
		_, err := p.ApproveComment(loc, "c1", true)
		require.Error(t, err)
		assert.Equal(t, "boom", err.Error())
	})
}
