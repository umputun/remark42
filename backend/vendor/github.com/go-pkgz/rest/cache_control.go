package rest

import (
	"crypto/sha1" //nolint not used for cryptography
	"fmt"
	"net/http"
	"strings"
	"time"
)

// CacheControl is a middleware setting cache expiration. Using url+version for etag
func CacheControl(expiration time.Duration, version string) func(http.Handler) http.Handler {

	etag := func(r *http.Request, version string) string {
		s := fmt.Sprintf("%s:%s", version, r.URL.String())
		return fmt.Sprintf("%x", sha1.Sum([]byte(s))) //nolint
	}

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			e := `"` + etag(r, version) + `"`
			w.Header().Set("Etag", e)
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, no-cache", int(expiration.Seconds())))

			if match := r.Header.Get("If-None-Match"); match != "" {
				if strings.Contains(match, e) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// CacheControlDynamic is a middleware setting cache expiration. Using url+ func(r) for etag
func CacheControlDynamic(expiration time.Duration, versionFn func(r *http.Request) string) func(http.Handler) http.Handler {

	etag := func(r *http.Request, version string) string {
		s := fmt.Sprintf("%s:%s", version, r.URL.String())
		return fmt.Sprintf("%x", sha1.Sum([]byte(s))) //nolint
	}

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			e := `"` + etag(r, versionFn(r)) + `"`
			w.Header().Set("Etag", e)
			w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d, no-cache", int(expiration.Seconds())))

			if match := r.Header.Get("If-None-Match"); match != "" {
				if strings.Contains(match, e) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
