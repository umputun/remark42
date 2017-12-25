package rest

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/didip/tollbooth"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/go-errors/errors"
	"github.com/gorilla/sessions"

	"github.com/umputun/remark/app/store"
)

var org = "Umputun"

// JSON is a map alias, just for convenience
type JSON map[string]interface{}

// Limiter middleware defines max recs/sec for given client. Client detected as a combination
// of source IP, auth key and user agent.  Requests rejected with 429 status code.
func Limiter(recSec int, excludeIps ...string) func(http.Handler) http.Handler {

	return func(h http.Handler) http.Handler {
		l := tollbooth.NewLimiter(int64(recSec), time.Second)

		fn := func(w http.ResponseWriter, r *http.Request) {

			for _, exclIP := range excludeIps {
				if strings.HasPrefix(r.RemoteAddr, exclIP) {
					h.ServeHTTP(w, r)
					return
				}
			}

			keys := []string{
				r.Header.Get("RemoteAddr"),
				r.Header.Get("X-Forwarded-For"),
				r.Header.Get("X-Real-IP"),
				r.Header.Get("User-Agent"),
				r.Header.Get("Authorization"),
			}

			if httpError := tollbooth.LimitByKeys(l, keys); httpError != nil {
				render.Status(r, httpError.StatusCode)
				render.JSON(w, r, JSON{"error": httpError.Message})
				return
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// AppInfo adds custom app-info to header
func AppInfo(app string, version string) func(http.Handler) http.Handler {
	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Org", org)
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

// Ping middleware response with pong. Stops chain if ping request detected
func Ping(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "GET" && strings.HasSuffix(strings.ToLower(r.URL.Path), "/ping") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("pong")); err != nil {
				log.Printf("[WARN] can't send pong, %s", err)
			}
			return
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// Recoverer is a middleware that recovers from panics, logs the panic and returns a HTTP 500 status if possible.
func Recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				log.Printf("[ERROR] request panic, %v", rvr)
				debug.PrintStack()
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

type contextKey string

const (
	anonymous = iota
	developer
	full
)

// Auth adds auth from session and populate user info
func Auth(sessionStore *sessions.FilesystemStore, admins []string, modes ...int) func(http.Handler) http.Handler {

	inModes := func(mode int) bool {
		for _, m := range modes {
			if m == mode {
				return true
			}
		}
		return false
	}

	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			// for dev mode skip all real auth, make dev admin user
			if inModes(developer) {
				user := store.User{
					ID:      "dev",
					Name:    "developer one",
					Picture: "https://friends.radio-t.com/resources/images/rt_logo_64.png",
					Profile: "https://radio-t.com/info/",
					Admin:   true,
				}
				ctx := r.Context()
				ctx = context.WithValue(ctx, contextKey("user"), user)
				r = r.WithContext(ctx)
				h.ServeHTTP(w, r)
				return
			}

			session, err := sessionStore.Get(r, "remark")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			uinfoData, ok := session.Values["uinfo"]
			if !ok && inModes(full) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if ok {
				user := uinfoData.(store.User)
				for _, admin := range admins {
					if admin == user.ID {
						user.Admin = true
						break
					}
				}

				ctx := r.Context()
				ctx = context.WithValue(ctx, contextKey("user"), user)
				r = r.WithContext(ctx)
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}

// AdminOnly allows access to admins
func AdminOnly(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		user, err := GetUserInfo(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !user.Admin {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// GetUserInfo extracts user, or and token from request's context
func GetUserInfo(r *http.Request) (user store.User, err error) {

	ctx := r.Context()
	if ctx == nil {
		return store.User{}, errors.New("user not defined")
	}

	if u, ok := ctx.Value(contextKey("user")).(store.User); ok {
		return u, nil
	}

	return store.User{}, errors.New("user can't be parsed")
}

// LoggerFlag type
type LoggerFlag int

// logger flags enum
const (
	LogAll LoggerFlag = iota
	LogUser
	LogBody
)
const maxBody = 1024

var reMultWhtsp = regexp.MustCompile(`[\s\p{Zs}]{2,}`)

// Logger middleware prints http log. Customized by set of LoggerFlag
func Logger(flags ...LoggerFlag) func(http.Handler) http.Handler {

	inFlags := func(f LoggerFlag) bool {
		for _, flg := range flags {
			if flg == LogAll || flg == f {
				return true
			}
		}
		return false
	}

	f := func(h http.Handler) http.Handler {

		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, 1)

			body, user := func() (body string, user string) {
				ctx := r.Context()
				if ctx == nil {
					return "", ""
				}

				if inFlags(LogBody) {
					if content, err := ioutil.ReadAll(r.Body); err == nil {
						body = string(content)
						r.Body = ioutil.NopCloser(bytes.NewReader(content))

						if len(body) > 0 {
							body = strings.Replace(body, "\n", " ", -1)
							body = reMultWhtsp.ReplaceAllString(body, " ")
						}

						if len(body) > maxBody {
							body = body[:maxBody] + "..."
						}
					}
				}

				if inFlags(LogUser) {
					u, err := GetUserInfo(r)
					if err == nil && u.Name != "" {
						user = fmt.Sprintf(" - %s %q", u.ID, u.Name)
					}
				}

				return body, user
			}()

			t1 := time.Now()
			defer func() {
				t2 := time.Now()

				q := r.URL.String()
				if qun, err := url.QueryUnescape(q); err == nil {
					q = qun
				}

				log.Printf("[INFO] REST %s%s - %s - %s - %d (%d) - %v %s",
					r.Method, user, q, strings.Split(r.RemoteAddr, ":")[0],
					ww.Status(), ww.BytesWritten(), t2.Sub(t1), body)
			}()

			h.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}

	return f

}
