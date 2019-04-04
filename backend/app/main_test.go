package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Main(t *testing.T) {

	os.Args = []string{"test", "server", "--secret=123456", "--store.bolt.path=/tmp/xyz", "--backup=/tmp",
		"--avatar.fs.path=/tmp", "--port=18202", "--url=https://demo.remark42.com", "--dbg", "--notify.type=none"}

	go func() {
		time.Sleep(1000 * time.Millisecond)
		err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.Nil(t, err)
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		st := time.Now()
		main()
		assert.True(t, time.Since(st).Seconds() < 2, "should take under 1s")
		wg.Done()
	}()

	time.Sleep(500 * time.Millisecond) // let server start

	// send ping
	resp, err := http.Get("http://localhost:18202/api/v1/ping")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "pong", string(body))

	wg.Wait()
}

func TestGetDump(t *testing.T) {
	dump := getDump()
	assert.True(t, strings.Contains(dump, "goroutine"))
	assert.True(t, strings.Contains(dump, "[running]"))
	assert.True(t, strings.Contains(dump, "backend/app/main.go"))
	log.Printf("\n dump: %s", dump)
}
