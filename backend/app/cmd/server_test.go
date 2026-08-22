package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/go-pkgz/auth/v2/provider"
	"github.com/go-pkgz/auth/v2/token"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jessevdk/go-flags"
	"go.uber.org/goleak"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// budget for a server to bind and answer, generous enough for a loaded CI runner
	serverStartTimeout = 30 * time.Second
	serverStartPoll    = 10 * time.Millisecond

	// budget for a server to stop once asked. tight enough to catch a shutdown that hangs,
	// loose enough not to depend on how loaded the runner is
	serverStopTimeout = 10 * time.Second

	// connect budget for a single probe. kept off the poll interval so a slow loopback connect
	// on a loaded runner does not look like a server that is not listening
	probeDialTimeout = time.Second

	// the /auth/ group is limited to 2 req/s, so retries sit at its refill interval rather than
	// above it, which would only manufacture more 429s
	authRetryPoll = 500 * time.Millisecond
)

func TestServerApp(t *testing.T) {
	port := chooseUnusedPort(t)
	app, ctx, cancel := prepServerApp(t, func(o ServerCommand) ServerCommand {
		o.Port = port
		return o
	})

	go func() { _ = app.run(ctx) }()
	waitForHTTPServerStart(t, port)

	// send ping
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/ping", port))
	defer http.DefaultClient.CloseIdleConnections()
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "pong", string(body))

	// add comment
	client := http.Client{Timeout: 10 * time.Second}
	defer client.CloseIdleConnections()
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment?site=remark", port),
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "remark"}}`))
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body, _ = io.ReadAll(resp.Body)
	t.Log(string(body))

	email, err := app.dataService.AdminStore.Email("")
	assert.NoError(t, err)
	assert.Equal(t, "admin@demo.remark42.com", email, "default admin email")

	cancel()
	app.Wait()
}

func TestServerApp_DevMode(t *testing.T) {
	port := chooseUnusedPort(t)
	app, ctx, cancel := prepServerApp(t, func(o ServerCommand) ServerCommand {
		o.Port = port
		o.AdminPasswd = "password"
		o.Auth.Dev = true
		return o
	})

	go func() { _ = app.run(ctx) }()
	waitForHTTPServerStart(t, port)

	providers := app.restSrv.Authenticator.Providers()
	require.Equal(t, 11+1, len(providers), "extra auth provider")
	assert.Equal(t, "dev", providers[len(providers)-2].Name(), "dev auth provider")
	// send ping
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/ping", port))
	defer http.DefaultClient.CloseIdleConnections()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, "pong", string(body))

	cancel()
	app.Wait()
}

func TestServerApp_CustomOAuthProvider(t *testing.T) {
	port := chooseUnusedPort(t)
	app, ctx, cancel := prepServerApp(t, func(o ServerCommand) ServerCommand {
		o.Port = port
		o.Auth.Custom.Name = "oidc"
		o.Auth.Custom.CID = "cid"
		o.Auth.Custom.CSEC = "csec"
		o.Auth.Custom.AuthURL = "https://example.com/oauth2/authorize"
		o.Auth.Custom.TokenURL = "https://example.com/oauth2/token"
		o.Auth.Custom.InfoURL = "https://example.com/oauth2/userinfo"
		return o
	})

	go func() { _ = app.run(ctx) }()
	waitForHTTPServerStart(t, port)

	providers := app.restSrv.Authenticator.Providers()
	require.Equal(t, 11+1, len(providers), "extra auth provider")
	assert.Equal(t, "oidc", providers[len(providers)-2].Name(), "custom auth provider")

	cancel()
	app.Wait()
}

func TestServerApp_AnonMode(t *testing.T) {
	port := chooseUnusedPort(t)
	app, ctx, cancel := prepServerApp(t, func(o ServerCommand) ServerCommand {
		o.Port = port
		o.Auth.Anonymous = true
		return o
	})

	go func() { _ = app.run(ctx) }()
	waitForHTTPServerStart(t, port)

	providers := app.restSrv.Authenticator.Providers()
	require.Equal(t, 11+1, len(providers), "extra auth provider for anon")
	assert.Equal(t, "anonymous", providers[len(providers)-1].Name(), "anon auth provider")

	client := http.Client{Timeout: 10 * time.Second}
	defer client.CloseIdleConnections()

	// send ping
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/api/v1/ping", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "pong", string(body))

	// try to login with good name
	resp = getRetryThrottled(t, &client, fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=blah123&aud=remark", port))
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// try to add a comment as good anonymous
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment?site=remark", port),
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "remark"}}`))
	require.NoError(t, err)

	tkn, claims := getAuthFromCookie(t, app, resp)
	require.NotEmpty(t, tkn)
	assert.False(t, claims.User.BoolAttr("blocked"), "should not be blocked")
	req.Header.Add("X-JWT", tkn)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// try to login with non-latin name
	nonLatin := fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=Раз_Два%20%20Три_34567&aud=remark", port)
	resp = getRetryThrottled(t, &client, nonLatin)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// try to login with bad name
	resp = getRetryThrottled(t, &client, fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=**blah123&aud=remark", port))
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// try to login with short name
	resp = getRetryThrottled(t, &client, fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=bl%%20%%20&aud=remark", port))
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// try to login with name what have space in prefix
	resp = getRetryThrottled(t, &client, fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=%%20somebody&aud=remark", port))
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// try to login with name what have space in suffix
	resp = getRetryThrottled(t, &client, fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=somebody%%20&aud=remark", port))
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// try to login with long name
	ln := strings.Repeat("x", 65)
	resp = getRetryThrottled(t, &client, fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=%s&aud=remark", port, ln))
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// try to login with admin name
	resp = getRetryThrottled(t, &client, fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=umpUtun&aud=remark", port))
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// try to add a comment as anonymous with admin name
	req, err = http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment?site=remark", port),
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "remark"}}`))
	require.NoError(t, err)

	tkn, claims = getAuthFromCookie(t, app, resp)
	require.NotEmpty(t, tkn)
	assert.True(t, claims.User.BoolAttr("blocked"), "should be blocked")
	req.Header.Add("X-JWT", tkn)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	cancel()
	app.Wait()
}

func getAuthFromCookie(t *testing.T, app *serverApp, resp *http.Response) (tkn string, claims token.Claims) {
	var err error
	for _, c := range resp.Cookies() {
		if c.Name == "JWT" {
			tkn = c.Value
			claims, err = app.restSrv.Authenticator.TokenService().Parse(c.Value)
			require.NoError(t, err)
		}
	}
	return tkn, claims
}

func TestServerApp_WithSSL(t *testing.T) {
	opts := ServerCommand{}
	sslPort := chooseUnusedPort(t)
	opts.SetCommon(CommonOpts{RemarkURL: fmt.Sprintf("https://localhost:%d", sslPort), SharedSecret: "123456"})

	// prepare options
	p := flags.NewParser(&opts, flags.Default)
	port := chooseUnusedPort(t)
	_, err := p.ParseArgs([]string{"--admin-passwd=password", "--port=" + strconv.Itoa(port), "--store.bolt.path=/tmp/xyz", "--backup=/tmp",
		"--avatar.type=bolt", "--avatar.bolt.file=/tmp/ava-test.db",
		"--ssl.type=static", "--ssl.cert=testdata/cert.pem", "--ssl.key=testdata/key.pem",
		"--ssl.port=" + strconv.Itoa(sslPort), "--image.fs.path=/tmp"})
	require.NoError(t, err)
	defer os.Remove("/tmp/xyz")
	defer os.Remove("/tmp/xyz/remark.db")
	defer os.Remove("/tmp/ava-test.db")

	// create app
	app, err := opts.newServerApp(context.Background())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // this context is not the one createAppFromCmd registers for cleanup
	go func() { _ = app.run(ctx) }()
	waitForServerStart(t, sslPort, port) // the redirect check below uses the plain http port

	client := http.Client{
		// prevent http redirect
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},

		// allow self-signed certificate
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	defer client.CloseIdleConnections()

	// check http to https redirect response
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/blah?param=1", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
	assert.Equal(t, fmt.Sprintf("https://localhost:%d/blah?param=1", sslPort), resp.Header.Get("Location"))

	// check https server
	resp, err = client.Get(fmt.Sprintf("https://localhost:%d/ping", sslPort))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "pong", string(body))

	cancel()
	app.Wait()
}

func TestServerApp_WithRemote(t *testing.T) {
	opts := ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	// prepare options
	p := flags.NewParser(&opts, flags.Default)
	port := chooseUnusedPort(t)
	_, err := p.ParseArgs([]string{"--admin-passwd=password", "--cache.type=none",
		"--store.type=rpc", "--store.rpc.api=http://127.0.0.1",
		"--port=" + strconv.Itoa(port), "--avatar.fs.path=/tmp",
		"--admin.type=rpc", "--admin.rpc.secret_per_site", "--admin.rpc.api=http://127.0.0.1"})
	require.NoError(t, err)
	opts.Auth.Github.CSEC, opts.Auth.Github.CID = "csec", "cid"
	opts.BackupLocation, opts.Image.FS.Path = "/tmp", "/tmp"

	// create app
	app, err := opts.newServerApp(context.Background())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // this context is not the one createAppFromCmd registers for cleanup
	go func() { _ = app.run(ctx) }()
	waitForHTTPServerStart(t, port)

	// send ping
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/ping", port))
	defer http.DefaultClient.CloseIdleConnections()
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "pong", string(body))

	cancel()
	app.Wait()
}

func TestServerApp_Failed(t *testing.T) {
	opts := ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	p := flags.NewParser(&opts, flags.Default)

	// RO bolt location
	_, err := p.ParseArgs([]string{"--backup=/tmp", "--store.bolt.path=/dev/null", "--image.fs.path=/tmp"})
	assert.NoError(t, err)
	_, err = opts.newServerApp(context.Background())
	assert.EqualError(t, err, "failed to make data store engine: failed to create bolt store: can't make directory /dev/null: mkdir /dev/null: not a directory")
	t.Log(err)

	// RO backup location
	opts = ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	_, err = p.ParseArgs([]string{"--store.bolt.path=/tmp", "--backup=/dev/null/not-writable"})
	assert.NoError(t, err)
	defer os.Remove("/tmp/remark.db")
	_, err = opts.newServerApp(context.Background())
	assert.EqualError(t, err, "failed to create backup store: can't make directory /dev/null/not-writable: mkdir /dev/null: not a directory")
	t.Log(err)

	// invalid url
	opts = ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "demo.remark42.com", SharedSecret: "123456"})

	_, err = p.ParseArgs([]string{"--backup=/tmp", "----store.bolt.path=/tmp"})
	assert.NoError(t, err)
	_, err = opts.newServerApp(context.Background())
	assert.EqualError(t, err, "invalid remark42 url demo.remark42.com")
	t.Log(err)

	// invalid trusted proxy CIDR fails fast, before any resource is created
	opts = ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})
	p = flags.NewParser(&opts, flags.Default)
	_, err = p.ParseArgs([]string{"--backup=/tmp", "--trusted-proxy=nonsense"})
	assert.NoError(t, err)
	_, err = opts.newServerApp(context.Background())
	assert.EqualError(t, err, `invalid --trusted-proxy: invalid trusted proxy "nonsense"`)
	t.Log(err)

	// wrong store type
	opts = ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	_, err = p.ParseArgs([]string{"--backup=/tmp", "--store.type=blah"})
	assert.Error(t, err, "blah is invalid type")

	opts.Store.Type = "blah"
	_, err = opts.newServerApp(context.Background())
	assert.EqualError(t, err, "failed to make data store engine: unsupported store type blah")
	t.Log(err)

	// wrong redis location
	opts = ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})
	p = flags.NewParser(&opts, flags.Default)
	_, err = p.ParseArgs([]string{"--store.bolt.path=/tmp", "--cache.type=redis_pub_sub", "--cache.redis_addr=wrong_address"})
	assert.NoError(t, err)
	_, err = opts.newServerApp(context.Background())
	assert.EqualError(t, err,
		"failed to make cache: cache backend initialization, redis PubSub initialisation: "+
			"problem subscribing to channel remark42-cache on address wrong_address: "+
			"dial tcp: address wrong_address: missing port in address")
	t.Log(err)

	// wrong apple private key type
	opts = ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})
	p = flags.NewParser(&opts, flags.Default)
	_, err = p.ParseArgs([]string{"--auth.apple.cid=123", "--auth.apple.tid=123",
		"--auth.apple.kid=123", "--auth.apple.private-key-filepath=testdata/apple-bad.p8"})
	assert.NoError(t, err)
	_, err = opts.newServerApp(context.Background())
	assert.EqualError(t, err,
		"failed to make authenticator: an AppleProvider creating failed: "+
			"provided private key is not ECDSA")
	t.Log(err)

	// incomplete custom oauth config
	opts = ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})
	p = flags.NewParser(&opts, flags.Default)
	_, err = p.ParseArgs([]string{"--store.bolt.path=/tmp", "--backup=/tmp", "--image.fs.path=/tmp", "--auth.custom.name=oidc", "--auth.custom.cid=123"})
	assert.NoError(t, err)
	_, err = opts.newServerApp(context.Background())
	assert.EqualError(t, err,
		"failed to make authenticator: custom oauth provider configuration is incomplete, missing: "+
			"AUTH_CUSTOM_CSEC, AUTH_CUSTOM_AUTH_URL, AUTH_CUSTOM_TOKEN_URL, AUTH_CUSTOM_INFO_URL")
	t.Log(err)
}

func TestIsReservedCustomProviderName(t *testing.T) {
	reserved := []string{
		"email", "anonymous", "google", "github", "facebook", "yandex", "twitter",
		"microsoft", "patreon", "discord", "telegram", "dev", "apple",
	}

	for _, name := range reserved {
		t.Run(name, func(t *testing.T) {
			assert.True(t, isReservedCustomProviderName(name))
		})
	}

	assert.False(t, isReservedCustomProviderName("oidc"))
}

func TestIsValidCustomProviderName(t *testing.T) {
	valid := []string{"oidc", "codeberg", "provider_1", "provider-1", "a1"}
	for _, name := range valid {
		t.Run("valid_"+name, func(t *testing.T) {
			assert.True(t, isValidCustomProviderName(name))
		})
	}

	invalid := []string{"", " has-space", "has space", "Uppercase", "provider!", "-provider", "_provider"}
	for _, name := range invalid {
		t.Run("invalid_"+strings.ReplaceAll(name, " ", "_"), func(t *testing.T) {
			assert.False(t, isValidCustomProviderName(name))
		})
	}
}

func TestCustomProviderSourceID(t *testing.T) {
	cfg := CustomAuthGroup{IDField: "sub", EmailField: "email", NameField: "name", PictureField: "picture"}

	assert.Equal(t, "user-1", customProviderSourceID(provider.UserData{"sub": "user-1", "email": "a@example.com"}, cfg))
	assert.Equal(t, "a@example.com", customProviderSourceID(provider.UserData{"email": "a@example.com"}, cfg))
	assert.Equal(t, "alice", customProviderSourceID(provider.UserData{"name": "alice"}, cfg))
	assert.Equal(t, "https://example.com/avatar.png", customProviderSourceID(provider.UserData{"picture": "https://example.com/avatar.png"}, cfg))
	assert.Equal(t, `{"login":"alice"}`, customProviderSourceID(provider.UserData{"login": "alice"}, cfg))
	assert.Equal(t, "{}", customProviderSourceID(provider.UserData{}, cfg))
}

func TestServerApp_InvalidCustomOAuthProviderName(t *testing.T) {
	baseArgs := []string{
		"--store.bolt.path=/tmp",
		"--backup=/tmp",
		"--image.fs.path=/tmp",
		"--auth.custom.cid=123",
		"--auth.custom.csec=456",
		"--auth.custom.auth-url=https://example.com/oauth2/authorize",
		"--auth.custom.token-url=https://example.com/oauth2/token",
		"--auth.custom.info-url=https://example.com/oauth2/userinfo",
	}

	t.Run("reserved", func(t *testing.T) {
		opts := ServerCommand{}
		opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})
		p := flags.NewParser(&opts, flags.Default)
		_, err := p.ParseArgs(append(baseArgs, "--auth.custom.name=twitter"))
		require.NoError(t, err)

		_, err = opts.newServerApp(context.Background())
		assert.EqualError(t, err, `failed to make authenticator: custom oauth provider name "twitter" is reserved`)
	})

	t.Run("not_url_safe", func(t *testing.T) {
		opts := ServerCommand{}
		opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})
		p := flags.NewParser(&opts, flags.Default)
		_, err := p.ParseArgs(append(baseArgs, "--auth.custom.name=bad name"))
		require.NoError(t, err)

		_, err = opts.newServerApp(context.Background())
		assert.EqualError(t, err, `failed to make authenticator: custom oauth provider name "bad name" is invalid, expected pattern "^[a-z0-9][a-z0-9_-]*$"`)
	})
}

func TestServerApp_Shutdown(t *testing.T) {
	port := chooseUnusedPort(t)
	app, ctx, cancel := prepServerApp(t, func(o ServerCommand) ServerCommand {
		o.Port = port
		return o
	})

	// cancel once the server actually answers, so the test measures shutdown and not startup.
	// the deferred cancel also covers a failed wait, keeping app.run from racing the next test
	errCh := make(chan error, 1)
	go func() { errCh <- app.run(ctx) }()
	defer cancel()
	waitForHTTPServerStart(t, port)
	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(serverStopTimeout):
		t.Fatal("server app did not stop after context cancel")
	}
	app.Wait()
}

// TestServerApp_ClaimsUpd covers the hook the authenticator runs on every token mint, refresh
// included: it stamps admin, blocked and email onto the claims and blocks impersonation of a
// restricted name. Calling the updater directly keeps it independent of when a token expires.
func TestServerApp_ClaimsUpd(t *testing.T) {
	port := chooseUnusedPort(t)
	app, ctx, cancel := prepServerApp(t, func(o ServerCommand) ServerCommand {
		o.Port = port
		return o
	})

	// the app owns stores and services that only run closes, so it goes through the usual
	// lifecycle here rather than being built and abandoned
	go func() { _ = app.run(ctx) }()
	waitForHTTPServerStart(t, port)
	defer app.Wait()
	defer cancel()

	upd := app.restSrv.Authenticator.TokenService().ClaimsUpd
	require.NotNil(t, upd, "claims updater wired into the token service")

	claimsFor := func(id, name string) token.Claims {
		return token.Claims{
			RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{"remark"}},
			User:             &token.User{ID: id, Name: name},
		}
	}

	t.Run("plain user gets no attributes", func(t *testing.T) {
		res := upd.Update(claimsFor("provider1_dev", "developer"))
		assert.False(t, res.User.IsAdmin(), "not an admin")
		assert.False(t, res.User.BoolAttr("blocked"), "not blocked")
		assert.Empty(t, res.User.Email, "no email on file")
	})

	t.Run("admin from the admin store", func(t *testing.T) {
		res := upd.Update(claimsFor("id1", "admin one"))
		assert.True(t, res.User.IsAdmin(), "id1 is listed as admin")
	})

	t.Run("blocked user carries the blocked attribute", func(t *testing.T) {
		require.NoError(t, app.restSrv.DataService.SetBlock("remark", "blocked_user", true, time.Hour))
		res := upd.Update(claimsFor("blocked_user", "blocked"))
		assert.True(t, res.User.BoolAttr("blocked"), "block is reflected on refresh")
	})

	t.Run("email is read from the store", func(t *testing.T) {
		_, err := app.restSrv.DataService.SetUserEmail("remark", "with_email", "user@example.com")
		require.NoError(t, err)
		res := upd.Update(claimsFor("with_email", "someone"))
		assert.Equal(t, "user@example.com", res.User.Email)
	})

	t.Run("anonymous impersonating a restricted name is blocked", func(t *testing.T) {
		res := upd.Update(claimsFor("anonymous_x", " UmpUtun "))
		assert.True(t, res.User.BoolAttr("blocked"), "restricted name matched case and space insensitively")
	})

	t.Run("email user impersonating a restricted name is blocked", func(t *testing.T) {
		res := upd.Update(claimsFor("email_x", "bobuk"))
		assert.True(t, res.User.BoolAttr("blocked"))
	})

	t.Run("regular user may carry a restricted name", func(t *testing.T) {
		res := upd.Update(claimsFor("provider1_someone", "umputun"))
		assert.False(t, res.User.BoolAttr("blocked"), "only anonymous and email logins are checked")
	})

	t.Run("claims without a user pass through", func(t *testing.T) {
		res := upd.Update(token.Claims{RegisteredClaims: jwt.RegisteredClaims{Audience: jwt.ClaimStrings{"remark"}}})
		assert.Nil(t, res.User)
	})

	t.Run("claims without exactly one audience pass through", func(t *testing.T) {
		c := claimsFor("id1", "admin one")
		c.Audience = jwt.ClaimStrings{"remark", "second"}
		res := upd.Update(c)
		assert.False(t, res.User.IsAdmin(), "attributes need a single audience to resolve the site")
	})
}

func TestServerApp_MainSignal(t *testing.T) {
	sigErr := make(chan error, 1)

	s := ServerCommand{}
	s.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	p := flags.NewParser(&s, flags.Default)
	port := chooseUnusedPort(t)
	args := []string{"test", "--store.bolt.path=/tmp/xyz", "--backup=/tmp", "--avatar.type=bolt",
		"--avatar.bolt.file=/tmp/ava-test.db", "--port=" + strconv.Itoa(port), "--image.fs.path=/tmp"}
	defer os.Remove("/tmp/xyz")
	defer os.Remove("/tmp/xyz/remark.db")
	defer os.Remove("/tmp/ava-test.db")
	_, err := p.ParseArgs(args)
	require.NoError(t, err)
	// the signal goes out only once the server answers: SIGTERM landing before the handler is
	// installed kills the test process, so a wait that timed out reports instead of sending it
	go func() {
		started := waitForServerPort(port, serverStartTimeout)
		// signal either way: Execute blocks until it gets one, so bailing out here would hang
		// the test until the package timeout instead of failing with the reason
		killErr := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		if !started {
			killErr = fmt.Errorf("server on port %d didn't start", port)
		}
		sigErr <- killErr
	}()

	err = s.Execute(args)
	assert.NoError(t, err, "execute should be without errors")
	require.NoError(t, <-sigErr, "SIGTERM not delivered")
}

func TestServerApp_RunCanceledBeforeRESTStart(t *testing.T) {
	port := chooseUnusedPort(t)
	app, ctx, cancel := prepServerApp(t, func(o ServerCommand) ServerCommand {
		o.Port = port
		return o
	})
	cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- app.run(ctx) }()

	// the budget is generous on purpose: the assertion is that run exits rather than hangs, and
	// store construction can take a while on a loaded runner
	select {
	case err := <-errCh:
		require.NoError(t, err)
		app.Wait()
	case <-time.After(serverStartTimeout):
		waitForHTTPServerStart(t, port)
		app.restSrv.Shutdown()
		select {
		case <-errCh:
			app.Wait()
		case <-time.After(serverStartTimeout):
			t.Fatal("server app did not stop after forced REST shutdown")
		}
		t.Fatal("server app should exit when context is canceled before REST server starts")
	}
}

func TestServerApp_DeprecatedArgs(t *testing.T) {
	s := ServerCommand{}
	s.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	p := flags.NewParser(&s, flags.Default)
	args := []string{
		"test",
		"--notify.type=email",
		"--notify.type=telegram",
		"--notify.users=none",
		"--notify.admins=none",
		"--img-proxy",
		"--notify.email.notify_admin",
		"--auth.email.host=smtp.example.org",
		"--auth.email.port=666",
		"--auth.email.tls",
		"--auth.email.user=test_user",
		"--auth.email.passwd=test_password",
		"--auth.email.timeout=15s",
		"--auth.email.template=file.tmpl",
		"--notify.telegram.token=abcd",
		"--notify.telegram.timeout=3m",
		"--notify.telegram.api=http://example.org",
		"--auth.twitter.cid=123",
		"--auth.twitter.csec=456",
	}
	assert.Empty(t, s.SMTP.Host)
	assert.Empty(t, s.SMTP.Port)
	assert.Empty(t, s.SMTP.TLS)
	assert.Empty(t, s.SMTP.Username)
	assert.Empty(t, s.SMTP.Password)
	assert.Empty(t, s.SMTP.TimeOut)
	_, err := p.ParseArgs(args)
	require.NoError(t, err)
	deprecatedFlags := s.HandleDeprecatedFlags()
	assert.ElementsMatch(t,
		[]DeprecatedFlag{
			{Old: "auth.email.host", New: "smtp.host", Version: "1.5"},
			{Old: "auth.email.port", New: "smtp.port", Version: "1.5"},
			{Old: "auth.email.tls", New: "smtp.tls", Version: "1.5"},
			{Old: "auth.email.user", New: "smtp.username", Version: "1.5"},
			{Old: "auth.email.passwd", New: "smtp.password", Version: "1.5"},
			{Old: "auth.email.timeout", New: "smtp.timeout", Version: "1.5"},
			{Old: "auth.email.template", Version: "1.5"},
			{Old: "img-proxy", New: "image-proxy.http2https", Version: "1.5"},
			{Old: "notify.email.notify_admin", New: "notify.admins=email", Version: "1.9"},
			{Old: "notify.type", New: "notify.(users|admins)", Version: "1.9"},
			{Old: "notify.telegram.token", New: "telegram.token", Version: "1.9"},
			{Old: "notify.telegram.timeout", New: "telegram.timeout", Version: "1.9"},
			{Old: "notify.telegram.api", Version: "1.9"},
			{Old: "auth.twitter.cid", Version: "1.14"},
			{Old: "auth.twitter.csec", Version: "1.14"},
		},
		deprecatedFlags)
	assert.Equal(t, "smtp.example.org", s.SMTP.Host)
	assert.Equal(t, 666, s.SMTP.Port)
	assert.Equal(t, true, s.SMTP.TLS)
	assert.Equal(t, "test_user", s.SMTP.Username)
	assert.Equal(t, "test_password", s.SMTP.Password)
	assert.Equal(t, 15*time.Second, s.SMTP.TimeOut)
}

func TestServerApp_DeprecatedArgsCollisions(t *testing.T) {
	s := ServerCommand{}
	s.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	p := flags.NewParser(&s, flags.Default)
	args := []string{
		"test",
		"--auth.email.host=smtp-old.example.org",
		"--smtp.host=smtp-new.example.org",
		"--auth.email.port=666",
		"--smtp.port=999",
		"--auth.email.user=test_user",
		"--smtp.username=new_test_user",
		"--auth.email.passwd=test_password",
		"--smtp.password=new_test_password",
		"--auth.email.timeout=15s",
		"--smtp.timeout=20s",
		"--notify.type=telegram",
		"--notify.users=telegram",
		"--notify.admins=none",
		"--notify.telegram.token=abcd",
		"--telegram.token=dcba",
		"--notify.telegram.timeout=3m",
		"--telegram.timeout=5m",
	}
	_, err := p.ParseArgs(args)
	require.NoError(t, err)
	deprecatedFlagsCollisions := s.findDeprecatedFlagsCollisions()
	assert.ElementsMatch(t,
		[]DeprecatedFlag{
			{Old: "notify.type", New: "notify.(users|admins)", Collision: true},
			{Old: "auth.email.host", New: "smtp.host", Collision: true},
			{Old: "auth.email.port", New: "smtp.port", Collision: true},
			{Old: "auth.email.user", New: "smtp.username", Collision: true},
			{Old: "auth.email.passwd", New: "smtp.password", Collision: true},
			{Old: "auth.email.timeout", New: "smtp.timeout", Collision: true},
			{Old: "notify.telegram.token", New: "telegram.token", Collision: true},
			{Old: "notify.telegram.timeout", New: "telegram.timeout", Collision: true},
		},
		deprecatedFlagsCollisions)

	// case which should return nothing
	s = ServerCommand{}
	s.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})
	p = flags.NewParser(&s, flags.Default)
	args = []string{
		"test",
		"--auth.email.host=smtp-old.example.org",
		"--smtp.host=''",
	}
	_, err = p.ParseArgs(args)
	require.NoError(t, err)
	deprecatedFlagsCollisions = s.findDeprecatedFlagsCollisions()
	assert.Empty(t, []DeprecatedFlag{}, deprecatedFlagsCollisions)
}

func Test_ACMEEmail(t *testing.T) {
	cmd := ServerCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: "https://remark.com:443", SharedSecret: "123456"})
	p := flags.NewParser(&cmd, flags.Default)
	args := []string{"--ssl.type=auto"}
	_, err := p.ParseArgs(args)
	require.NoError(t, err)
	cfg, err := cmd.makeSSLConfig()
	require.NoError(t, err)
	assert.Equal(t, "admin@remark.com", cfg.ACMEEmail)

	cmd = ServerCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: "https://remark.com", SharedSecret: "123456"})
	p = flags.NewParser(&cmd, flags.Default)
	args = []string{"--ssl.type=auto", "--ssl.acme-email=adminname@adminhost.com"}
	_, err = p.ParseArgs(args)
	require.NoError(t, err)
	cfg, err = cmd.makeSSLConfig()
	require.NoError(t, err)
	assert.Equal(t, "adminname@adminhost.com", cfg.ACMEEmail)

	cmd = ServerCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: "https://remark.com", SharedSecret: "123456"})
	p = flags.NewParser(&cmd, flags.Default)
	args = []string{"--ssl.type=auto", "--admin.type=shared", "--admin.shared.email=superadmin@admin.com"}
	_, err = p.ParseArgs(args)
	require.NoError(t, err)
	cfg, err = cmd.makeSSLConfig()
	require.NoError(t, err)
	assert.Equal(t, "superadmin@admin.com", cfg.ACMEEmail)

	cmd = ServerCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: "https://remark.com:443", SharedSecret: "123456"})
	p = flags.NewParser(&cmd, flags.Default)
	args = []string{"--ssl.type=auto", "--admin.type=shared"}
	_, err = p.ParseArgs(args)
	require.NoError(t, err)
	cfg, err = cmd.makeSSLConfig()
	require.NoError(t, err)
	assert.Equal(t, "admin@remark.com", cfg.ACMEEmail)
}

func TestServerAuthHooks(t *testing.T) {
	port := chooseUnusedPort(t)
	app, ctx, cancel := prepServerApp(t, func(o ServerCommand) ServerCommand {
		o.Port = port
		return o
	})

	go func() { _ = app.run(ctx) }()
	waitForHTTPServerStart(t, port)

	// make a token for user dev. nothing here checks expiry, so the lifetime only has to
	// outlast the whole test
	tkService := app.restSrv.Authenticator.TokenService()
	tkService.TokenDuration = time.Hour

	claims := token.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{"remark"},
			Issuer:    "remark",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		},
		User: &token.User{
			ID:   "github_dev",
			Name: "developer one",
		},
	}
	tk, err := tkService.Token(claims)
	require.NoError(t, err)
	t.Log(tk)

	client := http.Client{Timeout: 10 * time.Second}
	defer client.CloseIdleConnections()

	// add comment
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment?site=remark", port),
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/p/2018/12/29/podcast-630/", "site": "remark"}}`))
	require.NoError(t, err)
	req.Header.Set("X-JWT", tk)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "non-blocked user able to post")

	// try to add comment with no-aud claim
	badClaimsNoAud := claims
	badClaimsNoAud.Audience = jwt.ClaimStrings{""}
	tkNoAud, err := tkService.Token(badClaimsNoAud)
	require.NoError(t, err)
	t.Logf("no-aud claims: %s", tkNoAud)
	req, err = http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment?site=remark", port),
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/p/2018/12/29/podcast-631/",
	"site": "remark"}}`))
	require.NoError(t, err)
	req.Header.Set("X-JWT", tkNoAud)
	resp, err = client.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "user without aud claim rejected, \n"+tkNoAud+"\n"+string(body))

	// try to add comment with multiple auds
	badClaimsMultipleAud := claims
	badClaimsMultipleAud.Audience = jwt.ClaimStrings{"remark", "second_aud"}
	tkMultipleAuds, err := tkService.Token(badClaimsMultipleAud)
	require.NoError(t, err)
	t.Logf("multiple aud claims: %s", tkMultipleAuds)
	req, err = http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment?site=remark", port),
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/p/2018/12/29/podcast-631/",
	"site": "remark"}}`))
	require.NoError(t, err)
	req.Header.Set("X-JWT", tkMultipleAuds)
	resp, err = client.Do(req)
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "user with multiple auds claim rejected, \n"+tkMultipleAuds+"\n"+string(body))

	// try to add comment without user set
	badClaimsNoUser := claims
	badClaimsNoUser.Audience = jwt.ClaimStrings{"remark"}
	badClaimsNoUser.User = nil
	tkNoUser, err := tkService.Token(badClaimsNoUser)
	require.NoError(t, err)
	t.Logf("no user claims: %s", tkNoUser)
	req, err = http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment?site=remark", port),
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/p/2018/12/29/podcast-631/",
	"site": "remark"}}`))
	require.NoError(t, err)
	req.Header.Set("X-JWT", tkNoUser)
	resp, err = client.Do(req)
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "user without user information rejected, \n"+tkNoUser+"\n"+string(body))

	// block user github_dev as admin
	req, err = http.NewRequest(http.MethodPut,
		fmt.Sprintf("http://localhost:%d/api/v1/admin/user/github_dev?site=remark&block=1&ttl=10d", port), http.NoBody)
	assert.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "user github_dev blocked")
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	t.Log(string(b))

	// try add a comment with blocked user
	req, err = http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment?site=remark", port),
		strings.NewReader(`{"text": "test 123 blah", "locator":{"url": "https://radio-t.com/blah1", "site": "remark"}}`))
	require.NoError(t, err)
	req.Header.Set("X-JWT", tk)
	resp, err = client.Do(req)
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "blocked user can't post, \n"+tk+"\n"+string(body))

	cancel()
	app.Wait()
	client.CloseIdleConnections()
}

func TestServerCommand_parseSameSite(t *testing.T) {
	tbl := []struct {
		inp string
		res http.SameSite
	}{
		{"", http.SameSiteDefaultMode},
		{"default", http.SameSiteDefaultMode},
		{"blah", http.SameSiteDefaultMode},
		{"none", http.SameSiteNoneMode},
		{"lax", http.SameSiteLaxMode},
		{"strict", http.SameSiteStrictMode},
	}

	cmd := ServerCommand{}
	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.res, cmd.parseSameSite(tt.inp))
		})
	}
}

func Test_splitAtCommas(t *testing.T) {
	tbl := []struct {
		inp string
		res []string
	}{
		{"a string", []string{"a string"}},
		{"vv1, vv2, vv3", []string{"vv1", "vv2", "vv3"}},
		{`"vv1, blah", vv2, vv3`, []string{"vv1, blah", "vv2", "vv3"}},
		{
			`Access-Control-Allow-Headers:"DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type",header123:val, foo:"bar1,bar2"`,
			[]string{"Access-Control-Allow-Headers:\"DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type\"", "header123:val", "foo:\"bar1,bar2\""},
		},
		{"", []string{}},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.res, splitAtCommas(tt.inp))
		})
	}
}

func Test_getAllowedDomains(t *testing.T) {
	tbl := []struct {
		s              ServerCommand
		allowedDomains []string
	}{
		// correct example, parsed and returned as allowed domain
		{ServerCommand{AllowedHosts: []string{}, CommonOpts: CommonOpts{RemarkURL: "https://remark42.example.org"}}, []string{"example.org"}},
		{ServerCommand{AllowedHosts: []string{}, CommonOpts: CommonOpts{RemarkURL: "http://remark42.example.org"}}, []string{"example.org"}},
		{ServerCommand{AllowedHosts: []string{}, CommonOpts: CommonOpts{RemarkURL: "http://localhost"}}, []string{"localhost"}},
		// incorrect URLs, so Hostname is empty but returned list doesn't include empty string as it would allow any domain
		{ServerCommand{AllowedHosts: []string{}, CommonOpts: CommonOpts{RemarkURL: "bad hostname"}}, []string{}},
		{ServerCommand{AllowedHosts: []string{}, CommonOpts: CommonOpts{RemarkURL: "not_a_hostname"}}, []string{}},
		// test removal of 'self', multiple AllowedHosts. No deduplication is expected
		{ServerCommand{AllowedHosts: []string{"'self'", "example.org", "test.example.org", "remark42.com"}, CommonOpts: CommonOpts{RemarkURL: "https://example.org"}}, []string{"example.org", "test.example.org", "remark42.com", "example.org"}},
	}
	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.allowedDomains, tt.s.getAllowedDomains())
		})
	}
}

func Test_getAllowedRedirectHosts(t *testing.T) {
	tbl := []struct {
		name  string
		hosts []string
		want  []string
	}{
		{name: "empty", hosts: nil, want: []string{}},
		{name: "bare hostnames pass through", hosts: []string{"example.com", "admin.example.com"}, want: []string{"example.com", "admin.example.com"}},
		{name: "https scheme stripped", hosts: []string{"https://example.com"}, want: []string{"example.com"}},
		{name: "http scheme stripped", hosts: []string{"http://example.com"}, want: []string{"example.com"}},
		{name: "scheme with path strips path", hosts: []string{"https://example.com/embed"}, want: []string{"example.com"}},
		{name: "explicit port preserved as host:port", hosts: []string{"example.com:8080"}, want: []string{"example.com:8080"}},
		{name: "scheme with explicit port preserved", hosts: []string{"https://example.com:8443"}, want: []string{"example.com:8443"}},
		{name: "scheme without port stays bare host", hosts: []string{"https://example.com"}, want: []string{"example.com"}},
		{name: "self sentinel filtered", hosts: []string{"'self'", "self", `"self"`, "example.com"}, want: []string{"example.com"}},
		{name: "wildcards filtered", hosts: []string{"*", "*.example.com", "https://*.example.com", "example.com"}, want: []string{"example.com"}},
		{name: "empty entries filtered", hosts: []string{"", "  ", "example.com"}, want: []string{"example.com"}},
		{name: "mixed real-world", hosts: []string{"'self'", "https://blog.example.com", "admin.example.com:8443", "*.cdn.example.com"},
			want: []string{"blog.example.com", "admin.example.com:8443"}},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			s := ServerCommand{AllowedHosts: tt.hosts}
			assert.Equal(t, tt.want, s.getAllowedRedirectHosts())
		})
	}
}

// chooseUnusedPort asks the kernel for a free port from the ephemeral range, which makes a
// collision between concurrently running package test binaries very unlikely
func chooseUnusedPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "no free port available")
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())
	return port
}

// waitForHTTPServerStart blocks until the server on port answers, failing the test naming the
// port if it never does
func waitForHTTPServerStart(t *testing.T, port int) {
	t.Helper()
	client := http.Client{Timeout: time.Second}
	defer client.CloseIdleConnections()
	require.Eventually(t, func() bool {
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port))
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return true
	}, serverStartTimeout, serverStartPoll, "http server on port %d didn't start", port)
}

// waitForServerStart blocks until something accepts on every listed port, failing the test
// naming the port that never came up
func waitForServerStart(t *testing.T, ports ...int) {
	t.Helper()
	for _, port := range ports {
		require.True(t, waitForServerPort(port, serverStartTimeout), "server on port %d didn't start", port)
	}
}

// getRetryThrottled issues a GET and retries while the auth routes answer 429, since the /auth/
// group is limited to 2 req/s and this test logs in more often than that. a transport error is
// retried a couple of times and then reported as itself, so a dead server is not read as throttling
func getRetryThrottled(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()
	const transportRetries = 2
	errCount := 0
	for deadline := time.Now().Add(serverStartTimeout); time.Now().Before(deadline); time.Sleep(authRetryPoll) {
		r, err := client.Get(url)
		if err != nil {
			errCount++
			require.LessOrEqual(t, errCount, transportRetries, "request to %s failed: %v", url, err)
			continue
		}
		if r.StatusCode == http.StatusTooManyRequests {
			_ = r.Body.Close()
			continue
		}
		return r
	}
	t.Fatalf("request to %s kept being rate limited", url)
	return nil
}

// waitForServerPort blocks until something accepts on port, reporting whether it came up.
// unlike the require-based helpers it is safe to call off the test goroutine.
func waitForServerPort(port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), probeDialTimeout)
		if err == nil {
			_ = conn.Close()
			return true
		}
		time.Sleep(serverStartPoll)
	}
	return false
}

func prepServerApp(t *testing.T, fn func(o ServerCommand) ServerCommand) (*serverApp, context.Context, context.CancelFunc) {
	cmd := ServerCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "secret"})

	// prepare options
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--admin-passwd=password", "--site=remark"})
	require.NoError(t, err)
	cmd.Avatar.FS.Path, cmd.Avatar.Type, cmd.BackupLocation, cmd.Image.FS.Path = "/tmp/remark42_test", "fs", "/tmp/remark42_test", "/tmp/remark42_test"
	cmd.Store.Bolt.Timeout = 10 * time.Second
	cmd.Auth.Apple.CID, cmd.Auth.Apple.KID, cmd.Auth.Apple.TID = "cid", "kid", "tid"
	cmd.Auth.Apple.PrivateKeyFilePath = "testdata/apple.p8"
	cmd.Auth.Github.CSEC, cmd.Auth.Github.CID = "csec", "cid"
	cmd.Auth.Google.CSEC, cmd.Auth.Google.CID = "csec", "cid"
	cmd.Auth.Facebook.CSEC, cmd.Auth.Facebook.CID = "csec", "cid"
	cmd.Auth.Yandex.CSEC, cmd.Auth.Yandex.CID = "csec", "cid"
	cmd.Auth.Microsoft.CSEC, cmd.Auth.Microsoft.CID = "csec", "cid"
	cmd.Auth.Twitter.CSEC, cmd.Auth.Twitter.CID = "csec", "cid"
	cmd.Auth.Patreon.CSEC, cmd.Auth.Patreon.CID = "csec", "cid"
	cmd.Auth.Discord.CSEC, cmd.Auth.Discord.CID = "csec", "cid"
	cmd.Auth.Telegram = true
	cmd.Telegram.Token = "token"
	cmd.Auth.Email.Enable = true
	cmd.Auth.Email.MsgTemplate = "testdata/email.tmpl"
	cmd.BackupLocation = "/tmp"
	cmd.Notify.Users = []string{"email"}
	cmd.Notify.Admins = []string{"email"}
	cmd.Notify.Email.From = "from@example.org"
	cmd.Notify.Email.VerificationSubject = "test verification email subject"
	cmd.SMTP.Host = "127.0.0.1"
	cmd.SMTP.Port = 25
	cmd.SMTP.Username = "test_user"
	cmd.SMTP.Password = "test_password"
	cmd.SMTP.TimeOut = time.Second
	cmd.UpdateLimit = 10
	cmd.Admin.Type = "shared"
	cmd.Admin.Shared.Admins = []string{"id1", "id2"}
	cmd.RestrictedNames = []string{"umputun", "bobuk"}
	cmd.emailMsgTemplatePath = "../../templates/email_reply.html.tmpl"
	cmd.emailVerificationTemplatePath = "../../templates/email_confirmation_subscription.html.tmpl"

	cmd = fn(cmd)
	// as is uses port, call it after fn which could set it
	cmd.Store.Bolt.Path = fmt.Sprintf("/tmp/%d", cmd.Port)

	app, ctx, cancel := createAppFromCmd(t, cmd)

	// cleanup the remark.db file after context is canceled
	go func() {
		<-ctx.Done()
		os.RemoveAll(cmd.Store.Bolt.Path)
		os.RemoveAll(cmd.Avatar.FS.Path)

	}()

	return app, ctx, cancel
}

func createAppFromCmd(t *testing.T, cmd ServerCommand) (*serverApp, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	// a require in a readiness wait exits the test goroutine, so without this an app started in
	// a goroutine would never be stopped and goleak would report it instead of the failure
	t.Cleanup(cancel)
	app, err := cmd.newServerApp(ctx)
	require.NoError(t, err)
	return app, ctx, cancel
}

func TestMain(m *testing.M) {
	// ignore is added only for GitHub Actions, can't reproduce locally
	goleak.VerifyTestMain(
		m,
		// the shutdown goroutine in serverApp.run is not joined by Wait, and Rest.Shutdown gives
		// httpServer.Shutdown a second, which can outlast goleak's retry budget on a loaded runner
		goleak.IgnoreTopFunction("net/http.(*Server).Shutdown"),
		// this will be fixed in https://github.com/hashicorp/golang-lru/issues/159
		goleak.IgnoreTopFunction("github.com/hashicorp/golang-lru/v2/expirable.NewLRU[...].func1"),
		// regexp2, pulled in by chroma for syntax highlighting, keeps one shared clock goroutine
		// alive for up to a second after the last match with a timeout, sleeping in 100ms ticks.
		// it ends on its own, but a binary that finishes inside that window is reported as leaking
		goleak.IgnoreAnyFunction("github.com/dlclark/regexp2/v2.runClock"),
	)
}
