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
)

// Authenticator is top level auth object providing middlewares
type Authenticator struct {
	JWTService        *JWT
	Providers         []Provider
	AdminStore        admin.Store
	DevPasswd         string
	PermissionChecker PermissionChecker
}

var devUser = store.User{
	ID:      "dev",
	Name:    "developer one",
	Picture: "/api/v1/avatar/remark.image",
	Admin:   true,
}

// PermissionChecker defines interface to get user flags
type PermissionChecker interface {
	IsVerified(siteID, userID string) bool
	IsBlocked(siteID, userID string) bool
	IsAdmin(siteID, userID string) bool
}

// Auth middleware adds auth from session and populates user info
func (a *Authenticator) Auth(reqAuth bool) func(http.Handler) http.Handler {

	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			if a.basicDevUser(w, r) { // fail-back to dev user if enabled
				user := devUser
				r = rest.SetUserInfo(r, user)
				h.ServeHTTP(w, r)
				return
			}

			claims, err := a.JWTService.Get(r)
			if err != nil && reqAuth { // in full auth lack of session causes Unauthorized
				log.Printf("[DEBUG] failed auth, %s", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if err != nil { // in anonymous mode just pass it to the next handler
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

				if a.JWTService.HasFlags(claims) {
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

// AdminOnly allows access to admins
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

func (a *Authenticator) basicDevUser(w http.ResponseWriter, r *http.Request) bool {

	if a.DevPasswd == "" {
		return false
	}

	log.Printf("[DEBUG] dev user auth")
	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 {
		log.Printf("[WARN] dev user auth failed, incorrect auth header %s", r.Header.Get("Authorization"))
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
