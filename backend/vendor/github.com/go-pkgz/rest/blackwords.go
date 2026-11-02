package rest

import (
	"bytes"
	"io"
	"net/http"
	"strings"
)

// BlackWords middleware doesn't allow some words in the request body
func BlackWords(words ...string) func(http.Handler) http.Handler {

	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			content, err := io.ReadAll(r.Body)
			if err != nil {
				// the body can't be inspected, refuse rather than pass a partially consumed one through
				_ = EncodeJSON(w, http.StatusBadRequest, JSON{"error": "can't read request body"})
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(content))

			body := strings.ToLower(string(content))
			if body != "" {
				for _, word := range words {
					if strings.Contains(body, strings.ToLower(word)) {
						_ = EncodeJSON(w, http.StatusForbidden, JSON{"error": "one of blacklisted words detected"})
						return
					}
				}
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// BlackWordsFn middleware uses func to get the list and doesn't allow some words in the request body
func BlackWordsFn(fn func() []string) func(http.Handler) http.Handler {
	return BlackWords(fn()...)
}
