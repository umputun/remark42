package rest

import (
	"fmt"
	"net/http"
	"strings"
)

// StrictImageCSP is the strictest default-deny Content-Security-Policy used both by
// image-serving handlers (/api/v1/img, /api/v1/picture/{user}/{id}) and by the api-wide
// apiCSPMiddleware (covering all /api/v1/* responses — JSON, XML/RSS, images). The name
// keeps the "image" prefix for historical reasons; the policy itself is generic and
// suitable for any non-document API response.
//
// Re-setting the same value inside the image handlers (after the middleware already set
// it) is intentional defense-in-depth: if the middleware ever stops applying (route
// refactor, mount point change), the handlers still emit the header.
const StrictImageCSP = "default-src 'none'; sandbox; frame-ancestors 'none'"

// SafeImgContentType returns the sniffed content type for provided bytes if and only
// if it is in the strict allowlist of image formats safe to serve from a same-origin
// proxy endpoint: image/png, image/jpeg, image/gif, image/webp, image/bmp, image/x-icon.
// Anything else — HTML, XML, SVG, plain text, application/octet-stream, or any future
// image format the stdlib sniffer may learn (e.g. AVIF, HEIC, JXL, TIFF) — is rejected.
// SVG would also be rejected as it sniffs as text/xml or text/plain, never image/svg+xml.
// The previous behavior silently mapped application/octet-stream to image/* and is gone.
func SafeImgContentType(img []byte) (string, error) {
	contentType := http.DetectContentType(img)
	base, _, _ := strings.Cut(contentType, ";")
	base = strings.TrimSpace(base)
	switch base {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/bmp", "image/x-icon":
		return base, nil
	}
	return "", fmt.Errorf("non-image content type %q", contentType)
}

// SetImageDefenseHeaders applies the layered defense headers shared by every response
// from image-serving endpoints (success, 304, or error). Each header survives content-type
// validation regressions, browser sniffing, and top-level navigation:
//   - Content-Security-Policy: strict, with sandbox — blocks inline scripts and event handlers
//   - X-Content-Type-Options: nosniff — prevents browsers from MIME-overriding the declared type
//   - Content-Disposition: inline; filename="image" — frames the response as a file, not a document
//
// CSP is duplicated by apiCSPMiddleware for /api/v1/* — re-setting the same value here is
// harmless and provides defense-in-depth if the middleware is bypassed or moved. The other
// two headers (nosniff, Content-Disposition with filename) are image-specific and not set
// by the middleware.
func SetImageDefenseHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", StrictImageCSP)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", `inline; filename="image"`)
}

// EtagMatches reports whether If-None-Match header value contains the given etag.
// Handles the * wildcard, comma-separated etag lists with the W/ weak-validator prefix.
// NOTE: This is intentionally a simple splitter — it does not handle opaque-tags that
// contain commas (allowed by RFC 7232 but never emitted by this codebase, whose etag
// format is `"v2:<base64-url>"` or `"<user>/<xid>"`). If the etag format ever changes
// to include comma-bearing values, revisit this parser.
// Replaces a substring search that could match unrelated entries (e.g. an etag that
// happens to be a prefix of another).
func EtagMatches(header, etag string) bool {
	header = strings.TrimSpace(header)
	if header == "*" {
		return true
	}
	for tag := range strings.SplitSeq(header, ",") {
		tag = strings.TrimSpace(tag)
		tag = strings.TrimPrefix(tag, "W/")
		if tag == etag {
			return true
		}
	}
	return false
}
