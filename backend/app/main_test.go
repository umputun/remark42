package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func Test_Main(t *testing.T) {

	dir, err := ioutil.TempDir(os.TempDir(), "remark42")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	port := chooseRandomUnusedPort()
	os.Args = []string{"test", "server", "--secret=123456", "--store.bolt.path=" + dir, "--backup=/tmp",
		"--avatar.fs.path=" + dir, "--port=" + strconv.Itoa(port), "--url=https://demo.remark42.com", "--dbg", "--notify.type=none"}

	done := make(chan struct{})
	go func() {
		<-done
		e := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.NoError(t, e)
	}()

	finished := make(chan struct{})
	go func() {
		main()
		close(finished)
	}()

	// defer cleanup because require check below can fail
	defer func() {
		close(done)
		<-finished
	}()

	waitForHTTPServerStart(port)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/ping", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "pong", string(body))
}

func TestMain_WithWebhook(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "remark42")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	var webhookSent int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.StoreInt32(&webhookSent, 1)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		b, e := ioutil.ReadAll(r.Body)
		defer r.Body.Close()

		assert.Nil(t, e)
		assert.Equal(t, "Comment: env test", string(b))
	}))
	defer ts.Close()

	port := chooseRandomUnusedPort()
	os.Args = []string{"test", "server", "--secret=123456", "--store.bolt.path=" + dir, "--backup=/tmp",
		"--avatar.fs.path=" + dir, "--port=" + strconv.Itoa(port), "--url=https://demo.remark42.com", "--dbg",
		"--admin-passwd=password", "--site=remark", "--notify.admins=webhook"}

	err = os.Setenv("NOTIFY_WEBHOOK_URL", ts.URL)
	assert.NoError(t, err)
	err = os.Setenv("NOTIFY_WEBHOOK_TEMPLATE", "Comment: {{.Orig}}")
	assert.NoError(t, err)
	err = os.Setenv("NOTIFY_WEBHOOK_HEADERS", "Content-Type:application/json")
	assert.NoError(t, err)

	done := make(chan struct{})
	go func() {
		<-done
		e := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.NoError(t, e)
	}()

	finished := make(chan struct{})
	go func() {
		main()
		assert.Eventually(t, func() bool {
			return atomic.LoadInt32(&webhookSent) == int32(1)
		}, time.Second, 100*time.Millisecond, "webhook was not sent")
		close(finished)
	}()

	// defer cleanup because require check below can fail
	defer func() {
		close(done)
		<-finished
	}()

	waitForHTTPServerStart(port)

	resp, err := http.Post(fmt.Sprintf("http://admin:password@localhost:%d/api/v1/comment", port), "",
		strings.NewReader(`{"text": "env test", "locator":{"url": "https://radio-t.com", "site": "remark"}}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 201, resp.StatusCode)
}

func TestGetDump(t *testing.T) {
	dump := getDump()
	assert.True(t, strings.Contains(dump, "goroutine"))
	assert.True(t, strings.Contains(dump, "[running]"))
	assert.True(t, strings.Contains(dump, "backend/app/main.go"))
	t.Logf("\n dump: %s", dump)
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
	// wait for up to 10 seconds for server to start before returning it
	client := http.Client{Timeout: time.Second}
	for i := 0; i < 100; i++ {
		time.Sleep(time.Millisecond * 100)
		if resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port)); err == nil {
			_ = resp.Body.Close()
			return
		}
	}
}

func TestMain(m *testing.M) {
	// both ignores are for leaks which are detected locally
	goleak.VerifyTestMain(
		m,
		goleak.IgnoreTopFunction("github.com/umputun/remark42/backend/app.init.0.func1"),
		goleak.IgnoreTopFunction("net/http.(*Server).Shutdown"),
	)
}
