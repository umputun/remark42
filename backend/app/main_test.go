package main

import (
	"fmt"
	"io"
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
	dir, err := os.MkdirTemp(os.TempDir(), "remark42")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	port := chooseUnusedPort(t)
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

	waitForHTTPServerStart(t, port)
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/ping", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "pong", string(body))
}

func TestMain_WithWebhook(t *testing.T) {
	dir, err := os.MkdirTemp(os.TempDir(), "remark42")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	var webhookSent atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		webhookSent.Store(1)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		b, e := io.ReadAll(r.Body)
		defer r.Body.Close()

		assert.Nil(t, e)
		assert.Equal(t, "Comment: env test", string(b))
	}))
	defer ts.Close()

	port := chooseUnusedPort(t)
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
		close(finished)
	}()

	// defer cleanup because require check below can fail
	defer func() {
		close(done)
		<-finished
	}()

	waitForHTTPServerStart(t, port)

	resp, err := http.Post(fmt.Sprintf("http://admin:password@localhost:%d/api/v1/comment", port), "",
		strings.NewReader(`{"text": "env test", "locator":{"url": "https://radio-t.com", "site": "remark"}}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// wait for webhook to be sent before shutting down
	assert.Eventually(t, func() bool {
		return webhookSent.Load() == int32(1)
	}, 30*time.Second, 10*time.Millisecond, "webhook was not sent")
}

func TestGetDump(t *testing.T) {
	dump := getDump()
	assert.Contains(t, dump, "goroutine")
	assert.Contains(t, dump, "[running]")
	assert.Contains(t, dump, "backend/app/main.go")
	t.Logf("\n dump: %s", dump)
}

// chooseUnusedPort asks the kernel for a free port from the ephemeral range, so concurrently
// running package test binaries never land on the same number
func chooseUnusedPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "no free port available")
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())
	return port
}

// waitForHTTPServerStart blocks until the server on port answers, failing the test naming the
// port if it never does
func waitForHTTPServerStart(t *testing.T, port int) {
	t.Helper()
	client := http.Client{Timeout: time.Second}
	defer client.CloseIdleConnections()
	require.Eventually(t, func() bool {
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d", port))
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return true
	}, 30*time.Second, 10*time.Millisecond, "http server on port %d didn't start", port)
}

func TestMain(m *testing.M) {
	// both ignores are for leaks which are detected locally
	goleak.VerifyTestMain(
		m,
		// the shutdown goroutine in serverApp.run is not joined by Wait, and Rest.Shutdown gives
		// httpServer.Shutdown a second, which can outlast goleak's retry budget on a loaded runner
		goleak.IgnoreTopFunction("net/http.(*Server).Shutdown"),
		goleak.IgnoreTopFunction("github.com/umputun/remark42/backend/app.init.0.func1"),
		// this will be fixed in https://github.com/hashicorp/golang-lru/issues/159
		goleak.IgnoreTopFunction("github.com/hashicorp/golang-lru/v2/expirable.NewLRU[...].func1"),
		// regexp2, pulled in by chroma for syntax highlighting, keeps one shared clock goroutine
		// alive for up to a second after the last match with a timeout, sleeping in 100ms ticks.
		// it ends on its own, but a binary that finishes inside that window is reported as leaking
		goleak.IgnoreAnyFunction("github.com/dlclark/regexp2/v2.runClock"),
	)
}
