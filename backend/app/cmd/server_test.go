package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-pkgz/auth/token"
	"github.com/umputun/go-flags"
	"go.uber.org/goleak"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerApp(t *testing.T) {
	port := chooseRandomUnusedPort()
	app, ctx, cancel := prepServerApp(t, func(o ServerCommand) ServerCommand {
		o.Port = port
		return o
	})

	go func() { _ = app.run(ctx) }()
	waitForHTTPServerStart(port)

	// send ping
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/ping", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "pong", string(body))

	// add comment
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment", port),
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "remark"}}`))
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body, _ = ioutil.ReadAll(resp.Body)
	t.Log(string(body))

	email, err := app.dataService.AdminStore.Email("")
	assert.NoError(t, err)
	assert.Equal(t, "admin@demo.remark42.com", email, "default admin email")

	cancel()
	app.Wait()
}

func TestServerApp_DevMode(t *testing.T) {
	port := chooseRandomUnusedPort()
	app, ctx, cancel := prepServerApp(t, func(o ServerCommand) ServerCommand {
		o.Port = port
		o.AdminPasswd = "password"
		o.Auth.Dev = true
		return o
	})

	go func() { _ = app.run(ctx) }()
	waitForHTTPServerStart(port)

	require.Equal(t, 5+1, len(app.restSrv.Authenticator.Providers()), "extra auth provider")
	assert.Equal(t, "dev", app.restSrv.Authenticator.Providers()[4].Name(), "dev auth provider")
	// send ping
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/ping", port))
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, "pong", string(body))

	cancel()
	app.Wait()
}

func TestServerApp_AnonMode(t *testing.T) {
	port := chooseRandomUnusedPort()
	app, ctx, cancel := prepServerApp(t, func(o ServerCommand) ServerCommand {
		o.Port = port
		o.Auth.Anonymous = true
		return o
	})

	go func() { _ = app.run(ctx) }()
	waitForHTTPServerStart(port)

	require.Equal(t, 5+1, len(app.restSrv.Authenticator.Providers()), "extra auth provider for anon")
	assert.Equal(t, "anonymous", app.restSrv.Authenticator.Providers()[5].Name(), "anon auth provider")

	// send ping
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/ping", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "pong", string(body))

	// try to login with good name
	resp, err = http.Get(fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=blah123&aud=remark", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// try to add a comment as good anonymous
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment", port),
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "remark"}}`))
	require.NoError(t, err)

	tkn, claims := getAuthFromCookie(t, app, resp)
	require.NotEmpty(t, tkn)
	req.Header.Add("X-JWT", tkn)
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// try to login with non-latin name
	resp, err = http.Get(fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=Раз_Два%20%20Три_34567&aud=remark", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// try to login with bad name
	resp, err = http.Get(fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=**blah123&aud=remark", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// try to login with short name
	resp, err = http.Get(fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=bl%20%20&aud=remark", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// try to login with long name
	ln := strings.Repeat("x", 65)
	resp, err = http.Get(fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=%s&aud=remark", port, ln))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// try to login with admin name
	resp, err = http.Get(fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=umputun&aud=remark", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// try to add a comment as anonymous with admin name
	client = http.Client{Timeout: 10 * time.Second}
	req, err = http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment", port),
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
	sslPort := chooseRandomUnusedPort()
	opts.SetCommon(CommonOpts{RemarkURL: fmt.Sprintf("https://localhost:%d", sslPort), SharedSecret: "123456"})

	// prepare options
	p := flags.NewParser(&opts, flags.Default)
	port := chooseRandomUnusedPort()
	_, err := p.ParseArgs([]string{"--admin-passwd=password", "--port=" + strconv.Itoa(port), "--store.bolt.path=/tmp/xyz", "--backup=/tmp",
		"--avatar.type=bolt", "--avatar.bolt.file=/tmp/ava-test.db", "--notify.type=none",
		"--ssl.type=static", "--ssl.cert=testdata/cert.pem", "--ssl.key=testdata/key.pem",
		"--ssl.port=" + strconv.Itoa(sslPort), "--image.fs.path=/tmp"})
	require.NoError(t, err)

	// create app
	app, err := opts.newServerApp()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = app.run(ctx) }()
	waitForHTTPSServerStart(sslPort)

	client := http.Client{
		// prevent http redirect
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},

		// allow self-signed certificate
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// check http to https redirect response
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/blah?param=1", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 307, resp.StatusCode)
	assert.Equal(t, fmt.Sprintf("https://localhost:%d/blah?param=1", sslPort), resp.Header.Get("Location"))

	// check https server
	resp, err = client.Get(fmt.Sprintf("https://localhost:%d/ping", sslPort))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
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
	port := chooseRandomUnusedPort()
	_, err := p.ParseArgs([]string{"--admin-passwd=password", "--cache.type=none",
		"--store.type=rpc", "--store.rpc.api=http://127.0.0.1",
		"--port=" + strconv.Itoa(port), "--admin.type=rpc", "--admin.rpc.api=http://127.0.0.1", "--avatar.fs.path=/tmp"})
	require.NoError(t, err)
	opts.Auth.Github.CSEC, opts.Auth.Github.CID = "csec", "cid"
	opts.BackupLocation, opts.Image.FS.Path = "/tmp", "/tmp"

	// create app
	app, err := opts.newServerApp()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = app.run(ctx) }()
	waitForHTTPServerStart(port)

	// send ping
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/ping", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
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
	_, err = opts.newServerApp()
	assert.EqualError(t, err, "failed to make data store engine: failed to create bolt store: can't make directory /dev/null: mkdir /dev/null: not a directory")
	t.Log(err)

	// RO backup location
	opts = ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	_, err = p.ParseArgs([]string{"--store.bolt.path=/tmp", "--backup=/dev/null/not-writable"})
	assert.NoError(t, err)
	_, err = opts.newServerApp()
	assert.EqualError(t, err, "can't make directory /dev/null/not-writable: mkdir /dev/null: not a directory")
	t.Log(err)

	// invalid url
	opts = ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "demo.remark42.com", SharedSecret: "123456"})

	_, err = p.ParseArgs([]string{"--backup=/tmp", "----store.bolt.path=/tmp"})
	assert.NoError(t, err)
	_, err = opts.newServerApp()
	assert.EqualError(t, err, "invalid remark42 url demo.remark42.com")
	t.Log(err)

	opts = ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	_, err = p.ParseArgs([]string{"--backup=/tmp", "--store.type=blah"})
	assert.Error(t, err, "blah is invalid type")

	opts.Store.Type = "blah"
	_, err = opts.newServerApp()
	assert.EqualError(t, err, "failed to make data store engine: unsupported store type blah")
	t.Log(err)
}

func TestServerApp_Shutdown(t *testing.T) {
	app, ctx, cancel := prepServerApp(t, func(o ServerCommand) ServerCommand {
		o.Port = chooseRandomUnusedPort()
		return o
	})
	time.AfterFunc(100*time.Millisecond, func() {
		cancel()
	})
	st := time.Now()
	err := app.run(ctx)
	assert.NoError(t, err)
	assert.True(t, time.Since(st).Seconds() < 1, "should take about 100msec")
	app.Wait()
}

func TestServerApp_MainSignal(t *testing.T) {

	done := make(chan struct{})
	go func() {
		<-done
		time.Sleep(250 * time.Millisecond)
		err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.NoError(t, err)
	}()

	s := ServerCommand{}
	s.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	p := flags.NewParser(&s, flags.Default)
	port := chooseRandomUnusedPort()
	args := []string{"test", "--store.bolt.path=/tmp/xyz", "--backup=/tmp", "--avatar.type=bolt",
		"--avatar.bolt.file=/tmp/ava-test.db", "--port=" + strconv.Itoa(port), "--notify.type=none", "--image.fs.path=/tmp"}
	defer os.Remove("/tmp/ava-test.db")
	_, err := p.ParseArgs(args)
	require.NoError(t, err)
	st := time.Now()
	close(done)
	err = s.Execute(args)
	assert.NoError(t, err, "execute should be without errors")
	assert.True(t, time.Since(st).Seconds() < 5, "should take under five sec", time.Since(st).Seconds())
}

func TestServerApp_DeprecatedArgs(t *testing.T) {
	s := ServerCommand{}
	s.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	p := flags.NewParser(&s, flags.Default)
	args := []string{
		"test",
		"--auth.email.host=smtp.example.org",
		"--auth.email.port=666",
		"--auth.email.tls",
		"--auth.email.user=test_user",
		"--auth.email.passwd=test_password",
		"--auth.email.timeout=15s",
		"--auth.email.template=file.tmpl",
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
			{Old: "auth.email.host", New: "smtp.host", RemoveVersion: "1.7.0"},
			{Old: "auth.email.port", New: "smtp.port", RemoveVersion: "1.7.0"},
			{Old: "auth.email.tls", New: "smtp.tls", RemoveVersion: "1.7.0"},
			{Old: "auth.email.user", New: "smtp.username", RemoveVersion: "1.7.0"},
			{Old: "auth.email.passwd", New: "smtp.password", RemoveVersion: "1.7.0"},
			{Old: "auth.email.timeout", New: "smtp.timeout", RemoveVersion: "1.7.0"},
			{Old: "auth.email.template", RemoveVersion: "1.9.0"},
		},
		deprecatedFlags)
	assert.Equal(t, "smtp.example.org", s.SMTP.Host)
	assert.Equal(t, 666, s.SMTP.Port)
	assert.Equal(t, true, s.SMTP.TLS)
	assert.Equal(t, "test_user", s.SMTP.Username)
	assert.Equal(t, "test_password", s.SMTP.Password)
	assert.Equal(t, 15*time.Second, s.SMTP.TimeOut)
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
	port := chooseRandomUnusedPort()
	app, ctx, cancel := prepServerApp(t, func(o ServerCommand) ServerCommand {
		o.Port = port
		return o
	})

	go func() { _ = app.run(ctx) }()
	waitForHTTPServerStart(port)

	// make a token for user dev
	tkService := app.restSrv.Authenticator.TokenService()
	tkService.TokenDuration = time.Second

	claims := token.Claims{
		StandardClaims: jwt.StandardClaims{
			Audience:  "remark",
			Issuer:    "remark",
			ExpiresAt: time.Now().Add(time.Second).Unix(),
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
		},
		User: &token.User{
			ID:   "dev",
			Name: "developer one",
		},
	}
	tk, err := tkService.Token(claims)
	require.NoError(t, err)
	t.Log(tk)

	// add comment
	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment", port),
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/p/2018/12/29/podcast-630/", "site": "remark"}}`))
	require.NoError(t, err)
	req.Header.Set("X-JWT", tk)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "non-blocked user able to post")

	// add comment with no-aud claim
	claimsNoAud := claims
	claimsNoAud.Audience = ""
	tkNoAud, err := tkService.Token(claimsNoAud)
	require.NoError(t, err)
	t.Logf("no-aud claims: %s", tkNoAud)
	req, err = http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment", port),
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/p/2018/12/29/podcast-631/",
	"site": "remark"}}`))
	require.NoError(t, err)
	req.Header.Set("X-JWT", tkNoAud)
	resp, err = client.Do(req)
	require.NoError(t, err)
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "user without aud claim rejected, \n"+tkNoAud+"\n"+string(body))

	// block user dev as admin
	req, err = http.NewRequest(http.MethodPut,
		fmt.Sprintf("http://localhost:%d/api/v1/admin/user/dev?site=remark&block=1&ttl=10d", port), nil)
	assert.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "user dev blocked")
	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	t.Log(string(b))

	// try add a comment with blocked user
	req, err = http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment", port),
		strings.NewReader(`{"text": "test 123 blah", "locator":{"url": "https://radio-t.com/blah1", "site": "remark"}}`))
	require.NoError(t, err)
	req.Header.Set("X-JWT", tk)
	resp, err = client.Do(req)
	require.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized,
		"blocked user can't post, \n"+tk+"\n"+string(body))

	cancel()
	app.Wait()
	client.CloseIdleConnections()
}

func TestServer_loadEmailTemplate(t *testing.T) {
	cmd := ServerCommand{}
	cmd.Auth.Email.MsgTemplate = "testdata/email.tmpl"
	r, err := cmd.loadEmailTemplate()
	assert.NoError(t, err)
	assert.Equal(t, "The token is {{.Token}}", r)

	cmd.Auth.Email.MsgTemplate = "badpath.tmpl"
	r, err = cmd.loadEmailTemplate()
	assert.EqualError(t, err, "failed to read file badpath.tmpl: open badpath.tmpl: no such file or directory")
	assert.Equal(t, r, "")
}

func chooseRandomUnusedPort() (port int) {
	for i := 0; i < 10; i++ {
		port = 40000 + int(rand.Int31n(10000))
		if ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port)); err == nil {
			_ = ln.Close()
			break
		}
	}
	return port
}

func waitForHTTPServerStart(port int) {
	// wait for up to 3 seconds for server to start before returning it
	client := http.Client{Timeout: time.Second}
	for i := 0; i < 300; i++ {
		time.Sleep(time.Millisecond * 10)
		if resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port)); err == nil {
			_ = resp.Body.Close()
			return
		}
	}
}

func waitForHTTPSServerStart(port int) {
	// wait for up to 3 seconds for HTTPS server to start
	for i := 0; i < 300; i++ {
		time.Sleep(time.Millisecond * 10)
		conn, _ := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), time.Millisecond*10)
		if conn != nil {
			_ = conn.Close()
			break
		}
	}
}

func prepServerApp(t *testing.T, fn func(o ServerCommand) ServerCommand) (*serverApp, context.Context, context.CancelFunc) {
	cmd := ServerCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "secret"})

	// prepare options
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--admin-passwd=password", "--site=remark"})
	require.NoError(t, err)
	cmd.Avatar.FS.Path, cmd.Avatar.Type, cmd.BackupLocation, cmd.Image.FS.Path = "/tmp", "fs", "/tmp", "/tmp"
	cmd.Store.Bolt.Path = fmt.Sprintf("/tmp/%d", cmd.Port)
	cmd.Store.Bolt.Timeout = 10 * time.Second
	cmd.Auth.Github.CSEC, cmd.Auth.Github.CID = "csec", "cid"
	cmd.Auth.Google.CSEC, cmd.Auth.Google.CID = "csec", "cid"
	cmd.Auth.Facebook.CSEC, cmd.Auth.Facebook.CID = "csec", "cid"
	cmd.Auth.Yandex.CSEC, cmd.Auth.Yandex.CID = "csec", "cid"
	cmd.Auth.Email.Enable = true
	cmd.Auth.Email.MsgTemplate = "testdata/email.tmpl"
	cmd.BackupLocation = "/tmp"
	cmd.Notify.Type = []string{"email"}
	cmd.Notify.Email.From = "from@example.org"
	cmd.Notify.Email.VerificationSubject = "test verification email subject"
	cmd.SMTP.Host = "127.0.0.1"
	cmd.SMTP.Port = 25
	cmd.SMTP.Username = "test_user"
	cmd.SMTP.Password = "test_password"
	cmd.SMTP.TimeOut = time.Second
	cmd.UpdateLimit = 10
	cmd.Admin.Type = "shared"
	cmd.Admin.Shared.Admins = []string{"umputun", "bobuk"}
	cmd = fn(cmd)

	os.Remove(cmd.Store.Bolt.Path + "/remark.db")

	// create app
	app, err := cmd.newServerApp()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	rand.Seed(time.Now().UnixNano())
	return app, ctx, cancel
}

func TestMain(m *testing.M) {
	// ignore is added only for GitHub Actions, can't reproduce locally
	goleak.VerifyTestMain(m, goleak.IgnoreTopFunction("net/http.(*Server).Shutdown"))
}
