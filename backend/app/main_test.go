package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(t *testing.T) {

	os.Args = []string{"test", "server", "--secret=123456", "--store.bolt.path=/tmp/xyz", "--backup=/tmp",
		"--avatar.fs.path=/tmp", "--port=18200", "--url=https://demo.remark42.com"}

	go func() {
		time.Sleep(100 * time.Millisecond)
		err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.Nil(t, err)
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		st := time.Now()
		main()
		assert.True(t, time.Since(st).Seconds() < 1, "should take about 500msec")
		wg.Done()
	}()

	// send ping
	resp, err := http.Get("http://localhost:18200/api/v1/ping")
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "pong", string(body))

	wg.Wait()
}
