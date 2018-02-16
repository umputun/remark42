package auth

import (
	"context"
	"net/http"

	"github.com/gorilla/sessions"

	"github.com/umputun/remark/app/rest/avatar"
	"github.com/umputun/remark/app/rest/common"
	"github.com/umputun/remark/app/store"
)

// Authenticator is top level auth object providing middlewares
type Authenticator struct {
	SessionStore sessions.Store
	AvatarProxy  *avatar.Proxy
	Admins       []string
	Providers    []Provider
}

// Mode defines behavior of Auth middleware
type Mode int

// auth modes
const (
	Anonymous Mode = iota // propagates user info only, doesn't protect resource
	Developer             // fake dev auth, admin too
	Full                  // real auth
)

var devUser = store.User{
	ID:      "dev",
	Name:    "developer one",
	Picture: "https://friends.radio-t.com/resources/images/rt_logo_64.png",
	Profile: "https://radio-t.com/info/",
	Admin:   true,
}

// Auth middleware adds auth from session and populates user info
func (a *Authenticator) Auth(modes []Mode) func(http.Handler) http.Handler {

	inModes := func(mode Mode) bool {
		for _, m := range modes {
			if m == mode {
				return true
			}
		}
		return false
	}

	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			// for dev mode skip all real auth, make dev admin user
			if inModes(Developer) {
				user := devUser
				ctx := r.Context()
				ctx = context.WithValue(ctx, common.ContextKey("user"), user)
				r = r.WithContext(ctx)
				h.ServeHTTP(w, r)
				return
			}

			session, err := a.SessionStore.Get(r, "remark")
			if err != nil && inModes(Full) { // in full auth lack of session causes Unauthorized
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			if err != nil { // in any other mode just pass it to next handler
				h.ServeHTTP(w, r)
				return
			}

			uinfoData, ok := session.Values["uinfo"]
			if !ok && inModes(Full) { // return StatusUnauthorized for full auth mode only
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
				ctx = context.WithValue(ctx, common.ContextKey("user"), user)
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

		user, err := common.GetUserInfo(r)
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
