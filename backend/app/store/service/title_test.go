package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-pkgz/syncs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/safehttp"
)

func TestTitle_GetTitle(t *testing.T) {
	tbl := []struct {
		page  string
		ok    bool
		title string
	}{
		{`<html><title>blah 123</title><body> 2222</body></html>`, true, "blah 123"},
		{`<html><title>blah 123 `, true, "blah 123"},
		{"<html><title>\n\n  blah 123 \n ", true, "blah 123"},
		{`<html><body> 2222</body></html>`, false, ""},
	}

	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{})
	defer ex.Close()
	for i, tt := range tbl {
		t.Run(fmt.Sprintf("check-%d", i), func(t *testing.T) {
			title, ok := ex.getTitle(strings.NewReader(tt.page))
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.title, title)
		})
	}
}

func TestTitle_Get(t *testing.T) {
	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{"127.0.0.1"})
	defer ex.Close()
	var hits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/good" {
			atomic.AddInt32(&hits, 1)
			_, err := w.Write([]byte("<html><title>\n\n blah 123\n</title><body> 2222</body></html>"))
			assert.NoError(t, err)
			return
		}
		w.WriteHeader(404)
	}))
	defer ts.Close()

	title, err := ex.Get(ts.URL + "/good")
	require.NoError(t, err)
	assert.Equal(t, "blah 123", title)

	_, err = ex.Get(ts.URL + "/bad")
	require.Error(t, err)

	for range 100 {
		r, err := ex.Get(ts.URL + "/good")
		require.NoError(t, err)
		assert.Equal(t, "blah 123", r)
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits))
}

func TestTitle_GetConcurrent(t *testing.T) {
	var body strings.Builder
	for range 1000 {
		body.WriteString("something something blah blah\n")
	}
	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{"127.0.0.1"})
	defer ex.Close()
	var hits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.String(), "/good") {
			atomic.AddInt32(&hits, 1)
			_, err := fmt.Fprintf(w, "<html><title>blah 123 %s</title><body>%s</body></html>", r.URL.String(), body.String())
			assert.NoError(t, err)
			return
		}
		w.WriteHeader(404)
	}))
	defer ts.Close()

	g := syncs.NewSizedGroup(10)

	for i := range 100 {
		g.Go(func(_ context.Context) {
			title, err := ex.Get(ts.URL + "/good/" + strconv.Itoa(i))
			require.NoError(t, err)
			assert.Equal(t, "blah 123 "+"/good/"+strconv.Itoa(i), title)
		})
	}
	g.Wait()
	assert.Equal(t, int32(100), atomic.LoadInt32(&hits))
}

func TestTitle_GetFailed(t *testing.T) {
	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{"127.0.0.1"})
	defer ex.Close()
	var hits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(404)
	}))
	defer ts.Close()

	_, err := ex.Get(ts.URL + "/bad")
	require.Error(t, err)

	for range 100 {
		r, err := ex.Get(ts.URL + "/bad")
		require.NoError(t, err)
		assert.Equal(t, "", r)
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "hit once, errors cached")
}

func TestTitle_DoubleClosed(t *testing.T) {
	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{})
	assert.NoError(t, ex.Close())
	// second call should not result in panic
	assert.NoError(t, ex.Close())
}

// TestTitle_GetBlocksPrivateIPViaSafeTransport reproduces the SSRF in TitleExtractor.
// In production (cmd/server.go) the TitleExtractor receives the comment's Locator.URL
// straight from the user JSON body. The domain allowlist alone is not enough — a
// hostname suffix-matching an allowed domain can resolve to a private IP (DNS rebinding)
// or an attacker can list 127.0.0.1 directly when AllowedHosts is empty.
//
// The fix is to wrap the http.Client with safehttp.Transport at construction time,
// matching what the image proxy already does. This test asserts the safehttp transport
// is honored by the title fetcher: even though "127.0.0.1" is in the allowed-domains
// list, the dialer refuses to connect to a private address.
//
// As a control, the second sub-test shows the same setup WITHOUT safehttp.Transport
// happily fetches the page — demonstrating the original vulnerability.
func TestTitle_GetBlocksPrivateIPViaSafeTransport(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><title>secret</title></html>`))
	}))
	defer ts.Close()

	t.Run("with safehttp transport: blocked", func(t *testing.T) {
		client := http.Client{Timeout: 2 * time.Second, Transport: safehttp.Transport()}
		ex := NewTitleExtractor(client, []string{"127.0.0.1"})
		defer ex.Close()
		_, err := ex.Get(ts.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access to private address is not allowed")
	})

	t.Run("control: default transport leaks", func(t *testing.T) {
		client := http.Client{Timeout: 2 * time.Second} // no safehttp.Transport — vulnerable
		ex := NewTitleExtractor(client, []string{"127.0.0.1"})
		defer ex.Close()
		title, err := ex.Get(ts.URL)
		require.NoError(t, err, "without safehttp.Transport the SSRF succeeds — this is the bug")
		assert.Equal(t, "secret", title)
	})
}
