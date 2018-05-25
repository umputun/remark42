// Package auth provides oauth2 support as well as related middlewares.
package auth

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"

	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/store"
)

// Authenticator is top level auth object providing middlewares
type Authenticator struct {
	JWTService *JWT
	Providers  []Provider
	Admins     []string
	DevPasswd  string
}

var devUser = store.User{
	ID:      "dev",
	Name:    "developer one",
	Picture: "/api/v1/avatar/remark.image",
	Admin:   true,
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
				user := *claims.User
				user.Admin = isAdmin(user.ID, a.Admins) // dbl-check for admin to reset admin flag even if token has it
				// refresh token if it close to expiration
				if _, err := a.JWTService.Refresh(w, r); err != nil {
					log.Printf("[DEBUG] can't refresh jwt, %s", err)
				}
				r = rest.SetUserInfo(r, user)
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
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

	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 {
		return false
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return false
	}

	if pair[0] != "dev" || pair[1] != a.DevPasswd {
		return false
	}

	return true
}

func isAdmin(userID string, admins []string) bool {
	for _, admin := range admins {
		if admin == userID {
			return true
		}
	}
	return false
}
