package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-pkgz/auth/v2"
	"github.com/go-pkgz/auth/v2/provider"
	"github.com/go-pkgz/auth/v2/token"
	"github.com/go-pkgz/lgr"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// browserJar is the cookies one storage partition of a browser holds: the popup's first-party jar,
// or the jar of the frame on another site. Only name and value matter to the server
type browserJar map[string]string

func (j browserJar) header() string {
	pairs := make([]string, 0, len(j))
	for k, v := range j {
		pairs = append(pairs, k+"="+v)
	}
	return strings.Join(pairs, "; ")
}

// take applies the Set-Cookie lines of a response and returns them parsed, by name
func (j browserJar) take(t *testing.T, hdr http.Header) map[string]*http.Cookie {
	t.Helper()
	set := map[string]*http.Cookie{}
	for _, line := range hdr.Values("Set-Cookie") {
		c, err := http.ParseSetCookie(line)
		require.NoError(t, err, line)
		set[c.Name] = c
		if c.MaxAge < 0 || c.Value == "" {
			delete(j, c.Name)
			continue
		}
		j[c.Name] = c.Value
	}
	return set
}

// fakeIDP is an OAuth2 provider on a third site. Its authorize step signs in whoever the test names
// with the "as" parameter, and passes that name on through the code and the access token
func fakeIDP(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /authorize", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		http.Redirect(w, r, q.Get("redirect_uri")+"?"+url.Values{"code": {"c-" + q.Get("as")}, "state": {q.Get("state")}}.Encode(),
			http.StatusFound)
	})
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.NotEmpty(t, r.PostForm.Get("code_verifier"), "the library keeps using PKCE")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"access_token":"at-%s","token_type":"bearer"}`, strings.TrimPrefix(r.PostForm.Get("code"), "c-"))
	})
	mux.HandleFunc("GET /user", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer at-")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"id":%q,"name":%q}`, name, name)
	})
	idp := httptest.NewServer(mux)
	t.Cleanup(idp.Close)
	return idp
}

// handoffEnv is a remark42 with the fake provider registered as "fake", and a browser: the popup's
// first-party jar and the jar of a frame embedded on another site, which shares nothing with it
type handoffEnv struct {
	t     *testing.T
	ts    *httptest.Server
	srv   *Rest
	idp   *httptest.Server
	popup browserJar
	frame browserJar
	seq   int // requests sent, for the client address of the next one
}

func newHandoffEnv(t *testing.T, hooks ...func(srv *Rest)) *handoffEnv {
	t.Helper()
	idp := fakeIDP(t)
	hooks = append([]func(srv *Rest){func(srv *Rest) {
		srv.handoffWait = 50 * time.Millisecond
		srv.Authenticator.AddCustomProvider("fake", auth.Client{Cid: "cid", Csecret: "csecret"}, provider.CustomHandlerOpt{
			Endpoint: oauth2.Endpoint{AuthURL: idp.URL + "/authorize", TokenURL: idp.URL + "/token", AuthStyle: oauth2.AuthStyleInParams},
			InfoURL:  idp.URL + "/user",
			MapUserFn: func(data provider.UserData, _ []byte) token.User {
				return token.User{ID: "fake_" + data.Value("id"), Name: data.Value("name")}
			},
		})
	}}, hooks...)
	ts, srv := startupT(t, hooks...)
	return &handoffEnv{t: t, ts: ts, srv: srv, idp: idp, popup: browserJar{}, frame: browserJar{}}
}

// local turns an absolute url on the configured remark42 address into one on the test server
func (e *handoffEnv) local(u string) string {
	if strings.HasPrefix(u, "/") {
		return e.ts.URL + u
	}
	return strings.Replace(u, e.srv.RemarkURL, e.ts.URL, 1)
}

// reply is what do keeps of a response, whose body it has read and closed
type reply struct {
	StatusCode int
	Header     http.Header
}

// do sends a request with the cookies of jar, applies what it sets, and does not follow redirects
func (e *handoffEnv) do(jar browserJar, method, u string, body any, hdr map[string]string) (resp reply, set map[string]*http.Cookie, text string) {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(e.t, err)
		rd = strings.NewReader(string(b))
	}
	req, err := http.NewRequest(method, e.local(u), rd)
	require.NoError(e.t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c := jar.header(); c != "" {
		req.Header.Set("Cookie", c)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	// the auth routes take two requests a second from one address on a path, and a test asks faster
	// than that. Each request comes from an address of its own, which the server takes from X-Real-IP
	// with no trusted proxy configured, so none waits on the limiter
	e.seq++
	req.Header.Set("X-Real-IP", fmt.Sprintf("8.8.%d.%d", e.seq/250, e.seq%250+1))
	client := http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	r, err := client.Do(req)
	require.NoError(e.t, err)
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	require.NoError(e.t, err)
	require.NotEqual(e.t, http.StatusTooManyRequests, r.StatusCode, "%s %s", method, u)
	return reply{StatusCode: r.StatusCode, Header: r.Header}, jar.take(e.t, r.Header), string(b)
}

// verifier returns a fresh verifier and its S256 challenge, as the widget makes one per attempt
func newVerifier(t *testing.T) (verifier, challenge string) {
	t.Helper()
	b := make([]byte, 32)
	_, err := rand.Read(b)
	require.NoError(t, err)
	verifier = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

const handoffFrom = "https://demo.remark42.com/web/iframe.html?selfClose"

// navigation is the fetch metadata of a top-level navigation, which the popup's request is
var navigation = map[string]string{"Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}

var dataAttrRe = regexp.MustCompile(`data-([a-z]+)="([^"]*)"`)

// start opens the popup on the start page and returns its data attributes
func (e *handoffEnv) start(challenge string) map[string]string {
	e.t.Helper()
	resp, set, body := e.do(e.popup, "GET", "/auth/handoff/start?"+url.Values{"provider": {"fake"}, "site": {"remark42"},
		"from": {handoffFrom}, "challenge": {challenge}}.Encode(), nil, navigation)
	require.Equal(e.t, http.StatusOK, resp.StatusCode, body)
	data := map[string]string{}
	for _, m := range dataAttrRe.FindAllStringSubmatch(body, -1) {
		data[m[1]] = html.UnescapeString(m[2])
	}
	require.NotEmpty(e.t, data["txn"], body)
	require.Contains(e.t, set, e.srv.handoffsForTest().bindingName(data["txn"]), "the popup is bound to this browser")
	return data
}

// login runs an ordinary provider round in the popup from loginURL, signing in as user, and returns
// the callback's response
func (e *handoffEnv) login(loginURL, user string) (callback reply, set map[string]*http.Cookie) {
	e.t.Helper()
	resp, _, body := e.do(e.popup, "GET", loginURL, nil, nil)
	require.Equal(e.t, http.StatusFound, resp.StatusCode, body)
	authorize := resp.Header.Get("Location") + "&as=" + user
	resp, _, _ = e.do(browserJar{}, "GET", authorize, nil, nil) // the provider's own site has no remark42 cookie
	require.Equal(e.t, http.StatusFound, resp.StatusCode)
	callback, set, body = e.do(e.popup, "GET", resp.Header.Get("Location"), nil, nil)
	require.Equal(e.t, http.StatusSeeOther, callback.StatusCode, body)
	return callback, set
}

func (e *handoffEnv) status(txn, verifier string) (code int, status, user string) {
	e.t.Helper()
	resp, _, body := e.do(e.frame, "POST", "/auth/handoff/status", handoffRequest{Txn: txn, Verifier: verifier}, nil)
	var out struct{ Status, User string }
	if resp.StatusCode == http.StatusOK {
		require.NoError(e.t, json.Unmarshal([]byte(body), &out), body)
	}
	return resp.StatusCode, out.Status, out.User
}

// user asks /api/v1/user from jar as the widget does, with the xsrf token of the cookie it reads
func (e *handoffEnv) user(jar browserJar, xsrfCookie string, hdr ...map[string]string) (code int, id string) {
	e.t.Helper()
	h := map[string]string{"X-XSRF-TOKEN": jar[xsrfCookie]}
	for _, m := range hdr {
		for k, v := range m {
			h[k] = v
		}
	}
	resp, _, body := e.do(jar, "GET", "/api/v1/user?site=remark42", nil, h)
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, ""
	}
	var u struct{ ID string }
	require.NoError(e.t, json.Unmarshal([]byte(body), &u), body)
	return resp.StatusCode, u.ID
}

// TestHandoff_CrossSite is the sign-in of a frame that sees none of remark42's own cookies: the
// popup signs in as usual, and the frame ends up with a session of its own for the same account
func TestHandoff_CrossSite(t *testing.T) {
	logs := &lockedBuffer{}
	lgr.Setup(lgr.Out(logs), lgr.Err(logs))
	t.Cleanup(func() { lgr.Setup() })
	e := newHandoffEnv(t)
	verifier, challenge := newVerifier(t)

	data := e.start(challenge)
	assert.Equal(t, e.srv.RemarkURL+"/auth/fake/login?"+url.Values{"from": {handoffFrom}, "site": {"remark42"}}.Encode(),
		data["fallback"], "the fallback is the login the widget links to")
	next, err := url.Parse(data["next"])
	require.NoError(t, err)
	assert.Equal(t, "/auth/fake/login", next.Path)
	assert.Equal(t, e.srv.RemarkURL+"/auth/handoff/done?txn="+data["txn"], next.Query().Get("from"))

	code, status, _ := e.status(data["txn"], verifier)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "pending", status)

	resp, set := e.login(data["next"], "bob")
	assert.Equal(t, e.srv.RemarkURL+"/auth/handoff/done?txn="+data["txn"], resp.Header.Get("Location"))
	assert.Contains(t, set, "JWT", "the popup's own first-party session is written as before")
	require.Contains(t, set, e.srv.handoffsForTest().bindingName(data["txn"]))
	assert.Negative(t, set[e.srv.handoffsForTest().bindingName(data["txn"])].MaxAge, "the binding is spent")

	code, status, user := e.status(data["txn"], verifier)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "done", status)
	assert.Equal(t, "fake_bob", user)

	code, _ = e.user(e.frame, "XSRF-TOKEN")
	require.Equal(t, http.StatusUnauthorized, code, "the frame has no session of its own yet")

	resp, set, body := e.do(e.frame, "POST", "/auth/handoff/redeem", handoffRequest{Txn: data["txn"], Verifier: verifier}, nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode, body)
	for _, name := range []string{embeddedJWTCookie, embeddedXSRFCookie, embeddedMarkerCookie} {
		require.Contains(t, set, name)
		assert.True(t, set[name].Partitioned, name)
		assert.True(t, set[name].Secure, name)
		assert.Equal(t, http.SameSiteNoneMode, set[name].SameSite, name)
	}
	assert.True(t, set[embeddedJWTCookie].HttpOnly, "no token reaches script")
	assert.NotContains(t, set, "JWT", "the ordinary pair is not touched")
	assert.NotContains(t, set, "XSRF-TOKEN")
	assert.Empty(t, resp.Header.Get("X-JWT"))
	assert.NotEqual(t, e.popup["XSRF-TOKEN"], e.frame[embeddedXSRFCookie], "a session of its own, not the popup's")

	code, id := e.user(e.frame, embeddedXSRFCookie)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "fake_bob", id)

	resp, _, _ = e.do(e.frame, "POST", "/auth/handoff/redeem", handoffRequest{Txn: data["txn"], Verifier: verifier}, nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "a transaction hands out one session")
	assert.NotContains(t, logs.String(), verifier, "the verifier redeems a sign-in, it stays out of the log")
	assert.Contains(t, logs.String(), "/auth/handoff/redeem", "the requests are logged")
}

// TestHandoff_SessionsStayApart signs the site in as one account and the frame as another, in a
// browser that sends the frame both, as Chrome does with SameSite=None
func TestHandoff_SessionsStayApart(t *testing.T) {
	e := newHandoffEnv(t)

	// alice on remark42's own site, before anything else
	e.login("/auth/fake/login?"+url.Values{"from": {handoffFrom}, "site": {"remark42"}}.Encode(), "alice")
	code, id := e.user(e.popup, "XSRF-TOKEN")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "fake_alice", id)

	verifier, challenge := newVerifier(t)
	data := e.start(challenge)
	e.login(data["next"], "bob")
	resp, _, body := e.do(e.frame, "POST", "/auth/handoff/redeem", handoffRequest{Txn: data["txn"], Verifier: verifier}, nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode, body)

	// the popup's first-party write made the site bob, as it does today; alice signs in there again
	e.login("/auth/fake/login?"+url.Values{"from": {handoffFrom}, "site": {"remark42"}}.Encode(), "alice")

	both := browserJar{}
	for k, v := range e.popup {
		both[k] = v
	}
	for k, v := range e.frame {
		both[k] = v
	}
	code, id = e.user(both, embeddedXSRFCookie)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "fake_bob", id, "the frame stays bob next to the site's alice")
	code, _ = e.user(both, "XSRF-TOKEN")
	assert.NotEqual(t, http.StatusOK, code, "the site's xsrf token does not open the frame's session")

	code, id = e.user(e.popup, "XSRF-TOKEN")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "fake_alice", id, "and the site stays alice")

	t.Run("either embedded cookie removed never shows the site's account", func(t *testing.T) {
		for _, gone := range []string{embeddedJWTCookie, embeddedMarkerCookie, embeddedXSRFCookie} {
			jar := browserJar{}
			for k, v := range both {
				jar[k] = v
			}
			xsrf := jar[embeddedXSRFCookie]
			delete(jar, gone)
			code, id := e.user(jar, "", map[string]string{"X-XSRF-TOKEN": xsrf})
			switch gone {
			case embeddedJWTCookie:
				assert.Equal(t, http.StatusUnauthorized, code, "without its JWT the frame is signed out")
			default:
				require.Equal(t, http.StatusOK, code, gone)
				assert.Equal(t, "fake_bob", id, gone)
			}
			code, _ = e.user(jar, "XSRF-TOKEN")
			assert.NotEqual(t, http.StatusOK, code, "the site's pair is not read in place of the frame's, %s gone", gone)
		}
	})

	t.Run("a JWT header of the site's session is not read either", func(t *testing.T) {
		alice := e.popup["JWT"]
		jar := browserJar{embeddedMarkerCookie: "1", "JWT": alice, "XSRF-TOKEN": e.popup["XSRF-TOKEN"]}
		code, _ := e.user(jar, "XSRF-TOKEN", map[string]string{"X-JWT": alice})
		assert.Equal(t, http.StatusUnauthorized, code)
		code, id := e.user(browserJar{}, "", map[string]string{"X-JWT": alice})
		require.Equal(t, http.StatusOK, code, "the same header outside the embedded session")
		assert.Equal(t, "fake_alice", id)
	})

	t.Run("signing the frame out leaves the site and keeps the frame out of it", func(t *testing.T) {
		resp, set, body := e.do(both, "GET", "/auth/logout", nil, map[string]string{"X-XSRF-TOKEN": both[embeddedXSRFCookie]})
		require.Equal(t, http.StatusOK, resp.StatusCode, body)
		require.Contains(t, set, embeddedJWTCookie)
		assert.Negative(t, set[embeddedJWTCookie].MaxAge)
		assert.True(t, set[embeddedJWTCookie].Partitioned, "the expiry addresses the partitioned cookie")
		assert.NotContains(t, set, "JWT")
		assert.NotContains(t, set, "XSRF-TOKEN")
		assert.NotContains(t, set, embeddedMarkerCookie, "the choice survives the sign-out")
		require.Equal(t, "1", both[embeddedMarkerCookie])

		code, _ := e.user(both, "XSRF-TOKEN")
		assert.Equal(t, http.StatusUnauthorized, code, "the frame is signed out, not alice")
		code, id := e.user(e.popup, "XSRF-TOKEN")
		require.Equal(t, http.StatusOK, code)
		assert.Equal(t, "fake_alice", id, "the site is untouched")
	})
}

// TestHandoff_EmbeddedRenewal renews an expired embedded token: the fresh one goes to the embedded
// pair and the marker, never the ordinary pair
func TestHandoff_EmbeddedRenewal(t *testing.T) {
	e := newHandoffEnv(t, func(srv *Rest) { srv.Authenticator.TokenService().SendJWTHeader = true })
	claims := token.Claims{User: &token.User{ID: "fake_bob", Name: "bob"}, RegisteredClaims: jwt.RegisteredClaims{
		ID: "old-id", Audience: []string{"remark42"}, Issuer: "remark42",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)), IssuedAt: jwt.NewNumericDate(time.Now().Add(-2 * time.Minute)),
	}}
	tkn, err := e.srv.Authenticator.TokenService().Token(claims)
	require.NoError(t, err)
	jar := browserJar{embeddedJWTCookie: tkn, embeddedXSRFCookie: "old-id", embeddedMarkerCookie: "1", "JWT": "ordinary", "XSRF-TOKEN": "x"}

	resp, set, body := e.do(jar, "GET", "/api/v1/user?site=remark42", nil, map[string]string{"X-XSRF-TOKEN": "old-id"})
	require.Equal(t, http.StatusOK, resp.StatusCode, body)
	require.Contains(t, set, embeddedJWTCookie, "renewed")
	assert.NotEqual(t, tkn, set[embeddedJWTCookie].Value)
	assert.True(t, set[embeddedJWTCookie].Partitioned)
	assert.Contains(t, set, embeddedMarkerCookie, "the marker lasts as long as the session it marks")
	assert.NotContains(t, set, "JWT")
	assert.Equal(t, "ordinary", jar["JWT"])
	assert.Empty(t, resp.Header.Get("X-JWT"), "header mode hands no token to script in an embedded session")
}

// TestHandoff_RecordsTheCallbacksAccount checks that the account a transaction records is the one
// its own callback signed in, whatever the browser's session is before or after
func TestHandoff_RecordsTheCallbacksAccount(t *testing.T) {
	e := newHandoffEnv(t)
	verifier, challenge := newVerifier(t)
	data := e.start(challenge)
	e.login(data["next"], "bob")

	// another tab signs the browser in as carol straight after
	e.login("/auth/fake/login?"+url.Values{"from": {handoffFrom}, "site": {"remark42"}}.Encode(), "carol")
	code, id := e.user(e.popup, "XSRF-TOKEN")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "fake_carol", id)

	_, status, user := e.status(data["txn"], verifier)
	assert.Equal(t, "done", status)
	assert.Equal(t, "fake_bob", user)
	resp, _, body := e.do(e.frame, "POST", "/auth/handoff/redeem", handoffRequest{Txn: data["txn"], Verifier: verifier}, nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode, body)
	_, id = e.user(e.frame, embeddedXSRFCookie)
	assert.Equal(t, "fake_bob", id)
}

// TestHandoff_BoundToTheBrowser runs the provider round of a transaction in a browser that did not
// start it, as a reader lured to a login link carrying someone else's transaction would
func TestHandoff_BoundToTheBrowser(t *testing.T) {
	e := newHandoffEnv(t)
	verifier, challenge := newVerifier(t)
	data := e.start(challenge)

	victim := &handoffEnv{t: t, ts: e.ts, srv: e.srv, idp: e.idp, popup: browserJar{}, frame: browserJar{}, seq: 500}
	resp, _ := victim.login(data["next"], "victim")
	assert.Equal(t, e.srv.RemarkURL+"/auth/handoff/done?txn="+data["txn"], resp.Header.Get("Location"))
	// and one whose binding cookie of that name is not the one the start page set
	forged := &handoffEnv{t: t, ts: e.ts, srv: e.srv, idp: e.idp, popup: browserJar{e.srv.handoffsForTest().bindingName(data["txn"]): "forged"}, seq: 1000}
	forged.login(data["next"], "forged")

	_, status, user := e.status(data["txn"], verifier)
	assert.Equal(t, "pending", status, "the victim's sign-in is not recorded for the transaction")
	assert.Empty(t, user)
	resp, _, _ = e.do(e.frame, "POST", "/auth/handoff/redeem", handoffRequest{Txn: data["txn"], Verifier: verifier}, nil)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestHandoff_Verifier(t *testing.T) {
	e := newHandoffEnv(t)
	verifier, challenge := newVerifier(t)
	data := e.start(challenge)
	e.login(data["next"], "bob")

	other, _ := newVerifier(t)
	for _, v := range []string{"", other, challenge} {
		code, _, _ := e.status(data["txn"], v)
		assert.Equal(t, http.StatusForbidden, code, "verifier %q", v)
		resp, set, _ := e.do(e.frame, "POST", "/auth/handoff/redeem", handoffRequest{Txn: data["txn"], Verifier: v}, nil)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode, "verifier %q", v)
		assert.Empty(t, set)
	}
	code, _, _ := e.status("nope", verifier)
	assert.Equal(t, http.StatusNotFound, code)

	resp, _, body := e.do(e.frame, "POST", "/auth/handoff/redeem", handoffRequest{Txn: data["txn"], Verifier: verifier}, nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, body, "the wrong verifiers did not spend it")
}

// TestHandoff_FallsBackToTheOrdinaryLogin covers every way the start page leaves the handoff out
func TestHandoff_FallsBackToTheOrdinaryLogin(t *testing.T) {
	fallback := "https://demo.remark42.com/auth/fake/login?" + url.Values{"from": {handoffFrom}, "site": {"remark42"}}.Encode()
	startURL := func(challenge string) string {
		return "/auth/handoff/start?" + url.Values{"provider": {"fake"}, "site": {"remark42"}, "from": {handoffFrom},
			"challenge": {challenge}}.Encode()
	}

	t.Run("no challenge", func(t *testing.T) {
		e := newHandoffEnv(t)
		for _, c := range []string{"", "short", strings.Repeat("a", 42) + "!"} {
			resp, set, _ := e.do(e.popup, "GET", startURL(c), nil, navigation)
			require.Equal(t, http.StatusFound, resp.StatusCode, c)
			assert.Equal(t, fallback, resp.Header.Get("Location"), c)
			assert.Empty(t, set, c)
		}
	})

	t.Run("too many sign-ins under way", func(t *testing.T) {
		e := newHandoffEnv(t)
		_, challenge := newVerifier(t)
		e.start(challenge)
		h := e.srv.handoffsForTest()
		h.store.mu.Lock()
		h.store.max = len(h.store.recs)
		h.store.mu.Unlock()
		resp, _, _ := e.do(e.popup, "GET", startURL(challenge), nil, navigation)
		require.Equal(t, http.StatusFound, resp.StatusCode)
		assert.Equal(t, fallback, resp.Header.Get("Location"))
	})

	t.Run("anything but the popup's navigation", func(t *testing.T) {
		e := newHandoffEnv(t)
		_, challenge := newVerifier(t)
		for _, hdr := range []map[string]string{
			nil, // no fetch metadata at all
			{"Sec-Fetch-Mode": "cors", "Sec-Fetch-Dest": "empty", "Origin": "https://evil.example"},
			{"Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "iframe"},
		} {
			resp, set, body := e.do(e.popup, "GET", startURL(challenge), nil, hdr)
			if hdr["Sec-Fetch-Dest"] == "iframe" {
				// a frame is refused ahead of the handler, as every auth route refuses one
				require.Equal(t, http.StatusForbidden, resp.StatusCode)
			} else {
				require.Equal(t, http.StatusFound, resp.StatusCode, "%v", hdr)
				assert.Equal(t, fallback, resp.Header.Get("Location"), "%v", hdr)
			}
			assert.Empty(t, set, "no transaction, no binding cookie: %v", hdr)
			assert.NotContains(t, body, "data-txn")
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		e := newHandoffEnv(t)
		_, challenge := newVerifier(t)
		resp, _, _ := e.do(e.popup, "GET", strings.Replace(startURL(challenge), "provider=fake", "provider=nope", 1), nil, navigation)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("the popup gives up and the frame learns of it", func(t *testing.T) {
		e := newHandoffEnv(t)
		verifier, challenge := newVerifier(t)
		data := e.start(challenge)

		resp, _, _ := e.do(browserJar{}, "POST", "/auth/handoff/cancel", handoffRequest{Txn: data["txn"]}, nil)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode, "neither binding nor verifier")

		resp, _, body := e.do(e.popup, "POST", "/auth/handoff/cancel", handoffRequest{Txn: data["txn"]}, nil)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, body)
		_, status, _ := e.status(data["txn"], verifier)
		assert.Equal(t, "canceled", status)

		// the popup went on to the ordinary login, and its callback records nothing
		e.login(data["fallback"], "bob")
		_, status, _ = e.status(data["txn"], verifier)
		assert.Equal(t, "canceled", status)
	})
}

func TestHandoff_Pages(t *testing.T) {
	e := newHandoffEnv(t)
	_, challenge := newVerifier(t)
	resp, _, body := e.do(e.popup, "GET", "/auth/handoff/start?"+url.Values{"provider": {"fake"}, "site": {`"><script>alert(1)</script>`},
		"from": {`javascript:alert(1)"'`}, "challenge": {challenge}}.Encode(), nil, navigation)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotContains(t, body, "<script>alert", "request data is never markup")
	csp := resp.Header.Get("Content-Security-Policy")
	assert.Contains(t, csp, "script-src 'sha256-")
	assert.Regexp(t, `(^|; )script-src 'sha256-[A-Za-z0-9+/]{43}='; `, csp, "no script but the page's own")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.Contains(t, body, "<script>"+handoffStartScript+"</script>", "the script is the fixed one, byte for byte")
	assert.Equal(t, handoffCSP(handoffStartScript), csp)

	resp, _, body = e.do(e.popup, "GET", "/auth/handoff/done?txn=x", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, "<script>"+handoffDoneScript+"</script>")
	assert.Equal(t, handoffCSP(handoffDoneScript), resp.Header.Get("Content-Security-Policy"))
}

func TestHandoffStore(t *testing.T) {
	now := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	s := newHandoffStore()
	s.now = func() time.Time { return now }
	verifier, challenge := newVerifier(t)
	bob := token.Claims{User: &token.User{ID: "bob"}}

	id, binding, err := s.create(challenge, "site")
	require.NoError(t, err)
	state, _, changed, err := s.status(id, verifier)
	require.NoError(t, err)
	assert.Equal(t, handoffPending, state)

	assert.False(t, s.complete(id, "other", bob), "another browser's callback")
	select {
	case <-changed:
		t.Fatal("woken by a callback that did not count")
	default:
	}
	assert.True(t, s.complete(id, binding, bob))
	select {
	case <-changed:
	default:
		t.Fatal("a waiting status request is not woken")
	}
	assert.False(t, s.complete(id, binding, token.Claims{User: &token.User{ID: "mallory"}}), "a replayed callback")
	_, user, _, err := s.status(id, verifier)
	require.NoError(t, err)
	assert.Equal(t, "bob", user)

	never := func(token.Claims) bool { return false }
	_, err = s.redeem(id, verifier, func(token.Claims) bool { return true })
	require.ErrorIs(t, err, errHandoffSeen)
	got, err := s.redeem(id, verifier, never)
	require.NoError(t, err, "a refused redeem keeps the transaction")
	assert.Equal(t, "bob", got.User.ID)
	_, err = s.redeem(id, verifier, never)
	assert.ErrorIs(t, err, errHandoffNotFound)

	t.Run("expiry", func(t *testing.T) {
		id, binding, err := s.create(challenge, "site")
		require.NoError(t, err)
		_, _, changed, err := s.status(id, verifier)
		require.NoError(t, err)
		now = now.Add(handoffTTL + time.Second)
		assert.False(t, s.complete(id, binding, bob))
		_, _, _, err = s.status(id, verifier)
		assert.ErrorIs(t, err, errHandoffNotFound)
		select {
		case <-changed:
		default:
			t.Fatal("a waiting status request is not woken by the expiry")
		}
	})

	t.Run("bounded", func(t *testing.T) {
		small := newHandoffStore()
		small.now = func() time.Time { return now }
		small.max = 2
		_, _, err := small.create(challenge, "site")
		require.NoError(t, err)
		_, _, err = small.create(challenge, "site")
		require.NoError(t, err)
		_, _, err = small.create(challenge, "site")
		require.ErrorIs(t, err, errHandoffFull)
		now = now.Add(handoffTTL + time.Second)
		_, _, err = small.create(challenge, "site")
		require.NoError(t, err, "expired ones make room")
		assert.Len(t, small.recs, 1)
	})

	t.Run("cancel", func(t *testing.T) {
		id, binding, err := s.create(challenge, "site")
		require.NoError(t, err)
		assert.ErrorIs(t, s.cancel(id, "", "wrong"), errHandoffDenied)
		require.NoError(t, s.cancel(id, "", binding))
		state, _, _, err := s.status(id, verifier)
		require.NoError(t, err)
		assert.Equal(t, handoffCanceled, state, "kept for the frame to see")
		assert.False(t, s.complete(id, binding, bob), "a canceled one records nothing")
		require.NoError(t, s.cancel(id, verifier, ""))
		_, _, _, err = s.status(id, verifier)
		assert.ErrorIs(t, err, errHandoffNotFound, "the frame's cancel removes it")
	})
}

func (s *Rest) handoffsForTest() *handoffs { return s.handoffs }

// TestHandoff_WaitersAreCapped fills every waiting slot and checks a further status request answers
// at once instead of holding one more of the server's connections
func TestHandoff_WaitersAreCapped(t *testing.T) {
	e := newHandoffEnv(t, func(srv *Rest) { srv.handoffWait = waitTimeout })
	verifier, challenge := newVerifier(t)
	data := e.start(challenge)
	h := e.srv.handoffsForTest()
	for range cap(h.waiters) {
		h.waiters <- struct{}{}
	}
	t.Cleanup(func() {
		for range cap(h.waiters) {
			<-h.waiters
		}
	})
	started := time.Now()
	code, status, _ := e.status(data["txn"], verifier)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "pending", status)
	assert.Less(t, time.Since(started), waitTimeout/2, "answered without waiting")
}

// TestHandoff_RedeemRefusedWhereTheSessionIsSeen redeems from a frame that sends remark42's own
// session for the account: on the same site, or where the browser allows the site's cookies. An
// embedded pair set there would be the site's own, so it is refused and the transaction kept
func TestHandoff_RedeemRefusedWhereTheSessionIsSeen(t *testing.T) {
	e := newHandoffEnv(t)
	verifier, challenge := newVerifier(t)
	data := e.start(challenge)
	e.login(data["next"], "bob")

	shared := browserJar{"JWT": e.popup["JWT"], "XSRF-TOKEN": e.popup["XSRF-TOKEN"]}
	resp, set, body := e.do(shared, "POST", "/auth/handoff/redeem", handoffRequest{Txn: data["txn"], Verifier: verifier}, nil)
	assert.Equal(t, http.StatusConflict, resp.StatusCode, body)
	assert.Empty(t, set, "no embedded pair where the frame sees the site's session")

	// the site's session for another account is not this one: the frame gets its own
	carol := &handoffEnv{t: t, ts: e.ts, srv: e.srv, idp: e.idp, popup: browserJar{}, seq: 2000}
	carol.login("/auth/fake/login?"+url.Values{"from": {handoffFrom}, "site": {"remark42"}}.Encode(), "carol")
	jar := browserJar{"JWT": carol.popup["JWT"], "XSRF-TOKEN": carol.popup["XSRF-TOKEN"]}
	resp, set, body = e.do(jar, "POST", "/auth/handoff/redeem", handoffRequest{Txn: data["txn"], Verifier: verifier}, nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode, body, "the refusal kept the transaction")
	assert.Contains(t, set, embeddedJWTCookie)
}

// TestHandoff_RedeemOnTheEmbeddedSession signs a frame in again after it signed out of its embedded
// session, in a browser that now sends it the site's session for the same account as well. The
// frame's requests read its embedded session alone, so the site's is no use to it: it is given its
// own, and does not loop on refusals until the marker expires
func TestHandoff_RedeemOnTheEmbeddedSession(t *testing.T) {
	e := newHandoffEnv(t)
	verifier, challenge := newVerifier(t)
	data := e.start(challenge)
	e.login(data["next"], "bob")

	jar := browserJar{embeddedMarkerCookie: "1", "JWT": e.popup["JWT"], "XSRF-TOKEN": e.popup["XSRF-TOKEN"]}
	resp, set, body := e.do(jar, "POST", "/auth/handoff/redeem", handoffRequest{Txn: data["txn"], Verifier: verifier}, nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode, body)
	assert.Contains(t, set, embeddedJWTCookie)
	code, id := e.user(jar, embeddedXSRFCookie)
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, "fake_bob", id)
}

// TestHandoff_NoCORS checks no other site may read a handoff response, whatever origin it names
func TestHandoff_NoCORS(t *testing.T) {
	e := newHandoffEnv(t)
	_, challenge := newVerifier(t)
	evil := map[string]string{"Origin": "https://evil.example", "Sec-Fetch-Mode": "navigate", "Sec-Fetch-Dest": "document"}
	resp, _, _ := e.do(e.popup, "GET", "/auth/handoff/start?"+url.Values{"provider": {"fake"}, "site": {"remark42"},
		"from": {handoffFrom}, "challenge": {challenge}}.Encode(), nil, evil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
	resp, _, _ = e.do(e.frame, "POST", "/auth/handoff/status", handoffRequest{Txn: "x", Verifier: "y"},
		map[string]string{"Origin": "https://evil.example"})
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))

	// the rest of the api is as it was
	resp, _, _ = e.do(e.frame, "GET", "/api/v1/config?site=remark42", nil, map[string]string{"Origin": "https://evil.example"})
	assert.Equal(t, "https://evil.example", resp.Header.Get("Access-Control-Allow-Origin"), "the control: CORS is on elsewhere")
}

// TestHandoff_BadRequest sends what is not a sign-in request, and gets a 400, not a panic
func TestHandoff_BadRequest(t *testing.T) {
	e := newHandoffEnv(t)
	for _, body := range []any{map[string]string{}, map[string]string{"verifier": "v"}} {
		for _, path := range []string{"/auth/handoff/status", "/auth/handoff/redeem", "/auth/handoff/cancel"} {
			resp, _, text := e.do(e.frame, "POST", path, body, nil)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "%s %v: %s", path, body, text)
			assert.Contains(t, text, "no txn", path)
		}
	}
}

// TestHandoff_BindingCookiePath checks the binding cookie reaches the provider callback where the
// browser sees it, and over https carries the __Host- prefix no other host can plant a cookie under
func TestHandoff_BindingCookiePath(t *testing.T) {
	tbl := []struct{ remarkURL, name, path string }{
		{"https://demo.remark42.com", "__Host-remark42-handoff-abc", "/"},
		{"https://example.com/remark42", "__Host-remark42-handoff-abc", "/"},
		{"https://example.com/caf%C3%A9", "__Host-remark42-handoff-abc", "/"},
		{"http://example.com", "remark42-handoff-abc", "/auth/"},
		{"http://example.com/remark42", "remark42-handoff-abc", "/remark42/auth/"},
		{"http://example.com/caf%C3%A9/", "remark42-handoff-abc", "/caf%C3%A9/auth/"},
	}
	for _, tt := range tbl {
		h := &handoffs{remarkURL: strings.TrimSuffix(tt.remarkURL, "/"), tokens: &token.Service{}}
		c := h.bindingCookie("abc", "v", 60)
		assert.Equal(t, tt.name, c.Name, tt.remarkURL)
		assert.Equal(t, tt.path, c.Path, tt.remarkURL)
		assert.Equal(t, strings.HasPrefix(tt.remarkURL, "https"), c.Secure, tt.remarkURL)
		assert.Empty(t, c.Domain)
		// what the browser is sent keeps the path as it is on the wire
		assert.Contains(t, c.String(), "; Path="+tt.path, tt.remarkURL)
		u, err := url.Parse(strings.TrimSuffix(tt.remarkURL, "/") + "/auth/github/callback")
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(u.EscapedPath(), c.Path), "the callback of %s is under the cookie's path", tt.remarkURL)
	}
}

// TestHandoff_JSONFromItsOwnOrigin sends the calls the way only another site would: a form's
// text/plain body that still parses as JSON, and a cross-site fetch. Both are refused, and the
// transaction is still there for the frame
func TestHandoff_JSONFromItsOwnOrigin(t *testing.T) {
	e := newHandoffEnv(t)
	verifier, challenge := newVerifier(t)
	data := e.start(challenge)
	e.login(data["next"], "bob")
	body := `{"txn":"` + data["txn"] + `","verifier":"` + verifier + `","p":"="}`
	for _, hdr := range []map[string]string{
		{"Content-Type": "text/plain", "Sec-Fetch-Site": "cross-site", "Sec-Fetch-Mode": "navigate"},
		{"Content-Type": "text/plain"},
		{"Content-Type": "application/x-www-form-urlencoded"},
		{"Content-Type": "application/json", "Sec-Fetch-Site": "cross-site"},
		{"Content-Type": "application/json", "Sec-Fetch-Site": "same-site"},
		// a browser with no fetch metadata still names the origin of a cross-origin fetch
		{"Content-Type": "application/json", "Origin": "https://evil.example"},
	} {
		req, err := http.NewRequest("POST", e.ts.URL+"/auth/handoff/redeem", strings.NewReader(body))
		require.NoError(t, err)
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		e.seq++
		req.Header.Set("X-Real-IP", fmt.Sprintf("8.8.%d.%d", e.seq/250, e.seq%250+1))
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		assert.Equal(t, http.StatusForbidden, resp.StatusCode, "%v", hdr)
		assert.Empty(t, resp.Header.Values("Set-Cookie"), "%v", hdr)
	}
	resp, _, text := e.do(e.frame, "POST", "/auth/handoff/redeem", handoffRequest{Txn: data["txn"], Verifier: verifier},
		map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": "https://demo.remark42.com"})
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "the frame's own call: %s", text)
}

// TestHandoff_SeenInAnyJWTCookie redeems from a frame holding two ordinary JWT cookies: a stale one
// for another site first, as a partitioned copy from header mode can be, and the recorded
// account's after it. The frame sees the account, so the redeem is refused
func TestHandoff_SeenInAnyJWTCookie(t *testing.T) {
	e := newHandoffEnv(t)
	verifier, challenge := newVerifier(t)
	data := e.start(challenge)
	e.login(data["next"], "bob")
	stale, err := e.srv.Authenticator.TokenService().Token(token.Claims{User: &token.User{ID: "fake_bob"},
		RegisteredClaims: jwt.RegisteredClaims{ID: "s", Audience: []string{"other-site"}, Issuer: "remark42"}})
	require.NoError(t, err)
	req, err := http.NewRequest("POST", e.ts.URL+"/auth/handoff/redeem",
		strings.NewReader(`{"txn":"`+data["txn"]+`","verifier":"`+verifier+`"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "JWT="+stale+"; JWT="+e.popup["JWT"])
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

// TestHandoff_PlantedMarkerIsIgnored sends the embedded cookies' names without the __Host- prefix,
// as a page on a sibling subdomain could plant them with a Domain attribute. They are not the
// embedded session, and the ordinary session is read as before
func TestHandoff_PlantedMarkerIsIgnored(t *testing.T) {
	e := newHandoffEnv(t)
	e.login("/auth/fake/login?"+url.Values{"from": {handoffFrom}, "site": {"remark42"}}.Encode(), "alice")
	for _, planted := range []string{"REMARK42-EMBEDDED", "JWT-EMBEDDED"} {
		jar := browserJar{"JWT": e.popup["JWT"], "XSRF-TOKEN": e.popup["XSRF-TOKEN"], planted: "1"}
		code, id := e.user(jar, "XSRF-TOKEN")
		require.Equal(t, http.StatusOK, code, planted)
		assert.Equal(t, "fake_alice", id, planted)
	}
	jar := browserJar{"JWT": e.popup["JWT"], "XSRF-TOKEN": e.popup["XSRF-TOKEN"], embeddedMarkerCookie: "1"}
	code, _ := e.user(jar, "XSRF-TOKEN")
	assert.Equal(t, http.StatusUnauthorized, code, "the control: the prefixed marker does switch")
}

// TestHandoff_OwnOriginAsTheBrowserWritesIt runs the frame's own call against REMARK_URL spelled
// with the default port and capitals, which a browser leaves out and lowercases in Origin
func TestHandoff_OwnOriginAsTheBrowserWritesIt(t *testing.T) {
	for _, remarkURL := range []string{"https://Demo.Remark42.com:443", "https://demo.remark42.com"} {
		e := newHandoffEnv(t, func(srv *Rest) { srv.RemarkURL = remarkURL })
		// no fetch metadata, so the Origin is what the handler goes by
		resp, _, text := e.do(e.frame, "POST", "/auth/handoff/status", handoffRequest{Txn: "x", Verifier: "y"},
			map[string]string{"Origin": "https://demo.remark42.com"})
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "%s: reaches the handler, %s", remarkURL, text)
		resp, _, _ = e.do(e.frame, "POST", "/auth/handoff/status", handoffRequest{Txn: "x", Verifier: "y"},
			map[string]string{"Origin": "https://evil.example"})
		assert.Equal(t, http.StatusForbidden, resp.StatusCode, remarkURL)
	}

	// REMARK_URL says http, a proxy serves the page over https: the browser's own same-origin word
	// stands, where the Origin's scheme cannot match
	e := newHandoffEnv(t, func(srv *Rest) { srv.RemarkURL = "http://demo.remark42.com" })
	resp, _, text := e.do(e.frame, "POST", "/auth/handoff/status", handoffRequest{Txn: "x", Verifier: "y"},
		map[string]string{"Origin": "https://demo.remark42.com", "Sec-Fetch-Site": "same-origin"})
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "reaches the handler: %s", text)
}

// TestOwnOrigin checks the origin a browser would send for each way REMARK_URL can be written
func TestOwnOrigin(t *testing.T) {
	tbl := []struct{ remarkURL, want string }{
		{"https://demo.remark42.com", "https://demo.remark42.com"},
		{"https://Demo.Remark42.com:443", "https://demo.remark42.com"},
		{"https://demo.remark42.com:8443", "https://demo.remark42.com:8443"},
		{"http://demo.remark42.com:80/remark42", "http://demo.remark42.com"},
		{"http://demo.remark42.com:443", "http://demo.remark42.com:443"},
		{"https://комменты.example.com", "https://xn--e1ajgagc6a2e.example.com"},
		{"https://[::1]:443", "https://[::1]"},
		{"https://[::1]:8443", "https://[::1]:8443"},
		{"https://[0:0:0:0:0:0:0:1]:8443", "https://[::1]:8443"},
		{"https://127.0.0.1:443", "https://127.0.0.1"},
		{"not a url", ""},
	}
	for _, tt := range tbl {
		assert.Equal(t, tt.want, ownOrigin(tt.remarkURL), tt.remarkURL)
	}
}
