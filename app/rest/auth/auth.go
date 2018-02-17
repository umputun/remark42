package auth

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"

	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/store"
)

// Authenticator is top level auth object providing middlewares
type Authenticator struct {
	SessionStore sessions.Store
	AvatarProxy  *AvatarProxy
	Admins       []string
	Providers    []Provider

	DevEnabled bool
	DevPasswd  string
}

var devUser = store.User{
	ID:      "dev",
	Name:    "developer one",
	Picture: "https://friends.radio-t.com/resources/images/rt_logo_64.png",
	Profile: "https://radio-t.com/info/",
	Admin:   true,
}

// Auth middleware adds auth from session and populates user info
func (a *Authenticator) Auth(reqAuth bool) func(http.Handler) http.Handler {

	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			// dev user - skip regular auth check and populate dev to context
			if a.basicDevUser(w, r) {
				user := devUser
				ctx := r.Context()
				ctx = context.WithValue(ctx, rest.ContextKey("user"), user)
				r = r.WithContext(ctx)
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

			if ok { // if uinfo in session, populate to context
				user := uinfoData.(store.User)
				for _, admin := range a.Admins {
					if admin == user.ID {
						user.Admin = true
						break
					}
				}

				ctx := r.Context()
				ctx = context.WithValue(ctx, rest.ContextKey("user"), user)
				r = r.WithContext(ctx)
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

	if a.DevPasswd == "" || !a.DevEnabled {
		return false
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

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
