package service

import (
	"errors"
	"net"
	"net/url"
	"strings"

	"golang.org/x/net/idna"
)

// maxWebsiteLength limits the stored user website
const maxWebsiteLength = 200

// ErrInvalidWebsite is returned for a website that is not an absolute http(s) URL of a public host
var ErrInvalidWebsite = errors.New("website must be an http or https URL of a public host")

// NormalizeWebsite validates a user website and returns it in canonical form.
// Only absolute http and https URLs of a host with a domain are accepted: no credentials,
// IP addresses, localhost, control characters or other schemes (javascript:, data: and so on).
// The host is checked as browsers read it: mapped to ASCII like browsers do (so fullwidth digits
// become digits and an internationalized domain is stored in punycode), and hosts that browsers
// parse as IPv4 (127.1, 0x7f.0.0.1, 0177.0.0.1) are rejected as IP addresses.
// It is a syntactic check: a domain is not resolved, so it may still point to a private address.
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
	host, err := idna.Lookup.ToASCII(u.Hostname())
	if err != nil || host == "" || !strings.Contains(host, ".") || strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") ||
		net.ParseIP(host) != nil || endsInNumber(host) || host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return "", ErrInvalidWebsite
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if port := u.Port(); port != "" {
		host = net.JoinHostPort(host, port)
	}
	u.Host = host
	u.Fragment = ""
	if res := u.String(); len(res) <= maxWebsiteLength {
		return res, nil
	}
	return "", ErrInvalidWebsite
}

// endsInNumber reports whether browsers parse the host as an IPv4 address: its last label is a
// decimal number or a 0x hex number (the "ends in a number" rule of the WHATWG URL standard)
func endsInNumber(host string) bool {
	last := strings.ToLower(host[strings.LastIndex(host, ".")+1:])
	if strings.HasPrefix(last, "0x") {
		return strings.Trim(last[2:], "0123456789abcdef") == ""
	}
	return last != "" && strings.Trim(last, "0123456789") == ""
}
