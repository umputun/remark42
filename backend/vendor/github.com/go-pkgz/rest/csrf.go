//go:build !go1.25

package rest

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// CrossOriginProtection provides CSRF protection using modern browser Fetch metadata.
// It validates requests using Sec-Fetch-Site and Origin headers, rejecting cross-origin
// state-changing requests. Safe methods (GET, HEAD, OPTIONS) are always allowed.
//
// For Go 1.25+, this wraps the stdlib http.CrossOriginProtection.
// For earlier versions, it provides an equivalent custom implementation.
type CrossOriginProtection struct {
	mu             sync.RWMutex
	trustedOrigins map[string]bool
	bypassPatterns []string
	denyHandler    http.Handler
}

// NewCrossOriginProtection creates a new CSRF protection middleware.
func NewCrossOriginProtection() *CrossOriginProtection {
	return &CrossOriginProtection{
		trustedOrigins: make(map[string]bool),
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

	normalized := strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)

	c.mu.Lock()
	c.trustedOrigins[normalized] = true
	c.mu.Unlock()
	return nil
}

// AddBypassPattern adds a URL pattern that should bypass CSRF protection.
// Patterns follow the same syntax as http.ServeMux (e.g., "/api/webhook", "/oauth/").
// Use sparingly and only for endpoints that have alternative authentication.
func (c *CrossOriginProtection) AddBypassPattern(pattern string) {
	c.mu.Lock()
	c.bypassPatterns = append(c.bypassPatterns, pattern)
	c.mu.Unlock()
}

// SetDenyHandler sets a custom handler for rejected requests.
// If not set, rejected requests receive a 403 Forbidden response.
func (c *CrossOriginProtection) SetDenyHandler(h http.Handler) {
	c.mu.Lock()
	c.denyHandler = h
	c.mu.Unlock()
}

// Check validates a request against CSRF protection rules.
// Returns nil if the request is allowed, or an error describing why it was rejected.
func (c *CrossOriginProtection) Check(r *http.Request) error {
	// safe methods are always allowed
	if isSafeMethod(r.Method) {
		return nil
	}

	// check bypass patterns
	if c.matchesBypassPattern(r.URL.Path) {
		return nil
	}

	// check Sec-Fetch-Site header (modern browsers)
	secFetchSite := r.Header.Get("Sec-Fetch-Site")
	if secFetchSite != "" {
		switch secFetchSite {
		case "same-origin", "none":
			return nil
		case "cross-site", "same-site":
			// check if origin is trusted
			origin := r.Header.Get("Origin")
			if origin != "" && c.isOriginTrusted(origin) {
				return nil
			}
			return fmt.Errorf("cross-origin request blocked: Sec-Fetch-Site=%s", secFetchSite)
		}
	}

	// fallback: check Origin header against Host
	origin := r.Header.Get("Origin")
	if origin != "" {
		// check if origin is trusted
		if c.isOriginTrusted(origin) {
			return nil
		}

		// compare origin host with request host
		originURL, err := url.Parse(origin)
		if err != nil {
			return fmt.Errorf("invalid Origin header: %w", err)
		}

		requestHost := r.Host
		if requestHost == "" {
			requestHost = r.URL.Host
		}

		// normalize hosts for comparison
		originHost := strings.ToLower(originURL.Host)
		requestHost = strings.ToLower(requestHost)

		if originHost != requestHost {
			return fmt.Errorf("cross-origin request blocked: origin %s does not match host %s", originHost, requestHost)
		}
		return nil
	}

	// no Sec-Fetch-Site or Origin headers - assume same-origin or non-browser request
	return nil
}

// Handler wraps an http.Handler with CSRF protection.
// Rejected requests receive a 403 Forbidden response (or custom deny handler).
func (c *CrossOriginProtection) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := c.Check(r); err != nil {
			c.mu.RLock()
			deny := c.denyHandler
			c.mu.RUnlock()

			if deny != nil {
				deny.ServeHTTP(w, r)
				return
			}
			http.Error(w, "Forbidden - CSRF check failed", http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// isSafeMethod returns true for HTTP methods that don't modify state.
func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

// isOriginTrusted checks if the origin is in the trusted list.
func (c *CrossOriginProtection) isOriginTrusted(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	normalized := strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)

	c.mu.RLock()
	trusted := c.trustedOrigins[normalized]
	c.mu.RUnlock()
	return trusted
}

// matchesBypassPattern checks if the path matches any bypass pattern.
func (c *CrossOriginProtection) matchesBypassPattern(path string) bool {
	c.mu.RLock()
	patterns := make([]string, len(c.bypassPatterns))
	copy(patterns, c.bypassPatterns)
	c.mu.RUnlock()

	for _, pattern := range patterns {
		if matchPattern(pattern, path) {
			return true
		}
	}
	return false
}

// matchPattern implements simple pattern matching similar to http.ServeMux.
func matchPattern(pattern, path string) bool {
	// exact match
	if pattern == path {
		return true
	}
	// prefix match for patterns ending with /
	if strings.HasSuffix(pattern, "/") && strings.HasPrefix(path, pattern) {
		return true
	}
	return false
}
