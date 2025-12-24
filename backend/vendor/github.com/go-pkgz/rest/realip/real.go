// Package realip extracts a real IP address from the request.
package realip

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type ipRange struct {
	start net.IP
	end   net.IP
}

// privateRanges contains the list of private and special-use IP ranges.
// reference: https://www.iana.org/assignments/iana-ipv4-special-registry/iana-ipv4-special-registry.xhtml
var privateRanges = []ipRange{
	// IPv4 Private Ranges
	{start: net.ParseIP("10.0.0.0"), end: net.ParseIP("10.255.255.255")},
	{start: net.ParseIP("172.16.0.0"), end: net.ParseIP("172.31.255.255")},
	{start: net.ParseIP("192.168.0.0"), end: net.ParseIP("192.168.255.255")},
	// IPv4 Link-Local
	{start: net.ParseIP("169.254.0.0"), end: net.ParseIP("169.254.255.255")},
	// IPv4 Shared Address Space (RFC 6598)
	{start: net.ParseIP("100.64.0.0"), end: net.ParseIP("100.127.255.255")},
	// IPv4 Benchmarking (RFC 2544)
	{start: net.ParseIP("198.18.0.0"), end: net.ParseIP("198.19.255.255")},
	// IPv6 Unique Local Addresses (ULA)
	{start: net.ParseIP("fc00::"), end: net.ParseIP("fdff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")},
	// IPv6 Link-local Addresses
	{start: net.ParseIP("fe80::"), end: net.ParseIP("febf:ffff:ffff:ffff:ffff:ffff:ffff:ffff")},
}

// Get returns real IP from the given request.
// It checks headers in the following priority order:
//  1. X-Real-IP - trusted proxy (nginx/reproxy) sets this to actual client
//  2. CF-Connecting-IP - Cloudflare's header for original client
//  3. X-Forwarded-For - leftmost public IP (original client in CDN chain)
//  4. RemoteAddr - fallback for direct connections
//
// Only public IPs are accepted from headers; private/loopback/link-local IPs are skipped.
func Get(r *http.Request) (string, error) {
	// check X-Real-IP first (single value, set by trusted proxy)
	if xRealIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); xRealIP != "" {
		if ip := net.ParseIP(xRealIP); isPublicIP(ip) {
			return xRealIP, nil
		}
	}

	// check CF-Connecting-IP (Cloudflare's header)
	if cfIP := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cfIP != "" {
		if ip := net.ParseIP(cfIP); isPublicIP(ip) {
			return cfIP, nil
		}
	}

	// check X-Forwarded-For, find leftmost public IP
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		addresses := strings.Split(xff, ",")
		for _, addr := range addresses {
			ip := strings.TrimSpace(addr)
			if parsedIP := net.ParseIP(ip); isPublicIP(parsedIP) {
				return ip, nil
			}
		}
	}

	// fall back to RemoteAddr
	return parseRemoteAddr(r.RemoteAddr)
}

// isPublicIP checks if the IP is a valid public (globally routable) IP address.
func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if !ip.IsGlobalUnicast() {
		return false
	}
	return !isPrivateSubnet(ip)
}

// parseRemoteAddr extracts and validates IP from RemoteAddr (handles both "ip" and "ip:port" formats).
func parseRemoteAddr(remoteAddr string) (string, error) {
	if remoteAddr == "" {
		return "", fmt.Errorf("empty remote address")
	}

	// try to extract host from host:port format
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		remoteAddr = host
	}

	// validate it's a proper IP address
	if netIP := net.ParseIP(remoteAddr); netIP == nil {
		return "", fmt.Errorf("no valid ip found in %q", remoteAddr)
	}

	return remoteAddr, nil
}

// isPrivateSubnet - check to see if this ip is in a private subnet
func isPrivateSubnet(ipAddress net.IP) bool {
	inRange := func(r ipRange, ipAddress net.IP) bool { // check to see if a given ip address is within a range given
		// ensure the IPs are in the same format for comparison
		ipAddress = ipAddress.To16()
		r.start = r.start.To16()
		r.end = r.end.To16()
		return bytes.Compare(ipAddress, r.start) >= 0 && bytes.Compare(ipAddress, r.end) <= 0
	}

	for _, r := range privateRanges {
		if inRange(r, ipAddress) {
			return true
		}
	}
	return false
}
