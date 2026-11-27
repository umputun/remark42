package cmd

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-pkgz/auth/v2/provider"
	"github.com/go-pkgz/auth/v2/token"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store/engine"
)

// the ids the GitHub provider makes: sha1 of the login, and of "gid:" and the numeric account id
const (
	octocatLoginID = "github_cfffe7aa92f4e3791c5c73c6dfc232a13ae38a6b" // login "octocat"
	octocatID      = "github_5a72980f7c56c75b96f744e228219172cd4d1c23" // account 583231
	otherAccountID = "github_8e7c4795e6e5ab66ee0aab2bd2aed7d447943d44" // account 1
)

// githubAccountsMock keeps the accounts per site and user, and fails reads or writes on request
type githubAccountsMock struct {
	accounts map[string]string
	writes   int
	readErr  error
	writeErr error
}

func (m *githubAccountsMock) GetUserGithubAccount(siteID, userID string) (string, error) {
	if m.readErr != nil {
		return "", m.readErr
	}
	return m.accounts[siteID+"/"+userID], nil
}

func (m *githubAccountsMock) SetUserGithubAccount(siteID, userID, value string) (string, error) {
	m.writes++
	if m.writeErr != nil {
		return "", m.writeErr
	}
	m.accounts[siteID+"/"+userID] = value
	return value, nil
}

func TestGithubLoginID(t *testing.T) {
	assert.Equal(t, octocatLoginID, githubLoginID("octocat"), "the id the GitHub provider has always made from the login")
}

func TestBindGithubAccount(t *testing.T) {
	// signIn returns the user as the GitHub provider makes it at a sign-in
	signIn := func(accountID, login string) *token.User {
		u := &token.User{ID: accountID, Name: login}
		u.SetStrAttr(githubLoginAttr, login)
		return u
	}

	t.Run("first sign-in records the account and keeps the login-based id", func(t *testing.T) {
		st := &githubAccountsMock{accounts: map[string]string{}}
		u := signIn(octocatID, "octocat")
		bindGithubAccount(u, "site", st)
		assert.Equal(t, octocatLoginID, u.ID)
		assert.Equal(t, octocatID, st.accounts["site/"+octocatLoginID], "account recorded")
		assert.NotContains(t, u.Attributes, githubLoginAttr, "the login does not go into the token")

		u = signIn(octocatID, "octocat")
		bindGithubAccount(u, "site", st)
		assert.Equal(t, octocatLoginID, u.ID, "the same account again")
		assert.Equal(t, 1, st.writes, "recorded once")
	})

	t.Run("another account under the same login keeps its own id", func(t *testing.T) {
		st := &githubAccountsMock{accounts: map[string]string{"site/" + octocatLoginID: octocatID}}
		u := signIn(otherAccountID, "octocat")
		bindGithubAccount(u, "site", st)
		assert.Equal(t, otherAccountID, u.ID)
		assert.Equal(t, octocatID, st.accounts["site/"+octocatLoginID], "the record stays with the first account")
		assert.Zero(t, st.writes)

		u = signIn(otherAccountID, "octocat")
		bindGithubAccount(u, "site", st)
		assert.Equal(t, otherAccountID, u.ID, "and keeps it at the next sign-in")
	})

	t.Run("the record is per site", func(t *testing.T) {
		st := &githubAccountsMock{accounts: map[string]string{"site/" + octocatLoginID: octocatID}}
		u := signIn(otherAccountID, "octocat")
		bindGithubAccount(u, "other-site", st)
		assert.Equal(t, octocatLoginID, u.ID, "first sign-in of the login on the other site")
		assert.Equal(t, otherAccountID, st.accounts["other-site/"+octocatLoginID])
	})

	t.Run("a refresh is left alone", func(t *testing.T) {
		st := &githubAccountsMock{accounts: map[string]string{"site/" + octocatLoginID: octocatID}}
		u := &token.User{ID: otherAccountID, Name: "octocat"}
		u.SetBoolAttr("blocked", false)
		bindGithubAccount(u, "site", st)
		assert.Equal(t, otherAccountID, u.ID)
		u = &token.User{ID: octocatLoginID, Name: "octocat"}
		bindGithubAccount(u, "site", st)
		assert.Equal(t, octocatLoginID, u.ID)
		assert.Zero(t, st.writes)
	})

	t.Run("another provider is left alone", func(t *testing.T) {
		st := &githubAccountsMock{accounts: map[string]string{}}
		u := &token.User{ID: "google_1234"}
		u.SetStrAttr(githubLoginAttr, "octocat")
		bindGithubAccount(u, "site", st)
		assert.Equal(t, "google_1234", u.ID)
		assert.Zero(t, st.writes)
	})

	// when the account can't be checked the user signs in under the login-based id, as before the check
	t.Run("no numeric account id", func(t *testing.T) {
		st := &githubAccountsMock{accounts: map[string]string{"site/" + octocatLoginID: octocatID}}
		u := signIn(octocatLoginID, "octocat")
		bindGithubAccount(u, "site", st)
		assert.Equal(t, octocatLoginID, u.ID)
		assert.Zero(t, st.writes, "nothing recorded without an account")
	})

	t.Run("store read fails", func(t *testing.T) {
		st := &githubAccountsMock{accounts: map[string]string{}, readErr: errors.New("read failed")}
		u := signIn(otherAccountID, "octocat")
		bindGithubAccount(u, "site", st)
		assert.Equal(t, octocatLoginID, u.ID)
		assert.Zero(t, st.writes)
	})

	t.Run("store write fails", func(t *testing.T) {
		st := &githubAccountsMock{accounts: map[string]string{}, writeErr: errors.New("unsupported detail")}
		u := signIn(octocatID, "octocat")
		bindGithubAccount(u, "site", st)
		assert.Equal(t, octocatLoginID, u.ID)
	})
}

// TestServerApp_GithubAccount covers the GitHub provider as the server registers it and the account check
// in the claims updater, through the token service and the store of a running server
func TestServerApp_GithubAccount(t *testing.T) {
	app := startServerApp(t, func(o ServerCommand) ServerCommand {
		o.Admin.Shared.Admins = []string{octocatLoginID}
		return o
	})

	p, err := app.restSrv.Authenticator.Provider("github")
	require.NoError(t, err)
	gh, ok := p.Provider.(provider.Oauth2Handler)
	require.True(t, ok, "github is an oauth2 provider")
	assert.True(t, gh.GithubNumericID, "the user id is made from the numeric account id")
	assert.Equal(t, provider.UserAttributes{"login": githubLoginAttr}, gh.UserAttributes, "the login goes into an attribute")
	assert.Same(t, app.restSrv.Authenticator.AvatarProxy(), gh.AvatarSaver, "the picture is stored as for other providers")
	// the rest as the authenticator gives its own providers
	assert.Equal(t, strings.TrimSuffix(app.RemarkURL, "/"), gh.URL)
	assert.Same(t, app.restSrv.Authenticator.TokenService(), gh.JwtService)
	assert.Equal(t, authIssuer, gh.Issuer)
	require.NotNil(t, gh.AllowedRedirectHosts, "a provider without allowed hosts redirects anywhere")
	hosts, err := gh.AllowedRedirectHosts.Get()
	require.NoError(t, err)
	wantHosts, err := app.allowedRedirectHosts().Get()
	require.NoError(t, err)
	assert.Equal(t, wantHosts, hosts)

	// signIn makes the token of a sign-in as the GitHub provider asks the token service for it, and
	// returns the user of the token
	signIn := func(accountID string) token.User {
		t.Helper()
		u := token.User{ID: accountID, Name: "octocat"}
		u.SetStrAttr(githubLoginAttr, "octocat")
		tkn, tknErr := app.restSrv.Authenticator.TokenService().Token(token.Claims{
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{"remark"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))},
			User:             &u,
		})
		require.NoError(t, tknErr)
		claims, parseErr := app.restSrv.Authenticator.TokenService().Parse(tkn)
		require.NoError(t, parseErr)
		return *claims.User
	}

	u := signIn(octocatID)
	assert.Equal(t, octocatLoginID, u.ID, "first sign-in keeps the login-based id")
	assert.NotContains(t, u.Attributes, githubLoginAttr, "and the login stays out of the token")
	account, err := app.dataService.GetUserGithubAccount("remark", octocatLoginID)
	require.NoError(t, err)
	assert.Equal(t, octocatID, account, "the account is recorded")

	assert.Equal(t, octocatLoginID, signIn(octocatID).ID, "the same account again")
	assert.True(t, u.IsAdmin(), "the login-based id is an admin")

	// the admin right and the block of the login-based id stay with its account, also after a deletion of
	// its data, and do not pass to the next holder of the login
	require.NoError(t, app.dataService.SetBlock("remark", octocatLoginID, true, 0))
	require.NoError(t, app.dataService.DeleteUserDetail("remark", octocatLoginID, engine.AllUserDetails))
	other := signIn(otherAccountID)
	assert.Equal(t, otherAccountID, other.ID, "another account under the login keeps its own id")
	assert.False(t, other.IsAdmin(), "without the admin right of the earlier holder")
	assert.False(t, other.BoolAttr("blocked"), "and without the block")
	owner := signIn(octocatID)
	assert.Equal(t, octocatLoginID, owner.ID, "the account the login-based id belongs to")
	assert.True(t, owner.IsAdmin(), "keeps the admin right")
	assert.True(t, owner.BoolAttr("blocked"), "and the block")
}
