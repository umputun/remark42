package rest

import (
	"context"
	"net/http"
	"os"
	"runtime/debug"
	"strings"

	"github.com/go-pkgz/rest/logger"
	"github.com/go-pkgz/rest/realip"
)

// Wrap converts a list of middlewares to nested calls (in reverse order)
func Wrap(handler http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	return handler
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

// Ping middleware response with pong to /ping. Stops chain if ping request detected.
// Handles both GET and HEAD methods - HEAD returns headers only without body,
// which is useful for lightweight health checks by monitoring tools.
func Ping(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if (r.Method == "GET" || r.Method == "HEAD") && strings.HasSuffix(strings.ToLower(r.URL.Path), "/ping") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			if r.Method == "GET" {
				_, _ = w.Write([]byte("pong"))
			}
			return
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// Health middleware response with health info and status (200 if healthy). Stops chain if health request detected
// passed checkers implements custom health checks and returns error if health check failed. The check has to return name
// regardless to the error state.
// For production usage this middleware should be used with throttler and, optionally, with BasicAuth middlewares
func Health(path string, checkers ...func(ctx context.Context) (name string, err error)) func(http.Handler) http.Handler {

	type hr struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" || !strings.EqualFold(r.URL.Path, path) {
				h.ServeHTTP(w, r) // not the health check request, continue the chain
				return
			}
			resp := []hr{}
			var anyError bool
			for _, check := range checkers {
				name, err := check(r.Context())
				hh := hr{Name: name, Status: "ok"}
				if err != nil {
					hh.Status = "failed"
					hh.Error = err.Error()
					anyError = true
				}
				resp = append(resp, hh)
			}
			status := http.StatusOK
			if anyError {
				status = http.StatusServiceUnavailable
			}
			_ = EncodeJSON(w, status, resp)
		}
		return http.HandlerFunc(fn)
	}
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

// RealIP is a middleware that sets a http.Request's RemoteAddr to the client's real IP.
// It checks headers in the following priority order:
//  1. X-Real-IP - trusted proxy (nginx/reproxy) sets this to actual client
//  2. CF-Connecting-IP - Cloudflare's header for original client
//  3. X-Forwarded-For - leftmost public IP (original client in CDN/proxy chain)
//  4. RemoteAddr - fallback for direct connections
//
// Only public IPs are accepted from headers; private/loopback/link-local IPs are skipped.
//
// This middleware should only be used if user can trust the headers sent with request.
// If reverse proxies are configured to pass along arbitrary header values from the client,
// or if this middleware used without a reverse proxy, malicious clients could set anything
// as these headers and spoof their IP address.
func RealIP(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if rip, err := realip.Get(r); err == nil {
			r.RemoteAddr = rip
		}
		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// Reject is a middleware that conditionally rejects requests with a given status code and message.
// user-defined condition function rejectFn is used to determine if the request should be rejected.
func Reject(errCode int, errMsg string, rejectFn func(r *http.Request) bool) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if rejectFn(r) {
				http.Error(w, errMsg, errCode)
				return
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
