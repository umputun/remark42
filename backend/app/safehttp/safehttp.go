// Package safehttp provides HTTP transports hardened against SSRF: outbound
// connections are dialed using a pre-resolved IP, with a check that all
// resolved IPs sit outside private/reserved ranges. This blocks both naive
// SSRF (private IP literals in user-supplied URLs) and DNS rebinding.
// When an outbound proxy is configured, the request's target host is checked
// instead of the proxy's address, which may be private.
package safehttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"time"
	"unicode/utf8"

	"golang.org/x/net/http/httpproxy"
	"golang.org/x/net/idna"
)

// ErrPrivateAddress is returned by the Transport for a host resolving to a private or reserved address
var ErrPrivateAddress = errors.New("access to private address is not allowed")

// Transport returns an *http.Transport whose DialContext refuses any address
// that resolves to a private/reserved IP, choosing the IP itself for the dial
// to defeat DNS rebinding (an attacker cannot have the resolver hand back a
// public IP at the check and a private one at the connect).
//
// The proxy comes from HTTP_PROXY, HTTPS_PROXY and NO_PROXY, read on each call.
// For a proxied request the dialer only sees the proxy's address, so the Proxy
// hook checks the target host instead and the dialer lets the proxy itself
// through. The proxy resolves the target's name on its own, so the check
// covers address literals and what the local resolver returns, not the
// address the proxy finally connects to.
//
// The returned transport is a clone of http.DefaultTransport with only
// Proxy and DialContext overridden, preserving HTTP/2, idle/keep-alive and
// TLS handshake timeouts that bare &http.Transport{} would lose.
func Transport() *http.Transport {
	cfg := httpproxy.FromEnvironment()
	proxyFunc := cfg.ProxyFunc()

	// the addresses the transport dials for a proxy, taken from the same config with NO_PROXY cleared
	proxies := map[string]bool{}
	anyHost := *cfg
	anyHost.NoProxy = ""
	anyHostProxy := anyHost.ProxyFunc()
	for _, scheme := range []string{"http", "https"} {
		if u, err := anyHostProxy(&url.URL{Scheme: scheme, Host: "example.com"}); err == nil && u != nil {
			proxies[hostPort(u)] = true
		}
	}

	dialer := &net.Dialer{Timeout: 30 * time.Second}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = func(req *http.Request) (*url.URL, error) {
		proxyURL, err := proxyFunc(req.URL)
		if err != nil {
			return nil, err
		}
		// the dialer passes a proxy's address unchecked, so a target is checked here when it goes
		// through a proxy and when it is a proxy's address reached directly (NO_PROXY, loopback)
		if proxyURL != nil || proxies[hostPort(req.URL)] {
			if _, err := publicIPs(req.Context(), asciiHost(req.URL.Hostname())); err != nil {
				return nil, err
			}
		}
		return proxyURL, nil
	}
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if proxies[addr] {
			return dialer.DialContext(ctx, network, addr)
		}
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address %s: %w", addr, err)
		}

		ips, err := publicIPs(ctx, host)
		if err != nil {
			return nil, err
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

// lookupIPAddr resolves a host name; tests replace it to avoid depending on DNS
var lookupIPAddr = net.DefaultResolver.LookupIPAddr

// publicIPs resolves host and returns its addresses, or ErrPrivateAddress if any of them is private
func publicIPs(ctx context.Context, host string) ([]net.IPAddr, error) {
	ips, err := lookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("can't resolve host %s: %w", host, err)
	}
	if len(ips) == 0 { // reported as the resolver reports a name it doesn't know, so callers treat the two alike
		return nil, fmt.Errorf("can't resolve host %s: %w", host, &net.DNSError{Err: "no IP addresses", Name: host, IsNotFound: true})
	}
	for _, ip := range ips {
		if IsPrivateIP(ip.IP) {
			return nil, ErrPrivateAddress
		}
	}
	return ips, nil
}

// hostPort returns the address http.Transport dials for u: the ASCII host and the port, or the scheme's default
func hostPort(u *url.URL) string {
	port := u.Port()
	if port == "" {
		port = map[string]string{"http": "80", "https": "443", "socks5": "1080", "socks5h": "1080"}[u.Scheme]
	}
	return net.JoinHostPort(asciiHost(u.Hostname()), port)
}

// asciiHost returns host IDNA-encoded unless it is already ASCII, the form http.Transport dials and resolves
func asciiHost(host string) string {
	for i := range len(host) {
		if host[i] >= utf8.RuneSelf {
			if v, err := idna.Lookup.ToASCII(host); err == nil {
				return v
			}
			return host
		}
	}
	return host
}

// nonPublic lists ranges that are reachable only inside a network, which the stdlib does not classify
var nonPublic = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),        // "this network" (RFC 791)
	netip.MustParsePrefix("100.64.0.0/10"),    // carrier-grade NAT, shared address space (RFC 6598)
	netip.MustParsePrefix("168.63.129.16/32"), // Azure platform endpoint, answers inside Azure networks only
	netip.MustParsePrefix("192.0.0.0/24"),     // IETF protocol assignments (RFC 6890)
	netip.MustParsePrefix("198.18.0.0/15"),    // benchmarking (RFC 2544), used for cloud VPCs and pods
	netip.MustParsePrefix("240.0.0.0/4"),      // reserved (RFC 1112) and broadcast, used for cloud VPCs
	netip.MustParsePrefix("64:ff9b:1::/48"),   // local-use NAT64 (RFC 8215), where the IPv4 part sits depends on the network
}

// embedded4 lists IPv6 ranges whose addresses carry the IPv4 address a gateway connects to, with its byte offset.
// IPv4-mapped addresses are unmapped before the check.
var embedded4 = []struct {
	prefix netip.Prefix
	offset int
}{
	{netip.MustParsePrefix("64:ff9b::/96"), 12},    // NAT64 well-known prefix (RFC 6052)
	{netip.MustParsePrefix("::ffff:0:0:0/96"), 12}, // IPv4-translated (RFC 2765)
	{netip.MustParsePrefix("::/96"), 12},           // IPv4-compatible (RFC 4291)
	{netip.MustParsePrefix("2002::/16"), 2},        // 6to4 (RFC 3056)
}

// IsPrivateIP reports whether ip is unspecified, loopback, private (RFC 1918, RFC 4193), link-local unicast
// or in one of the nonPublic ranges, in IPv4, IPv6 or IPv4-mapped IPv6 form. An IPv6 address that carries
// an IPv4 address (embedded4) is private when that IPv4 address is.
func IsPrivateIP(ip net.IP) bool {
	if ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
		return true
	}
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	if slices.ContainsFunc(nonPublic, func(p netip.Prefix) bool { return p.Contains(addr) }) {
		return true
	}
	for _, e := range embedded4 {
		if e.prefix.Contains(addr) {
			b := addr.As16()
			return IsPrivateIP(net.IP(b[e.offset : e.offset+4]))
		}
	}
	return false
}
