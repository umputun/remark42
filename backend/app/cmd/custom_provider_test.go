package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-pkgz/auth/v2/provider"
	"github.com/go-pkgz/auth/v2/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomProviderUser(t *testing.T) {
	cfg := CustomAuthGroup{IDField: "sub", EmailField: "email", NameField: "name", PictureField: "picture"}

	// user maps the user info the way a sign-in does: raw is the provider's answer, decoded as the auth library decodes it
	user := func(t *testing.T, raw string) token.User {
		t.Helper()
		data := provider.UserData{}
		require.NoError(t, json.Unmarshal([]byte(raw), &data))
		return customProviderUser("oidc", data, []byte(raw), cfg)
	}

	tbl := []struct {
		name   string
		raw    string
		wantID string // "oidc_" and the sha1 of the value the comment names
	}{
		{name: "string id", raw: `{"sub":"alice"}`, wantID: "oidc_522b276a356bdf39013dfabea2cd43e141ecc9e8"}, // alice
		{name: "id spelling the hash of another id is hashed like any other", // the 40 digits themselves
			raw: `{"sub":"522b276a356bdf39013dfabea2cd43e141ecc9e8"}`, wantID: "oidc_c7ce375e45f37b15ba20069e4c26e87af25a6523"},
		{name: "upper case of the same", raw: `{"sub":"522B276A356BDF39013DFABEA2CD43E141ECC9E8"}`,
			wantID: "oidc_b6220e9cb64f2e437618cfd856929546b09d8785"},
		{name: "number keeps the id its float64 format gave it", raw: `{"sub":1345027}`, // 1.345027e+06
			wantID: "oidc_47223acdc4a7f1be1a63d86170214b93687f3f78"},
		{name: "2^53, the last number in the exact range", raw: `{"sub":9007199254740992}`, // 9.007199254740992e+15
			wantID: "oidc_98edf1480c6e895258353c95a2d945fb7ebe7af2"},
		{name: "number beyond 2^53 is taken with all its digits", raw: `{"sub":1152921504606846977}`, // 1152921504606846977
			wantID: "oidc_ed2bdf22707789b16a97f0574256ba2efb391b92"},
		{name: "the next id, the same float64, gets an id of its own", raw: `{ "sub" : 1152921504606846980 }`, // 1152921504606846980
			wantID: "oidc_8be46ce66310773dbc100bb46a80509a1f8c21a0"},
		{name: "string of digits beyond 2^53 is a string", raw: `{"sub":"1152921504606846977"}`,
			wantID: "oidc_ed2bdf22707789b16a97f0574256ba2efb391b92"},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantID, user(t, tt.raw).ID)
		})
	}

	t.Run("fields", func(t *testing.T) {
		u := user(t, `{"sub":"alice","name":"Alice","email":"a@example.com","picture":"https://example.com/a.png"}`)
		assert.Equal(t, "Alice", u.Name)
		assert.Equal(t, "a@example.com", u.Email)
		assert.Equal(t, "https://example.com/a.png", u.Picture)
	})

	t.Run("no name", func(t *testing.T) {
		assert.Equal(t, "noname_522b", user(t, `{"sub":"alice"}`).Name)
	})
}

func TestCustomProviderLargeInt(t *testing.T) {
	tbl := []struct {
		name string
		raw  string
		want string
	}{
		{name: "beyond 2^53", raw: `{"id":9007199254740993}`, want: "9007199254740993"},
		{name: "negative beyond 2^53", raw: `{"id":-9007199254740993}`, want: "-9007199254740993"},
		{name: "beyond int64", raw: `{"id":123456789012345678901234567890}`, want: "123456789012345678901234567890"},
		{name: "spaces around the value", raw: "{\"id\" :\n 9007199254740993 }", want: "9007199254740993"},
		{name: "2^53", raw: `{"id":9007199254740992}`, want: ""},
		{name: "small", raw: `{"id":42}`, want: ""},
		{name: "string", raw: `{"id":"9007199254740993"}`, want: ""},
		{name: "fraction", raw: `{"id":9007199254740993.5}`, want: ""},
		{name: "exponent", raw: `{"id":1e30}`, want: ""},
		{name: "null", raw: `{"id":null}`, want: ""},
		{name: "object", raw: `{"id":{"n":9007199254740993}}`, want: ""},
		{name: "another field", raw: `{"uid":9007199254740993}`, want: ""},
		{name: "not an object", raw: `[9007199254740993]`, want: ""},
		{name: "no user info", raw: ``, want: ""},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, customProviderLargeInt([]byte(tt.raw), "id"))
		})
	}
}

// TestServerApp_CustomOAuthProviderSignIn signs users in through the custom provider and checks the
// id each of them gets, as the session token carries it
func TestServerApp_CustomOAuthProviderSignIn(t *testing.T) {
	idp := newCustomOAuthStub(t, map[string]string{
		"alice":           `{"sub":"alice","name":"Alice"}`,
		"hash-of-alice":   `{"sub":"522b276a356bdf39013dfabea2cd43e141ecc9e8","name":"Mallory"}`,
		"large-number":    `{"sub":1152921504606846977,"name":"Carol"}`,
		"the-next-number": `{"sub":1152921504606846980,"name":"Dave"}`,
	})
	app := startCustomOAuthApp(t, idp, nil)

	t.Run("id is the hash of the provider's id", func(t *testing.T) {
		status, userID := customOAuthSignIn(t, app, "alice", "")
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, "oidc_522b276a356bdf39013dfabea2cd43e141ecc9e8", userID)
	})

	t.Run("id spelling another user's hash gets an id of its own", func(t *testing.T) {
		status, userID := customOAuthSignIn(t, app, "hash-of-alice", "")
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, "oidc_c7ce375e45f37b15ba20069e4c26e87af25a6523", userID)
	})

	t.Run("numbers a float64 can't tell apart get ids of their own", func(t *testing.T) {
		status, userID := customOAuthSignIn(t, app, "large-number", "")
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, "oidc_ed2bdf22707789b16a97f0574256ba2efb391b92", userID)

		status, userID = customOAuthSignIn(t, app, "the-next-number", "")
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, "oidc_8be46ce66310773dbc100bb46a80509a1f8c21a0", userID)
	})
}

// newCustomOAuthStub starts an OAuth2 provider for the custom provider tests. users maps an
// authorization code to the user info answered for it: the token endpoint hands the code back as
// the access token, and the user info endpoint looks the user up by that token
func newCustomOAuthStub(t *testing.T, users map[string]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"access_token":%q,"token_type":"Bearer"}`, r.FormValue("code"))
	})
	mux.HandleFunc("GET /userinfo", func(w http.ResponseWriter, r *http.Request) {
		info, ok := users[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
		if !ok {
			http.Error(w, "unknown user", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, info)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// startCustomOAuthApp runs the app with the custom provider "oidc" pointed at idp. The app is served
// over plain http on its own address, so the sign-in cookies reach it from a cookie jar
func startCustomOAuthApp(t *testing.T, idp *httptest.Server, fn func(o ServerCommand) ServerCommand) *serverApp {
	t.Helper()
	return startServerApp(t, func(o ServerCommand) ServerCommand {
		o.RemarkURL = fmt.Sprintf("http://localhost:%d", o.Port)
		o.Auth.Custom.Name = "oidc"
		o.Auth.Custom.CID = "cid"
		o.Auth.Custom.CSEC = "csec"
		o.Auth.Custom.AuthURL = idp.URL + "/authorize"
		o.Auth.Custom.TokenURL = idp.URL + "/token"
		o.Auth.Custom.InfoURL = idp.URL + "/userinfo"
		if fn != nil {
			o = fn(o)
		}
		return o
	})
}

// customOAuthSignIn signs in through the custom provider "oidc" with the given authorization code
// and returns the status of the callback and the user id in the session it issued, empty when it
// issued none. It does what a browser does, except for the visit to the provider: the callback is
// requested straight away with the state of the login redirect. loginQuery is added to the login
// request
func customOAuthSignIn(t *testing.T, app *serverApp, code, loginQuery string) (status int, userID string) {
	t.Helper()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := http.Client{Timeout: 10 * time.Second, Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	defer client.CloseIdleConnections()

	base := fmt.Sprintf("http://localhost:%d/auth/oidc", app.Port)
	resp := getRetryThrottled(t, &client, base+"/login?site=remark"+loginQuery)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusFound, resp.StatusCode, "login redirects to the provider")
	loc, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	state := loc.Query().Get("state")
	require.NotEmpty(t, state, "login redirect carries the state, %s", loc)

	resp = getRetryThrottled(t, &client, base+"/callback?code="+url.QueryEscape(code)+"&state="+url.QueryEscape(state))
	require.NoError(t, resp.Body.Close())
	if _, claims := getAuthFromCookie(t, app, resp); claims.User != nil {
		userID = claims.User.ID
	}
	return resp.StatusCode, userID
}
