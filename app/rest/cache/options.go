package cache

import (
	"net/http"
	"strings"

	"github.com/umputun/remark/app/rest"
)

// Option func type
type Option func(lc *loadingCache) error

// MaxValSize functional option defines the largest value's size allowed to be cached
// By default it is 0, which means unlimited.
func MaxValSize(max int) Option {
	return func(lc *loadingCache) error {
		lc.maxValueSize = max
		return nil
	}
}

// MaxKeys functional option defines how many keys to keep.
// By default it is 0, which means unlimited.
func MaxKeys(max int) Option {
	return func(lc *loadingCache) error {
		lc.maxKeys = max
		return nil
	}
}

// PostFlushFn functional option defines how callback function called after each Flush.
func PostFlushFn(postFlushFn func()) Option {
	return func(lc *loadingCache) error {
		lc.postFlushFn = postFlushFn
		return nil
	}
}

// URLKey gets url from request to use it as cache key
// admins will have different keys in order to prevent leak of admin-only data to regular users
func URLKey(r *http.Request) string {
	adminPrefix := "admin!!"
	key := strings.TrimPrefix(r.URL.String(), adminPrefix)          // prevents attach with fake url to get admin view
	if user, err := rest.GetUserInfo(r); err == nil && user.Admin { // make separate cache key for admins
		key = adminPrefix + key
	}
	return key
}
