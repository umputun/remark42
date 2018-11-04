package main

import (
	"crypto/tls"
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
		"--avatar.fs.path=/tmp", "--port=18202", "--url=https://demo.remark42.com", "--dbg"}

	go func() {
		time.Sleep(500 * time.Millisecond)
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

	time.Sleep(200 * time.Millisecond) // let server start

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

func TestMain_SSLStaticMode(t *testing.T) {
	os.Args = []string{"test", "server", "--secret=123456", "--store.bolt.path=/tmp/xyz", "--backup=/tmp",
		"--avatar.fs.path=/tmp", "--port=18080", "--url=https://localhost:18443", "--dbg",
		"--ssl.type=static", "--ssl.cert=testdata/cert.pem", "--ssl.key=testdata/key.pem", "--ssl.port=18443"}

	go func() {
		time.Sleep(500 * time.Millisecond)
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

	time.Sleep(200 * time.Millisecond) // let server start

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

	wg.Wait()
}
