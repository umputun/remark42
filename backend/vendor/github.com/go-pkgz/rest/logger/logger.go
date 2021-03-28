// Package logger implements logging middleware
package logger

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Middleware is a logger for rest requests.
type Middleware struct {
	prefix      string
	logBody     bool
	maxBodySize int
	ipFn        func(ip string) string
	userFn      func(r *http.Request) (string, error)
	subjFn      func(r *http.Request) (string, error)
	log         Backend
}

// Backend is logging backend
type Backend interface {
	Logf(format string, args ...interface{})
}

type stdBackend struct{}

func (s stdBackend) Logf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// Logger is a default logger middleware with "REST" prefix
func Logger(next http.Handler) http.Handler {
	l := New(Prefix("REST"))
	return l.Handler(next)

}

// New makes rest logger with given options
func New(options ...Option) *Middleware {
	res := Middleware{
		prefix:      "",
		maxBodySize: 1024,
		log:         stdBackend{},
	}
	for _, opt := range options {
		opt(&res)
	}
	return &res
}

// Handler middleware prints http log
func (l *Middleware) Handler(next http.Handler) http.Handler {

	fn := func(w http.ResponseWriter, r *http.Request) {
		ww := newCustomResponseWriter(w)

		user := ""
		if l.userFn != nil {
			if u, err := l.userFn(r); err == nil {
				user = u
			}
		}

		body := l.getBody(r)
		t1 := time.Now()
		defer func() {
			t2 := time.Now()

			u := *r.URL // shallow copy
			u.RawQuery = l.sanitizeQuery(u.RawQuery)
			rawurl := u.String()
			if unescURL, err := url.QueryUnescape(rawurl); err == nil {
				rawurl = unescURL
			}

			remoteIP := l.remoteIP(r)
			if l.ipFn != nil { // mask ip with ipFn
				remoteIP = l.ipFn(remoteIP)
			}

			var bld strings.Builder
			if l.prefix != "" {
				_, _ = bld.WriteString(l.prefix)
				_, _ = bld.WriteString(" ")
			}

			_, _ = bld.WriteString(fmt.Sprintf("%s - %s - %s - %d (%d) - %v", r.Method, rawurl, remoteIP, ww.status, ww.size, t2.Sub(t1)))

			if user != "" {
				_, _ = bld.WriteString(" - ")
				_, _ = bld.WriteString(user)
			}

			if l.subjFn != nil {
				if subj, err := l.subjFn(r); err == nil {
					_, _ = bld.WriteString(" - ")
					_, _ = bld.WriteString(subj)
				}
			}

			if traceID := r.Header.Get("X-Request-ID"); traceID != "" {
				_, _ = bld.WriteString(" - ")
				_, _ = bld.WriteString(traceID)
			}

			if body != "" {
				_, _ = bld.WriteString(" - ")
				_, _ = bld.WriteString(body)
			}

			l.log.Logf("%s", bld.String())
		}()

		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}

var reMultWhtsp = regexp.MustCompile(`[\s\p{Zs}]{2,}`)

func (l *Middleware) getBody(r *http.Request) string {
	if !l.logBody {
		return ""
	}

	reader, body, hasMore, err := peek(r.Body, int64(l.maxBodySize))
	if err != nil {
		return ""
	}

	// "The Server will close the request body. The ServeHTTP Handler does not need to."
	// https://golang.org/pkg/net/http/#Request
	// So we can use ioutil.NopCloser() to make io.ReadCloser.
	// Note that below assignment is not approved by the docs:
	// "Except for reading the body, handlers should not modify the provided Request."
	// https://golang.org/pkg/net/http/#Handler
	r.Body = ioutil.NopCloser(reader)

	if len(body) > 0 {
		body = strings.Replace(body, "\n", " ", -1)
		body = reMultWhtsp.ReplaceAllString(body, " ")
	}

	if hasMore {
		body += "..."
	}

	return body
}

// peek the first n bytes as string
func peek(r io.Reader, n int64) (reader io.Reader, s string, hasMore bool, err error) {
	if n < 0 {
		n = 0
	}

	buf := new(bytes.Buffer)
	_, err = io.CopyN(buf, r, n+1)
	if err == io.EOF {
		str := buf.String()
		return buf, str, false, nil
	}
	if err != nil {
		return r, "", false, err
	}

	// one extra byte is successfully read
	s = buf.String()
	s = s[:len(s)-1]

	return io.MultiReader(buf, r), s, true, nil
}

var keysToHide = []string{"password", "passwd", "secret", "credentials", "token"}

// Hide query values for keysToHide. May change order of query params.
// May escape unescaped query params.
func (l *Middleware) sanitizeQuery(rawQuery string) string {
	// note that we skip non-nil error further
	query, err := url.ParseQuery(rawQuery)

	isHidden := func(key string) bool {
		for _, k := range keysToHide {
			if strings.EqualFold(k, key) {
				return true
			}
		}
		return false
	}

	present := false
	for key, values := range query {
		if isHidden(key) {
			present = true
			for i := range values {
				values[i] = "********"
			}
		}
	}

	// short circuit
	if (err == nil) && !present {
		return rawQuery
	}

	return query.Encode()
}

// remoteIP gets address from X-Forwarded-For and than from request's remote address
func (l *Middleware) remoteIP(r *http.Request) (remoteIP string) {

	if remoteIP = r.Header.Get("X-Forwarded-For"); remoteIP == "" {
		remoteIP = r.RemoteAddr
	}
	remoteIP = strings.Split(remoteIP, ":")[0]
	if strings.HasPrefix(remoteIP, "[") {
		remoteIP = strings.Split(remoteIP, "]:")[0] + "]"
	}
	return remoteIP
}

// customResponseWriter is an HTTP response logger that keeps HTTP status code and
// the number of bytes written.
// It implements http.ResponseWriter, http.Flusher and http.Hijacker.
// Note that type assertion from http.ResponseWriter(customResponseWriter) to
// http.Flusher and http.Hijacker is always succeed but underlying http.ResponseWriter
// may not implement them.
type customResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func newCustomResponseWriter(w http.ResponseWriter) *customResponseWriter {
	return &customResponseWriter{
		ResponseWriter: w,
		status:         200,
	}
}

// WriteHeader implements http.ResponseWriter and saves status
func (c *customResponseWriter) WriteHeader(status int) {
	c.status = status
	c.ResponseWriter.WriteHeader(status)
}

// Write implements http.ResponseWriter and tracks number of bytes written
func (c *customResponseWriter) Write(b []byte) (int, error) {
	size, err := c.ResponseWriter.Write(b)
	c.size += size
	return size, err
}

// Flush implements http.Flusher
func (c *customResponseWriter) Flush() {
	if f, ok := c.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker
func (c *customResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := c.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not implement the Hijacker interface") //nolint:golint //capital letter is OK here
}
