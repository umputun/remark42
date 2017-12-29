package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/sessions"

	"github.com/umputun/remark/app/store"
)

type contextKey string

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
func Auth(sessionStore *sessions.FilesystemStore, admins []string, modes []Mode) func(http.Handler) http.Handler {

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
				ctx = context.WithValue(ctx, contextKey("user"), user)
				r = r.WithContext(ctx)
				h.ServeHTTP(w, r)
				return
			}

			session, err := sessionStore.Get(r, "remark")
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
				for _, admin := range admins {
					if admin == user.ID {
						user.Admin = true
						break
					}
				}

				ctx := r.Context()
				ctx = context.WithValue(ctx, contextKey("user"), user)
				r = r.WithContext(ctx)
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}

// AdminOnly allows access to admins
func AdminOnly(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		user, err := GetUserInfo(r)
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

// GetUserInfo extracts user, or and token from request's context
func GetUserInfo(r *http.Request) (user store.User, err error) {

	ctx := r.Context()
	if ctx == nil {
		return store.User{}, errors.New("no info about user")
	}

	if u, ok := ctx.Value(contextKey("user")).(store.User); ok {
		return u, nil
	}

	return store.User{}, errors.New("user can't be parsed")
}
