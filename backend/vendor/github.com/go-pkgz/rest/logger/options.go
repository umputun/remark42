package logger

import (
	"net/http"

	"github.com/go-pkgz/lgr"
)

// Option func type
type Option func(l *Middleware)

// Flags functional option defines output modes
func Flags(flags ...Flag) Option {
	return func(l *Middleware) {
		l.flags = flags
	}
}

// MaxBodySize functional option defines the largest body size to log.
func MaxBodySize(max int) Option {
	return func(l *Middleware) {
		if max >= 0 {
			l.maxBodySize = max
		}
	}
}

// Prefix functional option defines log line prefix.
func Prefix(prefix string) Option {
	return func(l *Middleware) {
		l.prefix = prefix
	}
}

// IPfn functional option defines ip masking function.
func IPfn(ipFn func(ip string) string) Option {
	return func(l *Middleware) {
		l.ipFn = ipFn
	}
}

// UserFn functional option defines user name function.
func UserFn(userFn func(r *http.Request) (string, error)) Option {
	return func(l *Middleware) {
		l.userFn = userFn
	}
}

// Log functional option defines loging backend.
func Log(log lgr.L) Option {
	return func(l *Middleware) {
		l.log = log
	}
}
