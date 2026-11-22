package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
		{`<html><head><title></title></head><body> 2222</body></html>`, false, ""},
		{"<html><title>" + strings.Repeat("a", teMaxTitleLen+100) + "</title></html>", true, strings.Repeat("a", teMaxTitleLen)},
		{"<html><title>" + strings.Repeat("ы", teMaxTitleLen+100) + "</title></html>", true, strings.Repeat("ы", teMaxTitleLen)},
		{"<html><title>" + strings.Repeat("a", teMaxTitleLen-1) + " bbb</title></html>", true, strings.Repeat("a", teMaxTitleLen-1)},
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
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/good" {
			hits.Add(1)
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
	assert.Equal(t, int32(1), hits.Load())
}

func TestTitle_GetConcurrent(t *testing.T) {
	var body strings.Builder
	for range 1000 {
		body.WriteString("something something blah blah\n")
	}
	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{"127.0.0.1"})
	defer ex.Close()
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.String(), "/good") {
			hits.Add(1)
			_, err := fmt.Fprintf(w, "<html><title>blah 123 %s</title><body>%s</body></html>", r.URL.String(), body.String())
			assert.NoError(t, err)
			return
		}
		w.WriteHeader(404)
	}))
	defer ts.Close()

	var wg sync.WaitGroup
	for w := range 10 { // 10 workers, 10 pages each
		wg.Go(func() {
			for i := w * 10; i < w*10+10; i++ {
				title, err := ex.Get(ts.URL + "/good/" + strconv.Itoa(i))
				assert.NoError(t, err)
				assert.Equal(t, "blah 123 "+"/good/"+strconv.Itoa(i), title)
			}
		})
	}
	wg.Wait()
	assert.Equal(t, int32(100), hits.Load())
}

func TestTitle_CachesLongURL(t *testing.T) {
	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{})
	defer ex.Close()

	longURL := "https://example.com/" + strings.Repeat("a", 2*teCacheMaxRecs)
	_, err := ex.cache.Get(longURL, func() (string, error) { return "title", nil })
	require.NoError(t, err)
	assert.Equal(t, 1, ex.cache.Stat().Keys, "long url is cached")
}

func TestTitle_GetFailed(t *testing.T) {
	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{"127.0.0.1"})
	defer ex.Close()
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(404)
	}))
	defer ts.Close()

	_, err := ex.Get(ts.URL + "/bad")
	require.Error(t, err)

	for range 100 {
		r, err := ex.Get(ts.URL + "/bad")
		require.Error(t, err, "a cached failure is still a failure")
		assert.Equal(t, "", r)
	}
	assert.Equal(t, int32(1), hits.Load(), "hit once, errors cached")
}

func TestTitle_GetBlankTitle(t *testing.T) {
	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{"127.0.0.1"})
	defer ex.Close()
	var hits atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, err := w.Write([]byte("<html><title> \n </title><body> 2222</body></html>"))
		assert.NoError(t, err)
	}))
	defer ts.Close()

	for range 10 {
		r, err := ex.Get(ts.URL + "/blank")
		require.Error(t, err, "a blank title is no title")
		assert.Equal(t, "", r)
	}
	assert.Equal(t, int32(1), hits.Load(), "hit once, blank title cached")
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

func TestTitle_GetLimited(t *testing.T) {
	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{"127.0.0.1"})
	defer ex.Close()
	// filler returns a page start of the given size, ending right before the title element
	filler := func(size int) string {
		const open, end = "<html><!--", "-->"
		return open + strings.Repeat("a", size-len(open)-len(end)) + end
	}
	tail := "<body>" + strings.Repeat("b", teMaxPageSize) + "</body></html>" // makes any page longer than the limit
	pages := map[string]string{
		"/long-page":     "<html><title>blah 123</title>" + tail,
		"/long-title":    "<html><title>" + strings.Repeat("a", 2*teMaxPageSize) + "</title></html>",
		"/late-title":    filler(teMaxPageSize) + "<title>blah 123</title></html>",
		"/padded":        "<html><title>" + strings.Repeat(" ", teMaxPageSize) + "blah 123</title></html>",
		"/ends-at-limit": filler(teMaxPageSize-len("<title>blah 123</title>")) + "<title>blah 123</title>" + tail,
		// the limit falls inside a two-byte letter of the title, and after its first byte
		"/cut-letter":       filler(teMaxPageSize-len("<title>")-11) + "<title>" + strings.Repeat("ы", 100) + "</title>" + tail,
		"/cut-first-letter": filler(teMaxPageSize-len("<title>")-1) + "<title>" + strings.Repeat("ы", 100) + "</title>" + tail,
		// the limit falls inside the closing tag, of a title with text and of an empty one
		"/cut-closing-tag":       filler(teMaxPageSize-len("<title>blah 123</ti")) + "<title>blah 123</title>" + tail,
		"/cut-closing-tag-empty": filler(teMaxPageSize-len("<title></ti")) + "<title></title>" + tail,
		// a closing tag inside the limit that closes no title, here in a comment, does not let a cut title through
		"/cut-after-commented-tag": "<html><!-- </title> -->" + filler(teMaxPageSize - len("<html><!-- </title> -->") - len("<title>") - 11)[len("<html>"):] +
			"<title>" + strings.Repeat("ы", 100) + "</title>" + tail,
		// closing tags as the html parser also accepts them
		"/end-tag-with-slash":     "<html><title>blah 123</title/>" + tail,
		"/end-tag-with-attribute": "<html><title>blah 123</title x>" + tail,
		// titles in a single-byte encoding come back byte for byte, on a long page too
		"/latin1": "<html><title>Caf\xe9</title>" + tail,
		"/cp1251": "<html><title>\xcd\xee\xe2\xee\xf1\xf2\xe8</TITLE >" + tail,
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(pages[r.URL.Path])) // the client stops reading at the limit, so the write may fail
	}))
	defer ts.Close()

	tbl := []struct {
		path, title string
	}{
		{"/long-page", "blah 123"},
		{"/long-title", ""},
		{"/late-title", ""},
		{"/padded", ""},
		{"/ends-at-limit", "blah 123"},
		{"/cut-letter", ""},
		{"/cut-first-letter", ""},
		{"/cut-closing-tag", ""},
		{"/cut-closing-tag-empty", ""},
		{"/cut-after-commented-tag", ""},
		{"/end-tag-with-slash", "blah 123"},
		{"/end-tag-with-attribute", "blah 123"},
		{"/latin1", "Caf\xe9"},
		{"/cp1251", "\xcd\xee\xe2\xee\xf1\xf2\xe8"},
	}
	for _, tt := range tbl {
		t.Run(tt.path, func(t *testing.T) {
			require.Greater(t, len(pages[tt.path]), teMaxPageSize, "the page is longer than the limit")
			title, err := ex.Get(ts.URL + tt.path)
			if tt.title == "" {
				require.Error(t, err, "a title that does not end inside the limit is no title")
				assert.Empty(t, title)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.title, title)
		})
	}
}

func TestTitle_readHeadStopsAtLimit(t *testing.T) {
	body := strings.NewReader("<html><title>blah 123</title>" + strings.Repeat("b", 3*teMaxPageSize))
	size := body.Len()
	page, err := readHead(body)
	require.NoError(t, err)
	assert.Equal(t, "<html><title>blah 123</title>", string(page), "the page is cut after its closing title tag")
	// one byte past the limit is read, to tell a page at the limit from a longer one
	assert.Equal(t, teMaxPageSize+1, size-body.Len(), "nothing past the limit is read")

	body = strings.NewReader("<html><title>short page</title></html>")
	page, err = readHead(body)
	require.NoError(t, err)
	assert.Equal(t, "<html><title>short page</title></html>", string(page), "a page under the limit is left whole")
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTitle_GetAllowedDomains(t *testing.T) {
	tbl := []struct {
		url     string
		allowed bool
	}{
		{"http://example.com/post", true},
		{"http://www.example.com/post", true},
		{"http://a.b.example.com:8080/post", true},
		{"http://EXAMPLE.com/post", true}, // names are compared without regard to case
		{"http://Www.Example.COM/post", true},
		{"http://NOTexample.com/post", false},
		{"http://RADIO-T.com/post", true},
		{"http://rad\u0130o-t.com/post", false}, // a dotted capital I is not an ASCII letter and makes another name
		{"http://radio-t.\u212aom/post", false}, // nor is the Kelvin sign
		{"http://127.0.0.1:8080/post", true},
		{"http://[2001:db8::1]:8080/post", true},
		{"http://notexample.com/post", false},
		{"http://example.com.evil.org/post", false},
		{"http://10.0.0.1/post", false},
		{"http://[2001:db8::2]/post", false},
		// the zone of an IPv6 address is free text and is not a subdomain of an allowed domain
		{"http://[2001:db8::2%25.example.com]:9000/post", false},
		{"http://[2001:db8::2%25example.com]/post", false},
		{"http://[2001:db8::2%25.127.0.0.1]/post", false},
		{"http://[2001:db8::1%25.example.com]/post", false},
		{"http://[::ffff:10.0.0.1%25.example.com]/post", false},
	}

	var hits atomic.Int32
	client := http.Client{Timeout: 5 * time.Second, Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		hits.Add(1)
		rec := httptest.NewRecorder()
		_, _ = rec.WriteString("<html><title>blah 123</title></html>")
		return rec.Result(), nil
	})}
	ex := NewTitleExtractor(client, []string{"example.com", "radio-t.com", "radio-t.kom", "127.0.0.1", "2001:db8::1"})
	defer ex.Close()

	for _, tt := range tbl {
		t.Run(tt.url, func(t *testing.T) {
			hits.Store(0)
			title, err := ex.Get(tt.url)
			if !tt.allowed {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "is not allowed")
				assert.Equal(t, int32(0), hits.Load(), "no request to a domain which is not allowed")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "blah 123", title)
			assert.Equal(t, int32(1), hits.Load())
		})
	}
}

func TestTitle_GetRedirect(t *testing.T) {
	var targetHits atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetHits.Add(1)
		_, _ = w.Write([]byte(`<html><title>target</title></html>`))
	}))
	defer target.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/loop" {
			http.Redirect(w, r, "/loop", http.StatusFound)
			return
		}
		http.Redirect(w, r, r.URL.Query().Get("to"), http.StatusFound)
	}))
	defer ts.Close()

	// both servers listen on 127.0.0.1, "localhost" reaches the same server under a name which is not allowed
	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second}, []string{"127.0.0.1"})
	defer ex.Close()

	t.Run("to allowed domain", func(t *testing.T) {
		targetHits.Store(0)
		title, err := ex.Get(ts.URL + "/out?to=" + target.URL + "/allowed")
		require.NoError(t, err)
		assert.Equal(t, "target", title)
		assert.Equal(t, int32(1), targetHits.Load())
	})

	t.Run("to domain which is not allowed", func(t *testing.T) {
		targetHits.Store(0)
		notAllowed := strings.Replace(target.URL, "127.0.0.1", "localhost", 1)
		title, err := ex.Get(ts.URL + "/out?to=" + notAllowed + "/not-allowed")
		assert.Equal(t, int32(0), targetHits.Load(), "no request to the redirect target")
		assert.Empty(t, title)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "redirect to domain localhost")
		assert.Contains(t, err.Error(), "is not allowed")
	})

	t.Run("too many redirects", func(t *testing.T) {
		_, err := ex.Get(ts.URL + "/loop")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "stopped after 10 redirects")
	})
}
