// Package middleware provides oauth2 support as well as related middlewares.
package middleware

import (
	"encoding/base64"
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
	JWTService  *token.Service
	Providers   []provider.Service
	Validator   token.Validator
	AdminPasswd string
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
		a.Logf("[DEBUG] auth failed, %s", err)
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

				if a.JWTService.IsExpired(claims) {
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

// AdminOnly middleware allows access for admins only
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
	return http.HandlerFunc(fn)
}

// basic auth for admin user
func (a *Authenticator) basicAdminUser(r *http.Request) bool {

	if a.AdminPasswd == "" {
		return false
	}

	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 {
		return false
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		a.Logf("[WARN] admin user auth failed, can't to decode %s, %s", s[1], err)
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		a.Logf("[WARN] admin user auth failed, can't split basic auth %s", string(b))
		return false
	}

	if pair[0] != "admin" || pair[1] != a.AdminPasswd {
		a.Logf("[WARN] admin basic auth failed, user/passwd mismatch %+v", pair)
		return false
	}

	return true
}
