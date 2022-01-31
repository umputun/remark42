package rest

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzDefaultContentTypes = []string{
	"text/css",
	"text/javascript",
	"text/xml",
	"text/html",
	"text/plain",
	"application/javascript",
	"application/x-javascript",
	"application/json",
}

var gzPool = sync.Pool{
	New: func() interface{} { return gzip.NewWriter(io.Discard) },
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// Gzip is a middleware compressing response
func Gzip(contentTypes ...string) func(http.Handler) http.Handler {

	gzCts := gzDefaultContentTypes
	if len(contentTypes) > 0 {
		gzCts = contentTypes
	}

	contentType := func(r *http.Request) string {
		result := r.Header.Get("Content-type")
		if result == "" {
			return "application/octet-stream"
		}
		return result
	}

	f := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			var gzOk bool
			ctype := contentType(r)
			for _, c := range gzCts {
				if strings.HasPrefix(strings.ToLower(ctype), strings.ToLower(c)) {
					gzOk = true
					break
				}
			}

			if !gzOk {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Content-Encoding", "gzip")
			gz := gzPool.Get().(*gzip.Writer)
			defer gzPool.Put(gz)

			gz.Reset(w)
			defer gz.Close()

			next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, Writer: gz}, r)
		})
	}
	return f
}
