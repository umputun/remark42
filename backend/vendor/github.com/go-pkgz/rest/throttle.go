package rest

import (
	"net/http"
)

// Throttle middleware checks how many request in-fly and rejects with 503 if exceeded
func Throttle(limit int64) func(http.Handler) http.Handler {

	ch := make(chan struct{}, limit)
	return func(h http.Handler) http.Handler {

		fn := func(w http.ResponseWriter, r *http.Request) {

			if limit <= 0 {
				h.ServeHTTP(w, r)
				return
			}

			var acquired bool
			defer func() {
				if !acquired {
					return
				}
				select {
				case <-ch:
					return
				default:
					return
				}
			}()

			select {
			case ch <- struct{}{}:
				acquired = true
				h.ServeHTTP(w, r)
				return
			default:
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}

		return http.HandlerFunc(fn)
	}
}
