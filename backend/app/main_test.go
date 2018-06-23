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

	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplication(t *testing.T) {
	app, ctx := prepApp(t, 18080, 500*time.Millisecond)
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

func TestApplicationFailed(t *testing.T) {
	opts := Opts{}
	p := flags.NewParser(&opts, flags.Default)

	// RO bolt location
	_, err := p.ParseArgs([]string{"--secret=123456", "--url=https://demo.remark42.com", "--store.bolt.path=/dev/null"})
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
	_, err = p.ParseArgs([]string{"--secret=123456", "--url=demo.remark42.com", "----store.bolt.path=/tmp"})
	assert.Nil(t, err)
	_, err = New(opts)
	assert.EqualError(t, err, "invalid remark42 url demo.remark42.com")
	t.Log(err)

	opts = Opts{}
	_, err = p.ParseArgs([]string{"--secret=123456", "--url=https://demo.remark42.com", "--store.type=mongo"})
	assert.Nil(t, err)
	_, err = New(opts)
	assert.EqualError(t, err, "unsupported store type mongo")
	t.Log(err)
}

func TestApplicationShutdown(t *testing.T) {
	app, ctx := prepApp(t, 18090, 500*time.Millisecond)
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

func prepApp(t *testing.T, port int, duration time.Duration) (*Application, context.Context) {
	// prepare options
	opts := Opts{}
	p := flags.NewParser(&opts, flags.Default)
	_, err := p.ParseArgs([]string{"--secret=123456", "--dev-passwd=password", "--url=https://demo.remark42.com"})
	require.Nil(t, err)
	opts.Avatar.FS.Path, opts.Avatar.Type, opts.BackupLocation = "/tmp", "fs", "/tmp"
	opts.Store.Bolt.Path = fmt.Sprintf("/tmp/%d", port)
	opts.Store.Bolt.Timeout = 10 * time.Second
	opts.Auth.Github.CSEC, opts.Auth.Github.CID = "csec", "cid"
	opts.Auth.Google.CSEC, opts.Auth.Google.CID = "csec", "cid"
	opts.Auth.Facebook.CSEC, opts.Auth.Facebook.CID = "csec", "cid"
	opts.Auth.Yandex.CSEC, opts.Auth.Yandex.CID = "csec", "cid"
	opts.Port = port
	opts.BackupLocation = "/tmp"

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
