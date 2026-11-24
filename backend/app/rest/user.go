package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

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

// PaidSubPeriod is how long the paid status Patreon reports at a sign-in counts in subscribers-only mode.
// Patreon is asked only at the sign-in, and a session goes on for as long as it is used
const PaidSubPeriod = 14 * 24 * time.Hour

// paidSubUntilAttr is the token attribute with the end of the paid status, in unix seconds
const paidSubUntilAttr = "paid_sub_until"

// SetPaidSubUntil sets the end of the user's paid status
func SetPaidSubUntil(u *token.User, until time.Time) {
	u.SetStrAttr(paidSubUntilAttr, strconv.FormatInt(until.Unix(), 10))
}

// PaidSubActive tells whether the user is a paid subscriber whose paid status has not ended by now. A paid
// status without an end, from a token made before subscribers-only mode set one, does not count
func PaidSubActive(u token.User, now time.Time) bool {
	if !u.IsPaidSub() {
		return false
	}
	until, err := strconv.ParseInt(u.StrAttr(paidSubUntilAttr), 10, 64)
	return err == nil && now.Unix() < until
}

// GetUserOrEmpty attempts to get user info from request and returns empty object if failed
func GetUserOrEmpty(r *http.Request) store.User {
	user, err := GetUserInfo(r)
	if err != nil {
		return store.User{}
	}
	return user
}
