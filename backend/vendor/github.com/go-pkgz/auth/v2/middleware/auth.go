// Package middleware provides login middlewares:
// - Auth: adds auth from session and populates user info
// - Trace: populates user info if token presented
// - AdminOnly: restrict access to admin users only
package middleware

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/go-pkgz/auth/v2/logger"
	"github.com/go-pkgz/auth/v2/provider"
	"github.com/go-pkgz/auth/v2/token"
)

// Authenticator is top level auth object providing middlewares
type Authenticator struct {
	logger.L
	JWTService       TokenService
	Providers        []provider.Service
	Validator        token.Validator
	AdminPasswd      string
	BasicAuthChecker BasicAuthFunc
	RefreshCache     RefreshCache
	ErrorHandler     ErrorHandlerFunc // custom error handler for auth failures
}

// RefreshCache defines interface storing and retrieving refreshed tokens
type RefreshCache interface {
	Get(key string) (value token.Claims, ok bool)
	Set(key string, value token.Claims)
}

// TokenService defines interface accessing tokens. On a refresh the middleware passes the claims Set returns
// through JSON before the Validator and the handler see them, so attribute values have the types Get gives
// on a later request: float64 for a number, or json.Number when the claims Get returned for the expired
// token carry one. For a service whose Get decodes numbers as json.Number, a number ClaimsUpd adds to a
// token whose attributes hold none is float64 on the request that refreshes it
type TokenService interface {
	Parse(tokenString string) (claims token.Claims, err error)
	Set(w http.ResponseWriter, claims token.Claims) (token.Claims, error)
	Get(r *http.Request) (claims token.Claims, token string, err error)
	IsExpired(claims token.Claims) bool
	Reset(w http.ResponseWriter)
}

// BasicAuthFunc type is an adapter to allow the use of ordinary functions as BasicAuth.
// The second return parameter `User` need for add user claims into context of request.
type BasicAuthFunc func(user, passwd string) (ok bool, userInfo token.User, err error)

// ErrorHandlerFunc type is an adapter to allow custom error handling for auth failures.
// It receives the suggested HTTP status code and the error that caused the auth failure.
// The handler can respond with custom status codes, HTML pages, redirects, or JSON responses.
// Status codes are typically http.StatusUnauthorized (401) for auth failures
// or http.StatusForbidden (403) for permission denied.
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, statusCode int, err error)

// adminUser sets claims for an optional basic auth
var adminUser = token.User{
	ID:   "admin",
	Name: "admin",
	Attributes: map[string]any{
		"admin": true,
	},
}

// Auth middleware adds auth from session and populates user info
func (a *Authenticator) Auth(next http.Handler) http.Handler {
	return a.auth(true)(next)
}

// Trace middleware doesn't require valid user but if user info presented populates info
func (a *Authenticator) Trace(next http.Handler) http.Handler {
	return a.auth(false)(next)
}

// auth implements all logic for authentication (reqAuth=true) and tracing (reqAuth=false)
func (a *Authenticator) auth(reqAuth bool) func(http.Handler) http.Handler {

	onError := func(h http.Handler, w http.ResponseWriter, r *http.Request, err error) {
		if !reqAuth { // if no auth required allow to proceeded on error
			h.ServeHTTP(w, r)
			return
		}
		a.Logf("[DEBUG] auth failed, %v", err)
		a.errResponse(w, r, http.StatusUnauthorized, err)
	}

	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			// use admin user basic auth if enabled but ignore when BasicAuthChecker defined
			if a.BasicAuthChecker == nil && a.basicAdminUser(r) {
				r = token.SetUserInfo(r, adminUser)
				h.ServeHTTP(w, r)
				return
			}

			// use custom basic auth if BasicAuthChecker defined
			if a.BasicAuthChecker != nil {
				if user, passwd, isBasicAuth := r.BasicAuth(); isBasicAuth {
					ok, userInfo, err := a.BasicAuthChecker(user, passwd)
					if err != nil {
						onError(h, w, r, fmt.Errorf("basic auth check failed: %w", err))
						return
					}
					if !ok {
						onError(h, w, r, fmt.Errorf("credentials are wrong for basic auth: %w", err))
						return
					}
					r = token.SetUserInfo(r, userInfo) // pass user claims into context of incoming request
					h.ServeHTTP(w, r)
					return
				}
			}

			claims, tkn, err := a.JWTService.Get(r)
			if err != nil {
				onError(h, w, r, fmt.Errorf("can't get token: %w", err))
				return
			}

			if claims.Handshake != nil { // handshake in token indicates special use cases, not for login
				onError(h, w, r, fmt.Errorf("invalid kind of token"))
				return
			}

			if claims.User == nil {
				onError(h, w, r, fmt.Errorf("no user info presented in the claim"))
				return
			}

			if claims.User != nil { // if uinfo in token populate it to context
				// validator passed by client and performs check on token or/and claims
				if a.Validator != nil && !a.Validator.Validate(tkn, claims) {
					a.resetCookies(w, r, tkn)
					onError(h, w, r, fmt.Errorf("user %s/%s blocked", claims.User.Name, claims.User.ID))
					return
				}

				// check if user provider is allowed
				if !a.isProviderAllowed(&claims) {
					a.resetCookies(w, r, tkn)
					onError(h, w, r, fmt.Errorf("user %s/%s provider is not allowed", claims.User.Name, claims.User.ID))
					return
				}

				if a.JWTService.IsExpired(claims) {
					refreshed, err := a.refreshExpiredToken(w, claims, tkn)
					if errors.Is(err, errRefreshRejected) {
						a.resetCookies(w, r, tkn)
						onError(h, w, r, fmt.Errorf("user %s/%s blocked", claims.User.Name, claims.User.ID))
						return
					}
					if err != nil {
						a.resetCookies(w, r, tkn)
						onError(h, w, r, fmt.Errorf("can't refresh token: %w", err))
						return
					}
					claims = refreshed
				}

				r = token.SetUserInfo(r, *claims.User) // populate user info to request context
			}

			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}

// isProviderAllowed checks if user provider is allowed.
// If provider name is explicitly set in the token claims, then that provider is checked.
//
// If user id looks like "provider_1234567890",
// then there is an attempt to extract provider name from that user ID.
// Note that such read can fail if user id has multiple "_" separator symbols.
//
// This check is needed to reject users from providers what are used to be allowed but not anymore.
// Such users made token before the provider was disabled and should not be allowed to login anymore.
func (a *Authenticator) isProviderAllowed(claims *token.Claims) bool {
	// TODO: remove this read when old tokens expire and all new tokens have a provider name in them
	userIDProvider, _, _ := strings.Cut(claims.User.ID, "_")
	for _, p := range a.Providers {
		name := p.Name()
		if claims.AuthProvider != nil && claims.AuthProvider.Name == name {
			return true
		}
		if name == userIDProvider {
			return true
		}
	}
	return false
}

// errRefreshRejected is returned by refreshExpiredToken when the Validator refuses the refreshed claims
var errRefreshRejected = errors.New("refreshed claims rejected")

// refreshExpiredToken makes a new token with passed claims. The refresh runs ClaimsUpd, which can change
// what the Validator decides, so the Validator checks the refreshed claims, with the token string of
// the request, before the new cookies reach w. Refused claims are neither written nor cached
func (a *Authenticator) refreshExpiredToken(w http.ResponseWriter, claims token.Claims, tkn string) (token.Claims, error) {

	// cache refreshed claims for given token in order to eliminate multiple refreshes for concurrent requests.
	// a copy per request, as UpdateUser and handlers change user attributes; an expired entry is refreshed again
	if a.RefreshCache != nil {
		if stored, ok := a.RefreshCache.Get(tkn); ok && !a.JWTService.IsExpired(stored) {
			cached, err := decodedClaims(stored, hasNumber(stored))
			if err != nil {
				return token.Claims{}, fmt.Errorf("can't read cached claims: %w", err)
			}
			if a.Validator != nil && !a.Validator.Validate(tkn, cached) {
				return token.Claims{}, errRefreshRejected
			}
			return cached, nil
		}
	}

	claims.ExpiresAt = nil // this will cause now+duration for refreshed token
	hw := &headerWriter{header: w.Header().Clone()}
	if hw.header == nil {
		hw.header = http.Header{}
	}
	// how Get decodes numbers, read before Set, whose ClaimsUpd can change the user of claims in place
	numbers := hasNumber(claims)
	c, err := a.JWTService.Set(hw, claims) // set changes token
	if err != nil {
		return token.Claims{}, err
	}
	if c, err = decodedClaims(c, numbers); err != nil {
		return token.Claims{}, fmt.Errorf("can't read refreshed claims: %w", err)
	}
	if c.User == nil {
		return token.Claims{}, fmt.Errorf("no user info in the refreshed claims")
	}
	if len(c.Audience) == 1 { // as Get does for a token it reads
		c.User.Audience = c.Audience[0]
	}
	if a.Validator != nil && !a.Validator.Validate(tkn, c) {
		return token.Claims{}, errRefreshRejected
	}
	var stored token.Claims
	if a.RefreshCache != nil {
		// the cache keeps a copy apart from the one this request goes on with
		if stored, err = decodedClaims(c, numbers); err != nil {
			return token.Claims{}, fmt.Errorf("can't copy refreshed claims: %w", err)
		}
	}
	// the headers as Set left them, as if it had written to w
	for k := range w.Header() {
		if _, ok := hw.header[k]; !ok {
			delete(w.Header(), k)
		}
	}
	maps.Copy(w.Header(), hw.header)

	if a.RefreshCache != nil {
		a.RefreshCache.Set(tkn, stored)
	}

	a.Logf("[DEBUG] token refreshed for %s", claims.User.ID)
	return c, nil
}

// decodedClaims returns the claims as a later request reads them from the token, through JSON. ClaimsUpd
// may have put in values of other types than JSON gives back, an int for a number or a []string for a
// list, and the Validator and the handler see the decoded types on every other request. With numbers set,
// a number is decoded as json.Number, for a TokenService whose Get gives attributes so; the caller sets it
// when the claims Get returned carry one, as the claims ClaimsUpd returns say nothing of how Get decodes.
// So for such a service a number ClaimsUpd adds to a token whose attributes hold none comes out as float64
// on the request that refreshes it. The result shares nothing with c, so a user ClaimsUpd shares is left alone
func decodedClaims(c token.Claims, numbers bool) (token.Claims, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return token.Claims{}, err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	if numbers {
		dec.UseNumber()
	}
	res := token.Claims{}
	if err = dec.Decode(&res); err != nil {
		return token.Claims{}, err
	}
	return res, nil
}

// hasNumber reports whether a user attribute of the claims holds a json.Number, at any depth. Only a
// TokenService that decodes numbers so, such as one parsing with jwt's UseJSONNumber, gives one
func hasNumber(c token.Claims) bool {
	if c.User == nil {
		return false
	}
	var walk func(v any) bool
	walk = func(v any) bool {
		switch v := v.(type) {
		case json.Number:
			return true
		case map[string]any:
			for _, e := range v {
				if walk(e) {
					return true
				}
			}
		case []any:
			if slices.ContainsFunc(v, walk) {
				return true
			}
		}
		return false
	}
	return walk(c.User.Attributes)
}

// headerWriter gives TokenService.Set a copy of the response headers, which replace the live ones only
// once the refreshed claims are accepted. Set writes headers only; a body or status written to it is dropped
type headerWriter struct {
	header http.Header
}

func (hw *headerWriter) Header() http.Header         { return hw.header }
func (hw *headerWriter) Write(b []byte) (int, error) { return len(b), nil }
func (hw *headerWriter) WriteHeader(int)             {}

// resetCookies expires the auth cookies before a rejection is answered, but only when the rejected
// token is the request's own session cookie. JWTService.Get reads the token from the query or a header
// before the cookie; a rejected token from there belongs to whoever made the link or the request, and
// clearing the cookies would sign out the user who merely followed it. Reset also sets a text/plain
// Content-Type, meant for logout responses; here the response belongs to the error handler or, under
// Trace, to the wrapped handler, so the header is put back as it was
func (a *Authenticator) resetCookies(w http.ResponseWriter, r *http.Request, tkn string) {
	fromCookie := false
	for _, c := range r.Cookies() {
		if c.Value != "" && c.Value == tkn {
			fromCookie = true
			break
		}
	}
	if !fromCookie {
		return
	}
	ct, had := w.Header()["Content-Type"]
	a.JWTService.Reset(w)
	if had {
		w.Header()["Content-Type"] = ct
		return
	}
	w.Header().Del("Content-Type")
}

// AdminOnly middleware allows access for admins only
// this handler internally wrapped with auth(true) to avoid situation if AdminOnly defined without prior Auth
func (a *Authenticator) AdminOnly(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user, err := token.GetUserInfo(r)
		if err != nil {
			a.errResponse(w, r, http.StatusUnauthorized, err)
			return
		}

		if !user.IsAdmin() {
			a.errResponse(w, r, http.StatusForbidden, fmt.Errorf("user %s is not admin", user.Name))
			return
		}
		next.ServeHTTP(w, r)
	}
	return a.auth(true)(http.HandlerFunc(fn)) // enforce auth
}

// basic auth for admin user
func (a *Authenticator) basicAdminUser(r *http.Request) bool {

	if a.AdminPasswd == "" {
		return false
	}

	user, passwd, ok := r.BasicAuth()
	if !ok {
		return false
	}

	// using ConstantTimeCompare to avoid timing attack
	if user != "admin" || subtle.ConstantTimeCompare([]byte(passwd), []byte(a.AdminPasswd)) != 1 {
		a.Logf("[WARN] admin basic auth failed for user %q", user)
		return false
	}

	return true
}

// errResponse calls ErrorHandler if set, otherwise returns default http error
func (a *Authenticator) errResponse(w http.ResponseWriter, r *http.Request, code int, err error) {
	if a.ErrorHandler != nil {
		a.ErrorHandler(w, r, code, err)
		return
	}
	// preserve original error messages for backward compatibility
	switch code {
	case http.StatusForbidden:
		http.Error(w, "Access denied", code)
	default:
		http.Error(w, "Unauthorized", code)
	}
}

// RBAC middleware allows role based control for routes
// this handler internally wrapped with auth(true) to avoid situation if RBAC defined without prior Auth
func (a *Authenticator) RBAC(roles ...string) func(http.Handler) http.Handler {

	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			user, err := token.GetUserInfo(r)
			if err != nil {
				a.errResponse(w, r, http.StatusUnauthorized, err)
				return
			}

			var matched bool
			for _, role := range roles {
				if strings.EqualFold(role, user.Role) {
					matched = true
					break
				}
			}
			if !matched {
				a.errResponse(w, r, http.StatusForbidden, fmt.Errorf("user %s role %s not in allowed roles", user.Name, user.Role))
				return
			}
			h.ServeHTTP(w, r)
		}
		return a.auth(true)(http.HandlerFunc(fn)) // enforce auth
	}
	return f
}
