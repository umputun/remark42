package rest

import (
	"expvar"
	"fmt"
	"net/http"
	"strings"
)

// Metrics responds to GET /metrics with list of expvar, limited to the given source ips.
// Called without any ip it rejects every request, as an endpoint nobody can reach is the safe
// default for one that publishes expvar; use MetricsAllowAll to serve it to everyone on purpose.
func Metrics(onlyIps ...string) func(http.Handler) http.Handler {
	return metricsHandler(false, onlyIps)
}

// MetricsAllowAll responds to GET /metrics with list of expvar for any source, without any ip check.
// expvar exposes cmdline, which usually carries the flag values the process was started with, so
// only use this where something else already keeps the endpoint private.
func MetricsAllowAll() func(http.Handler) http.Handler {
	return metricsHandler(true, nil)
}

func metricsHandler(allowAll bool, onlyIps []string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" && strings.HasSuffix(strings.ToLower(r.URL.Path), "/metrics") {
				if !allowAll {
					if matched, ip, err := matchSourceIP(r, onlyIps); !matched || err != nil {
						_ = EncodeJSON(w, http.StatusForbidden, JSON{"error": fmt.Sprintf("ip %s rejected", ip)})
						return
					}
				}
				expvar.Handler().ServeHTTP(w, r)
				return
			}
			h.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
