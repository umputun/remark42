package rest

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEtagMatches covers the strict If-None-Match parser that replaced a substring
// search prone to false positives (etag "abc" being matched inside "fooabc").
func TestEtagMatches(t *testing.T) {
	tbl := []struct {
		name   string
		header string
		etag   string
		want   bool
	}{
		{"exact match", `"v2:abc"`, `"v2:abc"`, true},
		{"comma-separated, second matches", `"x", "v2:abc"`, `"v2:abc"`, true},
		{"weak validator prefix", `W/"v2:abc"`, `"v2:abc"`, true},
		{"wildcard matches anything", `*`, `"v2:abc"`, true},
		{"leading/trailing whitespace", `   "v2:abc"   `, `"v2:abc"`, true},
		{"substring not enough", `"v2:abcdef"`, `"v2:abc"`, false},
		{"prefix-only mismatch", `"v2:ab"`, `"v2:abc"`, false},
		{"pre-fix etag no longer matches v2", `"abc"`, `"v2:abc"`, false},
		{"empty header", ``, `"v2:abc"`, false},
		{"different etag", `"v2:xyz"`, `"v2:abc"`, false},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EtagMatches(tt.header, tt.etag))
		})
	}
}

func TestSetImageDefenseHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	SetImageDefenseHeaders(w)
	assert.Equal(t, StrictImageCSP, w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, `inline; filename="image"`, w.Header().Get("Content-Disposition"))
}

// TestSafeImgContentType exercises the strict allowlist. The previous behavior
// (HasPrefix "image/" with an explicit image/svg+xml carve-out) is gone — the
// allowlist is the source of truth, and the explicit svg branch was dead code
// because http.DetectContentType never returns image/svg+xml (real SVG bodies
// sniff as text/xml or text/plain depending on whether they carry an XML decl,
// so they are rejected implicitly by not matching the allowlist).
func TestSafeImgContentType(t *testing.T) {
	// minimal magic-byte bodies — verified via http.DetectContentType to produce
	// the expected image/* result without needing testdata files for every format
	pngMagic := []byte("\x89PNG\r\n\x1a\n")
	jpegMagic := []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00")
	gifBytes := []byte("GIF89a")
	webpBytes := []byte("RIFF\x00\x00\x00\x00WEBPVP8 ")
	bmpBytes := []byte("BM\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	icoBytes := []byte("\x00\x00\x01\x00\x01\x00")

	// SVG with XML decl sniffs as text/xml — rejected because it's not in the allowlist
	svgWithXMLDecl := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"></svg>`)
	// SVG without XML decl sniffs as text/plain — also rejected
	svgPlain := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10"></svg>`)

	tbl := []struct {
		name    string
		body    []byte
		wantCT  string
		wantErr bool
	}{
		{name: "nil rejected", body: nil, wantErr: true},
		{name: "empty rejected", body: []byte{}, wantErr: true},
		{name: "png magic accepted", body: pngMagic, wantCT: "image/png"},
		{name: "jpeg magic accepted", body: jpegMagic, wantCT: "image/jpeg"},
		{name: "gif accepted", body: gifBytes, wantCT: "image/gif"},
		{name: "webp accepted", body: webpBytes, wantCT: "image/webp"},
		{name: "bmp accepted", body: bmpBytes, wantCT: "image/bmp"},
		{name: "ico accepted", body: icoBytes, wantCT: "image/x-icon"},
		{name: "html doc rejected", body: []byte(`<!DOCTYPE html><html></html>`), wantErr: true},
		{name: "html fragment rejected", body: []byte(`<body><img></body>`), wantErr: true},
		{name: "plain text rejected", body: []byte("hello world"), wantErr: true},
		{name: "octet-stream rejected", body: []byte{0x00, 0x01, 0x02, 0x03, 0x04}, wantErr: true},
		{name: "svg with xml decl rejected (sniffs as text/xml)", body: svgWithXMLDecl, wantErr: true},
		{name: "svg without xml decl rejected (sniffs as text/plain)", body: svgPlain, wantErr: true},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeImgContentType(tt.body)
			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, got)
				assert.Contains(t, err.Error(), "non-image content type")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCT, got)
			// returned type must never carry a charset suffix (the strip code path)
			assert.NotContains(t, got, ";")
		})
	}
}
