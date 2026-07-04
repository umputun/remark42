package provider

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-pkgz/rest"
	"github.com/golang-jwt/jwt/v5"

	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/logger"
	"github.com/go-pkgz/auth/v2/token"
)

// VerifyHandler implements non-oauth2 provider authorizing users with some confirmation.
// can be email, IM or anything else implementing Sender interface
//
// Identity caveat: the local user id returned to the application is derived
// from the verified address (ProviderName + "_" + HashID(address)). The
// confirmation round-trip proves current control of the address at login
// time; it does not guarantee a stable+unique identity over time. The owner
// of an address can change without the address changing — employer
// offboarding, lapsed free-mail accounts, and recycled domains all hand
// control of an address to the next person who claims it. Integrators that
// need stable identity should map the verified address to a server-side
// immutable user id at first successful verify and key their records on
// that id, not on the value returned here. See the "Email-as-identity
// caveat" section of the README for guidance.
type VerifyHandler struct {
	logger.L
	ProviderName string
	TokenService VerifTokenService
	Issuer       string
	AvatarSaver  AvatarSaver
	Sender       Sender
	Template     string
	UseGravatar  bool

	// URL is the service's own root URL; its host is always permitted as
	// a "from" redirect target. Optional but recommended.
	URL string
	// AllowedRedirectHosts lists additional hostnames permitted as "from"
	// redirect targets. Setting this field enables host validation: the
	// host of URL is always implicit, and any other host must appear
	// here. Nil disables validation and preserves legacy permissive
	// behavior — any non-empty "from" value is honored.
	AllowedRedirectHosts token.AllowedHosts

	// ConfirmationStore enforces one-shot consumption of confirmation tokens.
	// When non-nil, a token cannot be redeemed twice within its TTL window.
	// Leave nil to keep the legacy behavior (token replayable until expiry).
	ConfirmationStore VerifConfirmationStore
}

// VerifConfirmationStore tracks consumed confirmation tokens to prevent replay.
// Implementations must be safe for concurrent use.
type VerifConfirmationStore interface {
	// MarkUsed records key as consumed and returns alreadyUsed=true if it was
	// already recorded. The implementation MUST retain the marker for at
	// least the supplied ttl, or return a non-nil err if it cannot --
	// dropping a marker before its ttl while the underlying JWT is still
	// valid reopens the replay window the store is meant to close. err
	// signals a backend failure (network, disk, capacity, etc.); callers
	// MUST treat a non-nil err as fail-closed (reject the redemption).
	//
	// Adapter authors: do NOT embed key (or any caller-supplied data) in
	// returned errors. The handler logs err on the fail-closed branch, and
	// although key is the SHA-256 of the raw token rather than the token
	// itself, it still uniquely identifies the live, unredeemed JWT in
	// log destinations. Wrap the underlying backend error with a generic
	// description (e.g. "redis SET failed: %w") instead.
	MarkUsed(key string, ttl time.Duration) (alreadyUsed bool, err error)
}

// VerifConfirmationStoreFunc is an adapter to use ordinary functions as
// VerifConfirmationStore, mirroring the SenderFunc / token.AllowedHostsFunc
// house pattern for closure-based config.
type VerifConfirmationStoreFunc func(key string, ttl time.Duration) (alreadyUsed bool, err error)

// MarkUsed calls f(key, ttl) to implement VerifConfirmationStore.
func (f VerifConfirmationStoreFunc) MarkUsed(key string, ttl time.Duration) (bool, error) {
	return f(key, ttl)
}

// NewInMemoryVerifStore returns a process-local default VerifConfirmationStore.
// Suitable for single-instance deployments. Multi-instance deployments behind
// a load balancer MUST supply a shared backend (e.g. Redis) -- otherwise an
// attacker who lands on a different instance from the legitimate user can
// replay the token there. The default's failure is silent: the request
// completes normally and no log indicates the protection was bypassed.
func NewInMemoryVerifStore() VerifConfirmationStore {
	return &inMemoryVerifStore{used: make(map[string]time.Time)}
}

type inMemoryVerifStore struct {
	mu          sync.Mutex
	used        map[string]time.Time // key -> expiry
	insertCount int
}

// inMemoryVerifStoreSweepEvery is the in-memory store's amortization cadence.
// Walking the whole map on every redemption is O(n) under a single mutex,
// which serializes the hot path. Sweeping every N inserts keeps the map size
// bounded by ~N + (concurrent redemptions during the gap) without holding
// the lock through a full walk on most calls. Declared as a var rather than
// a const so tests can lower it to exercise the sweep branch.
var inMemoryVerifStoreSweepEvery = 256

func (s *inMemoryVerifStore) MarkUsed(key string, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if exp, ok := s.used[key]; ok && exp.After(now) {
		return true, nil
	}
	// amortized eviction: walk the map only every Nth insert, not on every
	// hot-path call. The lookup above already rejects unexpired duplicates,
	// so worst-case staleness is bounded by N inserts between sweeps.
	s.insertCount++
	if s.insertCount >= inMemoryVerifStoreSweepEvery {
		s.insertCount = 0
		for k, exp := range s.used {
			if !exp.After(now) {
				delete(s.used, k)
			}
		}
	}
	s.used[key] = now.Add(ttl)
	return false, nil
}

// confirmationKey hashes the raw token so the store key length is bounded
// regardless of token size, and so the in-memory map doesn't retain the
// signed token itself.
func confirmationKey(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

// scrubTokenFromRequest returns a shallow clone of r with the "token" query
// parameter replaced by "<redacted>". rest.SendErrorJSON logs r.URL, and the
// fail-closed branches in LoginHandler fire while the confirmation JWT is
// still live (store didn't record consumption) -- a single log line equals
// an unredeemed magic link without this scrub.
func scrubTokenFromRequest(r *http.Request) *http.Request {
	if r == nil || r.URL == nil || r.URL.Query().Get("token") == "" {
		return r
	}
	rc := r.Clone(r.Context())
	q := rc.URL.Query()
	q.Set("token", "<redacted>")
	rc.URL.RawQuery = q.Encode()
	return rc
}

// Sender defines interface to deliver a verification message (email, IM, or anything else).
type Sender interface {
	Send(address, text string) error
}

// SenderFunc type is an adapter to allow the use of ordinary functions as Sender.
type SenderFunc func(address, text string) error

// Send calls f(address,text) to implement Sender interface
func (f SenderFunc) Send(address, text string) error {
	return f(address, text)
}

// VerifTokenService defines interface accessing tokens
type VerifTokenService interface {
	Token(claims token.Claims) (string, error)
	Parse(tokenString string) (claims token.Claims, err error)
	IsExpired(claims token.Claims) bool
	Set(w http.ResponseWriter, claims token.Claims) (token.Claims, error)
	Reset(w http.ResponseWriter)
}

// Name of the handler
func (e VerifyHandler) Name() string { return e.ProviderName }

// LoginHandler gets name and address from query, makes confirmation token and sends it to user.
// In case if confirmation token presented in the query uses it to create auth token.
//
// Consumption is final when ConfirmationStore is configured: the token is
// marked used before any further side effects (avatar fetch, token issuance),
// so a transient downstream failure burns the token and the user must request
// a new confirmation email rather than retry the same link. This trade-off
// keeps the replay check atomic with the security boundary.
func (e VerifyHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {

	// GET /login?site=site&user=name&address=someone@example.com
	tkn := r.URL.Query().Get("token")
	if tkn == "" { // no token, ask confirmation via email
		e.sendConfirmation(w, r)
		return
	}

	// confirmation token presented
	// GET /login?token=confirmation-jwt&sess=1
	confClaims, err := e.TokenService.Parse(tkn)
	if err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusForbidden, err, "failed to verify confirmation token")
		return
	}

	if e.TokenService.IsExpired(confClaims) {
		rest.SendErrorJSON(w, r, e.L, http.StatusForbidden, fmt.Errorf("expired"), "failed to verify confirmation token")
		return
	}

	store := e.ConfirmationStore
	// guard against a typed-nil VerifConfirmationStoreFunc: a non-nil
	// interface wrapping a nil func survives the != nil check above and
	// would panic at MarkUsed. Treat it as no store configured. Mirrors
	// the AllowedHostsFunc nil-guard in token/jwt.go.
	if fn, ok := store.(VerifConfirmationStoreFunc); ok && fn == nil {
		store = nil
	}
	if store != nil {
		ttl := time.Minute
		if confClaims.ExpiresAt != nil {
			if remaining := time.Until(confClaims.ExpiresAt.Time); remaining > 0 {
				ttl = remaining
			}
		}
		alreadyUsed, markErr := store.MarkUsed(confirmationKey(tkn), ttl)
		if markErr != nil {
			// fail-closed: a backend outage must not let attackers replay
			// tokens. Reject with the token scrubbed from the logged URL,
			// since on this branch the store did NOT record consumption so
			// the JWT in the URL is still live.
			rest.SendErrorJSON(w, scrubTokenFromRequest(r), e.L, http.StatusForbidden, markErr, "confirmation token store unavailable")
			return
		}
		if alreadyUsed {
			rest.SendErrorJSON(w, scrubTokenFromRequest(r), e.L, http.StatusForbidden, fmt.Errorf("token already used"), "confirmation token already consumed")
			return
		}
	}

	elems := strings.Split(confClaims.Handshake.ID, "::")
	if len(elems) != 2 {
		rest.SendErrorJSON(w, r, e.L, http.StatusBadRequest, fmt.Errorf("%s", confClaims.Handshake.ID), "invalid handshake token")
		return
	}
	user, address := elems[0], elems[1]
	sessOnly := r.URL.Query().Get("sess") == "1"

	u := token.User{
		Name: user,
		ID:   e.ProviderName + "_" + token.HashID(sha1.New(), address),
	}
	// try to get gravatar for email
	if e.UseGravatar && strings.Contains(address, "@") { // TODO: better email check to avoid silly hits to gravatar api
		if picURL, e := avatar.GetGravatarURL(address); e == nil {
			u.Picture = picURL
		}
	}

	if u, err = setAvatar(e.AvatarSaver, u, &http.Client{Timeout: 5 * time.Second}); err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusInternalServerError, err, "failed to save avatar to proxy")
		return
	}

	cid, err := randToken()
	if err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusInternalServerError, err, "can't make token id")
		return
	}

	claims := token.Claims{
		User: &u,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:       cid,
			Issuer:   e.Issuer,
			Audience: confClaims.Audience,
		},
		SessionOnly: sessOnly,
		AuthProvider: &token.AuthProvider{
			Name: e.ProviderName,
		},
	}

	if _, err = e.TokenService.Set(w, claims); err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusInternalServerError, err, "failed to set token")
		return
	}
	if confClaims.Handshake != nil && confClaims.Handshake.From != "" {
		if !isAllowedRedirect(confClaims.Handshake.From, e.URL, e.AllowedRedirectHosts) {
			e.Logf("[WARN] rejected redirect to disallowed host: %s", redirectHostForLog(confClaims.Handshake.From))
			rest.RenderJSON(w, claims.User)
			return
		}
		http.Redirect(w, r, confClaims.Handshake.From, http.StatusTemporaryRedirect)
		return
	}
	rest.RenderJSON(w, claims.User)
}

// GET /login?site=site&user=name&address=someone@example.com
func (e VerifyHandler) sendConfirmation(w http.ResponseWriter, r *http.Request) {

	user, address, site := r.URL.Query().Get("user"), r.URL.Query().Get("address"), r.URL.Query().Get("site")

	if user == "" || address == "" {
		rest.SendErrorJSON(w, r, e.L, http.StatusBadRequest, fmt.Errorf("wrong request"), "can't get user and address")
		return
	}

	claims := token.Claims{
		Handshake: &token.Handshake{
			State: "",
			ID:    user + "::" + address,
			// without copying "from" here the redirect validator at the
			// other end has nothing to validate or to redirect to. The
			// docs (and #275) advertise ?from=<url> on the verify login
			// path, but the original sendConfirmation never put it on
			// the handshake JWT, so production verify flows could never
			// honor from at all.
			From: r.URL.Query().Get("from"),
		},
		SessionOnly: r.URL.Query().Get("session") != "" && r.URL.Query().Get("session") != "0",
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  []string{site},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
			Issuer:    e.Issuer,
		},
		AuthProvider: &token.AuthProvider{
			Name: e.ProviderName,
		},
	}

	tkn, err := e.TokenService.Token(claims)
	if err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusForbidden, err, "failed to make login token")
		return
	}

	tmpl := msgTemplate
	if e.Template != "" {
		tmpl = e.Template
	}
	emailTmpl, err := template.New("confirm").Parse(tmpl)
	if err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusInternalServerError, err, "can't parse confirmation template")
		return
	}

	tmplData := struct {
		User    string
		Address string
		Token   string
		Site    string
	}{
		User:    trim(user),
		Address: trim(address),
		Token:   tkn,
		Site:    site,
	}
	buf := bytes.Buffer{}
	if err = emailTmpl.Execute(&buf, tmplData); err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusInternalServerError, err, "can't execute confirmation template")
		return
	}

	if err := e.Sender.Send(address, buf.String()); err != nil {
		rest.SendErrorJSON(w, r, e.L, http.StatusInternalServerError, err, "failed to send confirmation")
		return
	}

	rest.RenderJSON(w, rest.JSON{"user": user, "address": address})
}

// AuthHandler is a no-op for verify login — the flow has no provider callback.
func (e VerifyHandler) AuthHandler(http.ResponseWriter, *http.Request) {}

// LogoutHandler - GET /logout
func (e VerifyHandler) LogoutHandler(w http.ResponseWriter, _ *http.Request) {
	e.TokenService.Reset(w)
}

var msgTemplate = `
Confirmation for {{.User}} {{.Address}}, site {{.Site}}

Token: {{.Token}}
`

func trim(inp string) string {
	res := strings.ReplaceAll(inp, "\n", "")
	res = strings.TrimSpace(res)
	if len(res) > 128 {
		return res[:128]
	}
	return res
}
