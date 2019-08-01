// Package tollbooth provides rate-limiting logic to HTTP request handler.
package tollbooth

import (
	"net/http"
	"strings"

	"fmt"
	"math"

	"github.com/didip/tollbooth/errors"
	"github.com/didip/tollbooth/libstring"
	"github.com/didip/tollbooth/limiter"
)

// setResponseHeaders configures X-Rate-Limit-Limit and X-Rate-Limit-Duration
func setResponseHeaders(lmt *limiter.Limiter, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Rate-Limit-Limit", fmt.Sprintf("%.2f", lmt.GetMax()))
	w.Header().Add("X-Rate-Limit-Duration", "1")
	w.Header().Add("X-Rate-Limit-Request-Forwarded-For", r.Header.Get("X-Forwarded-For"))
	w.Header().Add("X-Rate-Limit-Request-Remote-Addr", r.RemoteAddr)
}

// NewLimiter is a convenience function to limiter.New.
func NewLimiter(max float64, tbOptions *limiter.ExpirableOptions) *limiter.Limiter {
	return limiter.New(tbOptions).SetMax(max).SetBurst(int(math.Max(1, max)))
}

// LimitByKeys keeps track number of request made by keys separated by pipe.
// It returns HTTPError when limit is exceeded.
func LimitByKeys(lmt *limiter.Limiter, keys []string) *errors.HTTPError {
	if lmt.LimitReached(strings.Join(keys, "|")) {
		return &errors.HTTPError{Message: lmt.GetMessage(), StatusCode: lmt.GetStatusCode()}
	}

	return nil
}

// BuildKeys generates a slice of keys to rate-limit by given limiter and request structs.
func BuildKeys(lmt *limiter.Limiter, r *http.Request) [][]string {
	remoteIP := libstring.RemoteIP(lmt.GetIPLookups(), lmt.GetForwardedForIndexFromBehind(), r)
	path := r.URL.Path
	sliceKeys := make([][]string, 0)

	// Don't BuildKeys if remoteIP is blank.
	if remoteIP == "" {
		return sliceKeys
	}

	lmtMethods := lmt.GetMethods()
	lmtHeaders := lmt.GetHeaders()
	lmtBasicAuthUsers := lmt.GetBasicAuthUsers()

	lmtHeadersIsSet := len(lmtHeaders) > 0
	lmtBasicAuthUsersIsSet := len(lmtBasicAuthUsers) > 0

	if lmtMethods != nil && lmtHeadersIsSet && lmtBasicAuthUsersIsSet {
		// Limit by HTTP methods and HTTP headers+values and Basic Auth credentials.
		if libstring.StringInSlice(lmtMethods, r.Method) {
			for headerKey, headerValues := range lmtHeaders {
				if (headerValues == nil || len(headerValues) <= 0) && r.Header.Get(headerKey) != "" {
					// If header values are empty, rate-limit all request containing headerKey.
					username, _, ok := r.BasicAuth()
					if ok && libstring.StringInSlice(lmtBasicAuthUsers, username) {
						sliceKeys = append(sliceKeys, []string{remoteIP, path, r.Method, headerKey, r.Header.Get(headerKey), username})
					}

				} else if len(headerValues) > 0 && r.Header.Get(headerKey) != "" {
					// If header values are not empty, rate-limit all request with headerKey and headerValues.
					for _, headerValue := range headerValues {
						if r.Header.Get(headerKey) == headerValue {
							username, _, ok := r.BasicAuth()
							if ok && libstring.StringInSlice(lmtBasicAuthUsers, username) {
								sliceKeys = append(sliceKeys, []string{remoteIP, path, r.Method, headerKey, headerValue, username})
							}
							break
						}
					}
				}
			}
		}

	} else if lmtMethods != nil && lmtHeadersIsSet {
		// Limit by HTTP methods and HTTP headers+values.
		if libstring.StringInSlice(lmtMethods, r.Method) {
			for headerKey, headerValues := range lmtHeaders {
				if (headerValues == nil || len(headerValues) <= 0) && r.Header.Get(headerKey) != "" {
					// If header values are empty, rate-limit all request with headerKey.
					sliceKeys = append(sliceKeys, []string{remoteIP, path, r.Method, headerKey, r.Header.Get(headerKey)})

				} else if len(headerValues) > 0 && r.Header.Get(headerKey) != "" {
					// We are only limiting if request's header value is defined inside `headerValues`.
					for _, headerValue := range headerValues {
						if r.Header.Get(headerKey) == headerValue {
							sliceKeys = append(sliceKeys, []string{remoteIP, path, r.Method, headerKey, headerValue})
							break
						}
					}
				}
			}
		}

	} else if lmtMethods != nil && lmtBasicAuthUsersIsSet {
		// Limit by HTTP methods and Basic Auth credentials.
		if libstring.StringInSlice(lmtMethods, r.Method) {
			username, _, ok := r.BasicAuth()
			if ok && libstring.StringInSlice(lmtBasicAuthUsers, username) {
				sliceKeys = append(sliceKeys, []string{remoteIP, path, r.Method, username})
			}
		}

	} else if lmtMethods != nil {
		// Limit by HTTP methods.
		if libstring.StringInSlice(lmtMethods, r.Method) {
			sliceKeys = append(sliceKeys, []string{remoteIP, path, r.Method})
		}

	} else if lmtHeadersIsSet {
		// Limit by HTTP headers+values.
		for headerKey, headerValues := range lmtHeaders {
			if (headerValues == nil || len(headerValues) <= 0) && r.Header.Get(headerKey) != "" {
				// If header values are empty, rate-limit all request with headerKey.
				sliceKeys = append(sliceKeys, []string{remoteIP, path, headerKey, r.Header.Get(headerKey)})

			} else if len(headerValues) > 0 && r.Header.Get(headerKey) != "" {
				// If header values are not empty, rate-limit all request with headerKey and headerValues.
				for _, headerValue := range headerValues {
					if r.Header.Get(headerKey) == headerValue {
						sliceKeys = append(sliceKeys, []string{remoteIP, path, headerKey, headerValue})
						break
					}
				}
			}
		}

	} else if lmtBasicAuthUsersIsSet {
		// Limit by Basic Auth credentials.
		username, _, ok := r.BasicAuth()
		if ok && libstring.StringInSlice(lmtBasicAuthUsers, username) {
			sliceKeys = append(sliceKeys, []string{remoteIP, path, username})
		}
	} else {
		// Default: Limit by remoteIP and path.
		sliceKeys = append(sliceKeys, []string{remoteIP, path})
	}

	return sliceKeys
}

// LimitByRequest builds keys based on http.Request struct,
// loops through all the keys, and check if any one of them returns HTTPError.
func LimitByRequest(lmt *limiter.Limiter, w http.ResponseWriter, r *http.Request) *errors.HTTPError {
	setResponseHeaders(lmt, w, r)

	sliceKeys := BuildKeys(lmt, r)

	// Loop sliceKeys and check if one of them has error.
	for _, keys := range sliceKeys {
		httpError := LimitByKeys(lmt, keys)
		if httpError != nil {
			return httpError
		}
	}

	return nil
}

// LimitHandler is a middleware that performs rate-limiting given http.Handler struct.
func LimitHandler(lmt *limiter.Limiter, next http.Handler) http.Handler {
	middle := func(w http.ResponseWriter, r *http.Request) {
		httpError := LimitByRequest(lmt, w, r)
		if httpError != nil {
			lmt.ExecOnLimitReached(w, r)
			w.Header().Add("Content-Type", lmt.GetMessageContentType())
			w.WriteHeader(httpError.StatusCode)
			w.Write([]byte(httpError.Message))
			return
		}

		// There's no rate-limit error, serve the next handler.
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(middle)
}

// LimitFuncHandler is a middleware that performs rate-limiting given request handler function.
func LimitFuncHandler(lmt *limiter.Limiter, nextFunc func(http.ResponseWriter, *http.Request)) http.Handler {
	return LimitHandler(lmt, http.HandlerFunc(nextFunc))
}
