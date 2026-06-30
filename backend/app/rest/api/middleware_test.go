package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-pkgz/auth/v2/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/umputun/remark42/backend/app/rest"
	"github.com/umputun/remark42/backend/app/store"
)

func TestTimeout(t *testing.T) {
	t.Run("fast handler passes through and gets a deadline", func(t *testing.T) {
		var gotDeadline bool
		h := timeout(time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, gotDeadline = r.Context().Deadline()
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("ok"))
		}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))
		assert.True(t, gotDeadline, "request context should carry a deadline")
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})

	t.Run("deadline exceeded writes 504", func(t *testing.T) {
		h := timeout(10 * time.Millisecond)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			<-r.Context().Done() // honor the context: return only once the deadline fires
		}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))
		assert.Equal(t, http.StatusGatewayTimeout, rec.Code)
	})
}

func TestRest_rejectAnonUser(t *testing.T) {
	ts := httptest.NewServer(fakeAuth(rejectAnonUser(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "Hello")
	}))))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "use not logged in")

	resp, err = http.Get(ts.URL + "?fake_id=anonymous_user123&fake_name=test")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "anon rejected")

	resp, err = http.Get(ts.URL + "?fake_id=real_user123&fake_name=test")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode, "real user")
}

func TestRest_cacheControl(t *testing.T) {
	tbl := []struct {
		url     string
		version string
		exp     time.Duration
		etag    string
		maxAge  int
	}{
		{"http://example.com/foo", "v1", time.Hour, "b433be1ea19edaee9dc92ca4b895b6bdf3c058cb", 3600},
		{"http://example.com/foo2", "v1", 10 * time.Hour, "6d8466aef3246c1057452561acddf7ad9d0d99e0", 36000},
		{"http://example.com/foo", "v2", time.Hour, "481700c52aab0dfbca99f3ffc2a4fbb27884c114", 3600},
		{"https://example.com/foo", "v2", time.Hour, "bebd4f1b87f474792c4e75e5affe31fbf67f5778", 3600},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, http.NoBody)
			w := httptest.NewRecorder()

			h := cacheControl(tt.exp, tt.version)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			h.ServeHTTP(w, req)
			resp := w.Result()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.NoError(t, resp.Body.Close())
			t.Logf("%+v", resp.Header)
			assert.Equal(t, `"`+tt.etag+`"`, resp.Header.Get("Etag"))
			assert.Equal(t, `max-age=`+strconv.Itoa(int(tt.exp.Seconds()))+", no-cache", resp.Header.Get("Cache-Control"))
		})
	}
}

// TestRest_apiCSP locks in that /api/v1/* responses get a strict default-src 'none'
// override regardless of what the global CSP allows. The widget HTML pages
// (/web/*.html) still get the global CSP (with 'unsafe-inline' for bootstrap),
// so the test asserts the two policies diverge across origins.
func TestRest_apiCSP(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()
	client := http.Client{}

	// JSON API endpoint — must carry the strict policy
	resp, err := client.Get(ts.URL + "/api/v1/config")
	require.NoError(t, err)
	defer resp.Body.Close()
	csp := resp.Header.Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'none'",
		"API responses must override the global CSP with default-src 'none'; got %q", csp)
	assert.Contains(t, csp, "sandbox", "API CSP must include sandbox; got %q", csp)
	assert.NotContains(t, csp, "'unsafe-inline'",
		"API CSP must not allow inline scripts/styles; got %q", csp)

	// RSS/XML endpoint — same strict policy, and the XML response itself must still be served
	respRSS, err := client.Get(ts.URL + "/api/v1/rss/site?site=remark42")
	require.NoError(t, err)
	defer respRSS.Body.Close()
	assert.Equal(t, http.StatusOK, respRSS.StatusCode, "RSS must still respond OK under strict CSP")
	cspRSS := respRSS.Header.Get("Content-Security-Policy")
	assert.Contains(t, cspRSS, "default-src 'none'", "RSS responses must carry the strict API CSP")
	assert.Contains(t, cspRSS, "sandbox", "RSS CSP must include sandbox")

	// widget HTML — must keep the global CSP (unchanged, lax to support inline bootstrap)
	resp2, err := client.Get(ts.URL + "/web/index.html")
	require.NoError(t, err)
	defer resp2.Body.Close()
	csp2 := resp2.Header.Get("Content-Security-Policy")
	assert.Contains(t, csp2, "'unsafe-inline'",
		"widget HTML CSP must keep unsafe-inline for bootstrap; got %q", csp2)
}

// check CSP, img-src should be 'self' with proxy enabled and * without it
func TestRest_securityHeaders(t *testing.T) {
	ts, _, teardown := startupT(t)

	// with proxy disabled
	client := http.Client{}
	resp, err := client.Get(ts.URL + "/web/index.html")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "img-src *;")
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
	teardown()

	// check CSP with proxy enabled
	ts, _, teardown = startupT(t, func(srv *Rest) {
		srv.ExternalImageProxy = true
	})
	defer teardown()
	resp, err = client.Get(ts.URL + "/web/index.html")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "img-src 'self';")
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
}

func TestRest_subscribersOnly(t *testing.T) {
	paidSubUser := &token.User{}
	paidSubUser.SetPaidSub(true)

	tbl := []struct {
		subsOnly bool
		user     token.User
		setUser  bool
		status   int
	}{
		{true, token.User{}, false, http.StatusUnauthorized},
		{true, token.User{}, true, http.StatusForbidden},
		{false, token.User{}, false, http.StatusOK},
		{false, token.User{}, true, http.StatusOK},
		{true, *paidSubUser, true, http.StatusOK},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com", http.NoBody)
			if tt.setUser {
				req = token.SetUserInfo(req, tt.user)
			}
			w := httptest.NewRecorder()
			h := subscribersOnly(tt.subsOnly)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			h.ServeHTTP(w, req)
			resp := w.Result()
			assert.Equal(t, tt.status, resp.StatusCode)
			assert.NoError(t, resp.Body.Close())
		})
	}
}

func Test_validEmailAuth(t *testing.T) {
	tbl := []struct {
		req    string
		status int
	}{
		{"/auth/email/login?site=remark42&address=umputun%example.com&user=someone", http.StatusOK},
		{"/auth/email/login?site=site-with-dash_and_underscore-and.dot&address=umputun%example.com&user=someone", http.StatusOK},
		{"/auth/email/login?site=remark42&address=umputun%example.com&user=someone+blah", http.StatusOK},
		{"/auth/email/login?site=remark42&address=umputun%example.com&user=Евгений+Умпутун", http.StatusOK},
		{"/auth/email/login?site=remark42&address=umputun%example.com&user=12", http.StatusForbidden},
		{"/auth/email/login?site=remark42&address=umputun%example.com&user=..blah+blah", http.StatusForbidden},
		{"/auth/email/login?site=remark42&address=umputun%example.com&user=someonelooong+loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooong", http.StatusForbidden},
		{"/auth/twitter/login?site=remark42&address=umputun%example.com&user=..blah+blah", http.StatusOK},
		{"/auth/email/login?site=remark42&address=umputun%example.com", http.StatusOK},
		{"/auth/email/login?site=remark42&address=umputun+example.com&user=someone", http.StatusForbidden},
		{"/auth/email/login?site=bad!site&address=umputun%example.com&user=someone", http.StatusForbidden},
		{"/auth/email/login?site=loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooongsite&address=umputun%example.com&user=someone", http.StatusForbidden},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com"+tt.req, http.NoBody)
			w := httptest.NewRecorder()
			h := validEmailAuth()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			h.ServeHTTP(w, req)
			resp := w.Result()
			assert.Equal(t, tt.status, resp.StatusCode)
			assert.NoError(t, resp.Body.Close())
		})
	}
}

// TestRest_matchSiteID reproduces the multi-tenant isolation gap in the matchSiteID
// middleware. Before the fix, the check `if siteID != "" && user.SiteID != siteID`
// silently allowed any authenticated request that omitted the ?site= query param.
// On admin and user-mutation routes this meant the cross-site check was bypassable
// just by dropping the parameter. The fix requires ?site= to be present and to match
// the user's bound site.
func TestRest_matchSiteID(t *testing.T) {
	wrapped := matchSiteID(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	cases := []struct {
		name     string
		userSite string
		query    string
		want     int
	}{
		{name: "matching site allowed", userSite: "site-a", query: "?site=site-a", want: http.StatusOK},
		{name: "mismatched site forbidden", userSite: "site-a", query: "?site=site-b", want: http.StatusForbidden},
		{name: "missing site param rejected", userSite: "site-a", query: "", want: http.StatusForbidden},
		{name: "empty site param rejected", userSite: "site-a", query: "?site=", want: http.StatusForbidden},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r = rest.SetUserInfo(r, store.User{ID: "u", Name: "u", SiteID: c.userSite})
				wrapped.ServeHTTP(w, r)
			})
			ts := httptest.NewServer(h)
			defer ts.Close()
			resp, err := http.Get(ts.URL + c.query)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			assert.Equal(t, c.want, resp.StatusCode)
		})
	}
}
