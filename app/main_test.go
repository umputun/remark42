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

	flags "github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplication(t *testing.T) {
	app, ctx := prepApp(t, 18080, 500*time.Millisecond)
	go app.Run(ctx)
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
	p.ParseArgs([]string{"--secret=123456", "--url=https://demo.remark42.com", "--bolt=/dev/null"})
	_, err := New(opts)
	assert.EqualError(t, err, "can't initialize data store: failed to make boltdb for /dev/null/remark.db: "+
		"open /dev/null/remark.db: not a directory")
	t.Log(err)

	//p.ParseArgs([]string{"--secret=123456", "--url=https://demo.remark42.com", "--bolt=/tmp", "--backup=/not-writable"})
	//_, err = New(opts)
	//assert.EqualError(t, err, "can't initialize data store: failed to make boltdb for /dev/null/remark.db: "+
	//	"open /dev/null/remark.db: not a directory")
	//t.Log(err)
}

func TestApplicationShutdown(t *testing.T) {
	app, ctx := prepApp(t, 18090, 500*time.Millisecond)
	st := time.Now()
	app.Run(ctx)
	assert.True(t, time.Since(st).Seconds() < 1, "should take about 500msec")
	app.Wait()
}

func TestApplicationMainSignal(t *testing.T) {
	os.Args = []string{"test", "--secret=123456", "--bolt=/tmp/xyz", "--backup=/tmp", "--avatars=/tmp",
		"--port=18100", "--url=https://demo.remark42.com"}

	go func() {
		time.Sleep(100 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()
	st := time.Now()
	main()
	assert.True(t, time.Since(st).Seconds() < 1, "should take about 500msec")
}

func prepApp(t *testing.T, port int, duration time.Duration) (*Application, context.Context) {
	// prepare options
	opts := Opts{}
	p := flags.NewParser(&opts, flags.Default)
	p.ParseArgs([]string{"--secret=123456", "--dev-passwd=password", "--url=https://demo.remark42.com"})
	opts.AvatarStore, opts.BackupLocation = "/tmp", "/tmp"
	opts.BoltPath = fmt.Sprintf("/tmp/%d", port)
	opts.GithubCSEC, opts.GithubCID = "csec", "cid"
	opts.GoogleCSEC, opts.GoogleCID = "csec", "cid"
	opts.FacebookCSEC, opts.FacebookCID = "csec", "cid"
	opts.YandexCSEC, opts.YandexCID = "csec", "cid"
	opts.Port = port

	os.Remove(opts.BoltPath + "/remark.db")

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
