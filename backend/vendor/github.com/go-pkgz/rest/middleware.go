package rest

import (
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/go-pkgz/rest/logger"
	"github.com/go-pkgz/rest/realip"
)

// Wrap converts a list of middlewares to nested calls (in reverse order)
func Wrap(handler http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	if len(mws) == 0 {
		return handler
	}
	res := handler
	for i := len(mws) - 1; i >= 0; i-- {
		res = mws[i](res)
	}
	return res
}

// AppInfo adds custom app-info to the response header
func AppInfo(app, author, version string) func(http.Handler) http.Handler {
	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Author", author)
			w.Header().Set("App-Name", app)
			w.Header().Set("App-Version", version)
			if mhost := os.Getenv("MHOST"); mhost != "" {
				w.Header().Set("Host", mhost)
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}

// Ping middleware response with pong to /ping. Stops chain if ping request detected
func Ping(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "GET" && strings.HasSuffix(strings.ToLower(r.URL.Path), "/ping") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("pong"))
			return
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// Recoverer is a middleware that recovers from panics, logs the panic and returns a HTTP 500 status if possible.
func Recoverer(l logger.Backend) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					l.Logf("request panic for %s from %s, %v", r.URL.String(), r.RemoteAddr, rvr)
					if rvr != http.ErrAbortHandler {
						l.Logf(string(debug.Stack()))
					}
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// Headers middleware adds headers to request
func Headers(headers ...string) func(http.Handler) http.Handler {

	return func(h http.Handler) http.Handler {

		fn := func(w http.ResponseWriter, r *http.Request) {
			for _, h := range headers {
				elems := strings.Split(h, ":")
				if len(elems) != 2 {
					continue
				}
				r.Header.Set(strings.TrimSpace(elems[0]), strings.TrimSpace(elems[1]))
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// Maybe middleware will allow you to change the flow of the middleware stack execution depending on return
// value of maybeFn(request). This is useful for example if you'd like to skip a middleware handler if
// a request does not satisfy the maybeFn logic.
// borrowed from https://github.com/go-chi/chi/blob/master/middleware/maybe.go
func Maybe(mw func(http.Handler) http.Handler, maybeFn func(r *http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if maybeFn(r) {
				mw(next).ServeHTTP(w, r)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

// RealIP is a middleware that sets a http.Request's RemoteAddr to the results
// of parsing either the X-Forwarded-For or X-Real-IP headers.
//
// This middleware should only be used if user can trust the headers sent with request.
// If reverse proxies are configured to pass along arbitrary header values from the client,
// or if this middleware used without a reverse proxy, malicious clients could set anything
// as X-Forwarded-For header and attack the server in various ways.
func RealIP(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if rip, err := realip.Get(r); err == nil {
			r.RemoteAddr = rip
		}
		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
