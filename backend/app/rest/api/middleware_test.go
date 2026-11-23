package api

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-pkgz/auth/v2/token"
	"github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/realip"
	"github.com/go-pkgz/routegroup"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/rest"
)

// routes() wraps bounded routes with the enforcing rest.Timeout and deliberately leaves the
// streaming/long-polling routes (GET /export, /userdata, /wait) without it. This checks that
// contract holds against the vendored middleware: a slow handler under R.Timeout is aborted with
// 504 at the deadline, while a route left without it runs to completion.
func TestRouteTimeout(t *testing.T) {
	slow := func(d time.Duration) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done(): // return promptly once the enforcing timeout cancels the context
			case <-time.After(d):
			}
			w.WriteHeader(http.StatusOK)
		}
	}

	router := routegroup.New(http.NewServeMux())
	router.With(R.Timeout(20*time.Millisecond)).HandleFunc("GET /bounded", slow(time.Second))
	router.HandleFunc("GET /streaming", slow(30*time.Millisecond)) // no timeout, like /export and /wait
	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/bounded")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode, "route under R.Timeout is aborted at the deadline")

	resp, err = http.Get(ts.URL + "/streaming")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode, "route without R.Timeout runs to completion")
}

// TestRateLimiter covers the middleware guarding every route group: a burst past the per-second
// allowance is refused with 429, and a client under the allowance is not. The limiter keys on
// RemoteAddr, so the two cases use different ones rather than waiting for a bucket to refill.
func TestRateLimiter(t *testing.T) {
	router := routegroup.New(http.NewServeMux())
	router.With(rateLimiter(1)).HandleFunc("GET /limited", func(http.ResponseWriter, *http.Request) {})
	ts := httptest.NewServer(router)
	defer ts.Close()

	call := func(remoteAddr string) int {
		req := httptest.NewRequest("GET", "http://example.com/limited", http.NoBody)
		req.RemoteAddr = remoteAddr
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		resp := w.Result()
		assert.NoError(t, resp.Body.Close())
		return resp.StatusCode
	}

	// one request a second is allowed, so the first of a burst passes and the rest are refused
	assert.Equal(t, http.StatusOK, call("1.2.3.4:1000"), "first request within the allowance")
	refused := 0
	for range 5 {
		if call("1.2.3.4:1000") == http.StatusTooManyRequests {
			refused++
		}
	}
	assert.Equal(t, 5, refused, "burst past the allowance is refused")

	// a different client has its own bucket and is unaffected
	assert.Equal(t, http.StatusOK, call("5.6.7.8:1000"), "limit is per client, not global")
}

func TestRealIPMiddleware(t *testing.T) {
	// call runs mw with the given peer and (optional) X-Real-IP header and returns what the
	// downstream handler observes; state is per-call, so subtests don't share closure locals.
	call := func(mw func(http.Handler) http.Handler, remoteAddr, xRealIP string) (addr, hdr string) {
		next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			addr, hdr = r.RemoteAddr, r.Header.Get("X-Real-IP")
		})
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.RemoteAddr = remoteAddr
		if xRealIP != "" {
			req.Header.Set("X-Real-IP", xRealIP)
		}
		mw(next).ServeHTTP(httptest.NewRecorder(), req)
		return addr, hdr
	}

	trusted, err := ParseTrustedProxies([]string{"172.16.0.0/12", "2001:db8::/32"})
	require.NoError(t, err)

	t.Run("no trusted proxies trusts the header from anyone (legacy)", func(t *testing.T) {
		addr, _ := call(realIPMiddleware(nil), "203.0.113.9:1234", "8.8.8.8")
		assert.Equal(t, "8.8.8.8", addr)
	})
	t.Run("trusted v4 peer: forwarding header sets the client IP", func(t *testing.T) {
		addr, _ := call(realIPMiddleware(trusted), "172.18.0.5:5555", "8.8.8.8")
		assert.Equal(t, "8.8.8.8", addr)
	})
	t.Run("trusted v6 peer: forwarding header honored", func(t *testing.T) {
		addr, _ := call(realIPMiddleware(trusted), "[2001:db8::5]:5555", "8.8.8.8")
		assert.Equal(t, "8.8.8.8", addr)
	})
	t.Run("trusted peer without a forwarding header falls back to the socket IP", func(t *testing.T) {
		addr, _ := call(realIPMiddleware(trusted), "172.18.0.5:5555", "")
		assert.Equal(t, "172.18.0.5", addr, "no header to honor, so the bare socket IP is used")
	})
	t.Run("untrusted peer: header stripped, RemoteAddr pinned to bare socket IP", func(t *testing.T) {
		addr, hdr := call(realIPMiddleware(trusted), "203.0.113.9:1234", "8.8.8.8")
		assert.Equal(t, "203.0.113.9", addr, "real socket IP with the port stripped")
		assert.Empty(t, hdr, "spoofed forwarding header removed so nothing downstream can read it")
	})
	t.Run("unparseable RemoteAddr is treated as untrusted, header stripped", func(t *testing.T) {
		addr, hdr := call(realIPMiddleware(trusted), "garbage", "8.8.8.8")
		assert.Equal(t, "garbage", addr, "unparseable peer left as-is, not overwritten")
		assert.Empty(t, hdr, "forwarding header still stripped for a non-trusted peer")
	})
}

// a trusted proxy's request carries headers the proxy wrote next to headers the client sent. The
// address has to come from the former only: X-Real-IP as the proxy set it, or the entries the trusted
// proxies appended to X-Forwarded-For. 8.8.8.8 and 9.9.9.9 stand for what a client forged.
func TestRealIPMiddleware_TrustedPeerHeaders(t *testing.T) {
	trusted, err := ParseTrustedProxies([]string{"172.16.0.0/12", "2001:db8::/32"})
	require.NoError(t, err)

	const proxyV4, proxyV6 = "172.18.0.18:5555", "[2001:db8::5]:5555"
	tbl := []struct {
		name     string
		peer     string
		xRealIP  string
		xff      []string // one element per header line
		cfIP     string
		expected string
	}{
		{name: "public X-Real-IP set by the proxy, client entry ahead of the proxy's in X-Forwarded-For",
			peer: proxyV4, xRealIP: "203.0.113.7", xff: []string{"8.8.8.8, 203.0.113.7"}, expected: "203.0.113.7"},
		{name: "proxy sets X-Real-IP only and relays the client's X-Forwarded-For",
			peer: proxyV4, xRealIP: "203.0.113.7", xff: []string{"8.8.8.8"}, cfIP: "9.9.9.9", expected: "203.0.113.7"},
		{name: "X-Real-IP inside the trusted range, forged X-Forwarded-For",
			peer: proxyV4, xRealIP: "172.18.0.1", xff: []string{"8.8.8.8, 172.18.0.1"}, expected: "172.18.0.1"},
		{name: "X-Real-IP inside the trusted range, forged CF-Connecting-IP",
			peer: proxyV4, xRealIP: "172.18.0.1", xff: []string{"172.18.0.1"}, cfIP: "9.9.9.9", expected: "172.18.0.1"},
		{name: "private X-Real-IP outside the trusted range, forged X-Forwarded-For",
			peer: proxyV4, xRealIP: "192.168.1.20", xff: []string{"8.8.8.8, 192.168.1.20"}, expected: "192.168.1.20"},
		{name: "v4-mapped X-Real-IP is kept as the proxy wrote it",
			peer: proxyV4, xRealIP: "::ffff:203.0.113.7", expected: "::ffff:203.0.113.7"},
		{name: "no X-Real-IP, forged entry ahead of the proxy's in X-Forwarded-For",
			peer: proxyV4, xff: []string{"8.8.8.8, 203.0.113.7"}, expected: "203.0.113.7"},
		{name: "no X-Real-IP, forged CF-Connecting-IP",
			peer: proxyV4, xff: []string{"203.0.113.7"}, cfIP: "9.9.9.9", expected: "203.0.113.7"},
		{name: "no X-Real-IP, forged X-Forwarded-For on a line of its own ahead of the proxy's line",
			peer: proxyV4, xff: []string{"8.8.8.8", "203.0.113.7"}, expected: "203.0.113.7"},
		{name: "X-Real-IP that is not an address is passed over for X-Forwarded-For",
			peer: proxyV4, xRealIP: "unknown", xff: []string{"8.8.8.8, 203.0.113.7"}, expected: "203.0.113.7"},
		{name: "CF-Connecting-IP alone is not read",
			peer: proxyV4, cfIP: "9.9.9.9", expected: "172.18.0.18"},
		{name: "X-Forwarded-For through two trusted proxies",
			peer: proxyV4, xff: []string{"8.8.8.8, 198.51.100.4, 172.18.0.7"}, expected: "198.51.100.4"},
		{name: "X-Forwarded-For with trusted addresses only gives the leftmost",
			peer: proxyV4, xff: []string{"172.18.0.9, 172.18.0.7"}, expected: "172.18.0.9"},
		{name: "malformed entry at the end of X-Forwarded-For gives the proxy's address",
			peer: proxyV4, xff: []string{"8.8.8.8, unknown"}, expected: "172.18.0.18"},
		{name: "empty entry at the end of X-Forwarded-For gives the proxy's address",
			peer: proxyV4, xff: []string{"8.8.8.8,"}, expected: "172.18.0.18"},
		{name: "malformed entry behind a trusted one gives the proxy's address",
			peer: proxyV4, xff: []string{"8.8.8.8, unknown, 172.18.0.7"}, expected: "172.18.0.18"},
		{name: "v6 proxy and v6 addresses in X-Forwarded-For",
			peer: proxyV6, xff: []string{"2001:4860:4860::8888, 2a00:1450:4001::1, 2001:db8::7"}, expected: "2a00:1450:4001::1"},
		{name: "v6 proxy, X-Real-IP set by the proxy",
			peer: proxyV6, xRealIP: "2a00:1450:4001::1", xff: []string{"8.8.8.8, 2a00:1450:4001::1"}, cfIP: "9.9.9.9",
			expected: "2a00:1450:4001::1"},
		{name: "untrusted peer sending all three headers is known by its socket address",
			peer: "203.0.113.9:1234", xRealIP: "8.8.8.8", xff: []string{"8.8.8.8"}, cfIP: "9.9.9.9", expected: "203.0.113.9"},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			var addr, downstream string
			var left http.Header
			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				addr, left = r.RemoteAddr, r.Header.Clone()
				// the request logger and anything else below resolve the address this way
				var getErr error
				downstream, getErr = realip.Get(r)
				require.NoError(t, getErr)
			})
			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			req.RemoteAddr = tt.peer
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}
			for _, line := range tt.xff {
				req.Header.Add("X-Forwarded-For", line)
			}
			if tt.cfIP != "" {
				req.Header.Set("CF-Connecting-IP", tt.cfIP)
			}
			realIPMiddleware(trusted)(next).ServeHTTP(httptest.NewRecorder(), req)

			assert.Equal(t, tt.expected, addr, "RemoteAddr the handlers see")
			assert.Equal(t, tt.expected, downstream, "address resolved again below the middleware")
			for _, h := range ipForwardingHeaders {
				assert.Empty(t, left.Values(h), "%s is not passed on", h)
			}
		})
	}
}

func TestParseTrustedProxies(t *testing.T) {
	t.Run("cidr, bare v4, bare v6, blanks", func(t *testing.T) {
		got, err := ParseTrustedProxies([]string{"172.16.0.0/12", " 10.0.0.1 ", "", "2001:db8::/32"})
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.True(t, got[0].Contains(net.ParseIP("172.18.0.5")))
		assert.True(t, got[1].Contains(net.ParseIP("10.0.0.1")))
		assert.False(t, got[1].Contains(net.ParseIP("10.0.0.2")), "a bare IP is a single host")
		assert.True(t, got[2].Contains(net.ParseIP("2001:db8::1")))
	})
	t.Run("v4-mapped IPv6 bare entry resolves to the v4 host", func(t *testing.T) {
		got, err := ParseTrustedProxies([]string{"::ffff:10.0.0.1"})
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.True(t, got[0].Contains(net.ParseIP("10.0.0.1")), "the intended /32 host")
		assert.False(t, got[0].Contains(net.ParseIP("10.0.0.2")), "not a wider range")
	})
	t.Run("malformed entry is a hard error", func(t *testing.T) {
		_, err := ParseTrustedProxies([]string{"172.16.0.0/12", "nonsense"})
		require.Error(t, err)
		_, err = ParseTrustedProxies([]string{"10.0.0.0/999"})
		require.Error(t, err)
	})
	t.Run("all blank yields nil", func(t *testing.T) {
		got, err := ParseTrustedProxies([]string{"", "  "})
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestTrustsAnyPeer(t *testing.T) {
	catchAll := func(entries ...string) bool {
		cidrs, err := ParseTrustedProxies(entries)
		require.NoError(t, err)
		return TrustsAnyPeer(cidrs)
	}
	assert.True(t, catchAll("10.0.0.0/8", "0.0.0.0/0"), "v4 catch-all")
	assert.True(t, catchAll("::/0"), "v6 catch-all")
	assert.False(t, catchAll("172.16.0.0/12", "10.0.0.5"), "scoped ranges are not catch-all")
	assert.False(t, catchAll(), "empty is not catch-all")
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
	serve := func(exp time.Duration, version, url, content, ifNoneMatch string) (int, http.Header) {
		req := httptest.NewRequest("GET", url, http.NoBody)
		if ifNoneMatch != "" {
			req.Header.Set("If-None-Match", ifNoneMatch)
		}
		w := httptest.NewRecorder()
		fingerprint := func(*http.Request) string { return content }
		cacheControl(exp, version, fingerprint)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(w, req)
		resp := w.Result()
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode, resp.Header
	}
	etagOf := func(version, url, content string) string {
		_, h := serve(time.Hour, version, url, content, "")
		return h.Get("Etag")
	}

	t.Run("cache-control carries the expiration", func(t *testing.T) {
		for _, exp := range []time.Duration{time.Hour, 10 * time.Hour} {
			_, h := serve(exp, "v1", "http://example.com/foo", "a", "")
			assert.Equal(t, "max-age="+strconv.Itoa(int(exp.Seconds()))+", no-cache", h.Get("Cache-Control"))
		}
	})

	t.Run("each input changes the etag", func(t *testing.T) {
		base := etagOf("v1", "http://example.com/foo", "a")
		require.NotEmpty(t, base)
		for name, other := range map[string]string{
			"version": etagOf("v2", "http://example.com/foo", "a"),
			"url":     etagOf("v1", "http://example.com/foo2", "a"),
			"scheme":  etagOf("v1", "https://example.com/foo", "a"),
			"content": etagOf("v1", "http://example.com/foo", "b"),
		} {
			assert.NotEqual(t, base, other, name)
		}
		assert.Equal(t, base, etagOf("v1", "http://example.com/foo", "a"), "same inputs, same etag")
	})

	t.Run("a matching etag answers 304, a stale one passes through", func(t *testing.T) {
		etag := etagOf("v1", "http://example.com/foo", "a")
		code, _ := serve(time.Hour, "v1", "http://example.com/foo", "a", etag)
		assert.Equal(t, http.StatusNotModified, code)
		code, _ = serve(time.Hour, "v1", "http://example.com/foo", "b", etag)
		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("nothing to fingerprint, no etag and no 304", func(t *testing.T) {
		code, h := serve(time.Hour, "v1", "http://example.com/foo", "", `"anything"`)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, h.Get("Etag"))
		assert.Equal(t, "max-age=3600, no-cache", h.Get("Cache-Control"))
	})
}

// TestRest_apiCSP locks in that /api/v1/* responses get a strict default-src 'none'
// override regardless of what the global CSP allows. The widget HTML pages
// (/web/*.html) still get the global CSP (with 'unsafe-inline' for bootstrap),
// so the test asserts the two policies diverge across origins.
func TestRest_apiCSP(t *testing.T) {
	ts, _ := startupT(t)
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
	ts, _ := startupT(t)

	// with proxy disabled
	client := http.Client{}
	resp, err := client.Get(ts.URL + "/web/index.html")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "img-src *;")
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
	// httptest.Server.Close waits on connections still in use, and the cleanup of startupT does not
	// run until the test ends, so the body has to be released before the server is closed here
	require.NoError(t, resp.Body.Close())
	client.CloseIdleConnections()
	ts.Close() // stop the first server here, its cleanup closing it again does nothing

	// check CSP with proxy enabled
	ts, _ = startupT(t, func(srv *Rest) {
		srv.ExternalImageProxy = true
	})
	resp, err = client.Get(ts.URL + "/web/index.html")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "img-src 'self';")
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
}

func TestRest_subscribersOnly(t *testing.T) {
	// paidUser is a paid subscriber whose paid status ends at until, or has no end when it is zero
	paidUser := func(until time.Time) token.User {
		u := token.User{}
		u.SetPaidSub(true)
		if !until.IsZero() {
			rest.SetPaidSubUntil(&u, until)
		}
		return u
	}

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
		{true, paidUser(time.Now().Add(time.Hour)), true, http.StatusOK},
		{true, paidUser(time.Now().Add(-time.Second)), true, http.StatusForbidden},
		{true, paidUser(time.Time{}), true, http.StatusForbidden},
		{false, paidUser(time.Time{}), true, http.StatusOK},
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
		{"/auth/email/login?site=remark42&address=umputun%40example.com&user=someone", http.StatusOK},
		{"/auth/email/login?site=site-with-dash_and_underscore-and.dot&address=umputun%40example.com&user=someone", http.StatusOK},
		{"/auth/email/login?site=remark42&address=umputun%40example.com&user=someone+blah", http.StatusOK},
		{"/auth/email/login?site=remark42&address=umputun%40example.com&user=Евгений+Умпутун", http.StatusOK},
		{"/auth/email/login?site=remark42&address=umputun%40example.com&user=12", http.StatusForbidden},
		{"/auth/email/login?site=remark42&address=umputun%40example.com&user=**blah+blah", http.StatusForbidden},
		{"/auth/email/login?site=remark42&address=umputun%40example.com&user=someonelooong+loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooong", http.StatusForbidden},
		{"/auth/twitter/login?site=remark42&address=umputun%40example.com&user=..blah+blah", http.StatusOK},
		{"/auth/email/login?site=remark42&address=umputun%40example.com", http.StatusOK},
		{"/auth/email/login?site=remark42&address=umputun+example.com&user=someone", http.StatusForbidden},
		{"/auth/email/login?site=bad!site&address=umputun%40example.com&user=someone", http.StatusForbidden},
		{"/auth/email/login?site=loooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooooongsite&address=umputun%40example.com&user=someone", http.StatusForbidden},
		// three characters is the shortest name the widget's sign-in form accepts
		{"/auth/email/login?site=remark42&address=umputun%40example.com&user=abc", http.StatusOK},
		{"/auth/email/login?site=remark42&address=umputun%40example.com&user=Ева", http.StatusOK},
		// an address and nothing else
		{"/auth/email/login?site=remark42&user=someone&address=" + url.QueryEscape("first#last?x@example.com"), http.StatusOK},
		{"/auth/email/login?site=remark42&user=someone&address=Some.User%2Btag%40sub.example.com", http.StatusOK},
		{"/auth/email/login?site=remark42&user=someone&address=" + url.QueryEscape("пользователь@example.com"), http.StatusOK},
		{"/auth/email/login?site=remark42&user=someone&address=" + url.QueryEscape(`"unlock it at https://evil.example.com" <victim@example.com>`), http.StatusForbidden},
		{"/auth/email/login?site=remark42&user=someone&address=" + url.QueryEscape(`Your account is locked <victim@example.com>`), http.StatusForbidden},
		{"/auth/email/login?site=remark42&user=someone&address=" + url.QueryEscape(`victim@example.com (unlock at evil.example.com)`), http.StatusForbidden},
		{"/auth/email/login?site=remark42&user=someone&address=" + url.QueryEscape(`unlock at evil.example.com: victim@example.com;`), http.StatusForbidden},
		{"/auth/email/login?site=remark42&user=someone&address=" + url.QueryEscape(`"free text"@example.com`), http.StatusForbidden},
		{"/auth/email/login?site=remark42&user=someone&address=" + url.QueryEscape("victim+see\u00a0evil.example.com@example.com"), http.StatusForbidden},
		{"/auth/email/login?site=remark42&user=someone&address=" + url.QueryEscape(" victim@example.com"), http.StatusForbidden},
		{"/auth/email/login?site=remark42&user=someone&address=" + url.QueryEscape("victim@example.com,other@example.com"), http.StatusForbidden},
		// hyphens are allowed; dots, other spaces than the ordinary one and a space at either end are not
		{"/auth/email/login?site=remark42&address=umputun%40example.com&user=Ann-Marie+Smith", http.StatusOK},
		{"/auth/email/login?site=remark42&address=umputun%40example.com&user=Ann-Marie+O.+Smith", http.StatusForbidden},
		{"/auth/email/login?site=remark42&address=umputun%40example.com&user=claim+your+prize+at+evil-site.com", http.StatusForbidden},
		{"/auth/email/login?site=remark42&address=umputun%40example.com&user=a%09bc", http.StatusForbidden},
		{"/auth/email/login?site=remark42&address=umputun%40example.com&user=abc+", http.StatusForbidden},
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

func Test_isPlainEmail(t *testing.T) {
	for _, s := range []string{"a@example.com", "Some.User+tag@sub.example.com", "пользователь@example.com", "a?b#c@example.com"} {
		assert.True(t, isPlainEmail(s), s)
	}
	for _, s := range []string{
		"", "example.com", "a@", `"Some User" <a@example.com>`, "Some User <a@example.com>", "<a@example.com>",
		"a@example.com (comment)", "group: a@example.com;", `"free text"@example.com`, " a@example.com", "a@example.com ",
		"a@example.com,b@example.com", "a\u00a0b@example.com", "a\u202eb@example.com", "a@example.com\r\nBcc: b@example.com",
	} {
		assert.False(t, isPlainEmail(s), "%q", s)
	}
}

func Test_singleAuthPath(t *testing.T) {
	registered := func(name string) bool { return name == "email" || name == "github" }

	tbl := []struct {
		path   string
		status int
	}{
		{"/auth/status", http.StatusOK},
		{"/auth/user", http.StatusOK},
		{"/auth/list", http.StatusOK},
		{"/auth/logout", http.StatusOK},
		{"/auth/email/login", http.StatusOK},
		{"/auth/github/login", http.StatusOK},
		{"/auth/github/callback", http.StatusOK},
		// in front of an endpoint
		{"/auth/x/email/login", http.StatusNotFound},
		{"/auth/1/2/email/login", http.StatusNotFound},
		{"/auth/x%2Femail/login", http.StatusNotFound},
		{"/auth/email%2Flogin/login", http.StatusNotFound},
		{"/auth/x/status", http.StatusNotFound},
		{"/auth/x/list", http.StatusNotFound},
		{"/auth/x/logout", http.StatusNotFound},
		// a provider in front of a call that has none
		{"/auth/email/status", http.StatusNotFound},
		{"/auth/email/user", http.StatusNotFound},
		{"/auth/github/list", http.StatusNotFound},
		{"/auth/github/logout", http.StatusNotFound},
		// a provider's call without the provider, which the handler would read from the mount point
		{"/auth/login", http.StatusNotFound},
		{"/auth/callback", http.StatusNotFound},
		// after an endpoint
		{"/auth/email/login/", http.StatusNotFound},
		{"/auth/status/", http.StatusNotFound},
		// no endpoint
		{"/auth/", http.StatusNotFound},
		{"/auth/nothing-of-the-kind", http.StatusNotFound},
		{"/auth/email/nothing-of-the-kind", http.StatusNotFound},
		{"/auth/twitter/login", http.StatusNotFound}, // not registered
		{"/auth/EMAIL/login", http.StatusNotFound},   // the handler matches the name exactly too
	}

	for _, tt := range tbl {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com"+tt.path+"?site=remark42", http.NoBody)
			w := httptest.NewRecorder()
			singleAuthPath(registered)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(w, req)
			assert.Equal(t, tt.status, w.Code)
		})
	}
}

// regression: a rejected login printed the user, address or site as typed, which can be an email address
func Test_validEmailAuthLogHidesEmail(t *testing.T) {
	logs := &lockedBuffer{}
	lgr.Setup(lgr.Out(logs), lgr.Err(io.Discard))
	t.Cleanup(func() { lgr.Setup() })

	tbl := []struct{ name, query, want string }{
		{"user", "site=remark42&address=a%40example.com&user=secret.person%40example.com", "suspicious user rejected: ********"},
		{"address", "site=remark42&address=a%40example.com,secret.person%40example.com&user=someone",
			`suspicious address rejected: mail: expected single address, got ",********"`},
		{"site", "site=secret.person%40example.com&address=a%40example.com&user=someone", "suspicious site rejected: ********"},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com/auth/email/login?"+tt.query, http.NoBody)
			w := httptest.NewRecorder()
			validEmailAuth()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.Contains(t, logs.String(), tt.want)
		})
	}
	assert.NotContains(t, logs.String(), "secret.person")
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
				r = token.SetUserInfo(r, token.User{ID: "u", Name: "u", Audience: c.userSite})
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

func TestRest_rejectCrossSiteBasicAuth(t *testing.T) {
	cases := []struct {
		name      string
		remarkURL string
		basic     bool
		fetchSite string
		origin    string
		want      int
	}{
		{name: "no basic auth, cross-site", remarkURL: "https://r.example.com", fetchSite: "cross-site", origin: "https://evil.example", want: http.StatusOK},
		{name: "basic auth without browser signals", remarkURL: "https://r.example.com", basic: true, want: http.StatusOK},
		{name: "basic auth, cross-site", remarkURL: "https://r.example.com", basic: true, fetchSite: "cross-site", want: http.StatusForbidden},
		{name: "basic auth, same origin", remarkURL: "https://r.example.com", basic: true, fetchSite: "same-origin", origin: "https://r.example.com", want: http.StatusOK},
		{name: "basic auth, origin case differs", remarkURL: "https://R.example.com", basic: true, origin: "https://r.example.com", want: http.StatusOK},
		{name: "basic auth, same-site sibling origin", remarkURL: "https://r.example.com", basic: true, fetchSite: "same-site", origin: "https://blog.example.com", want: http.StatusForbidden},
		{name: "basic auth, foreign origin", remarkURL: "https://r.example.com", basic: true, origin: "https://evil.example", want: http.StatusForbidden},
		{name: "basic auth, null origin", remarkURL: "https://r.example.com", basic: true, origin: "null", want: http.StatusForbidden},
		{name: "basic auth, origin of a remark url with path", remarkURL: "https://example.com/remark42", basic: true, origin: "https://example.com", want: http.StatusOK},
		{name: "basic auth, remark url with default https port", remarkURL: "https://r.example.com:443", basic: true, origin: "https://r.example.com", want: http.StatusOK},
		{name: "basic auth, remark url with default http port", remarkURL: "http://r.example.com:80", basic: true, origin: "http://r.example.com", want: http.StatusOK},
		{name: "basic auth, origin with another port", remarkURL: "https://r.example.com:8443", basic: true, origin: "https://r.example.com", want: http.StatusForbidden},
		{name: "basic auth, unparseable remark url", remarkURL: "://bad", basic: true, origin: "https://r.example.com", want: http.StatusForbidden},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := &Rest{RemarkURL: c.remarkURL}
			h := srv.rejectCrossSiteBasicAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/blocked", http.NoBody)
			if c.basic {
				req.SetBasicAuth("admin", "password")
			}
			if c.fetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", c.fetchSite)
			}
			if c.origin != "" {
				req.Header.Set("Origin", c.origin)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			assert.Equal(t, c.want, rec.Code)
		})
	}
}

func TestRest_rejectEmbedded(t *testing.T) {
	cases := []struct {
		name string
		dest string
		want int
	}{
		{name: "no header, as a command-line client sends", dest: "", want: http.StatusOK},
		{name: "a call from script", dest: "empty", want: http.StatusOK},
		{name: "a page or a sign-in window", dest: "document", want: http.StatusOK},
		{name: "an image", dest: "image", want: http.StatusForbidden},
		{name: "a script", dest: "script", want: http.StatusForbidden},
		{name: "a stylesheet", dest: "style", want: http.StatusForbidden},
		{name: "a frame", dest: "iframe", want: http.StatusForbidden},
		{name: "a value no browser sends", dest: "Document", want: http.StatusForbidden},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			reached := false
			h := rejectEmbedded(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				reached = true
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodGet, "/auth/logout", http.NoBody)
			if c.dest != "" {
				req.Header.Set("Sec-Fetch-Dest", c.dest)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			assert.Equal(t, c.want, rec.Code)
			assert.Equal(t, c.want == http.StatusOK, reached, "the handler runs only for a request let through")
		})
	}
}

func Test_withoutQueryParam(t *testing.T) {
	cases := []struct{ name, query, want string }{
		{name: "nothing to remove", query: "site=remark&url=https://example.com/post?a=1", want: "site=remark&url=https://example.com/post?a=1"},
		{name: "removed from the end", query: "site=remark&jwt=abc.def.ghi", want: "site=remark"},
		{name: "removed from the middle, the rest untouched", query: "b=2&jwt=abc&a=%D1%8F+1", want: "b=2&a=%D1%8F+1"},
		{name: "the only parameter", query: "jwt=abc", want: ""},
		{name: "every occurrence", query: "jwt=abc&site=remark&jwt=def", want: "site=remark"},
		{name: "without a value", query: "jwt&site=remark", want: "site=remark"},
		{name: "the name written escaped", query: "%6Awt=abc&site=remark", want: "site=remark"},
		{name: "another name that contains it", query: "jwt2=abc&xjwt=def", want: "jwt2=abc&xjwt=def"},
		{name: "the name as a value", query: "sort=jwt", want: "sort=jwt"},
		{name: "a name that can't be unescaped stays", query: "%zz=1&jwt=abc", want: "%zz=1"},
		{name: "empty pairs around it stay", query: "&&jwt=abc&&a=1&", want: "&&&a=1&"},
		{name: "two in a row at the start", query: "jwt=a&jwt=b&site=remark", want: "site=remark"},
		{name: "nothing else left but an empty pair", query: "jwt=abc&", want: ""},
		{name: "an empty pair ahead of it stays", query: "&jwt=a&b=1", want: "&b=1"},
		{name: "an empty query", query: "", want: ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, withoutQueryParam(c.query, QueryTokenParam))
		})
	}
}

// a token in the query must not authenticate a request, and must not get in the way of one that
// carries its token where it belongs
func TestRest_QueryTokenIgnored(t *testing.T) {
	ts, _ := startupT(t)

	status := func(t *testing.T, path, headerToken string) int {
		_, code := call(t, http.MethodGet, ts.URL+path, "", headerToken)
		return code
	}

	assert.Equal(t, http.StatusOK, status(t, "/api/v1/user?site=remark42", devToken), "the token is good in the header")
	assert.Equal(t, http.StatusUnauthorized, status(t, "/api/v1/user?site=remark42&jwt="+devToken, ""),
		"the same token in the query signs nobody in")
	assert.Equal(t, http.StatusOK, status(t, "/api/v1/user?site=remark42&jwt=not.a.token", devToken),
		"a token in the query does not replace the one in the header")
}

// the sign-in and sign-out routes set and expire the session cookies on a plain GET, so they must
// not answer a request the browser makes for an image
func TestRest_AuthRefusesEmbeddedRequests(t *testing.T) {
	const login = "/auth/anonymous/login?user=reader&passwd=x&aud=remark42"
	cases := []struct {
		name, path, dest string
		want             int
		cookies          bool // the answer sets or expires the session cookies
	}{
		{name: "sign-in from script", path: login, dest: "empty", want: http.StatusOK, cookies: true},
		{name: "sign-in as an image", path: login, dest: "image", want: http.StatusForbidden},
		{name: "sign-out from script", path: "/auth/logout", dest: "empty", want: http.StatusOK, cookies: true},
		{name: "sign-out from a page", path: "/auth/logout", dest: "document", want: http.StatusOK, cookies: true},
		{name: "sign-out by a client that sends no fetch metadata", path: "/auth/logout", want: http.StatusOK, cookies: true},
		{name: "sign-out as an image", path: "/auth/logout", dest: "image", want: http.StatusForbidden},
		{name: "sign-out as an image, named after a provider", path: "/auth/anonymous/logout", dest: "image", want: http.StatusForbidden},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// a server each: the auth routes take two requests a second from an address
			ts, _ := startupT(t)

			req, err := http.NewRequest(http.MethodGet, ts.URL+c.path, http.NoBody)
			require.NoError(t, err)
			if c.dest != "" {
				req.Header.Set("Sec-Fetch-Dest", c.dest)
			}
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			assert.Equal(t, c.want, resp.StatusCode)
			if c.cookies {
				assert.NotEmpty(t, resp.Header.Values("Set-Cookie"))
			} else {
				assert.Empty(t, resp.Header.Values("Set-Cookie"), "a refused request must not touch the session")
			}
		})
	}
}

// requests refused as embedded do not count against the two a second the auth routes take from
// an address, so they cannot use up the allowance of the reader's own sign-out
func TestRest_AuthRefusalsAreNotRateLimited(t *testing.T) {
	ts, _ := startupT(t)

	// the status, and the header the limiter puts on every request that reaches it
	get := func(dest string) (status int, limit string) {
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/auth/logout", http.NoBody)
		require.NoError(t, err)
		req.Header.Set("Sec-Fetch-Dest", dest)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode, resp.Header.Get("X-Rate-Limit-Limit")
	}

	for range 5 {
		status, limit := get("image")
		assert.Equal(t, http.StatusForbidden, status)
		assert.Empty(t, limit, "a refused request must not reach the limiter")
	}
	status, limit := get("empty")
	assert.Equal(t, http.StatusOK, status)
	assert.NotEmpty(t, limit, "a request let through is limited as before")
}

func TestCorsMiddleware(t *testing.T) {
	h := corsMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("credentialed cross-origin reflects the request origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		// AllowedOrigins "*" with credentials must reflect the origin, never a literal "*"
		assert.Equal(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
		assert.Equal(t, "Authorization", rec.Header().Get("Access-Control-Expose-Headers"))
	})

	t.Run("preflight advertises configured methods, headers and max-age", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/", http.NoBody)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "X-JWT")
		assert.Equal(t, "300", rec.Header().Get("Access-Control-Max-Age"))
		// preflight responses must vary on origin and the request method/headers so caches
		// don't reuse one preflight across different requests
		vary := rec.Header().Values("Vary")
		assert.Contains(t, vary, "Origin")
		assert.Contains(t, vary, "Access-Control-Request-Method")
		assert.Contains(t, vary, "Access-Control-Request-Headers")
	})

	t.Run("same-origin request (no Origin) gets no CORS headers", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))
		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestRecoverer_MasksAddressInURL(t *testing.T) {
	var lines []string
	l := lgr.Func(func(format string, args ...any) { lines = append(lines, fmt.Sprintf(format, args...)) })
	h := recoverer(l)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("oops") }))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/email/login?address=user@example.com&site=remark42", http.NoBody))
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	require.Len(t, lines, 2, "the recovery line and the stack trace")
	assert.Contains(t, lines[0], "request panic for /auth/email/login?address=********&site=remark42")
	assert.Contains(t, lines[0], "oops")
	assert.NotContains(t, strings.Join(lines, "\n"), "user@example.com", "the address stays out of the log")
	assert.Contains(t, lines[1], "TestRecoverer_MasksAddressInURL", "the stack trace names the panicking code")
}

func TestPanicLog_LeavesStackTraceAlone(t *testing.T) {
	var lines []string
	l := lgr.Func(func(format string, args ...any) { lines = append(lines, fmt.Sprintf(format, args...)) })

	// a module path holds a run the mask would take for an address
	const stack = "goroutine 1 [running]:\n/go/pkg/mod/github.com/foo/bar@v1.2.3/x.go:12 +0x1d"
	panicLog{l: l}.Logf(stack)
	assert.Equal(t, []string{stack}, lines)
}

// TestPartitionedAuthCookies covers the middleware on its own: which Set-Cookie lines gain a
// partitioned copy, where the copy goes, and that a request without the header is answered as before.
func TestPartitionedAuthCookies(t *testing.T) {
	const (
		jwtLine      = "JWT=tkn; Path=/; Max-Age=720000; HttpOnly; Secure; SameSite=Lax"
		jwtCopy      = "JWT=tkn; Path=/; Max-Age=720000; HttpOnly; Secure; SameSite=None; Partitioned"
		xsrfLine     = "XSRF-TOKEN=id; Path=/; Max-Age=720000; Secure; SameSite=Lax"
		xsrfCopy     = "XSRF-TOKEN=id; Path=/; Max-Age=720000; Secure; SameSite=None; Partitioned"
		otherLine    = "other=v; Path=/"
		expiredLine  = "JWT=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0; HttpOnly"
		expiredCopy  = "JWT=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0; HttpOnly; Secure; SameSite=None; Partitioned"
		partitionedL = "JWT=tkn; Path=/; HttpOnly; Secure; SameSite=None; Partitioned"
	)
	signIn := func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "JWT", Value: "tkn", Path: "/", MaxAge: 720000, HttpOnly: true,
			Secure: true, SameSite: http.SameSiteLaxMode})
		http.SetCookie(w, &http.Cookie{Name: "XSRF-TOKEN", Value: "id", Path: "/", MaxAge: 720000,
			Secure: true, SameSite: http.SameSiteLaxMode})
		http.SetCookie(w, &http.Cookie{Name: "other", Value: "v", Path: "/"})
	}
	expire := func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "JWT", Path: "/", MaxAge: -1, Expires: time.Unix(0, 0), HttpOnly: true})
	}

	tbl := []struct {
		name    string
		header  string
		noIssue bool // the server issues no partitioned pair, which is header mode switched off
		handler http.HandlerFunc
		want    []string
	}{
		{name: "header and a body written without a status", header: "1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				signIn(w, r)
				_, _ = w.Write([]byte("ok"))
			},
			// each copy goes before its original, so a browser that reads both lines as one cookie
			// keeps the original
			want: []string{jwtCopy, jwtLine, xsrfCopy, xsrfLine, otherLine}},
		{name: "header and an explicit status", header: "1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				signIn(w, r)
				w.WriteHeader(http.StatusUnauthorized)
			},
			want: []string{jwtCopy, jwtLine, xsrfCopy, xsrfLine, otherLine}},
		{name: "header and a flush before anything is written", header: "1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				signIn(w, r)
				assert.NoError(t, http.NewResponseController(w).Flush())
			},
			want: []string{jwtCopy, jwtLine, xsrfCopy, xsrfLine, otherLine}},
		{name: "header and a handler that writes nothing", header: "1", handler: signIn,
			// net/http sends the headers itself here and never calls the writer, so there is no
			// moment to add the copies at. no handler setting an auth cookie ends this way
			want: []string{jwtLine, xsrfLine, otherLine}},
		{name: "an expiry is copied as an expiry", header: "1",
			handler: func(w http.ResponseWriter, r *http.Request) {
				expire(w, r)
				w.WriteHeader(http.StatusOK)
			},
			want: []string{expiredCopy, expiredLine}},
		{name: "a cookie that is partitioned already is left alone", header: "1",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.SetCookie(w, &http.Cookie{Name: "JWT", Value: "tkn", Path: "/", HttpOnly: true, Secure: true,
					SameSite: http.SameSiteNoneMode, Partitioned: true})
				w.WriteHeader(http.StatusOK)
			},
			want: []string{partitionedL}},
		{name: "no header", header: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				signIn(w, r)
				w.WriteHeader(http.StatusOK)
			},
			want: []string{jwtLine, xsrfLine, otherLine}},
		{name: "a header with another value is not the request", header: "true",
			handler: func(w http.ResponseWriter, r *http.Request) {
				signIn(w, r)
				w.WriteHeader(http.StatusOK)
			},
			want: []string{jwtLine, xsrfLine, otherLine}},
		{name: "an expiry is copied without the header too", header: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				expire(w, r)
				w.WriteHeader(http.StatusOK)
			},
			want: []string{expiredCopy, expiredLine}},
		{name: "no pair is issued where the server issues none, header or not", header: "1", noIssue: true,
			handler: func(w http.ResponseWriter, r *http.Request) {
				signIn(w, r)
				w.WriteHeader(http.StatusOK)
			},
			want: []string{jwtLine, xsrfLine, otherLine}},
		{name: "an expiry is copied where the server issues no pair", header: "", noIssue: true,
			handler: func(w http.ResponseWriter, r *http.Request) {
				expire(w, r)
				w.WriteHeader(http.StatusOK)
			},
			want: []string{expiredCopy, expiredLine}},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			if tt.header != "" {
				req.Header.Set(partitionedCookieHeader, tt.header)
			}
			rec := httptest.NewRecorder()
			partitionedAuthCookies(!tt.noIssue, "JWT", "XSRF-TOKEN")(tt.handler).ServeHTTP(rec, req)
			assert.Equal(t, tt.want, rec.Result().Header.Values("Set-Cookie"))
		})
	}
}

// TestRest_ThirdPartyFrameCookies drives the real routes, since the middleware is only worth
// anything where the auth library writes its cookies: at sign-in, when the auth middleware renews an
// expired token on an api route, and at sign-out. A widget framed by another host has to be handed a
// partitioned pair at each of them, and nobody else's response may change.
func TestRest_ThirdPartyFrameCookies(t *testing.T) {
	// what a case reads of an answer; the body is closed where the request is made
	type response struct {
		status int
		header http.Header
	}
	headerMode := func(srv *Rest) { srv.SendJWTHeader = true }
	// the lines for one cookie name, in the order they were sent
	lines := func(resp response, name string) (res []string) {
		for _, line := range resp.header.Values("Set-Cookie") {
			if strings.HasPrefix(line, name+"=") {
				res = append(res, line)
			}
		}
		return res
	}
	get := func(t *testing.T, url string, headers map[string]string) response {
		req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
		require.NoError(t, err)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		return response{status: resp.StatusCode, header: resp.Header}
	}
	assertBothForms := func(t *testing.T, resp response) {
		for _, name := range []string{"JWT", "XSRF-TOKEN"} {
			got := lines(resp, name)
			require.Len(t, got, 2, "%s has to be set twice, partitioned and as before: %v", name, got)
			assert.Contains(t, got[0], "; Partitioned", "the partitioned %s comes first", name)
			assert.Contains(t, got[0], "; SameSite=None")
			assert.Contains(t, got[0], "; Secure")
			assert.NotContains(t, got[1], "Partitioned", "the %s set as before stays as it was", name)
		}
	}
	assertOneForm := func(t *testing.T, resp response) {
		for _, name := range []string{"JWT", "XSRF-TOKEN"} {
			got := lines(resp, name)
			require.Len(t, got, 1, "%s is set once: %v", name, got)
			assert.NotContains(t, got[0], "Partitioned")
		}
	}
	const login = "/auth/anonymous/login?user=reader&passwd=x&aud=remark42"
	frame := map[string]string{partitionedCookieHeader: "1"}

	t.Run("sign-in from a third-party frame", func(t *testing.T) {
		ts, _ := startupT(t, headerMode)

		resp := get(t, ts.URL+login, frame)
		require.Equal(t, http.StatusOK, resp.status)
		assertBothForms(t, resp)
	})

	t.Run("renewal on an api route from a third-party frame", func(t *testing.T) {
		ts, srv := startupT(t, headerMode)

		// expired but validly signed, which is what the auth middleware renews for a cookie
		tkn, err := srv.Authenticator.TokenService().Token(token.Claims{
			ID: "xsrf-id", Audience: jwt.ClaimStrings{"remark42"},
			Issuer: "remark42", ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			User: &token.User{ID: "anonymous_reader", Name: "reader"},
		})
		require.NoError(t, err)

		resp := get(t, ts.URL+"/api/v1/user?site=remark42", map[string]string{
			partitionedCookieHeader: "1", "Cookie": "JWT=" + tkn, "X-XSRF-TOKEN": "xsrf-id"})
		require.Equal(t, http.StatusOK, resp.status)
		assertBothForms(t, resp)
	})

	t.Run("sign-out from a third-party frame", func(t *testing.T) {
		ts, _ := startupT(t, headerMode)

		resp := get(t, ts.URL+"/auth/logout", frame)
		require.Equal(t, http.StatusOK, resp.status)
		assertBothForms(t, resp)
		for _, line := range lines(resp, "JWT") {
			assert.Contains(t, line, "Max-Age=0", "both forms have to be expired, or one outlives the sign-out")
		}
	})

	t.Run("sign-in without the header", func(t *testing.T) {
		ts, _ := startupT(t, headerMode)

		resp := get(t, ts.URL+login, nil)
		require.Equal(t, http.StatusOK, resp.status)
		assertOneForm(t, resp)
	})

	t.Run("sign-out without the header", func(t *testing.T) {
		ts, _ := startupT(t, headerMode)

		// remark42's own page signing out a reader whose widget, framed by another host of the
		// same site, was given the partitioned pair: that pair is sent here as well
		resp := get(t, ts.URL+"/auth/logout", nil)
		require.Equal(t, http.StatusOK, resp.status)
		assertBothForms(t, resp)
	})

	t.Run("sign-out with header mode off", func(t *testing.T) {
		ts, _ := startupT(t)

		// a reader given the partitioned pair while header mode was on still holds it. its JWT is
		// HttpOnly, so unless this response expires it, it outlives the sign-out and is sent ahead
		// of the next session's cookie
		for _, headers := range []map[string]string{frame, nil} {
			resp := get(t, ts.URL+"/auth/logout", headers)
			require.Equal(t, http.StatusOK, resp.status)
			assertBothForms(t, resp)
			for _, name := range []string{"JWT", "XSRF-TOKEN"} {
				for _, line := range lines(resp, name) {
					assert.Contains(t, line, "Max-Age=0", "both forms of %s have to be expired", name)
				}
			}
		}
	})

	t.Run("sign-in with header mode off", func(t *testing.T) {
		ts, _ := startupT(t)

		resp := get(t, ts.URL+login, frame)
		require.Equal(t, http.StatusOK, resp.status)
		assertOneForm(t, resp)
	})

	t.Run("served over http", func(t *testing.T) {
		ts, _ := startupT(t, headerMode, func(srv *Rest) { srv.RemarkURL = "http://demo.remark42.com" })

		// a partitioned cookie has to be Secure, and a browser refuses a Secure cookie over http
		resp := get(t, ts.URL+login, frame)
		require.Equal(t, http.StatusOK, resp.status)
		assertOneForm(t, resp)

		resp = get(t, ts.URL+"/auth/logout", frame)
		require.Equal(t, http.StatusOK, resp.status)
		assertOneForm(t, resp)
	})
}
