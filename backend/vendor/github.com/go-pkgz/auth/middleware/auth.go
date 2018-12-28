// Package middleware provides oauth2 support as well as related middlewares.
package middleware

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	"github.com/go-pkgz/auth/provider"
	"github.com/go-pkgz/auth/token"
)

// Authenticator is top level token object providing middlewares
type Authenticator struct {
	JWTService *token.Service
	Providers  []provider.Service
	Validator  token.Validator
	DevPasswd  string
}

var devUser = token.User{
	ID:   "dev",
	Name: "developer one",
	Attributes: map[string]interface{}{
		"admin": true,
	},
}

var adminUser = token.User{
	ID:   "admin",
	Name: "admin",
	Attributes: map[string]interface{}{
		"admin": true,
	},
}

// Auth middleware adds token from session and populates user info
func (a *Authenticator) Auth(next http.Handler) http.Handler {
	return a.auth(true)(next)
}

// Trace middleware doesn't require valid user but if user info presented populates info
func (a *Authenticator) Trace(next http.Handler) http.Handler {
	return a.auth(false)(next)
}

func (a *Authenticator) auth(reqAuth bool) func(http.Handler) http.Handler {

	onError := func(h http.Handler, w http.ResponseWriter, r *http.Request, err error) {
		if err == nil {
			return
		}
		if !reqAuth {
			h.ServeHTTP(w, r)
			return
		}
		log.Printf("[DEBUG] failed token, %s", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}

	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			// if secret key matches for given site (from request) return admin user
			if a.checkSecretKey(r) {
				r = token.SetUserInfo(r, adminUser)
				h.ServeHTTP(w, r)
				return
			}

			// use dev user basic token if enabled
			if a.basicDevUser(r) {
				r = token.SetUserInfo(r, devUser)
				h.ServeHTTP(w, r)
				return
			}

			claims, tkn, err := a.JWTService.Get(r)
			if err != nil {
				onError(h, w, r, errors.Wrap(err, "can't get token"))
				return
			}

			if claims.Handshake != nil { // handshake in token indicate special use cases, not for login
				onError(h, w, r, errors.Errorf("invalid kind of token for %s/%s", claims.User.Name, claims.User.ID))
				return
			}

			if claims.User == nil {
				onError(h, w, r, errors.New("failed token, no user info presented in the claim"))
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
					log.Printf("[DEBUG] token refreshed for %+v", claims.User)
				}

				r = token.SetUserInfo(r, *claims.User) // populate user info to request context
			}

			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}

func (a *Authenticator) checkSecretKey(r *http.Request) bool {
	if a.JWTService.SecretReader == nil {
		return false
	}

	aud := r.URL.Query().Get("aud")
	secret := r.URL.Query().Get("secret")

	skey, err := a.JWTService.SecretReader.Get(aud)
	if err != nil {
		return false
	}

	if strings.TrimSpace(secret) == "" || secret != skey {
		return false
	}
	return true
}

// refreshExpiredToken makes new token with passed claims, but only if permission allowed
func (a *Authenticator) refreshExpiredToken(w http.ResponseWriter, claims token.Claims) (token.Claims, error) {
	// refresh token
	if err := a.JWTService.Set(w, claims, false); err != nil {
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

func (a *Authenticator) basicDevUser(r *http.Request) bool {

	if a.DevPasswd == "" {
		return false
	}

	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 {
		return false
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		log.Printf("[WARN] dev user token failed, failed to decode %s, %s", s[1], err)
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		log.Printf("[WARN] dev user token failed, failed to split %s", string(b))
		return false
	}

	if pair[0] != "dev" || pair[1] != a.DevPasswd {
		log.Printf("[WARN] dev user token failed, user/passwd mismatch %+v", pair)
		return false
	}

	return true
}
