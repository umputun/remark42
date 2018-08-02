package main

import (
	"context"
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

func TestApplication(t *testing.T) {
	app, ctx := prepApp(t, 500*time.Millisecond, func(o Opts) Opts {
		o.Port = 18080
		return o
	})

	go func() { _ = app.Run(ctx) }()
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

	assert.Equal(t, "admin@demo.remark42.com", app.restSrv.Authenticator.AdminEmail, "default admin email")

	app.Wait()
}

func TestApplicationDevMode(t *testing.T) {
	app, ctx := prepApp(t, 500*time.Millisecond, func(o Opts) Opts {
		o.Port = 18085
		o.DevPasswd = "password"
		o.Auth.Dev = true
		return o
	})

	go func() { _ = app.Run(ctx) }()
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

func TestApplicationWithMongo(t *testing.T) {

	mongoURL := os.Getenv("MONGO_TEST")
	if mongoURL == "" {
		mongoURL = "mongodb://localhost:27017/test"
	}
	if mongoURL == "skip" {
		t.Skip("skip mongo app test")
	}

	opts := Opts{}
	// prepare options
	p := flags.NewParser(&opts, flags.Default)
	_, err := p.ParseArgs([]string{"--secret=123456", "--dev-passwd=password", "--url=https://demo.remark42.com",
		"--cache.type=mongo", "--store.type=mongo", "--avatar.type=mongo", "--mongo.url=" + mongoURL, "--mongo.db=test_remark", "--port=12345"})
	require.Nil(t, err)
	opts.Auth.Github.CSEC, opts.Auth.Github.CID = "csec", "cid"
	opts.BackupLocation = "/tmp"

	// create app
	app, err := New(opts)
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
	go func() { _ = app.Run(ctx) }()
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

func TestApplicationFailed(t *testing.T) {
	opts := Opts{}
	p := flags.NewParser(&opts, flags.Default)

	// RO bolt location
	_, err := p.ParseArgs([]string{"--secret=123456", "--url=https://demo.remark42.com", "--backup=/tmp",
		"--store.bolt.path=/dev/null"})
	assert.Nil(t, err)
	_, err = New(opts)
	assert.EqualError(t, err, "can't initialize data store: failed to make boltdb for /dev/null/remark.db: "+
		"open /dev/null/remark.db: not a directory")
	t.Log(err)

	// RO backup location
	opts = Opts{}
	_, err = p.ParseArgs([]string{"--secret=123456", "--url=https://demo.remark42.com", "--store.bolt.path=/tmp",
		"--backup=/dev/null/not-writable"})
	assert.Nil(t, err)
	_, err = New(opts)
	assert.EqualError(t, err, "can't check directory status for /dev/null/not-writable: stat /dev/null/not-writable: not a directory")
	t.Log(err)

	// invalid url
	opts = Opts{}
	_, err = p.ParseArgs([]string{"--secret=123456", "--url=demo.remark42.com", "--backup=/tmp", "----store.bolt.path=/tmp"})
	assert.Nil(t, err)
	_, err = New(opts)
	assert.EqualError(t, err, "invalid remark42 url demo.remark42.com")
	t.Log(err)

	opts = Opts{}
	_, err = p.ParseArgs([]string{"--secret=123456", "--url=https://demo.remark42.com", "--backup=/tmp", "--store.type=blah"})
	assert.NotNil(t, err, "blah is invalid type")

	opts.Store.Type = "blah"
	_, err = New(opts)
	assert.EqualError(t, err, "unsupported store type blah")
	t.Log(err)
}

func TestApplicationShutdown(t *testing.T) {
	app, ctx := prepApp(t, 500*time.Millisecond, func(o Opts) Opts {
		o.Port = 18090
		return o
	})
	st := time.Now()
	err := app.Run(ctx)
	assert.Nil(t, err)
	assert.True(t, time.Since(st).Seconds() < 1, "should take about 500msec")
	app.Wait()
}

func TestApplicationMainSignal(t *testing.T) {
	os.Args = []string{"test", "--secret=123456", "--store.bolt.path=/tmp/xyz", "--backup=/tmp", "--avatar.fs.path=/tmp",
		"--port=18100", "--url=https://demo.remark42.com"}

	go func() {
		time.Sleep(100 * time.Millisecond)
		err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.Nil(t, err)
	}()
	st := time.Now()
	main()
	assert.True(t, time.Since(st).Seconds() < 1, "should take about 500msec")
}

func prepApp(t *testing.T, duration time.Duration, fn func(o Opts) Opts) (*Application, context.Context) {
	opts := Opts{}
	// prepare options
	p := flags.NewParser(&opts, flags.Default)
	_, err := p.ParseArgs([]string{"--secret=123456", "--dev-passwd=password", "--url=https://demo.remark42.com"})
	require.Nil(t, err)
	opts.Avatar.FS.Path, opts.Avatar.Type, opts.BackupLocation = "/tmp", "fs", "/tmp"
	opts.Store.Bolt.Path = fmt.Sprintf("/tmp/%d", opts.Port)
	opts.Store.Bolt.Timeout = 10 * time.Second
	opts.Auth.Github.CSEC, opts.Auth.Github.CID = "csec", "cid"
	opts.Auth.Google.CSEC, opts.Auth.Google.CID = "csec", "cid"
	opts.Auth.Facebook.CSEC, opts.Auth.Facebook.CID = "csec", "cid"
	opts.Auth.Yandex.CSEC, opts.Auth.Yandex.CID = "csec", "cid"
	opts.BackupLocation = "/tmp"
	opts = fn(opts)

	os.Remove(opts.Store.Bolt.Path + "/remark.db")

	// create app
	app, err := New(opts)
	require.Nil(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(duration)
		log.Print("[TEST] terminate app")
		cancel()
	}()
	return app, ctx
}
