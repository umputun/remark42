//go:build go1.25

package rest

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// CrossOriginProtection provides CSRF protection using modern browser Fetch metadata.
// It validates requests using Sec-Fetch-Site and Origin headers, rejecting cross-origin
// state-changing requests. Safe methods (GET, HEAD, OPTIONS) are always allowed.
//
// For Go 1.25+, this wraps the stdlib http.CrossOriginProtection.
// For earlier versions, it provides an equivalent custom implementation.
type CrossOriginProtection struct {
	stdlib *http.CrossOriginProtection
}

// NewCrossOriginProtection creates a new CSRF protection middleware.
func NewCrossOriginProtection() *CrossOriginProtection {
	return &CrossOriginProtection{
		stdlib: http.NewCrossOriginProtection(),
	}
}

// AddTrustedOrigin adds an origin that should be allowed to make cross-origin requests.
// The origin must be in the format "scheme://host" or "scheme://host:port".
// Returns an error if the origin format is invalid.
func (c *CrossOriginProtection) AddTrustedOrigin(origin string) error {
	u, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("invalid origin: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("origin must have scheme and host: %s", origin)
	}
	if u.Path != "" && u.Path != "/" {
		return fmt.Errorf("origin must not have path: %s", origin)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("origin must not have query or fragment: %s", origin)
	}

	// normalize to lowercase for consistent case-insensitive matching
	normalized := strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)
	return c.stdlib.AddTrustedOrigin(normalized)
}

// AddBypassPattern adds a URL pattern that should bypass CSRF protection.
// Patterns follow the same syntax as http.ServeMux (e.g., "/api/webhook", "/oauth/").
// Use sparingly and only for endpoints that have alternative authentication.
func (c *CrossOriginProtection) AddBypassPattern(pattern string) {
	c.stdlib.AddInsecureBypassPattern(pattern)
}

// SetDenyHandler sets a custom handler for rejected requests.
// If not set, rejected requests receive a 403 Forbidden response.
func (c *CrossOriginProtection) SetDenyHandler(h http.Handler) {
	c.stdlib.SetDenyHandler(h)
}

// Check validates a request against CSRF protection rules.
// Returns nil if the request is allowed, or an error describing why it was rejected.
func (c *CrossOriginProtection) Check(r *http.Request) error {
	// the stdlib Check method panics or returns void, so we use our own check logic
	// by creating a test handler and seeing if it gets called

	// safe methods are always allowed
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return nil
	}

	// use a test to determine if request would be allowed
	allowed := false
	testHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		allowed = true
	})

	// create a response recorder to capture the result
	rec := &discardResponseWriter{}
	c.stdlib.Handler(testHandler).ServeHTTP(rec, r)

	if !allowed {
		return fmt.Errorf("cross-origin request blocked by CSRF protection")
	}
	return nil
}

// Handler wraps an http.Handler with CSRF protection.
// Rejected requests receive a 403 Forbidden response (or custom deny handler).
func (c *CrossOriginProtection) Handler(h http.Handler) http.Handler {
	return c.stdlib.Handler(h)
}

// discardResponseWriter is a minimal ResponseWriter for testing.
type discardResponseWriter struct {
	header http.Header
}

func (d *discardResponseWriter) Header() http.Header {
	if d.header == nil {
		d.header = make(http.Header)
	}
	return d.header
}
func (d *discardResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (d *discardResponseWriter) WriteHeader(_ int)           {}
