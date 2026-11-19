package rest

import (
	"fmt"
	"net/http"

	"github.com/go-pkgz/auth/v2/token"

	"github.com/umputun/remark42/backend/app/store"
)

// MustGetUserInfo fails if can't extract user data from the request.
// should be called from authed controllers only
func MustGetUserInfo(r *http.Request) store.User {
	user, err := GetUserInfo(r)
	if err != nil {
		panic(err)
	}
	return user
}

// GetUserInfo returns user from request context
func GetUserInfo(r *http.Request) (user store.User, err error) {
	u, err := token.GetUserInfo(r)
	if err != nil {
		return store.User{}, fmt.Errorf("can't extract user info from the token: %w", err)
	}

	return store.User{
		Name:     u.Name,
		ID:       u.ID,
		IP:       u.IP,
		Picture:  u.Picture,
		Admin:    u.IsAdmin(),
		Verified: u.BoolAttr("verified"),
		Blocked:  u.BoolAttr("blocked"),
		SiteID:   u.Audience,
		PaidSub:  u.IsPaidSub(),
	}, nil
}

// GetUserOrEmpty attempts to get user info from request and returns empty object if failed
func GetUserOrEmpty(r *http.Request) store.User {
	user, err := GetUserInfo(r)
	if err != nil {
		return store.User{}
	}
	return user
}
