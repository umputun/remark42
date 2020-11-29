// Package middleware provides login middlewares:
// - Auth: adds auth from session and populates user info
// - Trace: populates user info if token presented
// - AdminOnly: restrict access to admin users only
package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/go-pkgz/auth/logger"
	"github.com/go-pkgz/auth/provider"
	"github.com/go-pkgz/auth/token"
)

// Authenticator is top level auth object providing middlewares
type Authenticator struct {
	logger.L
	JWTService   TokenService
	Providers    []provider.Service
	Validator    token.Validator
	AdminPasswd  string
	RefreshCache RefreshCache
}

// RefreshCache defines interface storing and retrieving refreshed tokens
type RefreshCache interface {
	Get(key interface{}) (value interface{}, ok bool)
	Set(key, value interface{})
}

// TokenService defines interface accessing tokens
type TokenService interface {
	Parse(tokenString string) (claims token.Claims, err error)
	Set(w http.ResponseWriter, claims token.Claims) (token.Claims, error)
	Get(r *http.Request) (claims token.Claims, token string, err error)
	IsExpired(claims token.Claims) bool
	Reset(w http.ResponseWriter)
}

// adminUser sets claims for an optional basic auth
var adminUser = token.User{
	ID:   "admin",
	Name: "admin",
	Attributes: map[string]interface{}{
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
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}

	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			// use admin user basic auth if enabled
			if a.basicAdminUser(r) {
				r = token.SetUserInfo(r, adminUser)
				h.ServeHTTP(w, r)
				return
			}

			claims, tkn, err := a.JWTService.Get(r)
			if err != nil {
				onError(h, w, r, errors.Wrap(err, "can't get token"))
				return
			}

			if claims.Handshake != nil { // handshake in token indicate special use cases, not for login
				onError(h, w, r, errors.New("invalid kind of token"))
				return
			}

			if claims.User == nil {
				onError(h, w, r, errors.New("no user info presented in the claim"))
				return
			}

			if claims.User != nil { // if uinfo in token populate it to context
				// validator passed by client and performs check on token or/and claims
				if a.Validator != nil && !a.Validator.Validate(tkn, claims) {
					onError(h, w, r, errors.Errorf("user %s/%s blocked", claims.User.Name, claims.User.ID))
					a.JWTService.Reset(w)
					return
				}

				if a.JWTService.IsExpired(claims) {
					if claims, err = a.refreshExpiredToken(w, claims, tkn); err != nil {
						a.JWTService.Reset(w)
						onError(h, w, r, errors.Wrap(err, "can't refresh token"))
						return
					}
				}

				r = token.SetUserInfo(r, *claims.User) // populate user info to request context
			}

			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}

// refreshExpiredToken makes a new token with passed claims
func (a *Authenticator) refreshExpiredToken(w http.ResponseWriter, claims token.Claims, tkn string) (token.Claims, error) {

	// cache refreshed claims for given token in order to eliminate multiple refreshes for concurrent requests
	if a.RefreshCache != nil {
		if c, ok := a.RefreshCache.Get(tkn); ok {
			// already in cache
			return c.(token.Claims), nil
		}
	}

	claims.ExpiresAt = 0                  // this will cause now+duration for refreshed token
	c, err := a.JWTService.Set(w, claims) // Set changes token
	if err != nil {
		return token.Claims{}, err
	}

	if a.RefreshCache != nil {
		a.RefreshCache.Set(tkn, c)
	}

	a.Logf("[DEBUG] token refreshed for %+v", claims.User)
	return c, nil
}

// AdminOnly middleware allows access for admins only
// this handler internally wrapped with auth(true) to avoid situation if AdminOnly defined without prior Auth
func (a *Authenticator) AdminOnly(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user, err := token.GetUserInfo(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !user.IsAdmin() {
			http.Error(w, "Access denied", http.StatusForbidden)
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
		a.Logf("[WARN] admin basic auth failed, user/passwd mismatch, %s:%s", user, passwd)
		return false
	}

	return true
}

// RBAC middleware allows role based control for routes
// this handler internally wrapped with auth(true) to avoid situation if RBAC defined without prior Auth
func (a *Authenticator) RBAC(roles ...string) func(http.Handler) http.Handler {

	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			user, err := token.GetUserInfo(r)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
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
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}
			h.ServeHTTP(w, r)
		}
		return a.auth(true)(http.HandlerFunc(fn)) // enforce auth
	}
	return f
}
