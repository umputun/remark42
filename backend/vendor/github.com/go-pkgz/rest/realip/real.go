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

// Get returns real ip from the given request
// Prioritize public IPs over private IPs
func Get(r *http.Request) (string, error) {
	var firstIP string
	for _, h := range []string{"X-Forwarded-For", "X-Real-Ip"} {
		addresses := strings.Split(r.Header.Get(h), ",")
		for i := len(addresses) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(addresses[i])
			realIP := net.ParseIP(ip)
			if firstIP == "" && realIP != nil {
				firstIP = ip
			}
			if !realIP.IsGlobalUnicast() || isPrivateSubnet(realIP) {
				continue
			}
			return ip, nil
		}
	}

	if firstIP != "" {
		return firstIP, nil
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("can't parse ip %q: %w", r.RemoteAddr, err)
	}
	if netIP := net.ParseIP(ip); netIP == nil {
		return "", fmt.Errorf("no valid ip found")
	}

	return ip, nil
}

// isPrivateSubnet - check to see if this ip is in a private subnet
func isPrivateSubnet(ipAddress net.IP) bool {

	// inRange - check to see if a given ip address is within a range given
	inRange := func(r ipRange, ipAddress net.IP) bool {
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
