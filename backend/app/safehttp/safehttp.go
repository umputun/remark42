// Package safehttp provides HTTP transports hardened against SSRF: outbound
// connections are dialed using a pre-resolved IP, with a check that all
// resolved IPs sit outside private/reserved ranges. This blocks both naive
// SSRF (private IP literals in user-supplied URLs) and DNS rebinding.
package safehttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"time"
)

// ErrPrivateAddress is returned by the Transport dialer for a host resolving to a private or reserved address
var ErrPrivateAddress = errors.New("access to private address is not allowed")

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
				return nil, ErrPrivateAddress
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

// cgnat is the carrier-grade NAT range (RFC 6598), shared address space the stdlib does not classify
var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// IsPrivateIP reports whether ip is unspecified, loopback, private (RFC 1918, RFC 4193),
// link-local unicast or CGNAT, in IPv4, IPv6 or IPv4-mapped IPv6 form.
func IsPrivateIP(ip net.IP) bool {
	if ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
		return true
	}
	addr, ok := netip.AddrFromSlice(ip)
	return ok && cgnat.Contains(addr.Unmap())
}
