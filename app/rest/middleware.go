package rest

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/didip/tollbooth"
	"github.com/go-chi/render"
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
				r.Header.Get("Authorization"),
				r.Header.Get("X-Forwarded-For"),
				r.Header.Get("X-Real-IP"),
				r.Header.Get("RemoteAddr"),
				r.Header.Get("User-Agent"),
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

				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
