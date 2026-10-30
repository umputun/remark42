package service

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

// maxWebsiteLength limits the stored user website
const maxWebsiteLength = 200

// ErrInvalidWebsite is returned for a website that is not an absolute http(s) URL of a public host
var ErrInvalidWebsite = errors.New("website must be an http or https URL of a public host")

// NormalizeWebsite validates a user website and returns it in canonical form.
// Only absolute http and https URLs of a host with a domain are accepted: no credentials,
// IP addresses, localhost, control characters or other schemes (javascript:, data: and so on).
// A value without a scheme gets https://, so "example.com" becomes "https://example.com".
func NormalizeWebsite(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" || len(value) > maxWebsiteLength || strings.ContainsAny(value, " \t\r\n<>\"'`\\") {
		return "", ErrInvalidWebsite
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Opaque != "" {
		return "", ErrInvalidWebsite
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || !strings.Contains(host, ".") || strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") ||
		net.ParseIP(host) != nil || host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return "", ErrInvalidWebsite
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	return u.String(), nil
}
