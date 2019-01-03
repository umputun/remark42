// Package middleware provides oauth2 support as well as related middlewares.
package middleware

import (
	"math/rand"
	"net/http"

	"github.com/pkg/errors"

	"github.com/go-pkgz/auth/logger"
	"github.com/go-pkgz/auth/provider"
	"github.com/go-pkgz/auth/token"
)

// Authenticator is top level auth object providing middlewares
type Authenticator struct {
	logger.L
	JWTService    TokenService
	Providers     []provider.Service
	Validator     token.Validator
	AdminPasswd   string
	RefreshFactor int
}

// TokenService defines interface accessing tokens
type TokenService interface {
	Parse(tokenString string) (claims token.Claims, err error)
	Set(w http.ResponseWriter, claims token.Claims) error
	Get(r *http.Request) (claims token.Claims, token string, err error)
	IsExpired(claims token.Claims) bool
	Reset(w http.ResponseWriter)
}

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
		if err == nil {
			return
		}
		if !reqAuth {
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
				onError(h, w, r, errors.New("failed auth, no user info presented in the claim"))
				return
			}

			if claims.User != nil { // if uinfo in token populate it to context
				// validator passed by client and performs check on token or/and claims
				if a.Validator != nil && !a.Validator.Validate(tkn, claims) {
					onError(h, w, r, errors.Errorf("user %s/%s blocked", claims.User.Name, claims.User.ID))
					a.JWTService.Reset(w)
					return
				}

				if a.shouldRefresh(claims) {
					if claims, err = a.refreshExpiredToken(w, claims); err != nil {
						a.JWTService.Reset(w)
						onError(h, w, r, errors.Wrap(err, "can't refresh token"))
						return
					}
					a.Logf("[DEBUG] token refreshed for %+v", claims.User)
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
func (a *Authenticator) refreshExpiredToken(w http.ResponseWriter, claims token.Claims) (token.Claims, error) {
	claims.ExpiresAt = 0 // this will cause now+duration for refreshed token
	if err := a.JWTService.Set(w, claims); err != nil {
		return token.Claims{}, err
	}
	return claims, nil
}

// shouldRefresh checks if token expired with an optional random rejection of refresh.
// the goal is to prevent multiple refresh request executed at the same time by allowing only some of them
func (a *Authenticator) shouldRefresh(claims token.Claims) bool {
	if !a.JWTService.IsExpired(claims) {
		return false
	}

	// disable randomizing with 0 factor
	if a.RefreshFactor == 0 {
		return true
	}

	return rand.Int31n(int32(a.RefreshFactor)) == 0 // randomize selection
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

	if user != "admin" || passwd != a.AdminPasswd {
		a.Logf("[WARN] admin basic auth failed, user/passwd mismatch, %s:%s", user, passwd)
		return false
	}

	return true
}
