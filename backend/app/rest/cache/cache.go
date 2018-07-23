package cache

import (
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/umputun/remark/backend/app/rest"
)

// LoadingCache defines interface for caching
type LoadingCache interface {
	Get(key string, siteID string, fn func() ([]byte, error)) (data []byte, err error)
	Flush(scopes ...string)
}

// Key makes full key from primary key and scopes
func Key(key string, scopes ...string) string {
	return strings.Join(scopes, "$$") + "@@" + key
}

// ParseKey gets compound key created by Key func and split it to the actual key and scopes
func ParseKey(fullKey string) (key string, scopes []string, err error) {
	elems := strings.Split(fullKey, "@@")
	if len(elems) != 2 {
		return "", nil, errors.Errorf("can't parse cache key %s", key)
	}
	scopes = strings.Split(elems[0], "$$")
	if len(scopes) == 1 && scopes[0] == "" {
		scopes = []string{}
	}
	key = elems[1]
	return key, scopes, nil
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
