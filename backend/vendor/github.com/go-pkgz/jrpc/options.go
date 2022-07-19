package jrpc

import (
	"net/http"
)

// Option func type
type Option func(s *Server)

// Auth sets basic auth credentials, required
func Auth(user, password string) Option {
	return func(s *Server) {
		s.authUser = user
		s.authPasswd = password
	}
}

// WithTimeouts sets server timeout values such as ReadHeader, Write and Idle timeout, optional.
// If this option not defined server use default timeout values
func WithTimeouts(timeouts Timeouts) Option {

	// this option sets only server limits values and exclude middlewares limits fields of Limits struct.
	return func(s *Server) {
		s.timeouts.ReadHeaderTimeout = timeouts.ReadHeaderTimeout
		s.timeouts.WriteTimeout = timeouts.WriteTimeout
		s.timeouts.IdleTimeout = timeouts.IdleTimeout
		s.timeouts.CallTimeout = timeouts.CallTimeout // it need for middleware with custom timeout value
	}
}

// WithLimits sets value for client limit call/sec per client middleware
func WithLimits(limit float64) Option {

	// this option sets only server limits values and exclude middlewares limits fields of Limits struct.
	return func(s *Server) {
		s.limits.clientLimit = limit
	}
}

// WithThrottler sets throttler middleware with specify limit value, optional
func WithThrottler(limit int) Option {
	return func(s *Server) {
		s.limits.serverThrottle = limit
	}
}

// WithMiddlewares sets custom middlewares list, optional
func WithMiddlewares(middlewares ...func(http.Handler) http.Handler) Option {
	return func(s *Server) {
		s.customMiddlewares = append(s.customMiddlewares, middlewares...)
	}
}

// WithSignature sets signature data for server response headers
func WithSignature(appName, author, version string) Option {
	return func(s *Server) {
		s.signature = signaturePayload{
			appName: appName,
			author:  author,
			version: version,
		}
	}
}

// WithLogger sets custom logger, optional
func WithLogger(logger L) Option {
	return func(s *Server) {
		s.logger = logger
	}
}
