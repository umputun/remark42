package rest

import (
	"expvar"
	"fmt"
	"net/http"
	"strings"
)

// Metrics responds to GET /metrics with list of expvar
func Metrics(onlyIps ...string) func(http.Handler) http.Handler {

	return func(h http.Handler) http.Handler {

		fn := func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" && strings.HasSuffix(strings.ToLower(r.URL.Path), "/metrics") {
				if matched, ip := matchSourceIP(r, onlyIps); !matched {
					w.WriteHeader(http.StatusForbidden)
					RenderJSON(w, JSON{"error": fmt.Sprintf("ip %s rejected", ip)})
					return
				}
				expvar.Handler().ServeHTTP(w, r)
				return
			}
			h.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
