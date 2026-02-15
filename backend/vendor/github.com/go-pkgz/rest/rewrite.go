package rest

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// CleanPath middleware cleans double slashes from URL path.
// For example, if a request is made to /users//1 or //users////1,
// it will be cleaned to /users/1 before routing.
// Trailing slashes are preserved: /users//1/ becomes /users/1/.
// Dot segments (. and ..) are intentionally NOT cleaned to preserve routing semantics.
func CleanPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()
		// skip if already cleaned
		if _, ok := rctx.Value(contextKey("cleanpath")).(bool); ok {
			next.ServeHTTP(w, r)
			return
		}

		p := r.URL.Path
		cleaned := cleanDoubleSlashes(p)

		if cleaned != p {
			r.URL.Path = cleaned
			if r.URL.RawPath != "" {
				// clean double slashes in RawPath separately to preserve percent-encoding
				r.URL.RawPath = cleanDoubleSlashes(r.URL.RawPath)
			}
			rctx = context.WithValue(rctx, contextKey("cleanpath"), true)
			r = r.WithContext(rctx)
		}
		next.ServeHTTP(w, r)
	})
}

// cleanDoubleSlashes removes consecutive slashes from path while preserving
// trailing slashes and dot segments (. and ..).
func cleanDoubleSlashes(p string) string {
	if p == "" || p == "/" {
		return p
	}

	var b strings.Builder
	b.Grow(len(p))

	prevSlash := false
	for i := 0; i < len(p); i++ {
		c := p[i]
		if c == '/' {
			if !prevSlash {
				b.WriteByte(c)
			}
			prevSlash = true
		} else {
			b.WriteByte(c)
			prevSlash = false
		}
	}

	return b.String()
}

// StripSlashes middleware removes trailing slashes from URL path.
// For example, /users/1/ becomes /users/1.
// The root path "/" is preserved.
func StripSlashes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if len(p) > 1 && p[len(p)-1] == '/' {
			r.URL.Path = p[:len(p)-1]
			if r.URL.RawPath != "" {
				r.URL.RawPath = strings.TrimSuffix(r.URL.RawPath, "/")
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Rewrite middleware with from->to rule. Supports regex (like nginx) and prevents multiple rewrites
// example: Rewrite(`^/sites/(.*)/settings/$`, `/sites/settings/$1`
func Rewrite(from, to string) func(http.Handler) http.Handler {
	reFrom := regexp.MustCompile(from)

	f := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			ctx := r.Context()

			// prevent double rewrites
			if ctx != nil {
				if _, ok := ctx.Value(contextKey("rewrite")).(bool); ok {
					next.ServeHTTP(w, r)
					return
				}
			}

			if !reFrom.MatchString(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			ru := reFrom.ReplaceAllString(r.URL.Path, to)
			cru := path.Clean(ru)
			if strings.HasSuffix(ru, "/") { // don't drop trailing slash
				cru += "/"
			}
			u, e := url.Parse(cru)
			if e != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Header.Set("X-Original-URL", r.URL.RequestURI())
			r.URL.Path = u.Path
			r.URL.RawPath = u.RawPath
			if u.RawQuery != "" {
				r.URL.RawQuery = u.RawQuery
			}
			ctx = context.WithValue(ctx, contextKey("rewrite"), true)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
	return f
}
