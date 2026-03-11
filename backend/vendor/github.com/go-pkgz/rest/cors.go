package rest

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig defines CORS middleware configuration.
// Use CorsOpt functions to customize.
type CORSConfig struct {
	// AllowedOrigins is a list of origins that may access the resource.
	// use "*" to allow all origins (not recommended with credentials).
	// default: ["*"]
	AllowedOrigins []string
	// AllowedMethods is a list of methods the client is allowed to use.
	// default: GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD
	AllowedMethods []string
	// AllowedHeaders is a list of headers the client is allowed to send.
	// default: Accept, Content-Type, Authorization, X-Requested-With
	AllowedHeaders []string
	// ExposedHeaders is a list of headers that are safe to expose to the client.
	// default: empty
	ExposedHeaders []string
	// AllowCredentials indicates whether the request can include credentials.
	// when true, AllowedOrigins cannot be "*" (browser security restriction).
	// default: false
	AllowCredentials bool
	// MaxAge indicates how long (in seconds) the results of a preflight can be cached.
	// default: 0 (no caching)
	MaxAge int
}

// CorsOpt is a functional option for CORSConfig
type CorsOpt func(*CORSConfig)

// defaultCORSConfig returns config with sensible defaults
func defaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization", "X-Requested-With"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           0,
	}
}

// CorsAllowedOrigins sets the list of allowed origins.
// Use "*" to allow all origins (not recommended with credentials).
func CorsAllowedOrigins(origins ...string) CorsOpt {
	return func(c *CORSConfig) {
		c.AllowedOrigins = origins
	}
}

// CorsAllowedMethods sets the list of allowed HTTP methods.
func CorsAllowedMethods(methods ...string) CorsOpt {
	return func(c *CORSConfig) {
		c.AllowedMethods = methods
	}
}

// CorsAllowedHeaders sets the list of allowed request headers.
func CorsAllowedHeaders(headers ...string) CorsOpt {
	return func(c *CORSConfig) {
		c.AllowedHeaders = headers
	}
}

// CorsExposedHeaders sets the list of headers exposed to the client.
func CorsExposedHeaders(headers ...string) CorsOpt {
	return func(c *CORSConfig) {
		c.ExposedHeaders = headers
	}
}

// CorsAllowCredentials enables or disables credentials.
// When true, AllowedOrigins cannot be "*".
func CorsAllowCredentials(allow bool) CorsOpt {
	return func(c *CORSConfig) {
		c.AllowCredentials = allow
	}
}

// CorsMaxAge sets how long (in seconds) preflight results can be cached.
func CorsMaxAge(seconds int) CorsOpt {
	return func(c *CORSConfig) {
		c.MaxAge = seconds
	}
}

// CORS is middleware that handles Cross-Origin Resource Sharing.
// It handles preflight OPTIONS requests and sets appropriate headers.
// By default allows all origins with common methods and headers.
func CORS(opts ...CorsOpt) func(http.Handler) http.Handler {
	cfg := defaultCORSConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// pre-compute joined strings for performance
	methodsStr := strings.Join(cfg.AllowedMethods, ", ")
	headersStr := strings.Join(cfg.AllowedHeaders, ", ")
	exposedStr := strings.Join(cfg.ExposedHeaders, ", ")

	// check if wildcard is used
	allowAll := len(cfg.AllowedOrigins) == 1 && cfg.AllowedOrigins[0] == "*"

	// build origin lookup for O(1) check (only when not allowing all)
	var originSet map[string]bool
	if !allowAll {
		originSet = make(map[string]bool, len(cfg.AllowedOrigins))
		for _, o := range cfg.AllowedOrigins {
			originSet[strings.ToLower(o)] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// no origin header means same-origin or non-browser request
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// check if origin is allowed
			var allowed bool
			if allowAll {
				allowed = true
			} else {
				allowed = originSet[strings.ToLower(origin)]
			}
			if !allowed {
				// origin not allowed, proceed without CORS headers
				next.ServeHTTP(w, r)
				return
			}

			// set Vary header for caching
			w.Header().Add("Vary", "Origin")

			// set allowed origin
			if allowAll && !cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				// reflect the specific origin (required for credentials)
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			// set credentials header if enabled
			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			// handle preflight request
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				// preflight request
				w.Header().Set("Access-Control-Allow-Methods", methodsStr)
				w.Header().Set("Access-Control-Allow-Headers", headersStr)

				if cfg.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}

				w.WriteHeader(http.StatusNoContent)
				return
			}

			// actual request - set exposed headers
			if exposedStr != "" {
				w.Header().Set("Access-Control-Expose-Headers", exposedStr)
			}

			next.ServeHTTP(w, r)
		})
	}
}
