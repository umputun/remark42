package api

import (
	"bytes"
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

	"github.com/go-chi/chi/middleware"

	"github.com/umputun/remark/backend/app/rest"
)

// JSON is a map alias, just for convenience
type JSON map[string]interface{}

// AppInfo adds custom app-info to the response header
func AppInfo(app string, version string) func(http.Handler) http.Handler {
	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Org", "Umputun")
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
				log.Printf("[WARN] request panic, %v", rvr)
				debug.PrintStack()
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// LoggerFlag type
type LoggerFlag int

// logger flags enum
const (
	LogAll LoggerFlag = iota
	LogUser
	LogBody
	LogNone
)
const maxBody = 1024

var reMultWhtsp = regexp.MustCompile(`[\s\p{Zs}]{2,}`)

// Logger middleware prints http log. Customized by set of LoggerFlag
func Logger(ipFn func(ip string) string, flags ...LoggerFlag) func(http.Handler) http.Handler {

	f := func(h http.Handler) http.Handler {

		fn := func(w http.ResponseWriter, r *http.Request) {

			if inLogFlags(LogNone, flags) { // skip logging
				h.ServeHTTP(w, r)
				return
			}

			ww := middleware.NewWrapResponseWriter(w, 1)
			body, user := getBodyAndUser(r, flags)
			t1 := time.Now()
			defer func() {
				t2 := time.Now()

				q := r.URL.String()
				if qun, err := url.QueryUnescape(q); err == nil {
					q = qun
				}
				q = sanitizeQuery(q)

				remoteIP := strings.Split(r.RemoteAddr, ":")[0]
				if strings.HasPrefix(r.RemoteAddr, "[") {
					remoteIP = strings.Split(r.RemoteAddr, "]:")[0] + "]"
				}
				if ipFn != nil {
					remoteIP = ipFn(remoteIP)
				}

				log.Printf("[INFO] REST %s - %s - %s - %d (%d) - %v %s %s",
					r.Method, q, remoteIP, ww.Status(), ww.BytesWritten(), t2.Sub(t1), user, body)
			}()

			h.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}

	return f
}

func getBodyAndUser(r *http.Request, flags []LoggerFlag) (body string, user string) {
	ctx := r.Context()
	if ctx == nil {
		return "", ""
	}

	if inLogFlags(LogBody, flags) {
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

	if inLogFlags(LogUser, flags) {
		u, err := rest.GetUserInfo(r)
		if err == nil && u.Name != "" {
			user = fmt.Sprintf(" - %s %q", u.ID, u.Name)
		}
	}

	return body, user
}

func sanitizeQuery(u string) string {
	out := []rune(u)
	hide := []string{"password", "passwd", "secret", "credentials"}
	for _, h := range hide {
		if strings.Contains(strings.ToLower(u), h+"=") {
			stPos := strings.Index(strings.ToLower(u), h+"=") + len(h) + 1
			fnPos := strings.Index(u[stPos:], "&")
			if fnPos == -1 {
				fnPos = len(u)
			} else {
				fnPos = stPos + fnPos
			}
			for i := stPos; i < fnPos; i++ {
				out[i] = rune('*')
			}
		}
	}
	return string(out)
}

func inLogFlags(f LoggerFlag, flags []LoggerFlag) bool {
	for _, flg := range flags {
		if (flg == LogAll && f != LogNone) || flg == f {
			return true
		}
	}
	return false
}
