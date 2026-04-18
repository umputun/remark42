// Package safehttp provides HTTP transports hardened against SSRF: outbound
// connections are dialed using a pre-resolved IP, with a check that all
// resolved IPs sit outside private/reserved ranges. This blocks both naive
// SSRF (private IP literals in user-supplied URLs) and DNS rebinding.
package safehttp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Transport returns an *http.Transport whose DialContext refuses any address
// that resolves to a private/reserved IP, choosing the IP itself for the dial
// to defeat DNS rebinding (an attacker cannot have the resolver hand back a
// public IP at the check and a private one at the connect).
//
// The returned transport is a clone of http.DefaultTransport with only
// DialContext overridden, preserving Proxy, HTTP/2, idle/keep-alive and
// TLS handshake timeouts that bare &http.Transport{} would lose.
func Transport() *http.Transport {
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address %s: %w", addr, err)
		}

		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("can't resolve host %s: %w", host, err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no IP addresses resolved for host %s", host)
		}

		for _, ip := range ips {
			if IsPrivateIP(ip.IP) {
				return nil, fmt.Errorf("access to private address is not allowed")
			}
		}

		var lastErr error
		for _, ip := range ips {
			conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			lastErr = dialErr
		}
		return nil, fmt.Errorf("can't connect to %s: %w", host, lastErr)
	}
	return t
}

// privateCIDRs holds pre-parsed private/reserved CIDR blocks.
var privateCIDRs = func() []*net.IPNet {
	cidrs := []string{
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16",
		"::1/128", "fc00::/7", "fe80::/10",
	}
	blocks := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, block, _ := net.ParseCIDR(cidr)
		blocks = append(blocks, block)
	}
	return blocks
}()

// IsPrivateIP reports whether ip falls in any private, loopback, link-local,
// CGNAT, or reserved range — including IPv4 and IPv6 unspecified addresses.
func IsPrivateIP(ip net.IP) bool {
	if ip.IsUnspecified() {
		return true
	}
	for _, block := range privateCIDRs {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}
