package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/repeater"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Main(t *testing.T) {

	dir, err := ioutil.TempDir(os.TempDir(), "remark42")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	os.Args = []string{"test", "server", "--secret=123456", "--store.bolt.path=" + dir, "--backup=/tmp",
		"--avatar.fs.path=" + dir, "--port=18222", "--url=https://demo.remark42.com", "--dbg", "--notify.type=none"}

	go func() {
		time.Sleep(5000 * time.Millisecond)
		e := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.Nil(t, e)
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		st := time.Now()
		main()
		assert.True(t, time.Since(st).Seconds() >= 4, "should take about 5s, took %s", time.Since(st))
		wg.Done()
	}()

	var passed bool
	err = repeater.NewDefault(10, time.Millisecond*1000).Do(context.Background(), func() error {
		resp, e := http.Get("http://localhost:18222/api/v1/ping")
		if e != nil {
			t.Logf("%+v", e)
			return e
		}
		require.Nil(t, e)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
		body, e := ioutil.ReadAll(resp.Body)
		assert.Nil(t, e)
		assert.Equal(t, "pong", string(body))
		passed = true
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, true, passed, "at least on ping passed")

	wg.Wait()
}

func TestGetDump(t *testing.T) {
	dump := getDump()
	assert.True(t, strings.Contains(dump, "goroutine"))
	assert.True(t, strings.Contains(dump, "[running]"))
	assert.True(t, strings.Contains(dump, "backend/app/main.go"))
	log.Printf("\n dump: %s", dump)
}
