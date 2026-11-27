package api

import (
	"crypto/sha1" //nolint:gosec // the auth library's id scheme
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-pkgz/auth/v2"
	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/provider"
	"github.com/go-pkgz/auth/v2/token"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/templates"
)

// lastMessage keeps the message a verify provider sends
type lastMessage struct {
	mu   sync.Mutex
	text string
}

func (m *lastMessage) Send(_, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.text = text
	return nil
}

func (m *lastMessage) get() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.text
}

// startupWithEmailLogin starts the test server with the email login registered as the server
// registers it, with the message template the server ships, and returns what it sends
// opts, when given, adjust the authenticator's options before it is made
func startupWithEmailLogin(t *testing.T, opts ...func(*auth.Opts)) (ts string, srv *Rest, sent *lastMessage) {
	t.Helper()
	sent = &lastMessage{}
	tmpl, err := templates.Read("email_confirmation_login.html.tmpl")
	require.NoError(t, err)
	server, srv := startupT(t, func(srv *Rest) {
		// the shared setup registers "email" as a direct provider, and the first one by a name wins
		o := auth.Opts{
			URL:          srv.RemarkURL,
			Issuer:       "remark42",
			SecretReader: token.SecretFunc(func(string) (string, error) { return "secret", nil }),
			JWTQuery:     QueryTokenParam, // as in the server, "token" carries the delete-me token
			// as in the server, which marks a blocked user's session so that it is refused everywhere
			ClaimsUpd: token.ClaimsUpdFunc(func(c token.Claims) token.Claims {
				if c.User != nil && len(c.Audience) == 1 {
					c.User.SetBoolAttr("blocked", srv.DataService.IsBlocked(c.Audience[0], c.User.ID))
				}
				return c
			}),
		}
		for _, f := range opts {
			f(&o)
		}
		srv.Authenticator = auth.NewService(o)
		srv.Authenticator.AddVerifProvider("email", string(tmpl), sent)
	})
	return server.URL, srv, sent
}

// requestLoginEmail asks for a login message for "reader one" and returns the address of its link and the token to paste.
// extra goes into the request as it is
func requestLoginEmail(t *testing.T, ts string, sent *lastMessage, address string, extra ...string) (link, pasteToken string) {
	t.Helper()
	q := url.Values{"user": {"reader one"}, "address": {address}, "site": {"remark42"}}
	for i := 0; i+1 < len(extra); i += 2 {
		q.Set(extra[i], extra[i+1])
	}
	resp, err := http.Get(ts + "/auth/email/login?" + q.Encode())
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)

	msg := sent.get()
	m := regexp.MustCompile(`href="(https://demo\.remark42\.com/email/login\.html\?tkn=[^"]+)"`).FindStringSubmatch(msg)
	require.Len(t, m, 2, "no sign-in link in the message:\n%s", msg)
	p := regexp.MustCompile(`user-select: all;">([\w-]+\.[\w-]+\.[\w-]+)<`).FindStringSubmatch(msg)
	require.Len(t, p, 2, "no token to paste in the message:\n%s", msg)
	// the link names the public address, the test server listens elsewhere
	return strings.TrimPrefix(html.UnescapeString(m[1]), "https://demo.remark42.com"), p[1]
}

func TestRest_EmailLoginLink(t *testing.T) {
	ts, srv, sent := startupWithEmailLogin(t)
	link, pasteToken := requestLoginEmail(t, ts, sent, "reader@example.com")
	readerID := "email_" + token.HashID(sha1.New(), "reader@example.com") //nolint:gosec // the auth library's id scheme

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	browser := &http.Client{Jar: jar, Timeout: 10 * time.Second}
	other := &http.Client{Timeout: 10 * time.Second} // a browser holding no cookies
	t.Cleanup(browser.CloseIdleConnections)
	t.Cleanup(other.CloseIdleConnections)
	do := func(c *http.Client, method, path string) (status int, body string, hdr http.Header) {
		t.Helper()
		req, e := http.NewRequest(method, ts+path, http.NoBody)
		require.NoError(t, e)
		resp, e := c.Do(req)
		require.NoError(t, e)
		b, e := io.ReadAll(resp.Body)
		require.NoError(t, e)
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode, string(b), resp.Header
	}
	tsURL, err := url.Parse(ts)
	require.NoError(t, err)
	session := func() *token.Claims {
		for _, c := range jar.Cookies(tsURL) {
			if c.Name == "JWT" {
				claims, e := srv.Authenticator.TokenService().Parse(c.Value)
				require.NoError(t, e)
				return &claims
			}
		}
		return nil
	}

	// a session of the same reader on another site served from this host does not count
	otherSite, err := srv.Authenticator.TokenService().Token(token.Claims{
		User: &token.User{ID: readerID, Name: "reader one"},
		RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{"another-site"}, Issuer: "remark42", ID: "x",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodGet, ts+link, http.NoBody)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "JWT", Value: otherSite})
	resp, err := other.Do(req)
	require.NoError(t, err)
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Contains(t, string(b), `<form id="confirm" method="post">`, "signed in on another site only")

	// nor does a session of this reader under another name: the link signs in with the name it was asked for
	otherName, err := srv.Authenticator.TokenService().Token(token.Claims{
		User: &token.User{ID: readerID, Name: "reader two"},
		RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{"remark42"}, Issuer: "remark42", ID: "z",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	})
	require.NoError(t, err)
	req, err = http.NewRequest(http.MethodGet, ts+link, http.NoBody)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "JWT", Value: otherName})
	resp, err = other.Do(req)
	require.NoError(t, err)
	b, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Contains(t, string(b), `<form id="confirm" method="post">`, "signed in under another name")

	status, body, hdr := do(browser, http.MethodGet, link)
	require.Equal(t, http.StatusOK, status, body)
	assert.Contains(t, body, "Sign in as <b>reader one</b> on site <b>remark42</b>")
	assert.Contains(t, body, `<form id="confirm" method="post">`)
	assert.Contains(t, hdr.Get("Content-Security-Policy"), "form-action 'self'", "the global policy would block the form")
	assert.Contains(t, hdr.Get("Content-Security-Policy"), "frame-ancestors 'none'")
	assert.Equal(t, "no-referrer", hdr.Get("Referrer-Policy"), "the token in the address must not leave in a Referer")
	nonce := regexp.MustCompile(`script-src 'nonce-([A-Z2-7]{26})'`).FindStringSubmatch(hdr.Get("Content-Security-Policy"))
	require.Len(t, nonce, 2, hdr.Get("Content-Security-Policy"))
	assert.Contains(t, body, `<script nonce="`+nonce[1]+`">`)
	assert.Empty(t, hdr.Values("Set-Cookie"), "GET must not sign in, mail scanners fetch links")
	status, _, hdr = do(browser, http.MethodHead, link)
	assert.Equal(t, http.StatusOK, status)
	assert.Empty(t, hdr.Values("Set-Cookie"), "HEAD must not sign in")
	assert.Nil(t, session())

	status, body, _ = do(browser, http.MethodPost, link)
	require.Equal(t, http.StatusOK, status, body)
	assert.Contains(t, body, "Signed in as <b>reader one</b> on site <b>remark42</b>")
	claims := session()
	require.NotNil(t, claims, "the link must leave a session in the browser that opened it")
	assert.Equal(t, readerID, claims.User.ID)
	assert.Equal(t, "reader one", claims.User.Name)
	assert.Equal(t, []string{"remark42"}, []string(claims.Audience))

	// a reload or a second press in the browser the link signed in
	status, body, _ = do(browser, http.MethodPost, link)
	assert.Equal(t, http.StatusOK, status, body)
	assert.Contains(t, body, "Signed in as <b>reader one</b>")
	status, body, _ = do(browser, http.MethodGet, link)
	assert.Equal(t, http.StatusOK, status, body)
	assert.Contains(t, body, "Signed in as <b>reader one</b>")
	assert.NotContains(t, body, "<form")

	// a session whose short-lived token has expired is still this reader's: the cookie outlives the token
	// and the next request of the comments renews it
	expired, err := srv.Authenticator.TokenService().Token(token.Claims{
		User: &token.User{ID: readerID, Name: "reader one"},
		RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{"remark42"}, Issuer: "remark42", ID: "y",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))},
	})
	require.NoError(t, err)
	req, err = http.NewRequest(http.MethodGet, ts+link, http.NoBody)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "JWT", Value: expired})
	resp, err = other.Do(req)
	require.NoError(t, err)
	b, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Contains(t, string(b), "Signed in as <b>reader one</b>", "a session past its token's expiry is not counted")

	// the link works once: another browser gets the refusal and no session
	status, body, hdr = do(other, http.MethodPost, link)
	assert.Equal(t, http.StatusForbidden, status)
	assert.Contains(t, body, "this sign-in link was already used or has expired: ask for a new email from the comments.")
	assert.Empty(t, hdr.Values("Set-Cookie"))

	// the token to paste was not used up by the link
	status, body, hdr = do(other, http.MethodGet, "/auth/email/login?token="+pasteToken)
	assert.Equal(t, http.StatusOK, status, body)
	assert.NotEmpty(t, hdr.Values("Set-Cookie"))
}

func TestRest_EmailLoginLinkRejects(t *testing.T) {
	ts, srv, sent := startupWithEmailLogin(t)
	_, pasteToken := requestLoginEmail(t, ts, sent, "reader@example.com")

	mint := func(c token.Claims) string {
		t.Helper()
		if c.ExpiresAt == nil {
			c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(10 * time.Minute))
		}
		c.NotBefore = jwt.NewNumericDate(time.Now().Add(-1 * time.Minute))
		c.Issuer = "remark42"
		tkn, err := srv.Authenticator.TokenService().Token(c)
		require.NoError(t, err)
		return tkn
	}
	linkHandshake := &token.Handshake{ID: "reader one::reader@example.com", State: provider.VerifLinkTokenState}
	email := &token.AuthProvider{Name: "email"}
	site := jwt.ClaimStrings{"remark42"}

	tbl := []struct {
		name, tkn, details string
		code               int
	}{
		{name: "no token", code: http.StatusBadRequest, details: "token parameter is required"},
		{name: "not a token", tkn: "jwt", code: http.StatusForbidden, details: "this sign-in link is not valid"},
		{name: "the token to paste", tkn: pasteToken, code: http.StatusBadRequest, details: "this sign-in link is not valid"},
		{name: "expired", code: http.StatusForbidden, details: "this sign-in link has expired",
			tkn: mint(token.Claims{Handshake: linkHandshake, AuthProvider: email, Audience: site,
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute))})},
		{name: "subscription token", code: http.StatusBadRequest, details: "this sign-in link is not valid",
			tkn: handshakeToken(t, srv, "provider1_dev::reader@example.com", emailSubscribeProvider)},
		{name: "unsubscribe token", code: http.StatusBadRequest, details: "this sign-in link is not valid",
			tkn: handshakeToken(t, srv, "provider1_dev::reader@example.com", "")},
		{name: "session token", code: http.StatusBadRequest, details: "this sign-in link is not valid",
			tkn: mint(token.Claims{User: &token.User{ID: "email_123", Name: "reader one"}, Audience: site})},
		{name: "another provider's link", code: http.StatusBadRequest, details: "this sign-in link is not valid",
			tkn: mint(token.Claims{Handshake: linkHandshake, AuthProvider: &token.AuthProvider{Name: "telegram"}, Audience: site})},
		{name: "no provider", code: http.StatusBadRequest, details: "this sign-in link is not valid",
			tkn: mint(token.Claims{Handshake: linkHandshake, Audience: site})},
		{name: "no site", code: http.StatusBadRequest, details: "this sign-in link is not valid",
			tkn: mint(token.Claims{Handshake: linkHandshake, AuthProvider: email, Audience: jwt.ClaimStrings{""}})},
	}
	client := &http.Client{Timeout: 10 * time.Second}
	t.Cleanup(client.CloseIdleConnections)
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			for _, method := range []string{http.MethodGet, http.MethodPost} {
				path := "/email/login.html"
				if tt.tkn != "" {
					path += "?tkn=" + tt.tkn
				}
				req, err := http.NewRequest(method, ts+path, http.NoBody)
				require.NoError(t, err)
				resp, err := client.Do(req)
				require.NoError(t, err)
				b, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
				assert.Equal(t, tt.code, resp.StatusCode, "%s: %s", method, b)
				assert.Contains(t, string(b), tt.details, method)
				assert.Empty(t, resp.Header.Values("Set-Cookie"), method)
			}
		})
	}

	// none of the refusals spent the token to paste
	resp, err := client.Get(ts + "/auth/email/login?token=" + pasteToken)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRest_EmailLoginLinkBlockedUser(t *testing.T) {
	ts, srv, sent := startupWithEmailLogin(t)
	link, _ := requestLoginEmail(t, ts, sent, "reader@example.com")
	readerID := "email_" + token.HashID(sha1.New(), "reader@example.com") //nolint:gosec // the auth library's id scheme
	// a session issued before the block, which the comments have not renewed since
	before, err := srv.Authenticator.TokenService().Token(token.Claims{
		User: &token.User{ID: readerID, Name: "reader one"},
		RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{"remark42"}, Issuer: "remark42", ID: "z",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))},
	})
	require.NoError(t, err)
	require.NoError(t, srv.DataService.SetBlock("remark42", readerID, true, 0))

	client := &http.Client{Timeout: 10 * time.Second}
	t.Cleanup(client.CloseIdleConnections)
	req, err := http.NewRequest(http.MethodGet, ts+link, http.NoBody)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "JWT", Value: before})
	resp, err := client.Do(req)
	require.NoError(t, err)
	page, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.NotContains(t, string(page), "Signed in", "a session from before the block counted")

	resp, err = client.Post(ts+link, "", http.NoBody)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, string(body))
	assert.Contains(t, string(body), "user is blocked")
	assert.NotContains(t, string(body), "Signed in")
	assert.Empty(t, resp.Header.Values("Set-Cookie"), "a session refused everywhere is not handed out")
}

func TestRest_EmailLoginLinkKeepsSessionOnly(t *testing.T) {
	ts, _, sent := startupWithEmailLogin(t)
	client := &http.Client{Timeout: 10 * time.Second}
	t.Cleanup(client.CloseIdleConnections)

	for _, tc := range []struct {
		name    string
		extra   []string
		session bool
	}{
		{name: "asked for a session-only sign-in", extra: []string{"session", "1"}, session: true},
		{name: "asked for a lasting sign-in", session: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			link, _ := requestLoginEmail(t, ts, sent, strings.ReplaceAll(tc.name, " ", "")+"@example.com", tc.extra...)
			resp, err := client.Post(ts+link, "", http.NoBody)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			require.Equal(t, http.StatusOK, resp.StatusCode)
			var jwtCookie *http.Cookie
			for _, c := range resp.Cookies() {
				if c.Name == "JWT" {
					jwtCookie = c
				}
			}
			require.NotNil(t, jwtCookie, "no session cookie")
			assert.Equal(t, tc.session, jwtCookie.MaxAge == 0 && jwtCookie.Expires.IsZero(),
				"the cookie lasts %d s, expires %v", jwtCookie.MaxAge, jwtCookie.Expires)
		})
	}
}

// TestRest_EmailLoginLinkProviderFailure: a failure of the server itself, here an avatar store it cannot write,
// is reported as one and not as a used or expired link the reader would ask a new email for in vain
func TestRest_EmailLoginLinkProviderFailure(t *testing.T) {
	notADir := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(notADir, nil, 0o600))
	ts, _, sent := startupWithEmailLogin(t, func(o *auth.Opts) { o.AvatarStore = avatar.NewLocalFS(notADir) })
	link, _ := requestLoginEmail(t, ts, sent, "reader@example.com")

	client := &http.Client{Timeout: 10 * time.Second}
	t.Cleanup(client.CloseIdleConnections)
	resp, err := client.Post(ts+link, "", http.NoBody)
	require.NoError(t, err)
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, string(b))
	assert.NotContains(t, string(b), "already used")
	assert.Empty(t, resp.Header.Values("Set-Cookie"))
}
