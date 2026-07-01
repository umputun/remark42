package rest

import (
	"bytes"
	"context"
	"maps"
	"net/http"
	"sync"
	"time"
)

// Timeout is a middleware that enforces a maximum duration for handling a request.
// It runs the next handler with a context deadline and, if the handler has not finished
// by the time the deadline is reached, responds with StatusGatewayTimeout (504) at the
// deadline — regardless of whether the handler observes the context.
//
// The handler's output is buffered until it completes: on success the buffered response
// (status, headers and body) is written through unchanged; if the deadline fires first,
// the buffered output is discarded, a 504 is sent, and any further writes by the still
// running handler return http.ErrHandlerTimeout. If the parent request context is
// canceled (rather than the deadline being exceeded) the handler is stopped without a
// 504, since the request is being abandoned, and its later writes return the context's
// error (e.g. context.Canceled).
//
// Because the response is buffered, the wrapped ResponseWriter does not support
// http.Flusher or http.Hijacker (matching net/http.TimeoutHandler); streaming and
// connection hijacking are not available under Timeout.
//
// A non-positive timeout disables the middleware: the handler is called directly, with
// no deadline, buffering or 504.
func Timeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if timeout <= 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			r = r.WithContext(ctx)

			done := make(chan struct{})
			panicChan := make(chan any, 1)
			tw := &timeoutWriter{w: w, h: make(http.Header)}

			go func() {
				defer func() {
					if p := recover(); p != nil {
						panicChan <- p
					}
				}()
				next.ServeHTTP(tw, r)
				close(done)
			}()

			select {
			case p := <-panicChan:
				panic(p)
			case <-done:
				tw.mu.Lock()
				defer tw.mu.Unlock()
				maps.Copy(w.Header(), tw.h)
				code := http.StatusOK
				if tw.wroteHeader {
					code = tw.code
				}
				w.WriteHeader(code)
				_, _ = w.Write(tw.wbuf.Bytes())
			case <-ctx.Done():
				tw.mu.Lock()
				defer tw.mu.Unlock()
				// discard the buffer and stop further handler writes, recording the cause so a
				// late write returns the real error. Only the deadline yields a 504; a canceled
				// parent means the request is being abandoned, so there is nobody to send it to.
				switch err := ctx.Err(); err {
				case context.DeadlineExceeded:
					tw.err = http.ErrHandlerTimeout
					w.WriteHeader(http.StatusGatewayTimeout)
				default:
					tw.err = err
				}
			}
		})
	}
}

// timeoutWriter buffers a handler's response so the Timeout middleware can either flush
// it on success or discard it and send a 504 once the deadline is reached. All fields
// after mu are guarded by mu, which is also held by the middleware while it drains the
// buffer, so it is safe against a handler goroutine that keeps writing after the timeout.
type timeoutWriter struct {
	w http.ResponseWriter
	h http.Header

	mu          sync.Mutex
	wbuf        bytes.Buffer
	code        int
	wroteHeader bool
	err         error // set once the response is timed out or canceled; returned by Write
}

// Header implements http.ResponseWriter and returns the buffered header map.
func (tw *timeoutWriter) Header() http.Header { return tw.h }

// Write implements http.ResponseWriter, buffering the response body. Once the response has
// timed out or been canceled it buffers nothing and returns the recorded error
// (http.ErrHandlerTimeout on deadline, or the context error on cancellation).
func (tw *timeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.err != nil {
		return 0, tw.err
	}
	if !tw.wroteHeader {
		tw.setHeaderLocked(http.StatusOK)
	}
	return tw.wbuf.Write(p)
}

// WriteHeader implements http.ResponseWriter, recording the status for the buffered
// response. It is a no-op after the timeout has fired or after the first call.
func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.setHeaderLocked(code)
}

func (tw *timeoutWriter) setHeaderLocked(code int) {
	if tw.err != nil || tw.wroteHeader {
		return
	}
	tw.wroteHeader = true
	tw.code = code
}
