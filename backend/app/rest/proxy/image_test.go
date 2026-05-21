package proxy

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store/image"
)

// gopher png for test, from https://golang.org/src/image/png/example_test.go
const gopher = "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"

func gopherPNG() io.Reader { return base64.NewDecoder(base64.StdEncoding, strings.NewReader(gopher)) }
func gopherPNGBytes() []byte {
	img, _ := io.ReadAll(gopherPNG())
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
	// no image supposed to be cached
	imageStore := image.StoreMock{LoadFunc: func(string) ([]byte, error) { return nil, nil }}
	img := Image{
		HTTP2HTTPS:   true,
		RemarkURL:    "https://demo.remark42.com",
		RoutePath:    "/api/v1/proxy",
		ImageService: image.NewService(&imageStore, image.ServiceParams{}),
		Transport:    http.DefaultTransport,
	}

	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()
	httpSrv := imgHTTPTestsServer(t)
	defer httpSrv.Close()

	t.Run("valid image", func(t *testing.T) {
		encodedImgURL := base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "/image/img1.png"))
		resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
		require.NoError(t, err)
		assert.NoError(t, resp.Body.Close())
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "1462", resp.Header["Content-Length"][0])
		assert.Equal(t, "image/png", resp.Header["Content-Type"][0])
	})

	t.Run("no image", func(t *testing.T) {
		encodedImgURL := base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "/image/no-such-image.png"))
		resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
		require.NoError(t, err)
		assert.NoError(t, resp.Body.Close())
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("bad encoding", func(t *testing.T) {
		encodedImgURL := base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "bad encoding"))
		resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
		require.NoError(t, err)
		assert.NoError(t, resp.Body.Close())
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, 2, len(imageStore.LoadCalls()))
	})

	t.Run("non-image reference", func(t *testing.T) {
		encodedImgURL := base64.URLEncoding.EncodeToString([]byte("https://google.com"))
		resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
		require.NoError(t, err)
		assert.NoError(t, resp.Body.Close())
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, 3, len(imageStore.LoadCalls()))
	})
}

func TestImage_DisabledCachingAndHTTP2HTTPS(t *testing.T) {
	imageStore := image.StoreMock{LoadFunc: func(string) ([]byte, error) { return nil, nil }}
	img := Image{
		RemarkURL:    "https://demo.remark42.com",
		RoutePath:    "/api/v1/proxy",
		ImageService: image.NewService(&imageStore, image.ServiceParams{}),
		Transport:    http.DefaultTransport,
	}

	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()
	httpSrv := imgHTTPTestsServer(t)
	defer httpSrv.Close()

	encodedImgURL := base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "/image/img1.png"))

	resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "1462", resp.Header["Content-Length"][0])
	assert.Equal(t, "image/png", resp.Header["Content-Type"][0])

	assert.Equal(t, 1, len(imageStore.LoadCalls()))
}

func TestImage_RoutesCachingImage(t *testing.T) {
	imageStore := image.StoreMock{
		LoadFunc: func(string) ([]byte, error) {
			return nil, nil
		},
		SaveFunc: func(string, []byte) error {
			return nil
		},
	}
	img := Image{
		CacheExternal: true,
		RemarkURL:     "https://demo.remark42.com",
		RoutePath:     "/api/v1/proxy",
		ImageService:  image.NewService(&imageStore, image.ServiceParams{MaxSize: 1500}),
		Transport:     http.DefaultTransport,
	}

	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()
	httpSrv := imgHTTPTestsServer(t)
	defer httpSrv.Close()

	imgURL := httpSrv.URL + "/image/img1.png"
	encodedImgURL := base64.URLEncoding.EncodeToString([]byte(imgURL))

	resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "1462", resp.Header["Content-Length"][0])
	assert.Equal(t, "image/png", resp.Header["Content-Type"][0])

	assert.Equal(t, 1, len(imageStore.LoadCalls()))
	assert.Equal(t, 1, len(imageStore.SaveCalls()))
	assert.Equal(t, "cached_images/4b84b15bff6ee5796152495a230e45e3d7e947d9-"+image.Sha1Str(imgURL), imageStore.SaveCalls()[0].ID)
	assert.Equal(t, gopherPNGBytes(), imageStore.SaveCalls()[0].Img)
}

func TestImage_RoutesUsingCachedImage(t *testing.T) {
	t.Run("cached image is served", func(t *testing.T) {
		imageStore := image.StoreMock{LoadFunc: func(string) ([]byte, error) {
			return gopherPNGBytes(), nil
		}}
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
		resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
		require.NoError(t, err)
		assert.NoError(t, resp.Body.Close())
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
		assert.Equal(t, 1, len(imageStore.LoadCalls()))
	})

	t.Run("non-image cached bytes are rejected (cache poisoning defense)", func(t *testing.T) {
		nonImage := fmt.Appendf(nil, "%256s", "X")
		imageStore := image.StoreMock{LoadFunc: func(string) ([]byte, error) {
			return nonImage, nil
		}}
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
		resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
		require.NoError(t, err)
		body, _ := io.ReadAll(resp.Body)
		assert.NoError(t, resp.Body.Close())
		assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode,
			"non-image bytes from cache must be rejected, not served as text/plain (XSS defense)")
		assert.False(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html"),
			"reject response must not be text/html; got %q", resp.Header.Get("Content-Type"))
		assert.NotContains(t, string(body), "XXXXX", "non-image bytes must not be echoed back")
	})
}

func TestImage_RoutesTimedOut(t *testing.T) {
	// no image supposed to be cached
	imageStore := image.StoreMock{LoadFunc: func(string) ([]byte, error) { return nil, nil }}
	img := Image{
		HTTP2HTTPS:   true,
		RemarkURL:    "https://demo.remark42.com",
		RoutePath:    "/api/v1/proxy",
		Timeout:      50 * time.Millisecond,
		ImageService: image.NewService(&imageStore, image.ServiceParams{}),
		Transport:    http.DefaultTransport,
	}

	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()
	httpSrv := imgHTTPTestsServer(t)
	defer httpSrv.Close()

	encodedImgURL := base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "/image/img-slow.png"))

	resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	b, err := io.ReadAll(resp.Body)
	assert.NoError(t, resp.Body.Close())
	require.NoError(t, err)
	t.Log(string(b))
	assert.Contains(t, string(b), "failed to fetch")
	assert.NotContains(t, string(b), "deadline exceeded", "should not leak transport details")
	assert.Equal(t, 1, len(imageStore.LoadCalls()))
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

func TestImage_PrivateIPBlocking(t *testing.T) {
	imageStore := image.StoreMock{LoadFunc: func(string) ([]byte, error) { return nil, nil }}
	img := Image{
		HTTP2HTTPS:   true,
		RemarkURL:    "https://demo.remark42.com",
		RoutePath:    "/api/v1/proxy",
		Timeout:      100 * time.Millisecond,
		ImageService: image.NewService(&imageStore, image.ServiceParams{}),
		// no Transport override — uses SSRF-safe transport
	}

	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()

	tbl := []struct {
		name string
		url  string
	}{
		{"loopback", "http://127.0.0.1/image.png"},
		{"rfc1918 10.x", "http://10.0.0.1/image.png"},
		{"rfc1918 172.16.x", "http://172.16.0.1/image.png"},
		{"rfc1918 192.168.x", "http://192.168.1.1/image.png"},
		{"link-local", "http://169.254.1.1/image.png"},
		{"ipv6 loopback", "http://[::1]/image.png"},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			encodedImgURL := base64.URLEncoding.EncodeToString([]byte(tt.url))
			resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
			require.NoError(t, err)
			b, err := io.ReadAll(resp.Body)
			assert.NoError(t, resp.Body.Close())
			require.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
			assert.NotContains(t, string(b), "private address", "should not leak private IP check details")
			assert.Contains(t, string(b), "failed to fetch")
		})
	}
}

func TestImage_ErrorSanitization(t *testing.T) {
	// server that immediately closes connections to simulate transport errors
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		conn, _, _ := hj.Hijack()
		conn.Close() // forcefully close to trigger transport error
	}))
	defer httpSrv.Close()

	imageStore := image.StoreMock{LoadFunc: func(string) ([]byte, error) { return nil, nil }}
	img := Image{
		RemarkURL:    "https://demo.remark42.com",
		RoutePath:    "/api/v1/proxy",
		Timeout:      2 * time.Second,
		ImageService: image.NewService(&imageStore, image.ServiceParams{}),
		Transport:    http.DefaultTransport,
	}

	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()

	encodedImgURL := base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "/image.png"))
	resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	b, err := io.ReadAll(resp.Body)
	assert.NoError(t, resp.Body.Close())
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, string(b), "failed to fetch")
	assert.NotContains(t, string(b), "EOF", "should not leak transport details")
	assert.NotContains(t, string(b), "connection", "should not leak transport details")
}

func TestImage_ResponseSizeLimit(t *testing.T) {
	// create a test server that returns a large image
	largeImg := make([]byte, 2000)
	for i := range largeImg {
		largeImg[i] = 0xFF
	}
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(largeImg)
	}))
	defer httpSrv.Close()

	imageStore := image.StoreMock{LoadFunc: func(string) ([]byte, error) { return nil, nil }}
	img := Image{
		RemarkURL:    "https://demo.remark42.com",
		RoutePath:    "/api/v1/proxy",
		ImageService: image.NewService(&imageStore, image.ServiceParams{MaxSize: 1000}),
		Transport:    http.DefaultTransport,
	}

	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()

	encodedImgURL := base64.URLEncoding.EncodeToString([]byte(httpSrv.URL + "/big-image.png"))
	resp, err := http.Get(ts.URL + "/?src=" + encodedImgURL)
	require.NoError(t, err)
	b, err := io.ReadAll(resp.Body)
	assert.NoError(t, resp.Body.Close())
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, string(b), "failed to fetch")
}

// TestImage_ContentTypeHandling covers both the rock-solid acceptance of legitimate
// images and the rejection of content-type-spoofing payloads (the XSS vector where
// upstream lies about Content-Type and the proxy serves attacker HTML back from the
// remark42 origin). Every response — accept or reject — must carry the layered
// defense headers (strict CSP, nosniff, Content-Disposition: inline).
//
// The defense must not depend on the upstream Content-Type header: each row controls
// it independently of the body so the matrix exercises attackers who flip the upstream
// header on the fly, and polyglot bodies where image magic bytes prefix HTML payloads.
func TestImage_ContentTypeHandling(t *testing.T) {
	htmlBody := []byte("<html><body><script>alert(document.domain)</script></body></html>")
	// polyglot: real PNG magic + trailing HTML. Sniffs as image/png, must be served
	// as image/png so the browser renders as image (broken or otherwise) — never as HTML.
	polyglot := append(append([]byte{}, gopherPNGBytes()...), []byte("<script>alert(1)</script>")...)

	tbl := []struct {
		name          string
		upstreamCT    string // Content-Type header the upstream sends
		body          []byte
		accept        bool   // true: legitimate image, served back; false: attack, rejected
		wantCT        string // exact Content-Type if accept
		payloadMarker string // attack substring that must NOT appear in the response body
	}{
		// legitimate
		{name: "real png", upstreamCT: "image/png", body: gopherPNGBytes(), accept: true, wantCT: "image/png"},

		// upstream lies — body is HTML, header varies. All must be rejected at body-sniff.
		{name: "html body claimed as image/png", upstreamCT: "image/png", body: htmlBody, payloadMarker: "<script>"},
		{name: "html body claimed as image/jpeg", upstreamCT: "image/jpeg", body: htmlBody, payloadMarker: "<script>"},
		{name: "html body claimed as image/gif", upstreamCT: "image/gif", body: htmlBody, payloadMarker: "<script>"},
		// upstream claims svg+xml; body still sniffs as text/html (the stdlib sniffer
		// never returns image/svg+xml, see rest.SafeImgContentType godoc).
		{name: "html body upstream claims image/svg+xml", upstreamCT: "image/svg+xml", body: htmlBody, payloadMarker: "<script>"},
		{name: "html body claimed as image/webp", upstreamCT: "image/webp", body: htmlBody, payloadMarker: "<script>"},

		// svg payloads — even if upstream claims a valid image format, the sniffer sees XML/text and we must reject
		{
			name:          "svg with xml declaration and onload",
			upstreamCT:    "image/png",
			body:          []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"></svg>`),
			payloadMarker: "onload",
		},
		{
			name:          "html fragment without doctype",
			upstreamCT:    "image/png",
			body:          []byte(`<body><img src=x onerror=alert(1)></body>`),
			payloadMarker: "onerror",
		},

		// polyglot — image magic + appended HTML. Sniffs as image/png so we accept and serve as image/png.
		// Safety comes from the response headers (Content-Type: image/png + X-Content-Type-Options: nosniff),
		// not from body filtering: the bytes round-trip verbatim by design (assertion below). The browser
		// cannot execute the trailing HTML when the response type is image/png with nosniff.
		{name: "polyglot png+html served as png", upstreamCT: "image/png", body: polyglot, accept: true, wantCT: "image/png"},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", tt.upstreamCT)
				_, _ = w.Write(tt.body)
			}))
			defer upstream.Close()

			imageStore := image.StoreMock{LoadFunc: func(string) ([]byte, error) { return nil, nil }}
			img := Image{
				RemarkURL:    "https://demo.remark42.com",
				RoutePath:    "/api/v1/proxy",
				ImageService: image.NewService(&imageStore, image.ServiceParams{}),
				Transport:    http.DefaultTransport,
			}

			ts := httptest.NewServer(http.HandlerFunc(img.Handler))
			defer ts.Close()

			encodedURL := base64.URLEncoding.EncodeToString([]byte(upstream.URL + "/logo.png"))
			resp, err := http.Get(ts.URL + "/?src=" + encodedURL)
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())

			// every response — accept or reject — must carry the defense headers
			assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
			assert.Contains(t, resp.Header.Get("Content-Disposition"), "inline")
			csp := resp.Header.Get("Content-Security-Policy")
			assert.Contains(t, csp, "default-src 'none'", "strict CSP missing")
			assert.Contains(t, csp, "sandbox", "CSP sandbox missing")

			if tt.accept {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Equal(t, tt.wantCT, resp.Header.Get("Content-Type"))
				assert.Equal(t, tt.body, body, "body bytes must round-trip")
				assert.Contains(t, resp.Header.Get("Cache-Control"), "max-age=2592000",
					"validated success path carries the 30-day TTL")
				assert.True(t, strings.HasPrefix(resp.Header.Get("Etag"), `"v2:`),
					"validated success path carries the versioned etag")
				return
			}

			// reject path
			assert.GreaterOrEqual(t, resp.StatusCode, 400, "must reject non-image content")
			// reject responses must NOT inherit the success path's long-lived cache
			// headers — a transient 4xx would otherwise be pinned in browser/intermediary
			// caches alongside the versioned etag for 30 days.
			assert.Contains(t, resp.Header.Get("Cache-Control"), "no-store",
				"reject path must set Cache-Control: no-store; got %q", resp.Header.Get("Cache-Control"))
			assert.NotContains(t, resp.Header.Get("Cache-Control"), "max-age=2592000",
				"reject path must not carry the success-path 30-day TTL")
			assert.Empty(t, resp.Header.Get("Etag"),
				"reject path must not carry the versioned etag (would pin the failure in cache)")
			ct := resp.Header.Get("Content-Type")
			assert.False(t, strings.HasPrefix(ct, "text/html"),
				"reject response must not be text/html; got %q", ct)
			assert.NotContains(t, string(body), tt.payloadMarker,
				"reject response must not echo attack payload; got body=%q", string(body))
		})
	}
}

// TestEtagMatches lives in the rest package alongside the shared EtagMatches helper
// (see backend/app/rest/image_headers_test.go). The proxy handler delegates to it.

// TestImage_ContentTypeHandling_CacheHit exercises the cache-hit branch of the handler:
// the StoreMock returns attacker bytes directly, so the upstream is never contacted.
// Without the body-sniff at serve time, pre-fix code would have echoed cached HTML as
// text/html. After the fix the same content-type defense applies on the cache path.
func TestImage_ContentTypeHandling_CacheHit(t *testing.T) {
	htmlBody := []byte("<html><body><script>alert(document.domain)</script></body></html>")
	polyglot := append(append([]byte{}, gopherPNGBytes()...), []byte("<script>alert(1)</script>")...)

	tbl := []struct {
		name          string
		cached        []byte
		accept        bool
		wantCT        string
		payloadMarker string
	}{
		{name: "html in cache claimed as image/png", cached: htmlBody, payloadMarker: "<script>"},
		{name: "polyglot in cache served as png", cached: polyglot, accept: true, wantCT: "image/png"},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			imageStore := image.StoreMock{LoadFunc: func(string) ([]byte, error) {
				return tt.cached, nil
			}}
			img := Image{
				CacheExternal: true,
				RemarkURL:     "https://demo.remark42.com",
				RoutePath:     "/api/v1/proxy",
				ImageService:  image.NewService(&imageStore, image.ServiceParams{}),
				// no Transport — cache hit must not reach upstream
			}
			ts := httptest.NewServer(http.HandlerFunc(img.Handler))
			defer ts.Close()

			encodedURL := base64.URLEncoding.EncodeToString([]byte("https://attacker.example.com/logo.png"))
			resp, err := http.Get(ts.URL + "/?src=" + encodedURL)
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())

			assert.Equal(t, 1, len(imageStore.LoadCalls()), "served from cache")
			assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
			assert.Contains(t, resp.Header.Get("Content-Disposition"), "inline")
			assert.Contains(t, resp.Header.Get("Content-Security-Policy"),
				"default-src 'none'; sandbox; frame-ancestors 'none'")

			if tt.accept {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Equal(t, tt.wantCT, resp.Header.Get("Content-Type"))
				return
			}
			assert.GreaterOrEqual(t, resp.StatusCode, 400, "must reject non-image cached content")
			assert.False(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html"),
				"reject response must not be text/html; got %q", resp.Header.Get("Content-Type"))
			assert.NotContains(t, string(body), tt.payloadMarker,
				"reject response must not echo cached attack payload")
		})
	}
}

// TestImage_EtagVersioned proves browser/proxy caches with pre-fix etags (the
// unversioned base64 of src that used to be served alongside text/html bodies)
// no longer satisfy revalidation: the server returns a fresh 200 with image
// content instead of 304-ing the poisoned cached entry. The 30-day Cache-Control
// max-age is unchanged — local browser caches still serving pre-fix bytes within
// their TTL are not reached; the prefix only helps clients that revalidate during
// the cached lifetime (Ctrl+R, intermediaries, post-expiry). See etagVersionPrefix
// godoc for the tradeoff.
func TestImage_EtagVersioned(t *testing.T) {
	imageStore := image.StoreMock{LoadFunc: func(string) ([]byte, error) { return nil, nil }}
	img := Image{
		RemarkURL:    "https://demo.remark42.com",
		RoutePath:    "/api/v1/proxy",
		ImageService: image.NewService(&imageStore, image.ServiceParams{}),
		Transport:    http.DefaultTransport,
	}
	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()
	httpSrv := imgHTTPTestsServer(t)
	defer httpSrv.Close()

	srcRaw := httpSrv.URL + "/image/img1.png"
	encodedSrc := base64.URLEncoding.EncodeToString([]byte(srcRaw))
	preFixEtag := `"` + encodedSrc + `"` // what a pre-fix browser would have cached

	req, err := http.NewRequest("GET", ts.URL+"/?src="+encodedSrc, http.NoBody)
	require.NoError(t, err)
	req.Header.Set("If-None-Match", preFixEtag)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"pre-fix etag must NOT validate as 304 — old cached text/html must be replaced")
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))
	assert.NotEqual(t, preFixEtag, resp.Header.Get("Etag"), "new etag must differ from pre-fix")
	assert.True(t, strings.HasPrefix(resp.Header.Get("Etag"), `"v2:`), "new etag must carry the version prefix")
	cc := resp.Header.Get("Cache-Control")
	assert.Contains(t, cc, "max-age=2592000", "success path keeps 30-day TTL for cache efficiency")

	// sanity: the NEW etag round-trips as 304 when sent back
	loadsBefore := len(imageStore.LoadCalls())
	req2, err := http.NewRequest("GET", ts.URL+"/?src="+encodedSrc, http.NoBody)
	require.NoError(t, err)
	req2.Header.Set("If-None-Match", resp.Header.Get("Etag"))
	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusNotModified, resp2.StatusCode, "new etag must validate against itself")
	body, _ := io.ReadAll(resp2.Body)
	assert.Empty(t, body, "304 must have no body")
	// 304 path must skip the store lookup entirely — revalidation must not amplify load
	assert.Equal(t, loadsBefore, len(imageStore.LoadCalls()),
		"revalidation 304 must not trigger any store Load (avoids upstream DoS amplification)")
	// 304 path must still carry the layered defense headers
	assert.Equal(t, "nosniff", resp2.Header.Get("X-Content-Type-Options"))
	assert.Contains(t, resp2.Header.Get("Content-Disposition"), "inline")
	assert.Contains(t, resp2.Header.Get("Content-Security-Policy"), "default-src 'none'")
	assert.Contains(t, resp2.Header.Get("Content-Security-Policy"), "sandbox")
}

// TestImage_RevalidationSkipsIO proves that a matching current-version If-None-Match
// short-circuits before any cache lookup or upstream fetch. With no Transport and no
// upstream server reachable, the only way this test can pass with 304 is if Load is
// never called and downloadImage is never attempted. This closes the DoS amplification
// where every reuse on a hot comment page would otherwise re-hit the upstream when
// CacheExternal is false.
func TestImage_RevalidationSkipsIO(t *testing.T) {
	imageStore := image.StoreMock{LoadFunc: func(string) ([]byte, error) {
		t.Fatal("Load must not be called on the revalidation short-circuit path")
		return nil, nil
	}}
	img := Image{
		RemarkURL:    "https://demo.remark42.com",
		RoutePath:    "/api/v1/proxy",
		ImageService: image.NewService(&imageStore, image.ServiceParams{}),
		// no Transport — any downloadImage attempt would also fail
	}
	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()

	encodedSrc := base64.URLEncoding.EncodeToString([]byte("https://example.com/whatever.png"))
	currentEtag := `"v2:` + encodedSrc + `"`

	req, err := http.NewRequest("GET", ts.URL+"/?src="+encodedSrc, http.NoBody)
	require.NoError(t, err)
	req.Header.Set("If-None-Match", currentEtag)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotModified, resp.StatusCode,
		"matching current-version etag must short-circuit to 304 without I/O")
	assert.Equal(t, 0, len(imageStore.LoadCalls()),
		"revalidation must not trigger store Load")
	body, _ := io.ReadAll(resp.Body)
	assert.Empty(t, body, "304 must have no body")
	// defense headers must still be set on the short-circuit path
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "inline")
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "default-src 'none'")
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "sandbox")
	assert.Equal(t, currentEtag, resp.Header.Get("Etag"))
	assert.Contains(t, resp.Header.Get("Cache-Control"), "max-age=2592000")
}

// TestImage_PerRequestRevalidation proves the defense holds when upstream flips its
// response body between requests (give a real PNG once, HTML next time, etc.). Each
// proxy response is independently validated against the body actually returned, so
// trust never accumulates and an earlier "good" response cannot grant the next one a
// free pass.
func TestImage_PerRequestRevalidation(t *testing.T) {
	htmlBody := []byte("<html><script>alert(1)</script></html>")

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png") // always lie consistently
		switch r.URL.Path {
		case "/png":
			_, _ = w.Write(gopherPNGBytes())
		case "/html":
			_, _ = w.Write(htmlBody)
		}
	}))
	defer upstream.Close()

	imageStore := image.StoreMock{LoadFunc: func(string) ([]byte, error) { return nil, nil }}
	img := Image{
		RemarkURL:    "https://demo.remark42.com",
		RoutePath:    "/api/v1/proxy",
		ImageService: image.NewService(&imageStore, image.ServiceParams{}),
		Transport:    http.DefaultTransport,
	}
	ts := httptest.NewServer(http.HandlerFunc(img.Handler))
	defer ts.Close()

	// alternate calls: PNG, HTML, PNG, HTML — each must be judged on its own bytes.
	type step struct {
		path       string
		wantStatus int
		wantCT     string // prefix match
	}
	steps := []step{
		{path: "/png", wantStatus: http.StatusOK, wantCT: "image/png"},
		{path: "/html", wantStatus: http.StatusUnsupportedMediaType, wantCT: "application/json"},
		{path: "/png", wantStatus: http.StatusOK, wantCT: "image/png"},
		{path: "/html", wantStatus: http.StatusUnsupportedMediaType, wantCT: "application/json"},
	}
	for i, s := range steps {
		t.Run(fmt.Sprintf("step_%d_%s", i, s.path), func(t *testing.T) {
			encodedURL := base64.URLEncoding.EncodeToString([]byte(upstream.URL + s.path))
			resp, err := http.Get(ts.URL + "/?src=" + encodedURL)
			require.NoError(t, err)
			body, _ := io.ReadAll(resp.Body)
			require.NoError(t, resp.Body.Close())

			assert.Equal(t, s.wantStatus, resp.StatusCode)
			assert.True(t, strings.HasPrefix(resp.Header.Get("Content-Type"), s.wantCT),
				"expected Content-Type prefix %q, got %q", s.wantCT, resp.Header.Get("Content-Type"))
			assert.False(t, strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html"),
				"must never serve text/html under any flip")
			assert.NotContains(t, string(body), "<script>",
				"attacker payload must never appear in response body")
			// every response must still carry the defense headers
			assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
			assert.Contains(t, resp.Header.Get("Content-Disposition"), "inline")
			assert.Contains(t, resp.Header.Get("Content-Security-Policy"),
				"default-src 'none'; sandbox; frame-ancestors 'none'")
		})
	}
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
