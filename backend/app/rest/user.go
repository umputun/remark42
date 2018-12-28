package rest

import (
	"net/http"

	"github.com/go-pkgz/auth/token"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
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
		return store.User{}, errors.Wrap(err, "can't extract user info from the token")
	}

	return store.User{
		Name:     u.Name,
		ID:       u.ID,
		IP:       u.IP,
		Picture:  u.Picture,
		Admin:    u.IsAdmin(),
		Verified: u.BoolAttr("verified"),
		Blocked:  u.BoolAttr("blocked"),
	}, nil

}

// SetUserInfo sets user into request context
func SetUserInfo(r *http.Request, user store.User) *http.Request {
	u := token.User{
		ID:      user.ID,
		Name:    user.Name,
		Picture: user.Picture,
		IP:      user.IP,
		Attributes: map[string]interface{}{
			"blocked":  user.Blocked,
			"verified": user.Verified,
		},
	}
	u.SetAdmin(user.Admin)

	return token.SetUserInfo(r, u)
}
