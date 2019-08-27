package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-pkgz/auth/token"
	log "github.com/go-pkgz/lgr"
	"github.com/jessevdk/go-flags"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerApp(t *testing.T) {
	port := rand.Intn(40000) + 10000
	app, ctx := prepServerApp(t, 1500*time.Millisecond, func(o ServerCommand) ServerCommand {
		o.Port = port
		return o
	})

	go func() { _ = app.run(ctx) }()
	time.Sleep(250 * time.Millisecond) // let server start

	// send ping
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/ping", port))
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "pong", string(body))

	// add comment
	client := http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment", port),
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "remark"}}`))
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err = client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body, _ = ioutil.ReadAll(resp.Body)
	t.Log(string(body))

	email, err := app.dataService.AdminStore.Email("")
	assert.NoError(t, err)
	assert.Equal(t, "admin@demo.remark42.com", email, "default admin email")

	app.Wait()
}

func TestServerApp_DevMode(t *testing.T) {
	port := rand.Intn(40000) + 10000
	app, ctx := prepServerApp(t, 500*time.Millisecond, func(o ServerCommand) ServerCommand {
		o.Port = port
		o.AdminPasswd = "password"
		o.Auth.Dev = true
		return o
	})

	go func() { _ = app.run(ctx) }()
	time.Sleep(100 * time.Millisecond) // let server start

	assert.Equal(t, 4+1, len(app.restSrv.Authenticator.Providers()), "extra auth provider")
	assert.Equal(t, "dev", app.restSrv.Authenticator.Providers()[4].Name(), "dev auth provider")
	// send ping
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/ping", port))
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "pong", string(body))

	app.Wait()
}

func TestServerApp_AnonMode(t *testing.T) {
	port := rand.Intn(40000) + 10000
	app, ctx := prepServerApp(t, 1000*time.Millisecond, func(o ServerCommand) ServerCommand {
		o.Port = port
		o.Auth.Anonymous = true
		return o
	})

	go func() { _ = app.run(ctx) }()
	time.Sleep(100 * time.Millisecond) // let server start

	assert.Equal(t, 4+1, len(app.restSrv.Authenticator.Providers()), "extra auth provider for anon")
	assert.Equal(t, "anonymous", app.restSrv.Authenticator.Providers()[4].Name(), "anon auth provider")

	// send ping
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/ping", port))
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "pong", string(body))

	// try to login with good name
	resp, err = http.Get(fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=blah123&aud=remark42", port))
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// try to login with bad name
	resp, err = http.Get(fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=**blah123&aud=remark42", port))
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	// try to login with short name
	resp, err = http.Get(fmt.Sprintf("http://localhost:%d/auth/anonymous/login?user=bl%20%20&aud=remark42", port))
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	app.Wait()
}

func TestServerApp_WithSSL(t *testing.T) {
	opts := ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://localhost:18443", SharedSecret: "123456"})

	// prepare options
	p := flags.NewParser(&opts, flags.Default)
	_, err := p.ParseArgs([]string{"--admin-passwd=password", "--port=18080", "--store.bolt.path=/tmp/xyz", "--backup=/tmp",
		"--avatar.type=bolt", "--avatar.bolt.file=/tmp/ava-test.db", "--notify.type=none",
		"--ssl.type=static", "--ssl.cert=testdata/cert.pem", "--ssl.key=testdata/key.pem",
		"--ssl.port=18443", "--image.fs.path=/tmp"})
	require.Nil(t, err)

	// create app
	app, err := opts.newServerApp()
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(1 * time.Second)
		log.Print("[TEST] terminate app")
		cancel()
	}()
	go func() { _ = app.run(ctx) }()
	time.Sleep(100 * time.Millisecond) // let server start

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
	resp, err := client.Get("http://localhost:18080/blah?param=1")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 307, resp.StatusCode)
	assert.Equal(t, "https://localhost:18443/blah?param=1", resp.Header.Get("Location"))

	// check https server
	resp, err = client.Get("https://localhost:18443/ping")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "pong", string(body))

	app.Wait()
}

func TestServerApp_WithRemote(t *testing.T) {

	opts := ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	// prepare options
	p := flags.NewParser(&opts, flags.Default)
	_, err := p.ParseArgs([]string{"--admin-passwd=password", "--cache.type=none",
		"--store.type=rpc", "--store.rpc.api=http://127.0.0.1",
		"--port=12345", "--admin.type=rpc", "--admin.rpc.api=http://127.0.0.1", "--avatar.fs.path=/tmp"})
	require.Nil(t, err)
	opts.Auth.Github.CSEC, opts.Auth.Github.CID = "csec", "cid"
	opts.BackupLocation, opts.Image.FS.Path = "/tmp", "/tmp"

	// create app
	app, err := opts.newServerApp()
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Second)
		log.Print("[TEST] terminate app")
		cancel()
	}()
	go func() { _ = app.run(ctx) }()
	time.Sleep(100 * time.Millisecond) // let server start

	// send ping
	resp, err := http.Get("http://localhost:12345/api/v1/ping")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "pong", string(body))

	app.Wait()
}

func TestServerApp_Failed(t *testing.T) {
	opts := ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	p := flags.NewParser(&opts, flags.Default)

	// RO bolt location
	_, err := p.ParseArgs([]string{"--backup=/tmp", "--store.bolt.path=/dev/null", "--image.fs.path=/tmp"})
	assert.Nil(t, err)
	_, err = opts.newServerApp()
	assert.EqualError(t, err, "failed to make data store engine: failed to create bolt store: can't make directory /dev/null: mkdir /dev/null: not a directory")
	t.Log(err)

	// RO backup location
	opts = ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	_, err = p.ParseArgs([]string{"--store.bolt.path=/tmp", "--backup=/dev/null/not-writable"})
	assert.Nil(t, err)
	_, err = opts.newServerApp()
	assert.EqualError(t, err, "can't make directory /dev/null/not-writable: mkdir /dev/null: not a directory")
	t.Log(err)

	// invalid url
	opts = ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "demo.remark42.com", SharedSecret: "123456"})

	_, err = p.ParseArgs([]string{"--backup=/tmp", "----store.bolt.path=/tmp"})
	assert.Nil(t, err)
	_, err = opts.newServerApp()
	assert.EqualError(t, err, "invalid remark42 url demo.remark42.com")
	t.Log(err)

	opts = ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	_, err = p.ParseArgs([]string{"--backup=/tmp", "--store.type=blah"})
	assert.NotNil(t, err, "blah is invalid type")

	opts.Store.Type = "blah"
	_, err = opts.newServerApp()
	assert.EqualError(t, err, "failed to make data store engine: unsupported store type blah")
	t.Log(err)
}

func TestServerApp_Shutdown(t *testing.T) {
	app, ctx := prepServerApp(t, 500*time.Millisecond, func(o ServerCommand) ServerCommand {
		o.Port = rand.Intn(40000) + 10000
		return o
	})
	st := time.Now()
	err := app.run(ctx)
	assert.Nil(t, err)
	assert.True(t, time.Since(st).Seconds() < 1, "should take about 500msec")
	app.Wait()
}

func TestServerApp_MainSignal(t *testing.T) {

	go func() {
		time.Sleep(250 * time.Millisecond)
		err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.Nil(t, err)
	}()
	st := time.Now()

	s := ServerCommand{}
	s.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	p := flags.NewParser(&s, flags.Default)
	args := []string{"test", "--store.bolt.path=/tmp/xyz", "--backup=/tmp", "--avatar.type=bolt",
		"--avatar.bolt.file=/tmp/ava-test.db", "--port=18100", "--notify.type=none", "--image.fs.path=/tmp"}
	defer os.Remove("/tmp/ava-test.db")
	_, err := p.ParseArgs(args)
	require.Nil(t, err)
	err = s.Execute(args)
	assert.NoError(t, err, "execute failed")
	assert.True(t, time.Since(st).Seconds() < 1, "should take about 500msec")
}

func Test_ACMEEmail(t *testing.T) {
	cmd := ServerCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: "https://remark.com:443", SharedSecret: "123456"})
	p := flags.NewParser(&cmd, flags.Default)
	args := []string{"--ssl.type=auto"}
	_, err := p.ParseArgs(args)
	require.Nil(t, err)
	cfg, err := cmd.makeSSLConfig()
	require.Nil(t, err)
	assert.Equal(t, "admin@remark.com", cfg.ACMEEmail)

	cmd = ServerCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: "https://remark.com", SharedSecret: "123456"})
	p = flags.NewParser(&cmd, flags.Default)
	args = []string{"--ssl.type=auto", "--ssl.acme-email=adminname@adminhost.com"}
	_, err = p.ParseArgs(args)
	require.Nil(t, err)
	cfg, err = cmd.makeSSLConfig()
	require.Nil(t, err)
	assert.Equal(t, "adminname@adminhost.com", cfg.ACMEEmail)

	cmd = ServerCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: "https://remark.com", SharedSecret: "123456"})
	p = flags.NewParser(&cmd, flags.Default)
	args = []string{"--ssl.type=auto", "--admin.type=shared", "--admin.shared.email=superadmin@admin.com"}
	_, err = p.ParseArgs(args)
	require.Nil(t, err)
	cfg, err = cmd.makeSSLConfig()
	require.Nil(t, err)
	assert.Equal(t, "superadmin@admin.com", cfg.ACMEEmail)

	cmd = ServerCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: "https://remark.com:443", SharedSecret: "123456"})
	p = flags.NewParser(&cmd, flags.Default)
	args = []string{"--ssl.type=auto", "--admin.type=shared"}
	_, err = p.ParseArgs(args)
	require.Nil(t, err)
	cfg, err = cmd.makeSSLConfig()
	require.Nil(t, err)
	assert.Equal(t, "admin@remark.com", cfg.ACMEEmail)
}

func TestServerAuthHooks(t *testing.T) {
	port := rand.Intn(40000) + 10000
	app, ctx := prepServerApp(t, 5*time.Second, func(o ServerCommand) ServerCommand {
		o.Port = port
		return o
	})

	go func() { _ = app.run(ctx) }()
	time.Sleep(150 * time.Millisecond) // let server start

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
	client := http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment", port),
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/p/2018/12/29/podcast-630/", "site": "remark"}}`))
	require.NoError(t, err)
	req.Header.Set("X-JWT", tk)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
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
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "user without aud claim rejected, \n"+tkNoAud+"\n"+string(body))

	// block user dev as admin
	req, e := http.NewRequest(http.MethodPut,
		fmt.Sprintf("http://localhost:%d/api/v1/admin/user/dev?site=remark&block=1&ttl=10d", port), nil)
	assert.Nil(t, e)
	req.SetBasicAuth("admin", "password")
	resp, e = client.Do(req)
	require.Nil(t, e)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "user dev blocked")
	b, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	t.Log(string(b))

	// try add a comment with blocked user
	req, err = http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/v1/comment", port),
		strings.NewReader(`{"text": "test 123 blah", "locator":{"url": "https://radio-t.com/blah1", "site": "remark"}}`))
	require.NoError(t, err)
	req.Header.Set("X-JWT", tk)
	resp, err = client.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized,
		"blocked user can't post, \n"+tk+"\n"+string(body))

	app.Wait()
}

func TestServer_loadEmailTemplate(t *testing.T) {
	cmd := ServerCommand{}
	cmd.Auth.Email.MsgTemplate = "testdata/email.tmpl"
	r := cmd.loadEmailTemplate()
	assert.Equal(t, "The token is {{.Token}}", r)

	cmd.Auth.Email.MsgTemplate = ""
	r = cmd.loadEmailTemplate()
	assert.Contains(t, r, "Remark42</h1>")

	cmd.Auth.Email.MsgTemplate = "bad-file"
	r = cmd.loadEmailTemplate()
	assert.Contains(t, r, "Remark42</h1>")
}

func prepServerApp(t *testing.T, duration time.Duration, fn func(o ServerCommand) ServerCommand) (*serverApp, context.Context) {
	cmd := ServerCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "secret"})

	// prepare options
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--admin-passwd=password", "--site=remark"})
	require.Nil(t, err)
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
	cmd.Notify.Type = "telegram"
	cmd.Notify.Telegram.API = "http://127.0.0.1:12340/"
	cmd.Notify.Telegram.Token = "blah"
	cmd.UpdateLimit = 10
	cmd = fn(cmd)

	os.Remove(cmd.Store.Bolt.Path + "/remark.db")

	// create app
	app, err := cmd.newServerApp()
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(duration, func() {
		log.Print("[TEST] terminate app")
		cancel()
	})
	rand.Seed(time.Now().UnixNano())
	return app, ctx
}
