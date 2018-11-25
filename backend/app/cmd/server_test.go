package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/globalsign/mgo"
	"github.com/go-pkgz/mongo"
	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerApp(t *testing.T) {
	app, ctx := prepServerApp(t, 500*time.Millisecond, func(o ServerCommand) ServerCommand {
		o.Port = 18080
		return o
	})

	go func() { _ = app.run(ctx) }()
	time.Sleep(100 * time.Millisecond) // let server start

	// send ping
	resp, err := http.Get("http://localhost:18080/api/v1/ping")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "pong", string(body))

	// add comment
	resp, err = http.Post("http://dev:password@localhost:18080/api/v1/comment", "json",
		strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "remark"}}`))
	require.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	body, _ = ioutil.ReadAll(resp.Body)
	t.Log(string(body))

	assert.Equal(t, "admin@demo.remark42.com", app.dataService.AdminStore.Email(""), "default admin email")

	app.Wait()
}

func TestServerApp_DevMode(t *testing.T) {
	app, ctx := prepServerApp(t, 500*time.Millisecond, func(o ServerCommand) ServerCommand {
		o.Port = 18085
		o.DevPasswd = "password"
		o.Auth.Dev = true
		return o
	})

	go func() { _ = app.run(ctx) }()
	time.Sleep(100 * time.Millisecond) // let server start

	assert.Equal(t, 4+1, len(app.restSrv.Authenticator.Providers), "extra auth provider")
	assert.Equal(t, "dev", app.restSrv.Authenticator.Providers[4].Name, "dev auth provider")
	// send ping
	resp, err := http.Get("http://localhost:18085/api/v1/ping")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "pong", string(body))

	app.Wait()
}

func TestServerApp_WithMongo(t *testing.T) {

	mongoURL := os.Getenv("MONGO_TEST")
	if mongoURL == "" {
		mongoURL = "mongodb://localhost:27017/test"
	}
	if mongoURL == "skip" {
		t.Skip("skip mongo app test")
	}

	opts := ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	// prepare options
	p := flags.NewParser(&opts, flags.Default)
	_, err := p.ParseArgs([]string{"--dev-passwd=password", "--cache.type=mongo", "--store.type=mongo",
		"--avatar.type=mongo", "--mongo.url=" + mongoURL, "--mongo.db=test_remark", "--port=12345", "--admin.type=mongo"})
	require.Nil(t, err)
	opts.Auth.Github.CSEC, opts.Auth.Github.CID = "csec", "cid"
	opts.BackupLocation = "/tmp"

	// create app
	app, err := opts.newServerApp()
	require.Nil(t, err)

	defer func() {
		s, err := mongo.NewServerWithURL(mongoURL, 10*time.Second)
		assert.NoError(t, err)
		conn := mongo.NewConnection(s, "test_remark", "")
		_ = conn.WithDB(func(dbase *mgo.Database) error {
			assert.NoError(t, dbase.DropDatabase())
			return nil
		})
	}()

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

func TestServerApp_WithSSL(t *testing.T) {
	opts := ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://localhost:18443", SharedSecret: "123456"})

	// prepare options
	p := flags.NewParser(&opts, flags.Default)
	_, err := p.ParseArgs([]string{"--dev-passwd=password", "--port=18080", "--store.bolt.path=/tmp/xyz", "--backup=/tmp", "--avatar.type=bolt", "--avatar.bolt.file=/tmp/ava-test.db", "--notify.type=none",
		"--ssl.type=static", "--ssl.cert=testdata/cert.pem", "--ssl.key=testdata/key.pem", "--ssl.port=18443"})
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

func TestServerApp_Failed(t *testing.T) {
	opts := ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	p := flags.NewParser(&opts, flags.Default)

	// RO bolt location
	_, err := p.ParseArgs([]string{"--backup=/tmp", "--store.bolt.path=/dev/null"})
	assert.Nil(t, err)
	_, err = opts.newServerApp()
	assert.EqualError(t, err, "failed to make data store engine: can't initialize data store: failed to make boltdb for /dev/null/remark.db: "+
		"open /dev/null/remark.db: not a directory")
	t.Log(err)

	// RO backup location
	opts = ServerCommand{}
	opts.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	_, err = p.ParseArgs([]string{"--store.bolt.path=/tmp", "--backup=/dev/null/not-writable"})
	assert.Nil(t, err)
	_, err = opts.newServerApp()
	assert.EqualError(t, err, "can't check directory status for /dev/null/not-writable: stat /dev/null/not-writable: not a directory")
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
		o.Port = 18090
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
		"--avatar.bolt.file=/tmp/ava-test.db", "--port=18100", "--notify.type=none"}
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

func prepServerApp(t *testing.T, duration time.Duration, fn func(o ServerCommand) ServerCommand) (*serverApp, context.Context) {
	cmd := ServerCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: "https://demo.remark42.com", SharedSecret: "123456"})

	// prepare options
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--dev-passwd=password"})
	require.Nil(t, err)
	cmd.Avatar.FS.Path, cmd.Avatar.Type, cmd.BackupLocation = "/tmp", "fs", "/tmp"
	cmd.Store.Bolt.Path = fmt.Sprintf("/tmp/%d", cmd.Port)
	cmd.Store.Bolt.Timeout = 10 * time.Second
	cmd.Auth.Github.CSEC, cmd.Auth.Github.CID = "csec", "cid"
	cmd.Auth.Google.CSEC, cmd.Auth.Google.CID = "csec", "cid"
	cmd.Auth.Facebook.CSEC, cmd.Auth.Facebook.CID = "csec", "cid"
	cmd.Auth.Yandex.CSEC, cmd.Auth.Yandex.CID = "csec", "cid"
	cmd.BackupLocation = "/tmp"
	cmd.Notify.Type = "telegram"
	cmd.Notify.Telegram.API = "http://127.0.0.1:12340/"
	cmd.Notify.Telegram.Token = "blah"
	cmd = fn(cmd)

	os.Remove(cmd.Store.Bolt.Path + "/remark.db")

	// create app
	app, err := cmd.newServerApp()
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(duration)
		log.Print("[TEST] terminate app")
		cancel()
	}()
	return app, ctx
}
