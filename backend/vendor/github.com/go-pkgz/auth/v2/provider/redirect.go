package provider

import (
	"net/url"
	"strings"

	"github.com/go-pkgz/auth/v2/token"
)

// isAllowedRedirect reports whether the "from" URL is safe to redirect to
// after a successful auth handshake.
//
// The check is opt-in: when allowed is nil the function returns true for any
// non-empty input, preserving the behavior of versions before the redirect
// validator existed. This keeps a dependency bump from breaking existing
// consumers; hardening is enabled by setting Opts.AllowedRedirectHosts.
//
// When allowed is non-nil:
//   - only http/https schemes are accepted
//   - relative paths and unparseable URLs are rejected
//   - the service's own host (derived from serviceURL) is always allowed
//   - any other host must appear in the allowed list
//
// Hostname comparison is case-insensitive and ignores the port:
// https://app.example.com:443 and https://App.Example.Com are treated as the
// same host. Operators wanting strict port-aware checks should list each
// host:port form explicitly via AllowedHosts.
func isAllowedRedirect(from, serviceURL string, allowed token.AllowedHosts) bool {
	// permissive default: no allowlist configured = legacy behavior.
	// guard against typed-nil AllowedHostsFunc values (non-nil interface
	// wrapping a nil func) to avoid panicking in Get().
	if allowed == nil {
		return from != ""
	}
	if fn, ok := allowed.(token.AllowedHostsFunc); ok && fn == nil {
		return from != ""
	}
	u, err := url.Parse(from)
	if err != nil || u.Hostname() == "" {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	fromHost := u.Hostname()
	if svc, sErr := url.Parse(serviceURL); sErr == nil && svc.Hostname() != "" && strings.EqualFold(svc.Hostname(), fromHost) {
		return true
	}
	hosts, hErr := allowed.Get()
	if hErr != nil {
		return false
	}
	for _, h := range hosts {
		if strings.EqualFold(h, fromHost) || strings.EqualFold(h, u.Host) {
			return true
		}
	}
	return false
}

// redirectHostForLog extracts just the hostname from a from-URL for logging
// on rejection, so attacker-supplied paths/queries do not leak into operator
// logs. Returns a sentinel if the URL cannot be parsed.
func redirectHostForLog(from string) string {
	if u, err := url.Parse(from); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return "<unparseable>"
}
