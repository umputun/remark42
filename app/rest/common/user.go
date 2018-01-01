package common

import (
	"errors"
	"net/http"

	"github.com/umputun/remark/app/store"
)

// ContextKey is a type to match on context
type ContextKey string

// GetUserInfo returns user from request context
func GetUserInfo(r *http.Request) (user store.User, err error) {

	ctx := r.Context()
	if ctx == nil {
		return store.User{}, errors.New("no info about user")
	}

	if u, ok := ctx.Value(ContextKey("user")).(store.User); ok {
		return u, nil
	}

	return store.User{}, errors.New("user can't be parsed")
}
