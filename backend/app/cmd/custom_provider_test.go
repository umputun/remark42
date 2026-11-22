package cmd

import (
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomProviderUser(t *testing.T) {
	cfg := CustomAuthGroup{IDField: "sub", EmailField: "email", NameField: "name", PictureField: "picture"}

	tbl := []struct {
		name   string
		data   provider.UserData
		wantID string
	}{
		{name: "string id", data: provider.UserData{"sub": "alice"}, wantID: "oidc_522b276a356bdf39013dfabea2cd43e141ecc9e8"},
		{name: "id spelling the hash of another id is hashed like any other",
			data: provider.UserData{"sub": "522b276a356bdf39013dfabea2cd43e141ecc9e8"}, wantID: "oidc_c7ce375e45f37b15ba20069e4c26e87af25a6523"},
		{name: "upper case of the same", data: provider.UserData{"sub": "522B276A356BDF39013DFABEA2CD43E141ECC9E8"},
			wantID: "oidc_b6220e9cb64f2e437618cfd856929546b09d8785"},
		{name: "number as the user info decoder formats it", data: provider.UserData{"sub": float64(1345027)},
			wantID: "oidc_47223acdc4a7f1be1a63d86170214b93687f3f78"},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantID, customProviderUser("oidc", tt.data, cfg).ID)
		})
	}

	t.Run("fields", func(t *testing.T) {
		user := customProviderUser("oidc", provider.UserData{"sub": "alice", "name": "Alice", "email": "a@example.com",
			"picture": "https://example.com/a.png"}, cfg)
		assert.Equal(t, "Alice", user.Name)
		assert.Equal(t, "a@example.com", user.Email)
		assert.Equal(t, "https://example.com/a.png", user.Picture)
	})

	t.Run("no name", func(t *testing.T) {
		assert.Equal(t, "noname_522b", customProviderUser("oidc", provider.UserData{"sub": "alice"}, cfg).Name)
	})
}

// TestServerApp_CustomOAuthProviderSignIn signs users in through the custom provider and checks the
// id each of them gets, as the session token carries it
func TestServerApp_CustomOAuthProviderSignIn(t *testing.T) {
	idp := newCustomOAuthStub(t, map[string]string{
		"alice":         `{"sub":"alice","name":"Alice"}`,
		"hash-of-alice": `{"sub":"522b276a356bdf39013dfabea2cd43e141ecc9e8","name":"Mallory"}`,
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
