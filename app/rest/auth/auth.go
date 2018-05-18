// Package auth provides oauth2 support as well as related middlewares.
package auth

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/pkg/errors"

	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/rest/proxy"
	"github.com/umputun/remark/app/store"
)

// Authenticator is top level auth object providing middlewares
type Authenticator struct {
	SessionStore sessions.Store
	AvatarProxy  *proxy.Avatar
	Admins       []string
	Providers    []Provider
	DevPasswd    string
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

			session, err := a.SessionStore.Get(r, "remark")
			if err != nil && reqAuth { // in full auth lack of session causes Unauthorized
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if err != nil { // in anonymous mode just pass it to next handler
				h.ServeHTTP(w, r)
				return
			}

			uinfoData, ok := session.Values["uinfo"]
			if !ok && reqAuth {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if xsrfError := a.checkXSRF(r, session); xsrfError != nil {
				if reqAuth {
					log.Printf("[WARN] %s", xsrfError.Error())
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				h.ServeHTTP(w, r) // in anonymous mode just pass it to next handler
				return
			}

			if ok { // if uinfo in session, populate to context
				user := uinfoData.(store.User)
				for _, admin := range a.Admins {
					if admin == user.ID {
						user.Admin = true
						break
					}
				}

				r = rest.SetUserInfo(r, user)
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}

func (a *Authenticator) checkXSRF(r *http.Request, session *sessions.Session) error {
	xsrfToken := r.Header.Get("X-XSRF-TOKEN")
	sessionToken, headerOk := session.Values["xsrf_token"]
	if !headerOk || xsrfToken == "" || sessionToken == nil {
		return errors.New(" no xsrf_token in session")
	}

	if xsrfToken != sessionToken {
		return errors.Errorf("xsrf header not matched session token, %q != %q", xsrfToken, sessionToken)
	}
	return nil
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
