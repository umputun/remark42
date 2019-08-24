package rest

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

// SizeLimit middleware checks if body size is above the limit and returns StatusRequestEntityTooLarge (413)
func SizeLimit(size int64) func(http.Handler) http.Handler {

	return func(h http.Handler) http.Handler {

		fn := func(w http.ResponseWriter, r *http.Request) {

			// check ContentLength
			if r.ContentLength > size {
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				return
			}

			// check size of the actual body
			content, err := ioutil.ReadAll(io.LimitReader(r.Body, size+1))
			if err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			if int64(len(content)) > size {
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = ioutil.NopCloser(bytes.NewReader(content))
			h.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
