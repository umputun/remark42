// Package auth provides oauth2 support as well as related middlewares.
package auth

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"

	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/admin"
	"github.com/umputun/remark/backend/app/store/keys"
)

// Authenticator is top level auth object providing middlewares
type Authenticator struct {
	JWTService        *JWT
	Providers         []Provider
	AdminStore        admin.Store
	KeyStore          keys.Store
	DevPasswd         string
	PermissionChecker PermissionChecker
}

var devUser = store.User{
	ID:      "dev",
	Name:    "developer one",
	Picture: "/api/v1/avatar/remark.image",
	Admin:   true,
}

var adminUser = store.User{
	ID:      "admin",
	Name:    "admin",
	Picture: "/api/v1/avatar/remark.image",
	Admin:   true,
}

// PermissionChecker defines interface to check user flags
type PermissionChecker interface {
	IsVerified(siteID, userID string) bool
	IsBlocked(siteID, userID string) bool
	IsAdmin(siteID, userID string) bool
}

// Auth middleware adds auth from session and populates user info
func (a *Authenticator) Auth(reqAuth bool) func(http.Handler) http.Handler {

	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			// if secret key matches for given site (from request) return admin user
			if a.checkSecretKey(r) {
				r = rest.SetUserInfo(r, adminUser)
				h.ServeHTTP(w, r)
				return
			}

			// use dev user basic auth if enabled
			if a.basicDevUser(r) {
				r = rest.SetUserInfo(r, devUser)
				h.ServeHTTP(w, r)
				return
			}

			claims, err := a.JWTService.Get(r)
			if err != nil {
				if reqAuth { // in full auth lack of token causes Unauthorized
					log.Printf("[DEBUG] failed auth, %s", err)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				// if !reqAuth just pass it to the next handler, used for information only, like logs
				h.ServeHTTP(w, r)
				return
			}

			if claims.User == nil && reqAuth {
				log.Print("[DEBUG] failed auth, no user info presented in the claim")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if claims.User != nil { // if uinfo in token populate it to context
				if claims.User.Blocked {
					log.Printf("[DEBUG] user %s/%s blocked", claims.User.Name, claims.User.ID)
					a.JWTService.Reset(w)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				if a.JWTService.HasFlags(claims) { // flags in token indicate special use cases, not for login
					log.Printf("[DEBUG] invalid token flags for %s/%s", claims.User.Name, claims.User.ID)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				if a.JWTService.IsExpired(claims) {
					if claims, err = a.refreshExpiredToken(w, claims); err != nil {
						log.Printf("[DEBUG] can't refresh jwt, %s", err)
						http.Error(w, "Unauthorized", http.StatusUnauthorized)
					}
					log.Printf("[DEBUG] token refreshed for %+v", claims.User)
				}
				r = rest.SetUserInfo(r, *claims.User) // populate user info to request context
			}

			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}

func (a *Authenticator) checkSecretKey(r *http.Request) bool {
	if a.KeyStore == nil {
		return false
	}

	siteID := r.URL.Query().Get("site")
	secret := r.URL.Query().Get("secret")

	skey, err := a.KeyStore.Get(siteID)
	if err != nil {
		return false
	}

	if strings.TrimSpace(secret) == "" || secret != skey {
		return false
	}
	return true
}

// refreshExpiredToken makes new token with passed claims, but only if permission allowed
func (a *Authenticator) refreshExpiredToken(w http.ResponseWriter, claims *CustomClaims) (*CustomClaims, error) {
	if a.PermissionChecker != nil {
		claims.User.Admin = a.PermissionChecker.IsAdmin(claims.SiteID, claims.User.ID)
		claims.User.Blocked = a.PermissionChecker.IsBlocked(claims.SiteID, claims.User.ID)
		claims.User.Verified = a.PermissionChecker.IsVerified(claims.SiteID, claims.User.ID)
	}
	// refresh token
	if err := a.JWTService.Set(w, claims, false); err != nil {
		return nil, err
	}
	return claims, nil
}

// AdminOnly middleware allows access for admins only
func (a *Authenticator) AdminOnly(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		user, err := rest.GetUserInfo(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !user.Admin {
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
		log.Printf("[WARN] dev user auth failed, failed to decode %s, %s", s[1], err)
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		log.Printf("[WARN] dev user auth failed, failed to split %s", string(b))
		return false
	}

	if pair[0] != "dev" || pair[1] != a.DevPasswd {
		log.Printf("[WARN] dev user auth failed, user/passwd mismatch %+v", pair)
		return false
	}

	return true
}
