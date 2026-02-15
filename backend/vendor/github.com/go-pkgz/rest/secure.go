package rest

import (
	"net/http"
	"strconv"
	"strings"
)

// SecureConfig defines security headers configuration.
// Use SecOpt functions to customize.
type SecureConfig struct {
	// xFrameOptions sets X-Frame-Options header. Default: DENY
	XFrameOptions string
	// xContentTypeOptions sets X-Content-Type-Options. Default: nosniff
	XContentTypeOptions string
	// ReferrerPolicy sets Referrer-Policy header. Default: strict-origin-when-cross-origin
	ReferrerPolicy string
	// ContentSecurityPolicy sets Content-Security-Policy header. Default: empty (not set)
	ContentSecurityPolicy string
	// PermissionsPolicy sets Permissions-Policy header. Default: empty (not set)
	PermissionsPolicy string
	// sTSSeconds sets max-age for Strict-Transport-Security. 0 disables.
	// only sent when request uses HTTPS. Default: 31536000 (1 year)
	STSSeconds int
	// sTSIncludeSubdomains adds includeSubDomains to HSTS. Default: true
	STSIncludeSubdomains bool
	// sTSPreload adds preload flag to HSTS. Default: false
	STSPreload bool
	// xSSProtection sets X-XSS-Protection header. Default: 1; mode=block
	// note: this header is deprecated in modern browsers but still useful for older ones
	XSSProtection string
}

// SecOpt is a functional option for SecureConfig
type SecOpt func(*SecureConfig)

// defaultSecureConfig returns config with sensible defaults
func defaultSecureConfig() SecureConfig {
	return SecureConfig{
		XFrameOptions:        "DENY",
		XContentTypeOptions:  "nosniff",
		ReferrerPolicy:       "strict-origin-when-cross-origin",
		STSSeconds:           31536000, // 1 year
		STSIncludeSubdomains: true,
		STSPreload:           false,
		XSSProtection:        "1; mode=block",
	}
}

// SecFrameOptions sets X-Frame-Options header.
// Common values: "DENY", "SAMEORIGIN"
func SecFrameOptions(value string) SecOpt {
	return func(c *SecureConfig) {
		c.XFrameOptions = value
	}
}

// SecContentTypeNosniff enables or disables X-Content-Type-Options: nosniff
func SecContentTypeNosniff(enable bool) SecOpt {
	return func(c *SecureConfig) {
		if enable {
			c.XContentTypeOptions = "nosniff"
		} else {
			c.XContentTypeOptions = ""
		}
	}
}

// SecReferrerPolicy sets Referrer-Policy header.
// Common values: "no-referrer", "same-origin", "strict-origin", "strict-origin-when-cross-origin"
func SecReferrerPolicy(policy string) SecOpt {
	return func(c *SecureConfig) {
		c.ReferrerPolicy = policy
	}
}

// SecContentSecurityPolicy sets Content-Security-Policy header.
// Example: "default-src 'self'; script-src 'self'"
func SecContentSecurityPolicy(policy string) SecOpt {
	return func(c *SecureConfig) {
		c.ContentSecurityPolicy = policy
	}
}

// SecPermissionsPolicy sets Permissions-Policy header.
// Example: "geolocation=(), microphone=()"
func SecPermissionsPolicy(policy string) SecOpt {
	return func(c *SecureConfig) {
		c.PermissionsPolicy = policy
	}
}

// SecHSTS configures Strict-Transport-Security header.
// maxAge is in seconds (0 disables HSTS), includeSubdomains and preload are optional flags.
// Note: HSTS header is only sent when the request is over HTTPS.
func SecHSTS(maxAge int, includeSubdomains, preload bool) SecOpt {
	return func(c *SecureConfig) {
		c.STSSeconds = maxAge
		c.STSIncludeSubdomains = includeSubdomains
		c.STSPreload = preload
	}
}

// SecXSSProtection sets X-XSS-Protection header.
// Set to empty string to disable. Common values: "0", "1", "1; mode=block"
func SecXSSProtection(value string) SecOpt {
	return func(c *SecureConfig) {
		c.XSSProtection = value
	}
}

// SecAllHeaders is a convenience option to set common headers for secure web applications.
// Sets CSP with self-only policy and restrictive permissions.
func SecAllHeaders() SecOpt {
	return func(c *SecureConfig) {
		c.ContentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; form-action 'self'; frame-ancestors 'none'"
		c.PermissionsPolicy = "geolocation=(), microphone=(), camera=()"
	}
}

// Secure is middleware that adds security headers to responses.
// By default it sets: X-Frame-Options, X-Content-Type-Options, Referrer-Policy,
// X-XSS-Protection, and Strict-Transport-Security (for HTTPS only).
// Use SecOpt functions to customize the configuration.
func Secure(opts ...SecOpt) func(http.Handler) http.Handler {
	cfg := defaultSecureConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// set security headers
			if cfg.XFrameOptions != "" {
				w.Header().Set("X-Frame-Options", cfg.XFrameOptions)
			}
			if cfg.XContentTypeOptions != "" {
				w.Header().Set("X-Content-Type-Options", cfg.XContentTypeOptions)
			}
			if cfg.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)
			}
			if cfg.XSSProtection != "" {
				w.Header().Set("X-XSS-Protection", cfg.XSSProtection)
			}
			if cfg.ContentSecurityPolicy != "" {
				w.Header().Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
			}
			if cfg.PermissionsPolicy != "" {
				w.Header().Set("Permissions-Policy", cfg.PermissionsPolicy)
			}

			// HSTS only for HTTPS connections
			if cfg.STSSeconds > 0 && isHTTPS(r) {
				sts := "max-age=" + strconv.Itoa(cfg.STSSeconds)
				if cfg.STSIncludeSubdomains {
					sts += "; includeSubDomains"
				}
				if cfg.STSPreload {
					sts += "; preload"
				}
				w.Header().Set("Strict-Transport-Security", sts)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isHTTPS checks if the request is over HTTPS by examining TLS state and common proxy headers
func isHTTPS(r *http.Request) bool {
	// direct TLS connection
	if r.TLS != nil {
		return true
	}
	// check common proxy headers (case-insensitive)
	if strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		return true
	}
	// check RFC 7239 Forwarded header
	if forwarded := r.Header.Get("Forwarded"); forwarded != "" {
		if forwardedProtoIsHTTPS(forwarded) {
			return true
		}
	}
	return false
}

// forwardedProtoIsHTTPS parses RFC 7239 Forwarded header to check for proto=https.
// The header format is: Forwarded: for=1.2.3.4;proto=https;by=proxy, for=5.6.7.8
// Parameters are separated by semicolons, multiple forwarded elements by commas.
func forwardedProtoIsHTTPS(header string) bool {
	// split by comma to get individual forwarded elements
	for element := range strings.SplitSeq(header, ",") {
		// split by semicolon to get parameters within element
		for param := range strings.SplitSeq(element, ";") {
			param = strings.TrimSpace(param)
			// check for proto=https (case-insensitive per RFC 7239)
			if len(param) > 6 && strings.EqualFold(param[:6], "proto=") {
				if strings.EqualFold(strings.TrimSpace(param[6:]), "https") {
					return true
				}
			}
		}
	}
	return false
}
