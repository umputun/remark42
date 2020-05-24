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

	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second})
	defer ex.Close()
	for i, tt := range tbl {
		tt := tt
		t.Run(fmt.Sprintf("check-%d", i), func(t *testing.T) {
			title, ok := ex.getTitle(strings.NewReader(tt.page))
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.title, title)
		})
	}
}

func TestTitle_Get(t *testing.T) {
	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second})
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

	for i := 0; i < 100; i++ {
		r, err := ex.Get(ts.URL + "/good")
		require.NoError(t, err)
		assert.Equal(t, "blah 123", r)
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits))
}

func TestTitle_GetConcurrent(t *testing.T) {
	body := ""
	for n := 0; n < 1000; n++ {
		body += "something something blah blah\n"
	}
	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second})
	defer ex.Close()
	var hits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.String(), "/good") {
			atomic.AddInt32(&hits, 1)
			_, err := w.Write([]byte(fmt.Sprintf("<html><title>blah 123 %s</title><body>%s</body></html>", r.URL.String(), body)))
			assert.NoError(t, err)
			return
		}
		w.WriteHeader(404)
	}))
	defer ts.Close()

	g := syncs.NewSizedGroup(10)

	for i := 0; i < 100; i++ {
		ii := i
		g.Go(func(_ context.Context) {
			title, err := ex.Get(ts.URL + "/good/" + strconv.Itoa(ii))
			require.NoError(t, err)
			assert.Equal(t, "blah 123 "+"/good/"+strconv.Itoa(ii), title)
		})
	}
	g.Wait()
	assert.Equal(t, int32(100), atomic.LoadInt32(&hits))
}

func TestTitle_GetFailed(t *testing.T) {
	ex := NewTitleExtractor(http.Client{Timeout: 5 * time.Second})
	defer ex.Close()
	var hits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(404)
	}))
	defer ts.Close()

	_, err := ex.Get(ts.URL + "/bad")
	require.Error(t, err)

	for i := 0; i < 100; i++ {
		r, err := ex.Get(ts.URL + "/bad")
		require.NoError(t, err)
		assert.Equal(t, "", r)
	}
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "hit once, errors cached")
}
