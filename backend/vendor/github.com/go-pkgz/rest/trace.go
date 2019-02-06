package rest

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"net/http"
	"time"
)

type contextKey string

const traceHeader = "X-Request-ID"

// Trace looks for header X-Request-ID and makes it as random id if not found, then populates it to the result's header
// and to request context
func Trace(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(traceHeader)
		if traceID == "" {
			traceID = randToken()
		}
		w.Header().Set(traceHeader, traceID)
		ctx := context.WithValue(r.Context(), contextKey("requestID"), traceID)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// GetTraceID returns request id from the context
func GetTraceID(r *http.Request) string {
	if id, ok := r.Context().Value(contextKey("requestID")).(string); ok {
		return id
	}
	return ""
}

func randToken() string {
	fallback := func() string {
		return fmt.Sprintf("%x", time.Now().Nanosecond())
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fallback()
	}
	s := sha1.New()
	if _, err := s.Write(b); err != nil {
		return fallback()
	}
	return fmt.Sprintf("%x", s.Sum(nil))
}
