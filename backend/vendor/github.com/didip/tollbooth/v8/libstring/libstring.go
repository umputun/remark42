// Package libstring provides various string related functions.
package libstring

import (
	"net"
	"net/http"
	"strings"

	"github.com/didip/tollbooth/v8/limiter"
)

// StringInSlice finds needle in a slice of strings.
func StringInSlice(sliceString []string, needle string) bool {
	for _, b := range sliceString {
		if b == needle {
			return true
		}
	}
	return false
}

// RemoteIPFromIPLookup picks an ip address explicitly from limiter.IPLookup criteria.
// This function is intended to replace RemoteIP function.
func RemoteIPFromIPLookup(ipLookup limiter.IPLookup, r *http.Request) string {
	switch ipLookup.Name {
	case "RemoteAddr":
		// 1. Cover the basic use cases for both ipv4 and ipv6
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// 2. Upon error, just return the remote addr.
			return r.RemoteAddr
		}
		return ip

	case "X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP":
		ipAddrListCommaSeparated := r.Header.Values(ipLookup.Name)

		ipAddrCommaSeparated := strings.Join(ipAddrListCommaSeparated, ",")

		ips := strings.Split(ipAddrCommaSeparated, ",")
		for i, p := range ips {
			ips[i] = strings.TrimSpace(p)
		}

		ipIndex := len(ips) - 1 - ipLookup.IndexFromRight
		if ipIndex < 0 {
			ipIndex = 0
		}

		return ips[ipIndex]
	}

	return ""
}

// CanonicalizeIP returns a form of ip suitable for comparison to other IPs.
// For IPv4 addresses, this is simply the whole string.
// For IPv6 addresses, this is the /64 prefix.
func CanonicalizeIP(ip string) string {
	isIPv6 := false
	// This is how net.ParseIP decides if an address is IPv6
	// https://cs.opensource.google/go/go/+/refs/tags/go1.17.7:src/net/ip.go;l=704
	for i := 0; !isIPv6 && i < len(ip); i++ {
		switch ip[i] {
		case '.':
			// IPv4
			return ip
		case ':':
			// IPv6
			isIPv6 = true
		}
	}
	if !isIPv6 {
		// Not an IP address at all
		return ip
	}

	// By default, the string representation of a net.IPNet (masked IP address) is just
	// "full_address/mask_bits". But using that will result in different addresses with
	// the same /64 prefix comparing differently. So we need to zero out the last 64 bits
	// so that all IPs in the same prefix will be the same.
	//
	// Note: When 1.18 is the minimum Go version, this can be written more cleanly like:
	// netip.PrefixFrom(netip.MustParseAddr(ipv6), 64).Masked().Addr().String()
	// (With appropriate error checking.)

	ipv6 := net.ParseIP(ip)
	if ipv6 == nil {
		return ip
	}

	const bytesToZero = (128 - 64) / 8
	for i := len(ipv6) - bytesToZero; i < len(ipv6); i++ {
		ipv6[i] = 0
	}

	// Note that this doesn't have the "/64" suffix customary with a CIDR representation,
	// but those three bytes add nothing for us.
	return ipv6.String()
}
