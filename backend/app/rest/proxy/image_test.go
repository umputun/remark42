package proxy

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPicture_Extract(t *testing.T) {

	tbl := []struct {
		inp string
		res []string
	}{
		{
			`<p> blah <img src="http://radio-t.com/img.png"/> test</p>`,
			[]string{"http://radio-t.com/img.png"},
		},
		{
			`<p> blah <img src="https://radio-t.com/img.png"/> test</p>`,
			[]string{},
		},
		{
			`<img src="http://radio-t.com/img2.png"/>`,
			[]string{"http://radio-t.com/img2.png"},
		},
		{
			`<img src="http://radio-t.com/img3.png"/> <div>xyz <img src="http://images.pexels.com/67636/img4.jpeg"> </div>`,
			[]string{"http://radio-t.com/img3.png", "http://images.pexels.com/67636/img4.jpeg"},
		},
		{
			`<img src="https://radio-t.com/img3.png"/> <div>xyz <img src="http://images.pexels.com/67636/img4.jpeg"> </div>`,
			[]string{"http://images.pexels.com/67636/img4.jpeg"},
		},
		{
			`abcd <b>blah</b> <h1>xxx</h1>`,
			[]string{},
		},
	}
	img := Image{HTTP2HTTPS: true}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res, err := img.extract(tt.inp, func(src string) bool { return strings.HasPrefix(src, "http://") })
			assert.NoError(t, err)
			assert.Equal(t, tt.res, res)
		})
	}
}

func TestPicture_Replace(t *testing.T) {
	img := Image{HTTP2HTTPS: true, RoutePath: "/img"}
	r := img.replace(`<img src="http://radio-t.com/img3.png"/> xyz <img src="http://images.pexels.com/67636/img4.jpeg">`,
		[]string{"http://radio-t.com/img3.png", "http://images.pexels.com/67636/img4.jpeg"})
	assert.Equal(t, `<img src="/img?src=aHR0cDovL3JhZGlvLXQuY29tL2ltZzMucG5n"/> xyz <img src="/img?src=aHR0cDovL2ltYWdlcy5wZXhlbHMuY29tLzY3NjM2L2ltZzQuanBlZw==">`, r)
}

func TestImage_Routes(t *testing.T) {
	img := Image{HTTP2HTTPS: true, RemarkURL: "https://demo.remark42.com", RoutePath: "/api/v1/proxy"}

	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()
	httpSrv := imgHTTPServer(t)
	defer httpSrv.Close()

	encodedImgURL := base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "/image/img1.png"))

	resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	t.Logf("%+v", resp.Header)
	assert.Equal(t, "123", resp.Header["Content-Length"][0])
	assert.Equal(t, "image/*", resp.Header["Content-Type"][0])

	encodedImgURL = base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "/image/no-such-image.png"))
	resp, err = http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	encodedImgURL = base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "bad encoding"))
	resp, err = http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestImage_RoutesTimedOut(t *testing.T) {
	img := Image{HTTP2HTTPS: true, RemarkURL: "https://demo.remark42.com", RoutePath: "/api/v1/proxy", Timeout: 50 * time.Millisecond}

	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()
	httpSrv := imgHTTPServer(t)
	defer httpSrv.Close()

	encodedImgURL := base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "/image/img-slow.png"))
	resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	t.Log(string(b))
	assert.True(t, strings.Contains(string(b), "deadline exceeded"))
}

func TestPicture_Convert(t *testing.T) {
	img := Image{HTTP2HTTPS: true, RoutePath: "/img"}
	r := img.Convert(`<img src="http://radio-t.com/img3.png"/> xyz <img src="http://images.pexels.com/67636/img4.jpeg">`, "userID")
	assert.Equal(t, `<img src="/img?src=aHR0cDovL3JhZGlvLXQuY29tL2ltZzMucG5n"/> xyz <img src="/img?src=aHR0cDovL2ltYWdlcy5wZXhlbHMuY29tLzY3NjM2L2ltZzQuanBlZw==">`, r)

	r = img.Convert(`<img src="https://radio-t.com/img3.png"/> xyz <img src="http://images.pexels.com/67636/img4.jpeg">`, "userID")
	assert.Equal(t, `<img src="https://radio-t.com/img3.png"/> xyz <img src="/img?src=aHR0cDovL2ltYWdlcy5wZXhlbHMuY29tLzY3NjM2L2ltZzQuanBlZw==">`, r)

	img = Image{HTTP2HTTPS: true, RoutePath: "/img", RemarkURL: "http://example.com"}
	r = img.Convert(`<img src="http://radio-t.com/img3.png"/> xyz`, "userID")
	assert.Equal(t, `<img src="http://radio-t.com/img3.png"/> xyz`, r, "http:// remark url, no proxy")

	img = Image{HTTP2HTTPS: false, RoutePath: "/img"}
	r = img.Convert(`<img src="http://radio-t.com/img3.png"/> xyz`, "userID")
	assert.Equal(t, `<img src="http://radio-t.com/img3.png"/> xyz`, r, "disabled, no proxy")
}

func imgHTTPServer(t *testing.T) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/image/img1.png" {
			t.Log("http img request", r.URL)
			w.Header().Add("Content-Length", "123")
			w.Header().Add("Content-Type", "image/png")
			_, err := w.Write([]byte(fmt.Sprintf("%123s", "X")))
			assert.NoError(t, err)
			return
		}
		if r.URL.Path == "/image/img-slow.png" {
			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(500)
			return
		}
		t.Log("http img request - not found", r.URL)
		w.WriteHeader(404)
	}))

	return ts
}
