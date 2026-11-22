package cmd

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha1" //nolint:gosec // reproduces the unsalted anonymous ID the server must no longer issue
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html"
	"image/png"
	"io"
	"mime/quotedprintable"
	"net"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-pkgz/auth/v2"
	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/provider"
	"github.com/go-pkgz/auth/v2/provider/sender"
	"github.com/go-pkgz/auth/v2/token"
	log "github.com/go-pkgz/lgr"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jessevdk/go-flags"
	"go.uber.org/goleak"

	cache "github.com/go-pkgz/lcw/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/notify"
	"github.com/umputun/remark42/backend/app/store"
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
	app := startServerApp(t, nil)
	port := app.Port

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
}

func TestServerApp_DevMode(t *testing.T) {
	app := startServerApp(t, func(o ServerCommand) ServerCommand {
		o.AdminPasswd = "password"
		o.Auth.Dev = true
		return o
	})
	port := app.Port

	providers := app.restSrv.Authenticator.Providers()
	require.Equal(t, 10+1, len(providers), "extra auth provider")
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
}

func TestServerApp_CustomOAuthProvider(t *testing.T) {
	app := startServerApp(t, func(o ServerCommand) ServerCommand {
		o.Auth.Custom.Name = "oidc"
		o.Auth.Custom.CID = "cid"
		o.Auth.Custom.CSEC = "csec"
		o.Auth.Custom.AuthURL = "https://example.com/oauth2/authorize"
		o.Auth.Custom.TokenURL = "https://example.com/oauth2/token"
		o.Auth.Custom.InfoURL = "https://example.com/oauth2/userinfo"
		return o
	})

	providers := app.restSrv.Authenticator.Providers()
	require.Equal(t, 10+1, len(providers), "extra auth provider")
	assert.Equal(t, "oidc", providers[len(providers)-2].Name(), "custom auth provider")
}

func TestServerApp_AnonMode(t *testing.T) {
	app := startServerApp(t, func(o ServerCommand) ServerCommand {
		o.Auth.Anonymous = true
		return o
	})
	port := app.Port

	providers := app.restSrv.Authenticator.Providers()
	require.Equal(t, 10+1, len(providers), "extra auth provider for anon")
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
}

// TestServerCommand_AnonymousUserID signs in through the anonymous provider and reads the user ID out
// of the issued token. The ID is shown with every comment, and so is the name, so the ID must not be
// computable from the name and the client address without the shared secret.
func TestServerCommand_AnonymousUserID(t *testing.T) {
	login := func(t *testing.T, secret, name, addr string) string {
		t.Helper()
		s := ServerCommand{}
		s.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: secret})
		_, err := flags.NewParser(&s, flags.Default).ParseArgs([]string{"--auth.anon"})
		require.NoError(t, err)
		authenticator := auth.NewService(auth.Opts{
			URL:          "https://demo.remark42.com",
			SecretReader: token.SecretFunc(func(string) (string, error) { return "jwt-secret", nil }),
		})
		_, err = s.addAuthProviders(authenticator)
		require.NoError(t, err)
		authHandler, _ := authenticator.Handlers()

		req := httptest.NewRequest(http.MethodGet, "/auth/anonymous/login?aud=remark&user="+url.QueryEscape(name), http.NoBody)
		req.RemoteAddr = addr // as realIPMiddleware leaves it: the bare client IP
		rr := httptest.NewRecorder()
		authHandler.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
		for _, c := range rr.Result().Cookies() {
			if c.Name == "JWT" {
				claims, err := authenticator.TokenService().Parse(c.Value)
				require.NoError(t, err)
				require.NotNil(t, claims.User)
				return claims.User.ID
			}
		}
		t.Fatalf("no JWT cookie after anonymous login of %q from %s", name, addr)
		return ""
	}

	id := login(t, "secret", "blah123", "203.0.113.5")
	assert.Regexp(t, `^anonymous_[0-9a-f]{40}$`, id)

	unsalted := "anonymous_" + fmt.Sprintf("%x", sha1.Sum([]byte("blah123203.0.113.5"))) //nolint:gosec // the old scheme
	assert.NotEqual(t, unsalted, id, "ID must not be the plain SHA-1 of name and address")
	assert.NotEqual(t, id, login(t, "another secret", "blah123", "203.0.113.5"),
		"ID must depend on the shared secret")

	assert.Equal(t, id, login(t, "secret", "blah123", "203.0.113.5"), "same name and address keep the ID")
	assert.NotEqual(t, id, login(t, "secret", "blah123", "203.0.113.6"), "another address gets another ID")
	assert.NotEqual(t, login(t, "secret", "abc1", "2.3.4.5"), login(t, "secret", "abc", "12.3.4.5"),
		"name and address must not run together")
}

// TestServerCommand_AddAuthProvidersCount checks the count that decides the "no auth providers defined"
// warning. Email and anonymous sign-in need no OAuth2 provider, so a setup with only them must not warn.
func TestServerCommand_AddAuthProvidersCount(t *testing.T) {
	tbl := []struct {
		name string
		args []string
		want int
	}{
		{name: "nothing enabled", args: nil, want: 0},
		{name: "email only", args: []string{"--auth.email.enable"}, want: 1},
		{name: "anonymous only", args: []string{"--auth.anon"}, want: 1},
		{name: "email and anonymous", args: []string{"--auth.email.enable", "--auth.anon"}, want: 2},
		{name: "oauth2 and anonymous", args: []string{"--auth.github.cid=cid", "--auth.github.csec=csec", "--auth.anon"}, want: 2},
		{name: "telegram only", args: []string{"--auth.telegram", "--telegram.token=token"}, want: 1},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			s := ServerCommand{}
			s.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "secret"})
			_, err := flags.NewParser(&s, flags.Default).ParseArgs(tt.args)
			require.NoError(t, err)
			authenticator := auth.NewService(auth.Opts{
				URL:          "https://demo.remark42.com",
				SecretReader: token.SecretFunc(func(string) (string, error) { return "jwt-secret", nil }),
			})
			_, err = s.makeTelegramAuth(authenticator) // registered before the others, as newServerApp does
			require.NoError(t, err)
			got, err := s.addAuthProviders(authenticator)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got, "providers counted for %v", tt.args)
		})
	}
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
	tmp := t.TempDir()
	opts := ServerCommand{}
	sslPort := chooseUnusedPort(t)
	port := chooseUnusedPort(t)
	parseCmd(t, &opts, fmt.Sprintf("https://localhost:%d", sslPort), "--admin-passwd=password", "--port="+strconv.Itoa(port),
		"--store.bolt.path="+tmp+"/xyz", "--backup="+tmp, "--avatar.type=bolt", "--avatar.bolt.file="+tmp+"/ava-test.db",
		"--ssl.type=static", "--ssl.cert=testdata/cert.pem", "--ssl.key=testdata/key.pem",
		"--ssl.port="+strconv.Itoa(sslPort), "--image.fs.path="+tmp)

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
	tmp := t.TempDir()
	opts := ServerCommand{}
	port := chooseUnusedPort(t)
	parseCmd(t, &opts, "https://demo.remark42.com", "--admin-passwd=password", "--cache.type=none",
		"--store.type=rpc", "--store.rpc.api=http://127.0.0.1",
		"--port="+strconv.Itoa(port), "--avatar.fs.path="+tmp,
		"--admin.type=rpc", "--admin.rpc.secret_per_site", "--admin.rpc.api=http://127.0.0.1")
	opts.Auth.Github.CSEC, opts.Auth.Github.CID = "csec", "cid"
	opts.BackupLocation, opts.Image.FS.Path = tmp, tmp

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
	tmp := t.TempDir()
	tbl := []struct {
		name, url string
		args      []string
		storeType string // set after parsing, for a value the parser refuses
		err       string
	}{
		{name: "RO bolt location", args: []string{"--backup=" + tmp, "--store.bolt.path=/dev/null", "--image.fs.path=" + tmp},
			err: "failed to make data store engine: failed to create bolt store: can't make directory /dev/null: mkdir /dev/null: not a directory"},
		{name: "RO backup location", args: []string{"--store.bolt.path=" + tmp, "--backup=/dev/null/not-writable"},
			err: "failed to create backup store: can't make directory /dev/null/not-writable: mkdir /dev/null: not a directory"},
		{name: "invalid url", url: "demo.remark42.com", args: []string{"--backup=" + tmp, "--store.bolt.path=" + tmp},
			err: "invalid remark42 url demo.remark42.com"},
		{name: "invalid trusted proxy fails before any resource is created", args: []string{"--backup=" + tmp, "--trusted-proxy=nonsense"},
			err: `invalid --trusted-proxy: invalid trusted proxy "nonsense"`},
		{name: "wrong store type", args: []string{"--backup=" + tmp}, storeType: "blah",
			err: "failed to make data store engine: unsupported store type blah"},
		{name: "wrong redis location", args: []string{"--store.bolt.path=" + tmp, "--backup=" + tmp, "--image.fs.path=" + tmp,
			"--cache.type=redis_pub_sub", "--cache.redis_addr=wrong_address"},
			err: "failed to make cache: cache backend initialization, redis PubSub initialisation: " +
				"problem subscribing to channel remark42-cache on address wrong_address: " +
				"dial tcp: address wrong_address: missing port in address"},
		{name: "wrong apple private key type", args: []string{"--store.bolt.path=" + tmp, "--backup=" + tmp, "--image.fs.path=" + tmp,
			"--avatar.fs.path=" + tmp, "--auth.apple.cid=123", "--auth.apple.tid=123", "--auth.apple.kid=123",
			"--auth.apple.private-key-filepath=testdata/apple-bad.p8"},
			err: "failed to make authenticator: an AppleProvider creating failed: provided private key is not ECDSA"},
		{name: "incomplete custom oauth config", args: []string{"--store.bolt.path=" + tmp, "--backup=" + tmp, "--image.fs.path=" + tmp,
			"--avatar.fs.path=" + tmp, "--auth.custom.name=oidc", "--auth.custom.cid=123"},
			err: "failed to make authenticator: custom oauth provider configuration is incomplete, missing: " +
				"AUTH_CUSTOM_CSEC, AUTH_CUSTOM_AUTH_URL, AUTH_CUSTOM_TOKEN_URL, AUTH_CUSTOM_INFO_URL"},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			opts := ServerCommand{}
			parseCmd(t, &opts, cmp.Or(tt.url, "https://demo.remark42.com"), tt.args...)
			opts.Store.Type = cmp.Or(tt.storeType, opts.Store.Type)
			_, err := opts.newServerApp(context.Background())
			assert.EqualError(t, err, tt.err)
		})
	}

	// unreadable restricted words file fails fast, before any resource is created
	t.Run("unreadable restricted words file", func(t *testing.T) {
		opts := ServerCommand{}
		parseCmd(t, &opts, "https://demo.remark42.com", "--backup="+tmp, "--restricted-words-file="+filepath.Join(tmp, "missing.txt"))
		_, err := opts.newServerApp(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid --restricted-words-file: failed to stat")
	})

	// the parser itself refuses a store type outside the choices
	opts := ServerCommand{}
	_, err := flags.NewParser(&opts, flags.Default).ParseArgs([]string{"--store.type=blah"})
	assert.Error(t, err, "blah is invalid type")
}

func TestIsReservedCustomProviderName(t *testing.T) {
	reserved := []string{
		"email", "anonymous", "google", "github", "facebook", "yandex", "twitter",
		"microsoft", "patreon", "discord", "telegram", "dev", "apple",
	}

	for _, name := range reserved {
		t.Run(name, func(t *testing.T) {
			assert.True(t, slices.Contains(reservedCustomProviderNames, name))
		})
	}

	assert.False(t, slices.Contains(reservedCustomProviderNames, "oidc"))
}

func TestIsValidCustomProviderName(t *testing.T) {
	valid := []string{"oidc", "codeberg", "provider_1", "provider-1", "a1"}
	for _, name := range valid {
		t.Run("valid_"+name, func(t *testing.T) {
			assert.True(t, validCustomProviderName.MatchString(name))
		})
	}

	invalid := []string{"", " has-space", "has space", "Uppercase", "provider!", "-provider", "_provider"}
	for _, name := range invalid {
		t.Run("invalid_"+strings.ReplaceAll(name, " ", "_"), func(t *testing.T) {
			assert.False(t, validCustomProviderName.MatchString(name))
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
	tmp := t.TempDir()
	baseArgs := []string{
		"--store.bolt.path=" + tmp,
		"--backup=" + tmp,
		"--image.fs.path=" + tmp,
		"--avatar.fs.path=" + tmp,
		"--auth.custom.cid=123",
		"--auth.custom.csec=456",
		"--auth.custom.auth-url=https://example.com/oauth2/authorize",
		"--auth.custom.token-url=https://example.com/oauth2/token",
		"--auth.custom.info-url=https://example.com/oauth2/userinfo",
	}

	t.Run("reserved", func(t *testing.T) {
		opts := ServerCommand{}
		parseCmd(t, &opts, "https://demo.remark42.com", append(baseArgs, "--auth.custom.name=twitter")...)
		_, err := opts.newServerApp(context.Background())
		assert.EqualError(t, err, `failed to make authenticator: custom oauth provider name "twitter" is reserved`)
	})

	t.Run("not_url_safe", func(t *testing.T) {
		opts := ServerCommand{}
		parseCmd(t, &opts, "https://demo.remark42.com", append(baseArgs, "--auth.custom.name=bad name")...)
		_, err := opts.newServerApp(context.Background())
		assert.EqualError(t, err, `failed to make authenticator: custom oauth provider name "bad name" is invalid, expected pattern "^[a-z0-9][a-z0-9_-]*$"`)
	})
}

func TestServerApp_Shutdown(t *testing.T) {
	app, ctx, cancel := prepServerApp(t, nil)

	// cancel once the server actually answers, so the test measures shutdown and not startup.
	// the deferred cancel also covers a failed wait, keeping app.run from racing the next test
	errCh := make(chan error, 1)
	go func() { errCh <- app.run(ctx) }()
	defer cancel()
	waitForHTTPServerStart(t, app.Port)
	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(serverStopTimeout):
		t.Fatal("server app did not stop after context cancel")
	}
	app.Wait()
}

func TestServerApp_ActivateBackupStopWaitsForExport(t *testing.T) {
	loc := t.TempDir()
	exp := &endlessExporter{started: make(chan struct{})}
	app := &serverApp{ServerCommand: &ServerCommand{Sites: []string{"site1"}, BackupLocation: loc, MaxBackupFiles: 3},
		exporter: exp}
	stop := app.activateBackup(t.Context())
	defer stop()

	select {
	case <-exp.started: // the backup at start is running
	case <-time.After(serverStartTimeout):
		t.Fatal("the backup at start did not begin")
	}
	stop()
	assert.True(t, exp.stopped.Load(), "stop returns only after the export in progress has ended")
	ff, err := os.ReadDir(loc)
	require.NoError(t, err)
	assert.Empty(t, ff, "the stopped export leaves no file behind")
}

func TestWaitGroupTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var wg sync.WaitGroup
		release := make(chan struct{})
		wg.Go(func() { <-release })

		start := time.Now()
		assert.False(t, waitGroupTimeout(&wg, backupStopWait), "gives up on a group that does not finish")
		assert.Equal(t, backupStopWait, time.Since(start), "after the whole wait")

		close(release)
		assert.True(t, waitGroupTimeout(&wg, backupStopWait), "a finished group is reported at once")
	})
}

// endlessExporter writes until a write fails, the way an export of a large site outlasts a shutdown
type endlessExporter struct {
	started chan struct{}
	stopped atomic.Bool
}

func (e *endlessExporter) Export(w io.Writer, _ string) (int, error) {
	close(e.started)
	for {
		if _, err := w.Write([]byte("comment\n")); err != nil {
			e.stopped.Store(true)
			return 0, err
		}
		time.Sleep(time.Millisecond) // keeps the gzip stream small while the test stops the backup
	}
}

// TestServerApp_ClaimsUpd covers the hook the authenticator runs on every token mint, refresh
// included: it stamps admin, blocked and email onto the claims and blocks impersonation of a
// restricted name. Calling the updater directly keeps it independent of when a token expires.
func TestServerApp_ClaimsUpd(t *testing.T) {
	// the app owns stores and services that only run closes, so it goes through the usual
	// lifecycle here rather than being built and abandoned
	app := startServerApp(t, nil)

	upd := app.restSrv.Authenticator.TokenService().ClaimsUpd
	require.NotNil(t, upd, "claims updater wired into the token service")

	claimsFor := func(id, name string) token.Claims {
		return token.Claims{
			Audience: jwt.ClaimStrings{"remark"},
			User:     &token.User{ID: id, Name: name},
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

	// regression: the block was logged with the whole user, notification email included. The line is
	// checked through impersonationLogLine and not by capturing the log: the app's backup goroutine
	// logs on its own schedule, so replacing the global logger while the app runs is a data race.
	t.Run("impersonation is logged without the email", func(t *testing.T) {
		_, err := app.restSrv.DataService.SetUserEmail("remark", "email_y", "secret.person@example.com")
		require.NoError(t, err)
		res := upd.Update(claimsFor("email_y", "bobuk"))
		require.True(t, res.User.BoolAttr("blocked"))
		require.Equal(t, "secret.person@example.com", res.User.Email, "the blocked user carries the email")
		assert.Equal(t, "[INFO] blocked email_y (bobuk), attempt to impersonate (restricted names)",
			impersonationLogLine(res.User))
	})

	t.Run("regular user may carry a restricted name", func(t *testing.T) {
		res := upd.Update(claimsFor("provider1_someone", "umputun"))
		assert.False(t, res.User.BoolAttr("blocked"), "only anonymous and email logins are checked")
	})

	t.Run("claims without a user pass through", func(t *testing.T) {
		res := upd.Update(token.Claims{Audience: jwt.ClaimStrings{"remark"}})
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
	port := chooseUnusedPort(t)
	tmp := t.TempDir()
	args := []string{"test", "--store.bolt.path=" + tmp + "/xyz", "--backup=" + tmp, "--avatar.type=bolt",
		"--avatar.bolt.file=" + tmp + "/ava-test.db", "--port=" + strconv.Itoa(port), "--image.fs.path=" + tmp}
	parseCmd(t, &s, "https://demo.remark42.com", args...)
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

	err := s.Execute(args)
	assert.NoError(t, err, "execute should be without errors")
	require.NoError(t, <-sigErr, "SIGTERM not delivered")
}

func TestServerApp_RunCanceledBeforeRESTStart(t *testing.T) {
	app, ctx, cancel := prepServerApp(t, nil)
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
		waitForHTTPServerStart(t, app.Port)
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
		"--notify.telegram.api=http://example.org/bot",
		"--auth.twitter.cid=123",
		"--auth.twitter.csec=456",
	}
	assert.Empty(t, s.SMTP.Host)
	assert.Empty(t, s.SMTP.Port)
	assert.Empty(t, s.SMTP.TLS)
	assert.Empty(t, s.SMTP.Username)
	assert.Empty(t, s.SMTP.Password)
	assert.Empty(t, s.SMTP.TimeOut)
	parseCmd(t, &s, "https://demo.remark42.com", args...)
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
			{Old: "notify.telegram.api", New: "telegram.api-url", Version: "1.9"},
		},
		deprecatedFlags)
	assert.Equal(t, "smtp.example.org", s.SMTP.Host)
	assert.Equal(t, 666, s.SMTP.Port)
	assert.Equal(t, true, s.SMTP.TLS)
	assert.Equal(t, "test_user", s.SMTP.Username)
	assert.Equal(t, "test_password", s.SMTP.Password)
	assert.Equal(t, 15*time.Second, s.SMTP.TimeOut)
	assert.Equal(t, "http://example.org", s.Telegram.APIURL)
}

// findDeprecatedFlagsCollisions returns the entries of HandleDeprecatedFlags reported as collisions
func (s *ServerCommand) findDeprecatedFlagsCollisions() []DeprecatedFlag {
	var res []DeprecatedFlag
	for _, f := range s.HandleDeprecatedFlags() {
		if f.Collision {
			res = append(res, f)
		}
	}
	return res
}

func TestServerApp_DeprecatedArgsCollisions(t *testing.T) {
	s := ServerCommand{}
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
		"--notify.telegram.api=http://old.example.org/bot",
		"--telegram.api-url=http://new.example.org",
	}
	parseCmd(t, &s, "https://demo.remark42.com", args...)
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
			{Old: "notify.telegram.api", New: "telegram.api-url", Collision: true},
		},
		deprecatedFlagsCollisions)

	// an old flag next to an empty new one is no collision; without a shell, "--smtp.host=''"
	// would set the host to the two quote characters
	s = ServerCommand{}
	parseCmd(t, &s, "https://demo.remark42.com", "test", "--auth.email.host=smtp-old.example.org", "--smtp.host=")
	deprecatedFlagsCollisions = s.findDeprecatedFlagsCollisions()
	assert.Empty(t, deprecatedFlagsCollisions)
}

func Test_ACMEEmail(t *testing.T) {
	tbl := []struct {
		name, url string
		args      []string
		want      string
	}{
		{name: "from remark url", url: "https://remark.com:443", want: "admin@remark.com"},
		{name: "acme email set", url: "https://remark.com", args: []string{"--ssl.acme-email=adminname@adminhost.com"},
			want: "adminname@adminhost.com"},
		{name: "shared admin email", url: "https://remark.com", args: []string{"--admin.type=shared", "--admin.shared.email=superadmin@admin.com"},
			want: "superadmin@admin.com"},
		{name: "shared admin without email", url: "https://remark.com:443", args: []string{"--admin.type=shared"},
			want: "admin@remark.com"},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ServerCommand{}
			parseCmd(t, &cmd, tt.url, append([]string{"--ssl.type=auto"}, tt.args...)...)
			cfg, err := cmd.makeSSLConfig()
			require.NoError(t, err)
			assert.Equal(t, tt.want, cfg.ACMEEmail)
		})
	}
}

func TestServerAuthHooks(t *testing.T) {
	app := startServerApp(t, nil)

	// make a token for user dev. nothing here checks expiry, so the lifetime only has to
	// outlast the whole test
	tkService := app.restSrv.Authenticator.TokenService()
	tkService.TokenDuration = time.Hour

	claims := token.Claims{
		Audience:  jwt.ClaimStrings{"remark"},
		Issuer:    "remark",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
		User: &token.User{
			ID:   "github_dev",
			Name: "developer one",
		},
	}
	tk, err := tkService.Token(claims)
	require.NoError(t, err)

	client := http.Client{Timeout: 10 * time.Second}
	defer client.CloseIdleConnections()

	// do sends one request with the token, or with the admin's basic auth for tkn "admin", and
	// returns the status and the body
	do := func(method, path, body, tkn string) (int, string) {
		t.Helper()
		req, e := http.NewRequest(method, fmt.Sprintf("http://localhost:%d%s", app.Port, path), strings.NewReader(body))
		require.NoError(t, e)
		if tkn == "admin" {
			req.SetBasicAuth("admin", "password")
		} else {
			req.Header.Set("X-JWT", tkn)
		}
		resp, e := client.Do(req)
		require.NoError(t, e)
		b, e := io.ReadAll(resp.Body)
		require.NoError(t, e)
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode, string(b)
	}
	comment := func(postURL string) string {
		return `{"text": "test 123", "locator":{"url": "` + postURL + `", "site": "remark"}}`
	}
	const addComment = "/api/v1/comment?site=remark"

	status, body := do(http.MethodPost, addComment, comment("https://radio-t.com/p/2018/12/29/podcast-630/"), tk)
	assert.Equal(t, http.StatusCreated, status, "non-blocked user able to post, %s", body)

	// try to add comment with no-aud claim
	// the token service refuses to issue it, so sign it the way a holder of the secret could
	badClaimsNoAud := claims
	badClaimsNoAud.Audience = jwt.ClaimStrings{""}
	_, err = tkService.Token(badClaimsNoAud)
	require.Error(t, err, "token service must not issue a token with an empty audience")
	status, body = do(http.MethodPost, addComment, comment("https://radio-t.com/p/2018/12/29/podcast-631/"), signRawToken(t, badClaimsNoAud))
	assert.Equal(t, http.StatusUnauthorized, status, "user without aud claim rejected, %s", body)

	// a token for a site this instance does not serve is neither issued nor accepted
	badClaimsOtherSite := claims
	badClaimsOtherSite.Audience = jwt.ClaimStrings{"<b>other</b>"}
	_, err = tkService.Token(badClaimsOtherSite)
	require.Error(t, err, "token service must not issue a token for a site it does not serve")
	status, body = do(http.MethodGet, "/api/v1/user?site="+url.QueryEscape("<b>other</b>"), "", signRawToken(t, badClaimsOtherSite))
	assert.Equal(t, http.StatusUnauthorized, status, "token for a site not served rejected, %s", body)

	// the auth package matches audiences ignoring case, so it issues this token, but the bolt store
	// keys sites by the exact name and the session must not be accepted
	tkOtherCase, err := tkService.Token(func() token.Claims { c := claims; c.Audience = jwt.ClaimStrings{"REMARK"}; return c }())
	require.NoError(t, err)
	status, body = do(http.MethodGet, "/api/v1/user?site=REMARK", "", tkOtherCase)
	assert.Equal(t, http.StatusUnauthorized, status, "token for a differently cased site rejected, %s", body)

	// try to add comment with multiple auds
	badClaimsMultipleAud := claims
	badClaimsMultipleAud.Audience = jwt.ClaimStrings{"remark", "second_aud"}
	tkMultipleAuds, err := tkService.Token(badClaimsMultipleAud)
	require.NoError(t, err)
	status, body = do(http.MethodPost, addComment, comment("https://radio-t.com/p/2018/12/29/podcast-631/"), tkMultipleAuds)
	assert.Equal(t, http.StatusUnauthorized, status, "user with multiple auds claim rejected, %s", body)

	// try to add comment without user set
	badClaimsNoUser := claims
	badClaimsNoUser.User = nil
	tkNoUser, err := tkService.Token(badClaimsNoUser)
	require.NoError(t, err)
	status, body = do(http.MethodPost, addComment, comment("https://radio-t.com/p/2018/12/29/podcast-631/"), tkNoUser)
	assert.Equal(t, http.StatusUnauthorized, status, "user without user information rejected, %s", body)

	// a delete-me token, as the endpoint issues it, is a deletion request for the admin and not a session
	status, body = do(http.MethodPost, "/api/v1/deleteme?site=remark", "", tk)
	require.Equal(t, http.StatusOK, status, body)
	deleteMe := struct {
		Token string `json:"token"`
	}{}
	require.NoError(t, json.Unmarshal([]byte(body), &deleteMe))
	require.NotEmpty(t, deleteMe.Token)
	for _, probe := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/user?site=remark", ""},
		{http.MethodPost, addComment, comment("https://radio-t.com/p/2018/12/29/podcast-632/")},
	} {
		status, body = do(probe.method, probe.path, probe.body, deleteMe.Token)
		assert.Equal(t, http.StatusUnauthorized, status, "delete-me token rejected on %s %s, %s", probe.method, probe.path, body)
	}

	// block user github_dev as admin
	status, body = do(http.MethodPut, "/api/v1/admin/user/github_dev?site=remark&block=1&ttl=240h", "", "admin")
	assert.Equal(t, http.StatusOK, status, "user github_dev blocked, %s", body)

	// try add a comment with blocked user
	status, body = do(http.MethodPost, addComment, comment("https://radio-t.com/blah1"), tk)
	assert.Equal(t, http.StatusForbidden, status, "blocked user can't post, %s", body)
}

// a token in the query is not read at all, so a URL that carries one, requested by a reader's
// browser with that reader's cookies, can neither sign the reader out nor stand in for a session
func TestServerApp_QueryTokenIgnored(t *testing.T) {
	app := startServerApp(t, nil)

	tkService := app.restSrv.Authenticator.TokenService()
	tkService.TokenDuration = time.Hour
	sessionToken := func(id, tokenID string) string {
		tk, err := tkService.Token(token.Claims{
			ID:        tokenID,
			Audience:  jwt.ClaimStrings{"remark"},
			Issuer:    "remark",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
			User:      &token.User{ID: id, Name: id},
		})
		require.NoError(t, err)
		return tk
	}
	reader, commenter := sessionToken("github_reader", "reader-token-id"), sessionToken("github_commenter", "commenter-token-id")

	client := http.Client{Timeout: 10 * time.Second}
	defer client.CloseIdleConnections()
	// do sends a request with the given headers and returns the status, the Set-Cookie lines and the body
	do := func(method, path string, headers map[string]string) (int, []string, string) {
		t.Helper()
		req, err := http.NewRequest(method, fmt.Sprintf("http://localhost:%d%s", app.Port, path), http.NoBody)
		require.NoError(t, err)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := client.Do(req)
		require.NoError(t, err)
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode, resp.Header.Values("Set-Cookie"), string(b)
	}

	// the commenter's delete-me token is signed like a session and rejected by the validator
	status, _, body := do(http.MethodPost, "/api/v1/deleteme?site=remark", map[string]string{"X-JWT": commenter})
	require.Equal(t, http.StatusOK, status, body)
	deleteMe := struct {
		Token string `json:"token"`
	}{}
	require.NoError(t, json.Unmarshal([]byte(body), &deleteMe))
	require.NotEmpty(t, deleteMe.Token)

	// the reader's browser, asked for a URL that somebody else wrote
	asReader := map[string]string{"Cookie": "JWT=" + reader + "; XSRF-TOKEN=reader-token-id", "X-XSRF-TOKEN": "reader-token-id"}

	status, cookies, body := do(http.MethodGet, "/api/v1/user?site=remark", asReader)
	require.Equal(t, http.StatusOK, status, body)
	require.Contains(t, body, "github_reader")
	require.Empty(t, cookies)

	for _, path := range []string{"/api/v1/config?site=remark", "/api/v1/user?site=remark"} {
		status, cookies, body = do(http.MethodGet, path+"&jwt="+deleteMe.Token, asReader)
		assert.Equal(t, http.StatusOK, status, "%s: %s", path, body)
		assert.Empty(t, cookies, "%s: a rejected token in the query must not expire the reader's cookies", path)
	}

	status, _, body = do(http.MethodGet, "/api/v1/user?site=remark&jwt="+commenter, asReader)
	assert.Equal(t, http.StatusOK, status, body)
	assert.Contains(t, body, "github_reader", "the reader stays the reader, whatever token the URL carries")

	status, _, body = do(http.MethodGet, "/api/v1/user?site=remark&jwt="+commenter, nil)
	assert.Equal(t, http.StatusUnauthorized, status, "a token in the query is not a session, %s", body)

	// the sign-out of an OAuth provider wants the session it ends, and a token in the URL is not one
	status, cookies, body = do(http.MethodGet, "/auth/logout?jwt="+commenter, map[string]string{"Cookie": asReader["Cookie"]})
	assert.Equal(t, http.StatusForbidden, status, body)
	assert.Empty(t, cookies, "a sign-out request without the reader's XSRF header must not end the session")
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

func TestServerCommand_TelegramAuthAcceptsLoginBeforeFirstSharedPoll(t *testing.T) {
	telegramAPI := newTelegramAPITestServer(t)

	client := &http.Client{Timeout: time.Second}
	defer client.CloseIdleConnections()
	authAPI, err := provider.NewTelegramAPIWithBaseURL("stub-token", client, telegramAPI.URL)
	require.NoError(t, err)
	authHandler := &provider.TelegramHandler{ProviderName: "telegram", Telegram: authAPI, L: log.Default()}

	cmd := ServerCommand{}
	cmd.Telegram.Token = "stub-token"
	cmd.Telegram.Timeout = time.Second
	cmd.Telegram.APIURL = telegramAPI.URL

	ctx := t.Context()
	tg, err := cmd.startTelegramAuthAndNotify(ctx, authHandler)
	require.NoError(t, err)
	require.NotNil(t, tg, "telegram auth needs the shared update source even when notifications are off")

	recorder := httptest.NewRecorder()
	authHandler.LoginHandler(recorder, httptest.NewRequest(http.MethodGet, "/auth/telegram/login?site=remark", http.NoBody))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	var response struct {
		Token string `json:"token"`
		Bot   string `json:"bot"`
	}
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, "remark42_test_bot", response.Bot)
}

func TestServerApp_TelegramAuthAloneDoesNotOfferNotifications(t *testing.T) {
	telegramAPI := newTelegramAPITestServer(t)
	app := startServerApp(t, func(cmd ServerCommand) ServerCommand {
		cmd.Auth.Telegram = true
		cmd.Telegram.Token = "stub-token"
		cmd.Telegram.Timeout = time.Second
		cmd.Telegram.APIURL = telegramAPI.URL
		return cmd
	})

	require.Nil(t, app.restSrv.TelegramService,
		"telegram auth alone must not put a typed nil or an update-only client behind the subscription endpoint")
}

func TestServerCommand_TelegramCustomAPIErrorStopsRequestedFeature(t *testing.T) {
	t.Run("auth", func(t *testing.T) {
		cmd := ServerCommand{}
		cmd.Auth.Telegram = true
		cmd.Telegram.APIURL = "://bad"

		_, err := cmd.makeTelegramAuth(&auth.Service{})
		require.ErrorContains(t, err, "failed to make telegram auth")
	})

	t.Run("notifications", func(t *testing.T) {
		cmd := ServerCommand{}
		cmd.Notify.Users = []string{"telegram"}
		cmd.Telegram.APIURL = "://bad"

		_, err := cmd.startTelegramAuthAndNotify(context.Background(), nil)
		require.ErrorContains(t, err, "failed to make telegram update source")
	})
}

func newTelegramAPITestServer(t *testing.T) *httptest.Server {
	t.Helper()
	telegramAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/getUpdates") {
			_, _ = io.WriteString(w, `{"ok":true,"result":[]}`)
			return
		}
		_, _ = io.WriteString(w, `{"ok":true,"result":{"id":1,"is_bot":true,"username":"remark42_test_bot"}}`)
	}))
	t.Cleanup(telegramAPI.Close)
	return telegramAPI
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
		{"   ", []string{}},
		{"vv1, ,vv2", []string{"vv1", "", "vv2"}},
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
		{ServerCommand{AllowedHosts: []string{}, RemarkURL: "https://remark42.example.org"}, []string{"example.org"}},
		{ServerCommand{AllowedHosts: []string{}, RemarkURL: "http://remark42.example.org"}, []string{"example.org"}},
		{ServerCommand{AllowedHosts: []string{}, RemarkURL: "http://localhost"}, []string{"localhost"}},
		// incorrect URLs, so Hostname is empty but returned list doesn't include empty string as it would allow any domain
		{ServerCommand{AllowedHosts: []string{}, RemarkURL: "bad hostname"}, []string{}},
		{ServerCommand{AllowedHosts: []string{}, RemarkURL: "not_a_hostname"}, []string{}},
		// test removal of 'self', multiple AllowedHosts. No deduplication is expected
		{ServerCommand{AllowedHosts: []string{"'self'", "example.org", "test.example.org", "remark42.com"}, RemarkURL: "https://example.org"}, []string{"example.org", "test.example.org", "remark42.com", "example.org"}},
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
	// probe on the loopback address the tests dial: on macOS a wildcard bind succeeds on a port
	// another process holds on 127.0.0.1 only, and requests to that address then reach the other
	// process instead of the server under test
	ln, err := net.Listen("tcp", "127.0.0.1:0")
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
func getRetryThrottled(t *testing.T, client *http.Client, reqURL string) *http.Response {
	t.Helper()
	const transportRetries = 2
	errCount := 0
	for deadline := time.Now().Add(serverStartTimeout); time.Now().Before(deadline); time.Sleep(authRetryPoll) {
		r, err := client.Get(reqURL)
		if err != nil {
			errCount++
			require.LessOrEqual(t, errCount, transportRetries, "request to %s failed: %v", reqURL, err)
			continue
		}
		if r.StatusCode == http.StatusTooManyRequests {
			_ = r.Body.Close()
			continue
		}
		return r
	}
	t.Fatalf("request to %s kept being rate limited", reqURL)
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

// signRawToken signs claims with the shared secret prepServerApp configures, bypassing the token
// service and the checks it makes before issuing
func signRawToken(t *testing.T, claims token.Claims) string {
	t.Helper()
	tkn, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSharedSecret))
	require.NoError(t, err)
	return tkn
}

// TestServerApp_LoginRejectsUnknownSite covers the audience a user names at login: it ends up in
// the session token and is echoed back in responses, so a site the instance does not serve gets
// no session at all.
func TestServerApp_LoginRejectsUnknownSite(t *testing.T) {
	app := startServerApp(t, func(o ServerCommand) ServerCommand {
		o.Auth.Anonymous = true
		return o
	})
	port := app.Port

	client := http.Client{Timeout: 10 * time.Second}
	defer client.CloseIdleConnections()

	resp := getRetryThrottled(t, &client, fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=blah123&aud=%s",
		port, url.QueryEscape("<b>other</b>")))
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.NotEqual(t, http.StatusOK, resp.StatusCode, "login to a site not served must fail, %s", body)
	for _, c := range resp.Cookies() {
		assert.NotEqual(t, "JWT", c.Name, "no session cookie for a site not served")
	}

	// the configured site still signs in
	resp = getRetryThrottled(t, &client, fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=blah123&aud=remark", port))
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// startServerApp builds the app with prepServerApp, runs it and waits until it answers on app.Port.
// the app is stopped, and waited for, when the test ends
func startServerApp(t *testing.T, fn func(o ServerCommand) ServerCommand) *serverApp {
	t.Helper()
	app, ctx, cancel := prepServerApp(t, fn)
	go func() { _ = app.run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-app.terminated:
		case <-time.After(serverStopTimeout):
			t.Errorf("server app on port %d did not stop after context cancel", app.Port)
		}
	})
	waitForHTTPServerStart(t, app.Port)
	return app
}

// prepServerApp builds the app on a free port with the options the server tests share; fn, if
// not nil, changes them
func prepServerApp(t *testing.T, fn func(o ServerCommand) ServerCommand) (*serverApp, context.Context, context.CancelFunc) {
	cmd := ServerCommand{}
	parseCmd(t, &cmd, "https://demo.remark42.com", "--admin-passwd=password", "--site=remark")
	cmd.Port = chooseUnusedPort(t)
	tmp := t.TempDir()
	cmd.Avatar.FS.Path, cmd.Avatar.Type, cmd.Image.FS.Path = filepath.Join(tmp, "avatars"), "fs", filepath.Join(tmp, "images")
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
	cmd.Auth.Email.Enable = true
	cmd.Auth.Email.MsgTemplate = "testdata/email.tmpl"
	cmd.BackupLocation = tmp
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

	if fn != nil {
		cmd = fn(cmd)
	}
	// the test's own directory, also when fn set another bolt path
	cmd.Store.Bolt.Path = filepath.Join(tmp, "db")

	return createAppFromCmd(t, cmd)
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

func TestServerApp_ClearSecretEnv(t *testing.T) {
	s := ServerCommand{}
	s.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})
	p := flags.NewParser(&s, flags.Default)

	// every option whose variable holds a credential, found from the flag definitions, so a new one
	// is checked without being added here. the webhook headers have no env tag, they are read in code
	isSecret := func(name string) bool {
		for _, suffix := range []string{"_CSEC", "_KID", "_TOKEN", "_PASSWD", "_PASSWORD"} {
			if strings.HasSuffix(name, suffix) {
				return true
			}
		}
		return name == "SECRET"
	}
	secrets := []string{"SECRET", "NOTIFY_WEBHOOK_HEADERS"}
	var walk func(g *flags.Group)
	walk = func(g *flags.Group) {
		for _, opt := range g.Options() {
			if name := opt.EnvKeyWithNamespace(); isSecret(name) {
				secrets = append(secrets, name)
			}
		}
		for _, sub := range g.Groups() {
			walk(sub)
		}
	}
	walk(p.Group)
	require.Contains(t, secrets, "STORE_RPC_AUTH_PASSWD", "the walk reaches nested groups")

	for _, name := range secrets {
		t.Setenv(name, "x")
	}
	t.Setenv("NOTIFY_WEBHOOK_HEADERS", `Authorization:Bearer abc,X-List:"a,b"`)
	_, err := p.ParseArgs([]string{"--notify.admins=webhook"})
	require.NoError(t, err)

	s.clearSecretEnv()
	for _, name := range secrets {
		_, ok := os.LookupEnv(name)
		assert.False(t, ok, "%s is cleared from the environment", name)
	}
	assert.Equal(t, []string{"Authorization:Bearer abc", `X-List:"a,b"`}, s.Notify.Webhook.Headers,
		"the webhook headers are kept in the options")

	// the headers are parsed whether or not the webhook is enabled, so a blank value must not stop the start
	blank := ServerCommand{}
	t.Setenv("NOTIFY_WEBHOOK_HEADERS", " ")
	require.NotPanics(t, blank.clearSecretEnv)
	assert.Empty(t, blank.Notify.Webhook.Headers)
}

func TestServerApp_MakeCacheKeepsLargeValuesByDefault(t *testing.T) {
	// the test is about the defaults, so the caller's environment must not supply cache options
	for _, name := range []string{"CACHE_TYPE", "CACHE_MAX_ITEMS", "CACHE_MAX_VALUE", "CACHE_MAX_SIZE"} {
		if val, ok := os.LookupEnv(name); ok {
			require.NoError(t, os.Unsetenv(name))
			t.Cleanup(func() { _ = os.Setenv(name, val) })
		}
	}

	opts := ServerCommand{}
	parseCmd(t, &opts, "https://demo.remark42.com")

	c, err := opts.makeCache()
	require.NoError(t, err)

	// a rendered tree for a busy page is far larger than 64 KiB
	val := bytes.Repeat([]byte("x"), 300_000)
	key := cache.NewKey("remark").ID("find").Scopes("remark", "https://example.com/post")
	loads := 0
	for range 3 {
		data, err := c.Get(key, func() ([]byte, error) { loads++; return val, nil })
		require.NoError(t, err)
		assert.Len(t, data, len(val))
	}
	assert.Equal(t, 1, loads, "a 300 KB value must be served from cache with default limits")
}

func TestNewEmailSender_MasksRecipientInLog(t *testing.T) {
	var lines []string
	l := log.Func(func(format string, args ...any) { lines = append(lines, fmt.Sprintf(format, args...)) })
	sndr := newEmailSender(sender.EmailParams{Host: "127.0.0.1", Port: 1, TimeOut: time.Second}, l)

	// the debug line is written before the connection is tried, so the send itself may fail
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.Error(t, sndr.SendContext(ctx, "user@example.com", "sign-in link"))

	require.NotEmpty(t, lines, "the sender logs the send")
	logged := strings.Join(lines, "\n")
	assert.NotContains(t, logged, "user@example.com", "the recipient's address stays out of the log")
	assert.Contains(t, logged, "message to ********", "the send is still logged, with the address masked")
}

func TestServerCommand_identiconSize(t *testing.T) {
	tbl := []struct {
		name string
		args []string
		want int
	}{
		{name: "default", want: 48},
		{name: "set", args: []string{"--avatar.identicon-size=96"}, want: 96},
		{name: "resize limit below it", args: []string{"--avatar.rsz-lmt=24"}, want: 24},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ServerCommand{}
			_, err := flags.NewParser(&cmd, flags.Default).ParseArgs(append([]string{"--admin-passwd=password", "--site=remark"}, tt.args...))
			require.NoError(t, err)

			avatars := avatar.NewLocalFS(t.TempDir())
			authenticator := cmd.getAuthenticator(nil, avatars, nil, nil)
			avatarURL, err := authenticator.AvatarProxy().Put(token.User{ID: "anonymous_1", Name: "anon"}, nil)
			require.NoError(t, err)

			r, _, err := avatars.Get(path.Base(avatarURL))
			require.NoError(t, err)
			defer r.Close()
			cfg, err := png.DecodeConfig(r)
			require.NoError(t, err)
			assert.Equal(t, tt.want, cfg.Width, "the identicon of a user without a picture is drawn at the configured size")
			assert.Equal(t, tt.want, cfg.Height)
		})
	}
}

func TestServerCommand_tokenAudiences(t *testing.T) {
	cmd := ServerCommand{Sites: []string{"remark", "other"}}
	cmd.Store.Type = "bolt"
	auds := cmd.tokenAudiences()
	require.NotNil(t, auds, "the bolt store knows its sites, so the audience is limited to them")
	got, err := auds.Get()
	require.NoError(t, err)
	assert.Equal(t, []string{"remark", "other"}, got)

	cmd.Sites[0] = "changed"
	got, err = auds.Get()
	require.NoError(t, err)
	assert.Equal(t, []string{"remark", "other"}, got, "the list is fixed when the authenticator is built")

	cmd.Store.Type = "rpc"
	assert.Nil(t, cmd.tokenAudiences(), "a remote store decides which sites exist, so any audience passes")
}

func TestServerCommand_servesAudience(t *testing.T) {
	cmd := ServerCommand{Sites: []string{"remark", "other"}}
	cmd.Store.Type = "bolt"
	assert.True(t, cmd.servesAudience([]string{"remark"}))
	assert.True(t, cmd.servesAudience([]string{"remark", "other"}))
	assert.False(t, cmd.servesAudience([]string{"Remark"}), "sites are compared exactly")
	assert.False(t, cmd.servesAudience([]string{"remark", "third"}), "every audience must be served")

	cmd.Store.Type = "rpc"
	assert.True(t, cmd.servesAudience([]string{"Remark"}), "a remote store decides which sites exist")
}

func TestServerCommand_RPCClients(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, passwd, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "user:passwd", user+":"+passwd)
		assert.Equal(t, "/rpc", r.URL.Path)
		_, _ = w.Write([]byte(`{"error":"rpc called"}`))
	}))
	defer ts.Close()
	defer http.DefaultClient.CloseIdleConnections()

	rpc := RPCGroup{API: ts.URL + "/rpc", TimeOut: time.Second, AuthUser: "user", AuthPassword: "passwd"}
	cmd := ServerCommand{}
	cmd.Store.Type, cmd.Store.RPC = "rpc", rpc
	cmd.Image.Type, cmd.Image.RPC = "rpc", rpc
	cmd.Admin.Type, cmd.Admin.RPC.RPCGroup = "rpc", rpc

	dataStore, err := cmd.makeDataStore()
	require.NoError(t, err)
	err = dataStore.Close()
	assert.ErrorContains(t, err, "rpc called", "data store")

	pictures, err := cmd.makePicturesStore()
	require.NoError(t, err)
	_, err = pictures.Info()
	assert.ErrorContains(t, err, "rpc called", "pictures store")

	admins, err := cmd.makeAdminStore()
	require.NoError(t, err)
	_, err = admins.Key("remark")
	assert.ErrorContains(t, err, "rpc called", "admin store")
}

func TestServerCommand_demoSiteID(t *testing.T) {
	tbl := []struct {
		sites []string
		want  string
	}{
		{[]string{"remark"}, "remark"},
		{[]string{"myblog", "other"}, "myblog"},
		// the stores register a site exactly as given, so the demo pages must name it the same way
		{[]string{" myblog "}, " myblog "},
		{[]string{"  ", "second"}, "second"},
		{[]string{"", "second"}, "second"},
		{nil, ""},
	}
	for _, tt := range tbl {
		assert.Equal(t, tt.want, (&ServerCommand{Sites: tt.sites}).demoSiteID(), "sites %q", tt.sites)
	}
}

func TestServerApp_WebhookUsesConfiguredTimeout(t *testing.T) {
	cmd := ServerCommand{}
	cmd.Notify.Admins = []string{"webhook"}
	cmd.Notify.Webhook.URL = "http://127.0.0.1/hook"
	cmd.Notify.Webhook.Timeout = 17 * time.Second

	destinations, err := cmd.makeNotifyDestinations(nil)
	require.NoError(t, err)
	require.Len(t, destinations, 1)
	wh, ok := destinations[0].(*notify.Webhook)
	require.True(t, ok, "webhook destination, got %T", destinations[0])
	assert.Equal(t, 17*time.Second, wh.Timeout, "NOTIFY_WEBHOOK_TIMEOUT reaches the webhook")
}

// TestServerCommand_makeNotifyDestinationsBrokenWebhookKeepsOthers covers a webhook that cannot be built:
// it is reported, and Slack and email, including users' reply notifications, are still set up
func TestServerCommand_makeNotifyDestinationsBrokenWebhookKeepsOthers(t *testing.T) {
	tbl := []struct {
		name, url, template, wantErr string
	}{
		{name: "no url", wantErr: "webhook URL is required"},
		{name: "bad template", url: "http://127.0.0.1/hook", template: "{{.Text", wantErr: "unable to parse webhook template"},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ServerCommand{}
			cmd.Notify.Admins = []string{"webhook", "slack", "email"}
			cmd.Notify.Users = []string{"email"}
			cmd.Admin.Shared.Email = []string{"admin@example.com"}
			cmd.Notify.Webhook.URL = tt.url
			cmd.Notify.Webhook.Template = tt.template

			destinations, err := cmd.makeNotifyDestinations(nil)
			require.ErrorContains(t, err, tt.wantErr)
			var slack, email int
			for _, d := range destinations {
				switch d.(type) {
				case *notify.Slack:
					slack++
				case *notify.Email:
					email++
				default:
					t.Errorf("unexpected destination %T", d)
				}
			}
			assert.Equal(t, 1, slack, "slack is set up despite the webhook error")
			assert.Equal(t, 1, email, "email is set up despite the webhook error")
		})
	}
}

// TestServerCommand_makeNotifyDestinationsAdminEmailMissing covers NOTIFY_ADMINS=email with no address to
// send to: the default admin@ address of ADMIN_SHARED_EMAIL is not used for notifications, so the start-up
// reports it, and the email destination is still built for users' reply notifications
func TestServerCommand_makeNotifyDestinationsAdminEmailMissing(t *testing.T) {
	tbl := []struct {
		name       string
		emails     []string
		wantAdmins []string
		wantErr    bool
	}{
		{name: "not set", emails: nil, wantErr: true},
		{name: "empty env value", emails: []string{""}, wantErr: true},
		{name: "blank values", emails: []string{" ", ""}, wantErr: true},
		{name: "set", emails: []string{"a@example.com", "b@example.com"}, wantAdmins: []string{"a@example.com", "b@example.com"}},
		{name: "set with a blank", emails: []string{"a@example.com", " "}, wantAdmins: []string{"a@example.com"}},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ServerCommand{
				RemarkURL: "https://remark42.example.com"}
			cmd.Notify.Admins = []string{"email"}
			cmd.Admin.Shared.Email = tt.emails

			destinations, err := cmd.makeNotifyDestinations(nil)
			if tt.wantErr {
				require.EqualError(t, err, "admin email notifications are on, but --admin.shared.email (ADMIN_SHARED_EMAIL) "+
					"has no address, so no admin is notified by email")
			} else {
				require.NoError(t, err)
			}
			require.Len(t, destinations, 1)
			e, ok := destinations[0].(*notify.Email)
			require.True(t, ok, "email destination, got %T", destinations[0])
			assert.Equal(t, tt.wantAdmins, e.AdminEmails, "admin copies go to the configured addresses only")
		})
	}

	t.Run("user notifications only", func(t *testing.T) {
		cmd := ServerCommand{}
		cmd.Notify.Users = []string{"email"}
		_, err := cmd.makeNotifyDestinations(nil)
		require.NoError(t, err, "no admin address is needed without admin email notifications")
	})
}

// TestServerCommand_makeNotifyDestinationsAvatarURLs covers the pictures in notification emails: the email
// shows only avatars of this instance, so it must know the prefix the avatar proxy makes pictures with and
// the root route that pictures stored by older versions point to
func TestServerCommand_makeNotifyDestinationsAvatarURLs(t *testing.T) {
	cmd := ServerCommand{RemarkURL: "https://remark42.example.com/"} // a picture URL has no double slash
	cmd.Notify.Users = []string{"email"}
	authenticator := cmd.getAuthenticator(nil, avatar.NewLocalFS(t.TempDir()), nil, nil)

	destinations, err := cmd.makeNotifyDestinations(authenticator)
	require.NoError(t, err)
	require.Len(t, destinations, 1)
	e, ok := destinations[0].(*notify.Email)
	require.True(t, ok, "email destination, got %T", destinations[0])

	picture, err := authenticator.AvatarProxy().Put(token.User{ID: "anonymous_1", Name: "anon"}, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{
		strings.TrimSuffix(picture, path.Base(picture)), // where the avatar proxy makes pictures
		"https://remark42.example.com/avatar/",          // where pictures stored by older versions point
	}, e.AvatarURLs)

	// a host name that is not ASCII: the picture is stored through the URL sanitizer, which escapes the name
	cmd = ServerCommand{RemarkURL: "https://комменты.example.com"}
	cmd.Notify.Users = []string{"email"}
	authenticator = cmd.getAuthenticator(nil, avatar.NewLocalFS(t.TempDir()), nil, nil)
	destinations, err = cmd.makeNotifyDestinations(authenticator)
	require.NoError(t, err)
	require.Len(t, destinations, 1)
	e, ok = destinations[0].(*notify.Email)
	require.True(t, ok)
	picture, err = authenticator.AvatarProxy().Put(token.User{ID: "anonymous_1", Name: "anon"}, nil)
	require.NoError(t, err)
	c := store.Comment{User: store.User{ID: "anonymous_1", Name: "anon", Picture: picture}}
	c.Sanitize()
	require.NotEqual(t, picture, c.User.Picture, "the stored picture has the host name escaped")
	assert.Contains(t, e.AvatarURLs, strings.TrimSuffix(picture, path.Base(picture)), "the picture as the proxy makes it")
	assert.Contains(t, e.AvatarURLs, strings.TrimSuffix(c.User.Picture, path.Base(c.User.Picture)), "the picture as stored")
}

func TestServerCommand_makeTelegramNotifyNoChannel(t *testing.T) {
	cmd := ServerCommand{}
	cmd.Notify.Admins = []string{"telegram"}
	cmd.Telegram.Token = "token"
	_, err := cmd.makeTelegramNotify()
	require.EqualError(t, err, "--notify.telegram.chan (NOTIFY_TELEGRAM_CHAN) must be set for admin notifications to work",
		"the error names a flag that exists")
}

// TestServerApp_UnsubscribeTokenIsNotALogin covers the token in unsubscribe links: it names the user
// and their email, is signed with the shared secret and is valid for a century, so the email login
// must not accept it in place of a confirmation token it issued itself.
func TestServerApp_UnsubscribeTokenIsNotALogin(t *testing.T) {
	app := startServerApp(t, func(o ServerCommand) ServerCommand {
		o.Admin.Shared.Email = []string{"admin@example.org"} // admin email notifications are on in the shared setup
		return o
	})
	port := app.Port

	destinations, err := app.makeNotifyDestinations(app.authenticator)
	require.NoError(t, err)
	var tkn string
	for _, d := range destinations {
		if e, ok := d.(*notify.Email); ok {
			tkn, err = e.TokenGenFn("email_user1", "user1@example.org", "remark")
			require.NoError(t, err)
		}
	}
	require.NotEmpty(t, tkn, "email notifications are configured, so unsubscribe links carry a token")

	client := http.Client{Timeout: 10 * time.Second}
	defer client.CloseIdleConnections()
	resp := getRetryThrottled(t, &client, fmt.Sprintf("http://localhost:%d/auth/email/login?token=%s", port, tkn))
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "an unsubscribe token is not a login, %s", body)
	for _, c := range resp.Cookies() {
		assert.NotEqual(t, "JWT", c.Name, "no session from an unsubscribe token")
	}
}

// TestServerApp_UnsubscribeLinkPerRecipient follows a reply notification from the notifier through the
// mail sent to every subscriber up the thread to the unsubscribe link in it. A reply reaches the authors
// of all the comments above it, so each mail's link must name its own recipient, or the handler finds
// another user's address in it and refuses to unsubscribe anyone.
func TestServerApp_UnsubscribeLinkPerRecipient(t *testing.T) {
	smtpSrv := startSMTPCapture(t) // before the app, so it is closed after the app stops
	app := startServerApp(t, func(o ServerCommand) ServerCommand {
		o.SMTP.Host, o.SMTP.Port = "127.0.0.1", smtpSrv.port()
		o.SMTP.Username, o.SMTP.Password = "", "" // the capture server offers no AUTH
		o.Admin.Shared.Email = []string{"admin@example.org"}
		return o
	})

	// thread a <- b <- c, the authors of a and b subscribed, c's reply is the one notified about
	locator := store.Locator{SiteID: "remark", URL: "https://example.org/post"}
	create := func(userID, parentID string) store.Comment {
		id, err := app.dataService.Create(store.Comment{Locator: locator, ParentID: parentID, PostTitle: "post",
			Text: "comment of " + userID, User: store.User{ID: userID, Name: userID}})
		require.NoError(t, err)
		c, err := app.dataService.Get(locator, id, store.User{})
		require.NoError(t, err)
		return c
	}
	a := create("email_a", "")
	b := create("email_b", a.ID)
	c := create("email_c", b.ID)
	for user, address := range map[string]string{"email_a": "a@example.org", "email_b": "b@example.org"} {
		_, err := app.dataService.SetUserEmail("remark", user, address)
		require.NoError(t, err)
	}

	app.notifyService.Submit(notify.Request{Comment: c})
	require.Eventually(t, func() bool { return len(smtpSrv.messages()) == 3 }, serverStartTimeout, serverStartPoll,
		"mail to a, b and the admin")

	links := map[string]string{}
	for _, m := range smtpSrv.messages() {
		require.Len(t, m.to, 1, "one recipient per mail")
		header, body := readMail(t, m.data)
		if m.to[0] == "admin@example.org" {
			assert.Empty(t, header.Get("List-Unsubscribe"), "no unsubscribe header in the admin's mail")
			assert.NotContains(t, body, "unsubscribe.html", "no unsubscribe link in the admin's mail")
			continue
		}
		link := strings.TrimSuffix(strings.TrimPrefix(header.Get("List-Unsubscribe"), "<"), ">")
		require.NotEmpty(t, link, "unsubscribe header in the mail to %s", m.to[0])
		assert.Contains(t, body, `href="`+link+`"`, "the link in the text of the mail to %s is the one in its header", m.to[0])
		links[m.to[0]] = link
	}
	require.Len(t, links, 2, "mail to a and b, none to c, who wrote the reply")

	// each link names its own recipient
	handshake := func(link string) string {
		u, err := url.Parse(link)
		require.NoError(t, err)
		claims, err := app.authenticator.TokenService().Parse(u.Query().Get("tkn"))
		require.NoError(t, err)
		require.NotNil(t, claims.Handshake)
		return claims.Handshake.ID
	}
	assert.Equal(t, "email_a::a@example.org", handshake(links["a@example.org"]), "a's link names a")
	assert.Equal(t, "email_b::b@example.org", handshake(links["b@example.org"]), "b's link names b")

	client := http.Client{Timeout: 10 * time.Second}
	defer client.CloseIdleConnections()
	local := func(link string) string { // the links point at the configured remark URL, the test server is local
		u, err := url.Parse(link)
		require.NoError(t, err)
		return fmt.Sprintf("http://localhost:%d%s", app.Port, u.RequestURI())
	}
	email := func(user string) string {
		address, err := app.dataService.GetUserEmail("remark", user)
		require.NoError(t, err)
		return address
	}

	// a, who wrote the root comment, follows the link from the mail and is unsubscribed, b is not
	resp, err := client.Get(local(links["a@example.org"]))
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode, "a's link unsubscribes a: %s", body)
	assert.Empty(t, email("email_a"), "a unsubscribed")
	assert.Equal(t, "b@example.org", email("email_b"), "b still subscribed")

	// b, the direct parent, unsubscribes the way a mail client does it from the header, with a one-click POST
	resp, err = client.Post(local(links["b@example.org"]), "application/x-www-form-urlencoded",
		strings.NewReader("List-Unsubscribe=One-Click"))
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode, "b's link unsubscribes b: %s", body)
	assert.Empty(t, email("email_b"), "b unsubscribed")
}

// readMail returns the headers of a mail as sent and its text, decoded from quoted-printable and HTML
func readMail(t *testing.T, data []byte) (header mail.Header, text string) {
	t.Helper()
	m, err := mail.ReadMessage(bytes.NewReader(data))
	require.NoError(t, err)
	body, err := io.ReadAll(quotedprintable.NewReader(m.Body))
	require.NoError(t, err)
	return m.Header, html.UnescapeString(string(body))
}

// smtpCapture is an SMTP server on a loopback port which accepts every mail and keeps it
type smtpCapture struct {
	ln    net.Listener
	wg    sync.WaitGroup
	mu    sync.Mutex
	conns map[net.Conn]struct{}
	mails []capturedMail
}

type capturedMail struct {
	to   []string
	data []byte
}

// startSMTPCapture starts an smtpCapture, stopped when the test ends
func startSMTPCapture(t *testing.T) *smtpCapture {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := &smtpCapture{ln: ln, conns: map[net.Conn]struct{}{}}
	s.wg.Go(func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			s.mu.Lock()
			s.conns[conn] = struct{}{}
			s.mu.Unlock()
			s.wg.Go(func() { s.serve(conn) })
		}
	})
	t.Cleanup(func() {
		_ = ln.Close()
		s.mu.Lock()
		for conn := range s.conns {
			_ = conn.Close()
		}
		s.mu.Unlock()
		s.wg.Wait()
	})
	return s
}

func (s *smtpCapture) port() int { return s.ln.Addr().(*net.TCPAddr).Port }

func (s *smtpCapture) messages() []capturedMail {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.mails)
}

// serve answers one SMTP session with the replies net/smtp needs to deliver a mail without AUTH or TLS
func (s *smtpCapture) serve(conn net.Conn) {
	defer func() {
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
		_ = conn.Close()
	}()
	tp := textproto.NewConn(conn)
	if tp.PrintfLine("220 localhost ESMTP") != nil {
		return
	}
	var m capturedMail
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}
		verb, arg, _ := strings.Cut(line, " ")
		reply := "250 ok"
		switch strings.ToUpper(verb) {
		case "MAIL":
			m = capturedMail{}
		case "RCPT":
			_, addr, _ := strings.Cut(arg, ":")
			m.to = append(m.to, strings.Trim(addr, "<> "))
		case "DATA":
			if tp.PrintfLine("354 go ahead") != nil {
				return
			}
			if m.data, err = tp.ReadDotBytes(); err != nil {
				return
			}
			s.mu.Lock()
			s.mails = append(s.mails, m)
			s.mu.Unlock()
		case "QUIT":
			_ = tp.PrintfLine("221 bye")
			return
		}
		if tp.PrintfLine("%s", reply) != nil {
			return
		}
	}
}

func TestServerCommand_ExecuteReturnsSetupError(t *testing.T) {
	s := ServerCommand{}
	tmp := t.TempDir()
	args := []string{"--store.bolt.path=/dev/null", "--backup=" + tmp, "--image.fs.path=" + tmp,
		"--avatar.fs.path=" + filepath.Join(tmp, "avatars")}
	parseCmd(t, &s, "https://demo.remark42.com", args...)

	// a misconfiguration comes back as an error for main to report, without a stack dump and exit
	err := s.Execute(args)
	require.ErrorContains(t, err, "failed to setup application: failed to make data store engine")
}
