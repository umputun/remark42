package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/admin"
	"github.com/umputun/remark42/backend/app/store/engine"
)

func TestNormalizeWebsite(t *testing.T) {
	tbl := []struct {
		in, out string
		ok      bool
	}{
		{in: "https://example.com", out: "https://example.com", ok: true},
		{in: "  http://Example.COM/blog/  ", out: "http://example.com/blog/", ok: true},
		{in: "example.com", out: "https://example.com", ok: true},
		{in: "https://sub.example.co.uk/a?b=1#top", out: "https://sub.example.co.uk/a?b=1", ok: true},
		{in: "https://example.com:8443/path", out: "https://example.com:8443/path", ok: true},
		{in: "", ok: false},
		{in: "javascript:alert(1)", ok: false},
		{in: "data:text/html,hi", ok: false},
		{in: "ftp://example.com", ok: false},
		{in: "https://user:pass@example.com", ok: false},
		{in: "https://localhost", ok: false},
		{in: "https://app.localhost", ok: false},
		{in: "https://127.0.0.1", ok: false},
		{in: "https://[::1]/", ok: false},
		{in: "https://intranet", ok: false},
		{in: "https://example.com/<script>", ok: false},
		{in: `https://example.com/"onmouseover`, ok: false},
		{in: "https://exa mple.com", ok: false},
		{in: "https://example.com/" + string(make([]byte, 200)), ok: false},
		{in: "https://.example.com", ok: false},
		{in: "https://ExAmple.com.:443/", ok: false},
		// IPv4 as browsers parse it, loopback and others
		{in: "http://127.1", ok: false},
		{in: "http://0x7f.0.0.1", ok: false},
		{in: "http://0177.0.0.1", ok: false},
		{in: "http://0x7f000001", ok: false},
		{in: "http://2130706433", ok: false},
		{in: "http://192.168.0x1.1", ok: false},
		{in: "http://10.1", ok: false},
		{in: "http://1.2.3.0x", ok: false},
		{in: "http://example.0x1", ok: false},
		{in: "http://\uff11\uff12\uff17.\uff10.\uff10.\uff11", ok: false}, // fullwidth 127.0.0.1
		{in: "http://127\u30020\u30020\u30021", ok: false},                    // ideographic full stops
		{in: "http://ex%61mple.com", ok: false},
		// domains that only look numeric stay valid
		{in: "https://1.example.com", out: "https://1.example.com", ok: true},
		{in: "https://example.com1", out: "https://example.com1", ok: true},
		{in: "https://0xff.example.com/p?q=1", out: "https://0xff.example.com/p?q=1", ok: true},
		{in: "https://Espa\u00f1a.es/blog", out: "https://xn--espaa-rta.es/blog", ok: true},
		{in: "https://\uff45xample.com", out: "https://example.com", ok: true}, // fullwidth e
	}
	for _, tt := range tbl {
		t.Run(tt.in, func(t *testing.T) {
			out, err := NormalizeWebsite(tt.in)
			if !tt.ok {
				assert.ErrorIs(t, err, ErrInvalidWebsite)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.out, out)
		})
	}
}

func TestService_UserWebsite(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond, AdminStore: admin.NewStaticKeyStore("secret 123")}

	res, err := b.SetUserWebsite("radio-t", "u1", "example.com/me")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/me", res)

	res, err = b.GetUserWebsite("radio-t", "u1")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/me", res)

	_, err = b.SetUserWebsite("radio-t", "u1", "javascript:alert(1)")
	require.ErrorIs(t, err, ErrInvalidWebsite)
	res, err = b.GetUserWebsite("radio-t", "u1")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/me", res, "invalid value doesn't overwrite the stored one")

	require.NoError(t, b.DeleteUserDetail("radio-t", "u1", engine.UserWebsite))
	res, err = b.GetUserWebsite("radio-t", "u1")
	require.NoError(t, err)
	assert.Empty(t, res)

	_, err = b.SetUserWebsite("bad-site", "u1", "https://example.com")
	assert.Error(t, err)
}

func TestService_UserWebsiteOnComments(t *testing.T) {
	eng, teardown := prepStoreEngine(t)
	defer teardown()
	b := DataStore{Engine: eng, EditDuration: 100 * time.Millisecond, AdminStore: admin.NewStaticKeyStore("secret 123")}

	_, err := b.SetUserWebsite("radio-t", "user1", "https://user1.example.com")
	require.NoError(t, err)
	loc := store.Locator{URL: "https://radio-t.com", SiteID: "radio-t"}

	// stored comment carries a website in its user: it must never be shown as is
	c := store.Comment{Text: "hi", Locator: loc, User: store.User{ID: "user2", Name: "u2", Website: "https://spam.example.com"}}
	_, err = b.Create(c)
	require.NoError(t, err)

	websites := func() map[string]string {
		comments, err := b.Find(loc, "time", store.User{})
		require.NoError(t, err)
		res := map[string]string{}
		for _, c := range comments {
			res[c.User.ID] = c.User.Website
		}
		return res
	}

	// disabled: no website at all
	assert.Equal(t, map[string]string{"user1": "", "user2": ""}, websites())

	b.UserWebsite = true
	assert.Equal(t, map[string]string{"user1": "https://user1.example.com", "user2": ""}, websites())

	// blocked user loses the link
	require.NoError(t, b.SetBlock("radio-t", "user1", true, time.Hour))
	assert.Equal(t, map[string]string{"user1": "", "user2": ""}, websites())
}
