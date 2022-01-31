package rest

import (
	"fmt"
	"net/http"
	"time"
)

// Deprecation adds a header 'Deprecation: version="version", date="date" header'
// see https://tools.ietf.org/id/draft-dalal-deprecation-header-00.html
func Deprecation(version string, date time.Time) func(http.Handler) http.Handler {
	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			headerVal := fmt.Sprintf("version=%q, date=%q", version, date.Format(time.RFC3339))
			w.Header().Set("Deprecation", headerVal)
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}
