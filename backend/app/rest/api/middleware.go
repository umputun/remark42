// Package api middleware: request-scoped HTTP middlewares used by the REST router.
package api

import (
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/didip/tollbooth/v8"
	"github.com/didip/tollbooth/v8/limiter"
	"github.com/go-pkgz/auth/v2/token"
	log "github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/umputun/remark42/backend/app/rest"
	"github.com/umputun/remark42/backend/app/store"
)

// ipForwardingHeaders are the request headers R.RealIP and realip.Get derive the client IP from.
var ipForwardingHeaders = []string{"X-Real-IP", "X-Forwarded-For", "CF-Connecting-IP"}

// realIPMiddleware sets RemoteAddr to the client IP that per-IP controls (rate limiting, vote dedup,
// comment IP, anonymous id) and the request log key on. With trusted proxies configured, the address
// comes from the forwarding headers only when the direct peer is one of them, see forwardedClientIP;
// any other peer is known by its socket IP. Either way the forwarding headers are dropped from the
// request, so nothing downstream can resolve another address from a header the client sent.
//
// With no trusted proxies configured it falls back to trusting the headers from any client (the
// historical behavior). That is spoofable by design, so operators running behind a reverse proxy
// should set --trusted-proxy to the proxy's address — see the "trusted proxy" docs.
func realIPMiddleware(trustedProxies []*net.IPNet) func(http.Handler) http.Handler {
	if len(trustedProxies) == 0 {
		return R.RealIP
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peer := directPeerIP(r.RemoteAddr)
			if peer != nil {
				r.RemoteAddr = peer.String() // bare IP, with the port stripped
				if cidrsContain(trustedProxies, peer) {
					r.RemoteAddr = forwardedClientIP(r, trustedProxies, peer)
				}
			}
			for _, h := range ipForwardingHeaders {
				r.Header.Del(h)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// forwardedClientIP returns the client IP a trusted proxy reports, reading only what a trusted proxy
// wrote. X-Real-IP is taken as sent, private and trusted addresses included: the proxy sets it to the
// address it accepted the connection from. Without it, X-Forwarded-For is read right to left while
// each entry is a trusted proxy and the first other entry is the client; if every entry is trusted,
// the leftmost one is. A malformed entry ends the walk at the peer, so a client can't lead it past
// such an entry to a forged one. CF-Connecting-IP is not read, a proxy relays it from any client.
func forwardedClientIP(r *http.Request, trustedProxies []*net.IPNet, peer net.IP) string {
	if xRealIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); net.ParseIP(xRealIP) != nil {
		return xRealIP
	}
	// a client can send the header on a line of its own, the lines are one list in the order received
	hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	client := ""
	for _, hop := range slices.Backward(hops) {
		hop = strings.TrimSpace(hop)
		ip := net.ParseIP(hop)
		if ip == nil { // malformed or empty entry, or no header at all
			return peer.String()
		}
		if !cidrsContain(trustedProxies, ip) {
			return hop
		}
		client = hop
	}
	return client
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
//
// That combination is refused by default upstream, so it has to be asked for by name with
// CorsUnsafeAnyOriginWithCredentials. The wildcard stays because the comment widget is embedded on
// arbitrary third-party sites, which makes the set of origins unknowable. The consequence it carries
// is that any site a signed-in user visits can read authenticated responses, so state-changing
// requests have to keep being protected by something other than the origin: X-XSRF-Token for
// cookie sessions, rejectCrossSiteBasicAuth for Basic auth.
func corsMiddleware() func(http.Handler) http.Handler {
	return R.CORS(
		R.CorsAllowedOrigins("*"),
		R.CorsAllowedMethods("GET", "POST", "PUT", "DELETE", "OPTIONS"),
		R.CorsAllowedHeaders("Accept", "Authorization", "Content-Type", "X-XSRF-Token", "X-JWT"),
		R.CorsExposedHeaders("Authorization"),
		R.CorsAllowCredentials(true),
		R.CorsUnsafeAnyOriginWithCredentials(true),
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

// rejectCrossSiteBasicAuth refuses Basic-auth requests a browser marks as cross-site or sends
// from another origin. Basic auth skips the XSRF check, and the reflected CORS policy would let
// any page drive the API with credentials a browser has cached. CLI tools send neither header.
func (s *Rest) rejectCrossSiteBasicAuth(next http.Handler) http.Handler {
	own := ""
	if u, err := url.Parse(s.RemarkURL); err == nil && u.Scheme != "" && u.Host != "" {
		host := u.Host
		switch u.Scheme { // browsers leave the default port out of Origin
		case "https":
			host = strings.TrimSuffix(host, ":443")
		case "http":
			host = strings.TrimSuffix(host, ":80")
		}
		own = u.Scheme + "://" + host
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := r.BasicAuth(); !ok {
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" || (origin != "" && !strings.EqualFold(origin, own)) {
			log.Printf("[WARN] cross-site basic auth request rejected, origin %q, %s %s", origin, r.Method, r.URL.Path)
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// cacheControl is a middleware setting cache expiration and a validator. The etag is built from the
// version, the url and the fingerprint of what the request is answered with, and a request whose
// fingerprint is empty gets no etag, since there is nothing to validate it against
func cacheControl(expiration time.Duration, version string, fingerprint func(*http.Request) string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, no-cache", int(expiration.Seconds())))

			content := fingerprint(r)
			if content == "" {
				h.ServeHTTP(w, r)
				return
			}
			e := `"` + store.EncodeID(version+":"+r.URL.String()+":"+content) + `"`
			w.Header().Set("Etag", e)

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

// subscribersOnly is a middleware rejecting users without a paid status that is still running
func subscribersOnly(enable bool) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if enable {
				user, err := token.GetUserInfo(r)
				if err != nil {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				if !rest.PaidSubActive(user, time.Now()) {
					http.Error(w, "Access denied", http.StatusForbidden)
					return
				}
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// QueryTokenParam is the query parameter the auth library is told to read a token from. It has to
// be given a name, because left empty it falls back to "token", which carries the delete-me token.
// The parameter is taken out of every request before the library sees it, see ignoreQueryToken.
const QueryTokenParam = "jwt"

// ignoreQueryToken removes the token parameter from the query, so a token is taken from the X-JWT
// header or from the cookie and never from the URL. The auth library reads one in the query ahead
// of both, without the XSRF check, and when the token is then rejected it expires the cookies of
// whoever sent the request. A URL is where somebody else can put a token: as the source of an
// image in a comment it is requested by every reader's browser, with that reader's cookies.
// The rest of the query is passed on as it was sent.
func ignoreQueryToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			r.URL.RawQuery = withoutQueryParam(r.URL.RawQuery, QueryTokenParam)
		}
		next.ServeHTTP(w, r)
	})
}

// withoutQueryParam returns the raw query with every pair of the given key removed and everything
// else left byte for byte. A query without the key is returned as it is: a new one is built only
// when there is a pair to leave out.
func withoutQueryParam(rawQuery, key string) string {
	isKey := func(pair string) bool {
		name, _, _ := strings.Cut(pair, "=")
		if name == key {
			return true
		}
		if !strings.ContainsAny(name, "%+") {
			return false // nothing to unescape, so it is some other name
		}
		unescaped, err := url.QueryUnescape(name)
		return err == nil && unescaped == key
	}

	var kept strings.Builder
	found, separate := false, false
	for left, start := rawQuery, 0; ; {
		pair, tail, more := strings.Cut(left, "&")
		switch {
		case isKey(pair):
			if !found { // the first one: everything before it is kept
				found, separate = true, start > 0
				kept.Grow(len(rawQuery))
				kept.WriteString(strings.TrimSuffix(rawQuery[:start], "&"))
			}
		case found:
			if separate {
				kept.WriteByte('&')
			}
			kept.WriteString(pair)
			separate = true
		}
		if !more {
			break
		}
		start += len(pair) + 1
		left = tail
	}
	if !found {
		return rawQuery
	}
	return kept.String()
}

// rejectEmbedded refuses a request the browser makes for an embedded resource: an image, a script,
// a stylesheet, a frame. Browsers name what a request is for in Sec-Fetch-Dest, "empty" for a call
// made from script and "document" for a page or a window, and those two are how the widget and the
// sign-in windows reach the auth routes. The routes answer a plain GET by setting or expiring the
// session cookies, so one of their URLs must not work as the source of an image in a comment.
// A client that does not send the header, as command-line tools do not, is let through.
func rejectEmbedded(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Sec-Fetch-Dest") {
		case "", "empty", "document":
			next.ServeHTTP(w, r)
		default:
			http.Error(w, "Access denied", http.StatusForbidden)
		}
	})
}

// authRoutePrefix is where the auth handler is mounted
const authRoutePrefix = "/auth/"

// the calls the auth handler answers: authActions without a provider in the path and
// providerAuthActions for a provider
var (
	authActions         = []string{"list", "logout", "status", "user"}
	providerAuthActions = []string{"login", "callback"}
)

// singleAuthPath is a middleware for the auth routes that leaves every endpoint of the auth handler
// one path. The handler takes the action from the last element of the path and the provider from
// the one before it, whatever stands in front of them, so it answers /auth/x/email/login as the
// email sign-in and /auth/email/status as /auth/status. The rate limiter counts requests by path
// and validEmailAuth compares the path, so each such path had a limit of its own and skipped the
// check. Only /auth/<action> for the four calls without a provider and /auth/<provider>/<action>
// for a registered provider's sign-in and callback pass, anything else gets 404. The path checked
// is the decoded one, which the handler, the limiter and validEmailAuth read, so an encoded slash
// makes no difference.
func singleAuthPath(registered func(provider string) bool) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			elems := strings.Split(strings.TrimPrefix(r.URL.Path, authRoutePrefix), "/")
			known := false
			switch len(elems) {
			case 1:
				known = slices.Contains(authActions, elems[0])
			case 2:
				known = registered(elems[0]) && slices.Contains(providerAuthActions, elems[1])
			}
			if !known {
				http.NotFound(w, r)
				return
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// isPlainEmail reports whether s is an email address and nothing else. net/mail parses a mailbox
// the way a mail header writes it, so it also takes a display name, a comment, a group and a
// quoted local part, and the free text of those would end up in the mail sent to the address
func isPlainEmail(s string) bool {
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return false
	}
	// net/mail lets any non-ASCII rune into an address, spaces and invisible ones included
	return !strings.ContainsFunc(s, func(r rune) bool { return !unicode.IsPrint(r) })
}

// validEmailAuth is a middleware for auth endpoints for email method.
// it rejects login request if user, site or email are suspicious
func validEmailAuth() func(http.Handler) http.Handler {

	reSite := regexp.MustCompile(`^[a-zA-Z\d\s_.-]{1,64}$`)

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			if r.URL.Path != authRoutePrefix+"email/login" {
				// not email login, skip the check
				h.ServeHTTP(w, r)
				return
			}

			if u := r.URL.Query().Get("user"); u != "" {
				// the rule of anonymous login, without dots: the name goes into the confirmation mail to an
				// address nobody has confirmed yet, where a dotted name can be a domain a mail client links
				if store.CheckUserName(u) != nil || strings.Contains(u, ".") {
					log.Printf("[WARN] suspicious user rejected: %s", store.MaskForLog(u))
					http.Error(w, "Access denied", http.StatusForbidden)
					return
				}
			}

			if a := r.URL.Query().Get("address"); a != "" {
				if _, err := mail.ParseAddress(a); err != nil {
					log.Printf("[WARN] suspicious address rejected: %s", store.MaskForLog(err.Error())) // it can quote the input
					http.Error(w, "Access denied", http.StatusForbidden)
					return
				}
				if !isPlainEmail(a) {
					log.Print("[WARN] suspicious address rejected: more than an address")
					http.Error(w, "Access denied", http.StatusForbidden)
					return
				}
			}

			if s := r.URL.Query().Get("site"); s != "" {
				if !reSite.MatchString(s) {
					log.Printf("[WARN] suspicious site rejected: %s", store.MaskForLog(s))
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
//
// tollbooth echoes the client's address and X-Forwarded-For in X-Rate-Limit-Request-* headers on every
// response. The picture, avatar and image proxy responses are cacheable by shared caches, which would then
// hand one visitor's address to the next, so the two headers are removed: on a refused request before
// tollbooth writes the 429, on a passed one before the next handler runs.
func rateLimiter(maxReq float64) func(http.Handler) http.Handler {
	lmt := tollbooth.NewLimiter(maxReq, nil)
	lmt.SetIPLookup(limiter.IPLookup{
		Name:           "RemoteAddr",
		IndexFromRight: 0,
	})
	dropClientAddr := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Del("X-Rate-Limit-Request-Remote-Addr")
		w.Header().Del("X-Rate-Limit-Request-Forwarded-For")
	}
	lmt.SetOnLimitReached(dropClientAddr)
	limit := tollbooth.HTTPMiddleware(lmt)
	return func(next http.Handler) http.Handler {
		return limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dropClientAddr(w, r)
			next.ServeHTTP(w, r)
		}))
	}
}

// recoverer is the panic recovery middleware. Its first line carries the request URL, which holds the address
// during email sign-in, so that line is masked
func recoverer(l log.L) func(http.Handler) http.Handler {
	return R.Recoverer(panicLog{l: l})
}

// panicLog masks the recovery line and leaves the stack trace, logged as a bare string, as it is:
// operators paste it into bug reports, and module paths in it can look like addresses
type panicLog struct {
	l log.L
}

// Logf writes the line to l, masked when it is formatted from arguments
func (p panicLog) Logf(format string, args ...any) {
	if len(args) == 0 {
		p.l.Logf("%s", format)
		return
	}
	rest.MaskedLog{L: p.l}.Logf(format, args...)
}

// partitionedCookieHeader is what a client sends, with the value "1", to be given its auth cookies
// in the partitioned form as well. The widget sends it when the page framing it is on another host
const partitionedCookieHeader = "X-Remark42-Partitioned-Cookie"

// partitionedAuthCookies gives a widget framed by another host a session it can keep. With issue
// set, for a request carrying partitionedCookieHeader, every Set-Cookie for one of the named cookies
// is repeated in its third-party form: Partitioned, Secure and SameSite=None. A browser that blocks
// third-party cookies drops the original and keeps the copy, keyed to the embedding site.
//
// The copy goes before the original, so a browser that does not know Partitioned reads both lines
// as one cookie and ends up with the original and its configured SameSite.
//
// An expiry is repeated whoever asks. A partitioned cookie is also sent to a top-level page of the
// site it is keyed to, so signing out on remark42's own page has to clear the pair a widget framed
// by another host of that site was given, and that request does not carry the header.
// Nothing else changes in a response to a request without it.
//
// An expiry is repeated without issue as well. A reader may still hold a pair issued before the
// server stopped issuing them, and its JWT is HttpOnly, so only the server can clear it. Left
// behind by a sign-out, it is sent ahead of the cookie of the next session and that session is
// refused for as long as the stale one lasts.
func partitionedAuthCookies(issue bool, names ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			asked := issue && r.Header.Get(partitionedCookieHeader) == "1"
			next.ServeHTTP(&partitioningWriter{ResponseWriter: w, names: names, asked: asked}, r)
		})
	}
}

// partitioningWriter adds the partitioned copies just before the headers are sent
type partitioningWriter struct {
	http.ResponseWriter
	names []string
	asked bool // the request carried partitionedCookieHeader
	done  bool
}

func (p *partitioningWriter) WriteHeader(code int) {
	p.addCopies()
	p.ResponseWriter.WriteHeader(code)
}

func (p *partitioningWriter) Write(b []byte) (int, error) {
	p.addCopies()
	return p.ResponseWriter.Write(b)
}

// Flush sends the headers when nothing was written before it, so the copies have to be in by then
func (p *partitioningWriter) Flush() {
	p.addCopies()
	_ = http.NewResponseController(p.ResponseWriter).Flush()
}

// Unwrap lets http.ResponseController reach the writer below
func (p *partitioningWriter) Unwrap() http.ResponseWriter { return p.ResponseWriter }

func (p *partitioningWriter) addCopies() {
	if p.done {
		return
	}
	p.done = true

	lines := p.Header().Values("Set-Cookie")
	out := make([]string, 0, len(lines)*2)
	for _, line := range lines {
		c, err := http.ParseSetCookie(line)
		if err == nil && slices.Contains(p.names, c.Name) && !c.Partitioned && (p.asked || c.MaxAge < 0) {
			c.Partitioned, c.Secure, c.SameSite = true, true, http.SameSiteNoneMode
			out = append(out, c.String())
		}
		out = append(out, line)
	}
	if len(out) > len(lines) {
		p.Header()["Set-Cookie"] = out
	}
}

// writeStallTimeout is how long one chunk of a response may take to leave: a client that reads nothing for that
// long loses the connection, while a response the client keeps reading is written for as long as it takes, as an
// export is. On Linux, unsentLimit lets a blocked write return as the client reads: over loopback, at 8 KB a second,
// a write blocked for about 16 seconds, one per 128 KiB
const writeStallTimeout = 30 * time.Second

// writeStallChunk is the most a single write sends under one deadline, so a large write is timed by its progress
const writeStallChunk = 16 * 1024

// stalledWriteLimit cuts a response the client stops reading. Every request holds a slot of R.Throttle until its
// response is written, and the server has no write timeout, as an export can take minutes, so clients that never
// read their responses could otherwise hold all the slots. The write deadline is set before each chunk and cleared
// after it, as on HTTP/2 a deadline that passes resets the stream even with no write in progress, so a handler may
// take as long as it needs between writes. It is set once more when the handler returns, for the buffered rest the
// server writes after it
func stalledWriteLimit(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rc := http.NewResponseController(w)
			// a deadline left by the previous request on this connection must not cut this one
			_ = rc.SetWriteDeadline(time.Time{})
			defer func() { _ = rc.SetWriteDeadline(time.Now().Add(timeout)) }()
			next.ServeHTTP(&stallWriter{ResponseWriter: w, rc: rc, timeout: timeout}, r)
		})
	}
}

// stallWriter renews the write deadline before each chunk of a response
type stallWriter struct {
	http.ResponseWriter
	rc      *http.ResponseController
	timeout time.Duration
	err     error // the first failed write; the connection is gone after it
}

func (s *stallWriter) renew() { _ = s.rc.SetWriteDeadline(time.Now().Add(s.timeout)) }

func (s *stallWriter) clear() { _ = s.rc.SetWriteDeadline(time.Time{}) }

func (s *stallWriter) WriteHeader(code int) {
	s.renew()
	s.ResponseWriter.WriteHeader(code)
	s.clear()
}

func (s *stallWriter) Write(b []byte) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	written := 0
	for len(b) > 0 {
		chunk := b[:min(len(b), writeStallChunk)]
		s.renew()
		n, err := s.ResponseWriter.Write(chunk)
		written += n
		if err != nil {
			s.err = err
			return written, err
		}
		b = b[len(chunk):]
	}
	s.clear()
	return written, nil
}

// Flush sends what is buffered, under a fresh deadline
func (s *stallWriter) Flush() {
	s.renew()
	_ = s.rc.Flush()
	s.clear()
}

// Unwrap lets http.ResponseController reach the writer below
func (s *stallWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }
