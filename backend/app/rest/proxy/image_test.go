package proxy

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store/image"
)

// gopher png for test, from https://golang.org/src/image/png/example_test.go
const gopher = "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"

func gopherPNG() io.Reader { return base64.NewDecoder(base64.StdEncoding, strings.NewReader(gopher)) }
func gopherPNGBytes() []byte {
	img, _ := ioutil.ReadAll(gopherPNG())
	return img
}

func TestImage_Extract(t *testing.T) {
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
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res, err := img.extract(tt.inp, func(src string) bool { return strings.HasPrefix(src, "http://") })
			assert.NoError(t, err)
			assert.Equal(t, tt.res, res)
		})
	}
}

func TestImage_Replace(t *testing.T) {
	img := Image{HTTP2HTTPS: true, RoutePath: "/img"}
	r := img.replace(`<img src="http://radio-t.com/img3.png"/> xyz <img src="http://images.pexels.com/67636/img4.jpeg">`,
		[]string{"http://radio-t.com/img3.png", "http://images.pexels.com/67636/img4.jpeg"})
	assert.Equal(t, `<img src="/img?src=aHR0cDovL3JhZGlvLXQuY29tL2ltZzMucG5n"/> xyz <img src="/img?src=aHR0cDovL2ltYWdlcy5wZXhlbHMuY29tLzY3NjM2L2ltZzQuanBlZw==">`, r)
}

func TestImage_Routes(t *testing.T) {
	imageStore := image.MockStore{}
	img := Image{
		HTTP2HTTPS:   true,
		RemarkURL:    "https://demo.remark42.com",
		RoutePath:    "/api/v1/proxy",
		ImageService: image.NewService(&imageStore, image.ServiceParams{}),
	}

	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()
	httpSrv := imgHTTPTestsServer(t)
	defer httpSrv.Close()

	encodedImgURL := base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "/image/img1.png"))

	// no image supposed to be cached
	imageStore.On("Load", mock.Anything).Times(2).Return(nil, nil)

	resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "1462", resp.Header["Content-Length"][0])
	assert.Equal(t, "image/png", resp.Header["Content-Type"][0])

	encodedImgURL = base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "/image/no-such-image.png"))
	resp, err = http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	encodedImgURL = base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "bad encoding"))
	resp, err = http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestImage_DisabledCachingAndHTTP2HTTPS(t *testing.T) {
	imageStore := image.MockStore{}
	img := Image{
		RemarkURL:    "https://demo.remark42.com",
		RoutePath:    "/api/v1/proxy",
		ImageService: image.NewService(&imageStore, image.ServiceParams{}),
	}

	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()
	httpSrv := imgHTTPTestsServer(t)
	defer httpSrv.Close()

	encodedImgURL := base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "/image/img1.png"))

	imageStore.On("Load", mock.Anything).Once().Return(nil, nil)

	resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "1462", resp.Header["Content-Length"][0])
	assert.Equal(t, "image/png", resp.Header["Content-Type"][0])

	imageStore.AssertCalled(t, "Load", mock.Anything)
}

func TestImage_RoutesCachingImage(t *testing.T) {
	imageStore := image.MockStore{}
	img := Image{
		CacheExternal: true,
		RemarkURL:     "https://demo.remark42.com",
		RoutePath:     "/api/v1/proxy",
		ImageService:  image.NewService(&imageStore, image.ServiceParams{MaxSize: 1500}),
	}

	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()
	httpSrv := imgHTTPTestsServer(t)
	defer httpSrv.Close()

	imgURL := httpSrv.URL + "/image/img1.png"
	encodedImgURL := base64.URLEncoding.EncodeToString([]byte(imgURL))

	imageStore.On("Load", mock.Anything).Once().Return(nil, nil)
	imageStore.On("Save", mock.Anything, mock.Anything).Once().Return(nil)

	resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "1462", resp.Header["Content-Length"][0])
	assert.Equal(t, "image/png", resp.Header["Content-Type"][0])

	imageStore.AssertCalled(t, "Load", mock.Anything)
	imageStore.AssertCalled(t, "Save", "cached_images/4b84b15bff6ee5796152495a230e45e3d7e947d9-"+image.Sha1Str(imgURL), gopherPNGBytes())
}

func TestImage_RoutesUsingCachedImage(t *testing.T) {
	imageStore := image.MockStore{}
	img := Image{
		CacheExternal: true,
		RemarkURL:     "https://demo.remark42.com",
		RoutePath:     "/api/v1/proxy",
		ImageService:  image.NewService(&imageStore, image.ServiceParams{}),
	}

	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()
	httpSrv := imgHTTPTestsServer(t)
	defer httpSrv.Close()

	encodedImgURL := base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "/image/img1.png"))

	// In order to validate that cached data used cache "will return" some other data from what http server would
	testImage := []byte(fmt.Sprintf("%256s", "X"))
	imageStore.On("Load", mock.Anything).Once().Return(testImage, nil)

	resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "256", resp.Header["Content-Length"][0])
	assert.Equal(t, "text/plain; charset=utf-8", resp.Header["Content-Type"][0],
		"if you save text you receive text/plain in response, that's only fair option you got")

	imageStore.AssertCalled(t, "Load", mock.Anything)
}

func TestImage_RoutesTimedOut(t *testing.T) {
	imageStore := image.MockStore{}
	img := Image{
		HTTP2HTTPS:   true,
		RemarkURL:    "https://demo.remark42.com",
		RoutePath:    "/api/v1/proxy",
		Timeout:      50 * time.Millisecond,
		ImageService: image.NewService(&imageStore, image.ServiceParams{}),
	}

	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()
	httpSrv := imgHTTPTestsServer(t)
	defer httpSrv.Close()

	encodedImgURL := base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "/image/img-slow.png"))

	// no image supposed to be cached
	imageStore.On("Load", mock.Anything).Once().Return(nil, nil)

	resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, resp.Body.Close())
	require.NoError(t, err)
	t.Log(string(b))
	assert.True(t, strings.Contains(string(b), "deadline exceeded"))
}

func TestImage_ConvertProxyMode(t *testing.T) {
	img := Image{HTTP2HTTPS: true, RoutePath: "/img"}
	r := img.Convert(`<img src="http://radio-t.com/img3.png"/> xyz <img src="http://images.pexels.com/67636/img4.jpeg">`)
	assert.Equal(t, `<img src="/img?src=aHR0cDovL3JhZGlvLXQuY29tL2ltZzMucG5n"/> xyz <img src="/img?src=aHR0cDovL2ltYWdlcy5wZXhlbHMuY29tLzY3NjM2L2ltZzQuanBlZw==">`, r)

	r = img.Convert(`<img src="https://radio-t.com/img3.png"/> xyz <img src="http://images.pexels.com/67636/img4.jpeg">`)
	assert.Equal(t, `<img src="https://radio-t.com/img3.png"/> xyz <img src="/img?src=aHR0cDovL2ltYWdlcy5wZXhlbHMuY29tLzY3NjM2L2ltZzQuanBlZw==">`, r)

	img = Image{HTTP2HTTPS: true, RoutePath: "/img", RemarkURL: "http://example.com"}
	r = img.Convert(`<img src="http://radio-t.com/img3.png"/> xyz`)
	assert.Equal(t, `<img src="http://radio-t.com/img3.png"/> xyz`, r, "http:// remark url, no proxy")

	img = Image{HTTP2HTTPS: false, RoutePath: "/img"}
	r = img.Convert(`<img src="http://radio-t.com/img3.png"/> xyz`)
	assert.Equal(t, `<img src="http://radio-t.com/img3.png"/> xyz`, r, "disabled, no proxy")
}

func TestImage_ConvertCachingMode(t *testing.T) {
	img := Image{CacheExternal: true, RoutePath: "/img", RemarkURL: "https://remark42.com"}
	r := img.Convert(`<img src="http://radio-t.com/img3.png"/> xyz <img src="http://images.pexels.com/67636/img4.jpeg">`)
	assert.Equal(t, `<img src="https://remark42.com/img?src=aHR0cDovL3JhZGlvLXQuY29tL2ltZzMucG5n"/> xyz <img src="https://remark42.com/img?src=aHR0cDovL2ltYWdlcy5wZXhlbHMuY29tLzY3NjM2L2ltZzQuanBlZw==">`, r)

	r = img.Convert(`<img src="https://radio-t.com/img3.png"/> xyz <img src="https://images.pexels.com/67636/img4.jpeg">`)
	assert.Equal(t, `<img src="https://remark42.com/img?src=aHR0cHM6Ly9yYWRpby10LmNvbS9pbWczLnBuZw=="/> xyz <img src="https://remark42.com/img?src=aHR0cHM6Ly9pbWFnZXMucGV4ZWxzLmNvbS82NzYzNi9pbWc0LmpwZWc=">`, r)

	r = img.Convert(`<img src="https://remark42.com/pictures/1.png"/>`)
	assert.Equal(t, `<img src="https://remark42.com/pictures/1.png"/>`, r)

	img = Image{CacheExternal: false, RoutePath: "/img", RemarkURL: "https://remark42.com"}
	r = img.Convert(`<img src="http://radio-t.com/img3.png"/>`)
	assert.Equal(t, `<img src="http://radio-t.com/img3.png"/>`, r)

	// both Caching and Proxy enabled
	img = Image{CacheExternal: true, HTTP2HTTPS: true, RoutePath: "/img", RemarkURL: "https://remark42.com"}
	r = img.Convert(`<img src="http://radio-t.com/img3.png"/> xyz <img src="http://images.pexels.com/67636/img4.jpeg">`)
	assert.Equal(t, `<img src="https://remark42.com/img?src=aHR0cDovL3JhZGlvLXQuY29tL2ltZzMucG5n"/> xyz <img src="https://remark42.com/img?src=aHR0cDovL2ltYWdlcy5wZXhlbHMuY29tLzY3NjM2L2ltZzQuanBlZw==">`, r)
}

func imgHTTPTestsServer(t *testing.T) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/image/img1.png" {
			t.Log("http img request", r.URL)
			w.Header().Add("Content-Length", "1462")
			w.Header().Add("Content-Type", "image/png")
			_, err := w.Write(gopherPNGBytes())
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
