package rest

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/umputun/remark/backend/app/store"
)

type contextKey string

// MustGetUserInfo fails if can't extract user data from the request.
// should be called from authed controllers only
func MustGetUserInfo(r *http.Request) store.User {
	user, err := GetUserInfo(r)
	if err != nil {
		log.Fatalf("[ERROR] %s", err)
	}
	return user
}

// GetUserInfo returns user from request context
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

// SetUserInfo sets user into request context
func SetUserInfo(r *http.Request, user store.User) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, contextKey("user"), user)
	return r.WithContext(ctx)
}
