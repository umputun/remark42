package rest

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-pkgz/rest/realip"
)

// OnlyFrom middleware allows access for limited list of source IPs.
// Rules can be complete IPs (like 192.168.1.12), textual prefixes (129.168.), or CIDRs (192.168.0.0/16).
// Complete IPs use semantic address equality, CIDRs use network containment, and all other rules use prefix matching.
func OnlyFrom(onlyIps ...string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if len(onlyIps) == 0 {
				// no restrictions if no ips defined
				h.ServeHTTP(w, r)
				return
			}
			matched, ip, err := matchSourceIP(r, onlyIps)
			if err != nil {
				_ = EncodeJSON(w, http.StatusInternalServerError, JSON{"error": fmt.Sprintf("can't get realip: %s", err)})
				return
			}
			if matched {
				// matched ip - allow
				h.ServeHTTP(w, r)
				return
			}

			_ = EncodeJSON(w, http.StatusForbidden, JSON{"error": fmt.Sprintf("ip %q rejected", ip)})
		}
		return http.HandlerFunc(fn)
	}
}

// matchSourceIP returns true if request's ip matches any of ips
func matchSourceIP(r *http.Request, ips []string) (result bool, match string, err error) {
	ip, err := realip.Get(r)
	if err != nil {
		return false, "", fmt.Errorf("can't get realip: %w", err) // we can't get ip, so no match
	}
	parsedIP := net.ParseIP(ip)
	// check for cidr, complete ip, or ip prefix
	for _, exclIP := range ips {
		if _, cidrnet, err := net.ParseCIDR(exclIP); err == nil {
			if cidrnet.Contains(parsedIP) {
				return true, ip, nil
			}
			continue
		}
		if allowedIP := net.ParseIP(exclIP); allowedIP != nil {
			if allowedIP.Equal(parsedIP) {
				return true, ip, nil
			}
			continue
		}
		if strings.HasPrefix(ip, exclIP) {
			return true, ip, nil
		}
	}
	return false, ip, nil
}
