// Package api middleware: request-scoped HTTP middlewares used by the REST router.
package api

import (
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/didip/tollbooth/v8"
	"github.com/didip/tollbooth/v8/limiter"
	log "github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/umputun/remark42/backend/app/rest"
	"github.com/umputun/remark42/backend/app/store"
)

// ipForwardingHeaders are the request headers R.RealIP derives the client IP from.
var ipForwardingHeaders = []string{"X-Real-IP", "X-Forwarded-For", "CF-Connecting-IP"}

// realIPMiddleware derives the client IP from forwarding headers (X-Real-IP / X-Forwarded-For /
// CF-Connecting-IP) via R.RealIP, but honors those headers only for requests whose direct peer
// is one of the trusted proxies. For any other peer it drops those headers and pins RemoteAddr to
// the real socket IP, so an untrusted client can't spoof the IP that per-IP controls (rate limiting,
// vote dedup, comment IP, anonymous id) and the request log key on.
//
// With no trusted proxies configured it falls back to trusting the headers from any client (the
// historical behavior). That is spoofable by design, so operators running behind a reverse proxy
// should set --trusted-proxy to the proxy's network — see the "trusted proxy" docs.
func realIPMiddleware(trustedProxies []*net.IPNet) func(http.Handler) http.Handler {
	if len(trustedProxies) == 0 {
		return R.RealIP
	}
	return func(next http.Handler) http.Handler {
		fromTrusted := R.RealIP(next) // rewrites RemoteAddr from the forwarding headers
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peer := directPeerIP(r.RemoteAddr)
			if peer != nil && cidrsContain(trustedProxies, peer) {
				fromTrusted.ServeHTTP(w, r) // trusted proxy: honor the forwarding headers
				return
			}
			// untrusted peer: drop the forwarding headers and pin RemoteAddr to the real socket IP,
			// so nothing downstream can be fooled by a spoofed header (R.RealIP normalizes
			// RemoteAddr to a bare IP for trusted peers; do the same here for consistency)
			for _, h := range ipForwardingHeaders {
				r.Header.Del(h)
			}
			if peer != nil {
				r.RemoteAddr = peer.String()
			}
			next.ServeHTTP(w, r)
		})
	}
}

// directPeerIP extracts the IP from a "host:port" (or bare host) RemoteAddr, or nil if unparseable.
func directPeerIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr // may already be a bare IP with no port
	}
	return net.ParseIP(host)
}

// TrustsAnyPeer reports whether the trusted-proxy list contains a catch-all (0.0.0.0/0 or ::/0),
// which trusts forwarding headers from every client and re-opens the IP-spoofing bypass.
func TrustsAnyPeer(cidrs []*net.IPNet) bool {
	for _, c := range cidrs {
		if ones, _ := c.Mask.Size(); ones == 0 {
			return true
		}
	}
	return false
}

// cidrsContain reports whether ip falls within any of the CIDRs.
func cidrsContain(cidrs []*net.IPNet, ip net.IP) bool {
	for _, c := range cidrs {
		if c.Contains(ip) {
			return true
		}
	}
	return false
}

// ParseTrustedProxies parses a list of trusted-proxy entries into CIDRs. Each entry may be a CIDR
// (e.g. 172.16.0.0/12) or a bare IP (treated as a single host). Blank entries are skipped; a
// malformed entry is a hard error so a typo can't silently disable proxy trust.
func ParseTrustedProxies(entries []string) ([]*net.IPNet, error) {
	var out []*net.IPNet
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if !strings.Contains(e, "/") { // bare IP -> single-host CIDR
			ip := net.ParseIP(e)
			if ip == nil {
				return nil, fmt.Errorf("invalid trusted proxy %q", e)
			}
			// build the network from the normalized IP so a v4-mapped IPv6 (e.g. ::ffff:10.0.0.1)
			// yields the intended /32 host, not a huge ::/32 range
			bits := 128
			if v4 := ip.To4(); v4 != nil {
				ip, bits = v4, 32
			}
			out = append(out, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			continue
		}
		_, network, err := net.ParseCIDR(e)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy CIDR %q: %w", e, err)
		}
		out = append(out, network)
	}
	return out, nil
}

// corsMiddleware builds the CORS middleware for the public API. With AllowedOrigins
// "*" and credentials enabled, rest.CORS reflects the request Origin into
// Access-Control-Allow-Origin (rather than a literal "*"), which browsers require
// for credentialed cross-origin requests.
func corsMiddleware() func(http.Handler) http.Handler {
	return R.CORS(
		R.CorsAllowedOrigins("*"),
		R.CorsAllowedMethods("GET", "POST", "PUT", "DELETE", "OPTIONS"),
		R.CorsAllowedHeaders("Accept", "Authorization", "Content-Type", "X-XSRF-Token", "X-JWT"),
		R.CorsExposedHeaders("Authorization"),
		R.CorsAllowCredentials(true),
		R.CorsMaxAge(300),
	)
}

// rejectHead rejects HEAD requests with 405, advertising the given allowed methods in
// the Allow header. net/http.ServeMux routes HEAD to a "GET ..." handler, but per RFC
// 9110 GET/HEAD are safe methods; this guard is applied to the few GET routes whose
// handlers mutate state so they cannot be triggered by a (nominally side-effect-free)
// HEAD, preserving the pre-routegroup behavior. allow lists every method the resource
// supports (e.g. "GET" or "GET, POST") so the 405 Allow header is accurate.
func rejectHead(allow string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodHead {
				w.Header().Set("Allow", allow)
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// rejectAnonUser is a middleware rejecting anonymous users
func rejectAnonUser(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user, err := rest.GetUserInfo(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if strings.HasPrefix(user.ID, "anonymous_") {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// matchSiteID is a middleware rejecting users with mismatch between site param and and User.SiteID
func matchSiteID(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user, err := rest.GetUserInfo(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// skip for basic auth user
		if user.Name == "admin" && user.ID == "admin" {
			next.ServeHTTP(w, r)
			return
		}

		siteID := r.URL.Query().Get("site")
		// require an explicit site so the user.SiteID check below cannot be bypassed
		// by simply omitting the query parameter
		if siteID == "" || user.SiteID != siteID {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// cacheControl is a middleware setting cache expiration. Using url+version as etag
func cacheControl(expiration time.Duration, version string) func(http.Handler) http.Handler {
	etag := func(r *http.Request, version string) string {
		s := version + ":" + r.URL.String()
		return store.EncodeID(s)
	}

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			e := `"` + etag(r, version) + `"`
			w.Header().Set("Etag", e)
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, no-cache", int(expiration.Seconds())))

			if match := r.Header.Get("If-None-Match"); match != "" {
				if strings.Contains(match, e) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// apiCSPMiddleware overrides the global Content-Security-Policy on /api/v1 routes
// with a strict, default-deny policy. The global CSP (securityHeadersMiddleware) keeps
// 'self' 'unsafe-inline' for script-src/style-src because the widget HTML pages
// (/web/*.html) need inline bootstrap blocks. API responses serve JSON, XML/RSS, or
// images — none of those should ever execute scripts when rendered, so they get the
// strictest policy available as defense-in-depth against future trust-boundary bugs.
//
// Image-serving handlers (/api/v1/img, /api/v1/picture/{user}/{id}) re-apply the same
// rest.StrictImageCSP value at the handler level and additionally set Content-Disposition:
// inline; filename="image" (framing the response as a file rather than a renderable
// document) and X-Content-Type-Options: nosniff. The CSP re-apply is intentional belt-and-
// braces: if a future route refactor bypasses this middleware, the image handlers still
// emit the policy.
func apiCSPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", rest.StrictImageCSP)
		next.ServeHTTP(w, r)
	})
}

// securityHeadersMiddleware sets security-related headers:
//   - Content-Security-Policy: controls which resources the browser is allowed to load
//   - Permissions-Policy: disables browser features (camera, mic, etc.) not needed by a comment widget
//   - X-Content-Type-Options: prevents browsers from MIME-sniffing responses away from the declared type,
//     stopping e.g. a user-uploaded image from being reinterpreted as executable HTML/JS
//   - Referrer-Policy: controls how much URL information leaks in the Referer header on cross-origin
//     requests; "strict-origin-when-cross-origin" sends only the origin (no path) to other domains
//     and nothing at all on HTTPS→HTTP downgrades
func securityHeadersMiddleware(imageProxyEnabled bool, allowedAncestors []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			imgSrc := "*"
			if imageProxyEnabled {
				imgSrc = "'self'"
			}
			frameAncestors := "*"
			if len(allowedAncestors) > 0 {
				frameAncestors = strings.Join(allowedAncestors, " ")
			}
			// font-src is set to 'none' (no @font-face / no base64 fonts in the bundle).
			w.Header().Set("Content-Security-Policy", fmt.Sprintf("default-src 'none'; base-uri 'none'; form-action 'none'; connect-src 'self'; frame-src 'self' mailto:; img-src %s; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; font-src 'none'; object-src 'none'; frame-ancestors %s;", imgSrc, frameAncestors))
			w.Header().Set("Permissions-Policy", "accelerometer=(), autoplay=(), camera=(), cross-origin-isolated=(), display-capture=(), encrypted-media=(), fullscreen=(), geolocation=(), gyroscope=(), keyboard-map=(), magnetometer=(), microphone=(), midi=(), payment=(), picture-in-picture=(), publickey-credentials-get=(), screen-wake-lock=(), sync-xhr=(), usb=(), xr-spatial-tracking=(), clipboard-read=(), clipboard-write=(), gamepad=(), hid=(), idle-detection=(), interest-cohort=(), serial=(), unload=(), window-management=()")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			next.ServeHTTP(w, r)
		})
	}
}

// subscribersOnly is a middleware rejecting non-paid_sub users
func subscribersOnly(enable bool) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if enable {
				user, err := rest.GetUserInfo(r)
				if err != nil {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				if !user.PaidSub {
					http.Error(w, "Access denied", http.StatusForbidden)
					return
				}
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// validEmailAuth is a middleware for auth endpoints for email method.
// it rejects login request if user, site or email are suspicious
func validEmailAuth() func(http.Handler) http.Handler {

	reUser := regexp.MustCompile(`^[\p{L}\d._\- ]{3,64}$`) // matches default ui side validation, adding min/max limitation
	reSite := regexp.MustCompile(`^[a-zA-Z\d\s_.-]{1,64}$`)

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			if r.URL.Path != "/auth/email/login" {
				// not email login, skip the check
				h.ServeHTTP(w, r)
				return
			}

			if u := r.URL.Query().Get("user"); u != "" {
				if !reUser.MatchString(u) {
					log.Printf("[WARN] suspicious user rejected: %s", u)
					http.Error(w, "Access denied", http.StatusForbidden)
					return
				}
			}

			if a := r.URL.Query().Get("address"); a != "" {
				if _, err := mail.ParseAddress(a); err != nil {
					log.Printf("[WARN] suspicious address rejected: %s", a)
					http.Error(w, "Access denied", http.StatusForbidden)
					return
				}
			}

			if s := r.URL.Query().Get("site"); s != "" {
				if !reSite.MatchString(s) {
					log.Printf("[WARN] suspicious site rejected: %s", s)
					http.Error(w, "Access denied", http.StatusForbidden)
					return
				}
			}

			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// rateLimiter creates a rate limiting middleware with proper IP lookup configuration.
// tollbooth v8 requires explicit IP lookup method to be set.
// keys on RemoteAddr, which realIPMiddleware sets to the client IP (from the forwarding
// headers for trusted proxies, otherwise the real socket IP).
func rateLimiter(maxReq float64) func(http.Handler) http.Handler {
	lmt := tollbooth.NewLimiter(maxReq, nil)
	lmt.SetIPLookup(limiter.IPLookup{
		Name:           "RemoteAddr",
		IndexFromRight: 0,
	})
	return tollbooth.HTTPMiddleware(lmt)
}
