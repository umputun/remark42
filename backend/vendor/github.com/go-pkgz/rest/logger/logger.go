package logger

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-pkgz/lgr"
)

var reMultWhtsp = regexp.MustCompile(`[\s\p{Zs}]{2,}`)

// Middleware for logging rest requests
type Middleware struct {
	prefix      string
	maxBodySize int
	flags       []Flag
	ipFn        func(ip string) string
	userFn      func(r *http.Request) (string, error)
	log         lgr.L
}

// Flag type
type Flag int

// logger flags enum
const (
	All Flag = iota
	User
	Body
	None
)

// Logger returns default logger middleware
func Logger(next http.Handler) http.Handler {
	l := New(Flags(All), Prefix("REST"))
	return l.Handler(next)

}

// New makes rest Logger with given options
func New(options ...Option) *Middleware {
	res := Middleware{
		prefix:      "",
		maxBodySize: 1024,
		flags:       []Flag{All},
		log:         lgr.Default(),
	}
	for _, opt := range options {
		opt(&res)
	}
	return &res
}

// Handler middleware prints http log
func (l *Middleware) Handler(next http.Handler) http.Handler {

	fn := func(w http.ResponseWriter, r *http.Request) {

		if l.inLogFlags(None) { // skip logging
			next.ServeHTTP(w, r)
			return
		}

		ww := newCustomResponseWriter(w)
		body, user := l.getBodyAndUser(r)
		t1 := time.Now()
		defer func() {
			t2 := time.Now()

			q := l.sanitizeQuery(r.URL.String())
			if qun, err := url.QueryUnescape(q); err == nil {
				q = qun
			}

			remoteIP := strings.Split(r.RemoteAddr, ":")[0]
			if strings.HasPrefix(r.RemoteAddr, "[") {
				remoteIP = strings.Split(r.RemoteAddr, "]:")[0] + "]"
			}

			if l.ipFn != nil { // mask ip with ipFn
				remoteIP = l.ipFn(remoteIP)
			}

			l.log.Logf("%s %s - %s - %s - %d (%d) - %v %s %s",
				l.prefix, r.Method, q, remoteIP, ww.status, ww.size, t2.Sub(t1), user, body)
		}()

		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}

func (l *Middleware) getBodyAndUser(r *http.Request) (body string, user string) {
	ctx := r.Context()
	if ctx == nil {
		return "", ""
	}

	if l.inLogFlags(Body) {
		if content, err := ioutil.ReadAll(r.Body); err == nil {
			body = string(content)
			r.Body = ioutil.NopCloser(bytes.NewReader(content))

			if len(body) > 0 {
				body = strings.Replace(body, "\n", " ", -1)
				body = reMultWhtsp.ReplaceAllString(body, " ")
			}

			if len(body) > l.maxBodySize {
				body = body[:l.maxBodySize] + "..."
			}
		}
	}

	if l.inLogFlags(User) && l.userFn != nil {
		u, err := l.userFn(r)
		if err == nil && u != "" {
			user = fmt.Sprintf(" - %s", u)
		}
	}

	return body, user
}

func (l *Middleware) inLogFlags(f Flag) bool {
	for _, flg := range l.flags {
		if (flg == All && f != None) || flg == f {
			return true
		}
	}
	return false
}

func (l *Middleware) sanitizeQuery(inp string) string {
	out := []rune(inp)
	hide := []string{"password", "passwd", "secret", "credentials"}
	for _, h := range hide {
		if strings.Contains(strings.ToLower(inp), h+"=") {
			stPos := strings.Index(strings.ToLower(inp), h+"=") + len(h) + 1
			fnPos := strings.Index(inp[stPos:], "&")
			if fnPos == -1 {
				fnPos = len(inp)
			} else {
				fnPos = stPos + fnPos
			}
			for i := stPos; i < fnPos; i++ {
				out[i] = rune('*')
			}
		}
	}
	return string(out)
}

// customResponseWriter implements ResponseWriter and keeping status and size
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

// WriteHeader implements ResponseWriter and saves status
func (c *customResponseWriter) WriteHeader(status int) {
	c.status = status
	c.ResponseWriter.WriteHeader(status)
}

// WriteHeader implements ResponseWriter and tracking size
func (c *customResponseWriter) Write(b []byte) (int, error) {
	size, err := c.ResponseWriter.Write(b)
	c.size += size
	return size, err
}

// Flush implements ResponseWriter
func (c *customResponseWriter) Flush() {
	if f, ok := c.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements ResponseWriter
func (c *customResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := c.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not implement the Hijacker interface")
}
