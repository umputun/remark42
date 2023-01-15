package rest

import (
	"context"
	"crypto/subtle"
	"net/http"
)

const baContextKey = "authorizedWithBasicAuth"

// BasicAuth middleware requires basic auth and matches user & passwd with client-provided checker
func BasicAuth(checker func(user, passwd string) bool) func(http.Handler) http.Handler {

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			u, p, ok := r.BasicAuth()
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if !checker(u, p) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey(baContextKey), true)))
		}
		return http.HandlerFunc(fn)
	}
}

// BasicAuthWithUserPasswd middleware requires basic auth and matches user & passwd with client-provided values
func BasicAuthWithUserPasswd(user, passwd string) func(http.Handler) http.Handler {
	checkFn := func(reqUser, reqPasswd string) bool {
		matchUser := subtle.ConstantTimeCompare([]byte(user), []byte(reqUser))
		matchPass := subtle.ConstantTimeCompare([]byte(passwd), []byte(reqPasswd))
		return matchUser == 1 && matchPass == 1
	}
	return BasicAuth(checkFn)
}

// IsAuthorized returns true is user authorized.
// it can be used in handlers to check if BasicAuth middleware was applied
func IsAuthorized(ctx context.Context) bool {
	v := ctx.Value(contextKey(baContextKey))
	return v != nil && v.(bool)
}
