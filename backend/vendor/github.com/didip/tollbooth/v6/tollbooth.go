// Package tollbooth provides rate-limiting logic to HTTP request handler.
package tollbooth

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/didip/tollbooth/v6/errors"
	"github.com/didip/tollbooth/v6/libstring"
	"github.com/didip/tollbooth/v6/limiter"
)

// setResponseHeaders configures X-Rate-Limit-Limit and X-Rate-Limit-Duration
func setResponseHeaders(lmt *limiter.Limiter, w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Rate-Limit-Limit", fmt.Sprintf("%.2f", lmt.GetMax()))
	w.Header().Add("X-Rate-Limit-Duration", "1")

	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if strings.TrimSpace(xForwardedFor) != "" {
		w.Header().Add("X-Rate-Limit-Request-Forwarded-For", xForwardedFor)
	}

	w.Header().Add("X-Rate-Limit-Request-Remote-Addr", r.RemoteAddr)
}

// NewLimiter is a convenience function to limiter.New.
func NewLimiter(max float64, tbOptions *limiter.ExpirableOptions) *limiter.Limiter {
	return limiter.New(tbOptions).
		SetMax(max).
		SetBurst(int(math.Max(1, max))).
		SetIPLookups([]string{"X-Forwarded-For", "X-Real-IP", "RemoteAddr"})
}

// LimitByKeys keeps track number of request made by keys separated by pipe.
// It returns HTTPError when limit is exceeded.
func LimitByKeys(lmt *limiter.Limiter, keys []string) *errors.HTTPError {
	if lmt.LimitReached(strings.Join(keys, "|")) {
		return &errors.HTTPError{Message: lmt.GetMessage(), StatusCode: lmt.GetStatusCode()}
	}

	return nil
}

// ShouldSkipLimiter is a series of filter that decides if request should be limited or not.
func ShouldSkipLimiter(lmt *limiter.Limiter, r *http.Request) bool {
	// ---------------------------------
	// Filter by remote ip
	// If we are unable to find remoteIP, skip limiter
	remoteIP := libstring.RemoteIP(lmt.GetIPLookups(), lmt.GetForwardedForIndexFromBehind(), r)
	if remoteIP == "" {
		return true
	}

	// ---------------------------------
	// Filter by request method
	lmtMethods := lmt.GetMethods()
	lmtMethodsIsSet := len(lmtMethods) > 0

	if lmtMethodsIsSet {
		// If request does not contain all of the methods in limiter,
		// skip limiter
		requestMethodDefinedInLimiter := libstring.StringInSlice(lmtMethods, r.Method)

		if !requestMethodDefinedInLimiter {
			return true
		}
	}

	// ---------------------------------
	// Filter by request headers
	lmtHeaders := lmt.GetHeaders()
	lmtHeadersIsSet := len(lmtHeaders) > 0

	if lmtHeadersIsSet {
		// If request does not contain all of the headers in limiter,
		// skip limiter
		requestHeadersDefinedInLimiter := false

		for headerKey := range lmtHeaders {
			reqHeaderValue := r.Header.Get(headerKey)
			if reqHeaderValue != "" {
				requestHeadersDefinedInLimiter = true
				break
			}
		}

		if !requestHeadersDefinedInLimiter {
			return true
		}

		// ------------------------------
		// If request contains the header key but not the values,
		// skip limiter
		requestHeadersDefinedInLimiter = false

		for headerKey, headerValues := range lmtHeaders {
			for _, headerValue := range headerValues {
				if r.Header.Get(headerKey) == headerValue {
					requestHeadersDefinedInLimiter = true
					break
				}
			}
		}

		if !requestHeadersDefinedInLimiter {
			return true
		}
	}

	// ---------------------------------
	// Filter by context values
	lmtContextValues := lmt.GetContextValues()
	lmtContextValuesIsSet := len(lmtContextValues) > 0

	if lmtContextValuesIsSet {
		// If request does not contain all of the contexts in limiter,
		// skip limiter
		requestContextValuesDefinedInLimiter := false

		for contextKey := range lmtContextValues {
			reqContextValue := fmt.Sprintf("%v", r.Context().Value(contextKey))
			if reqContextValue != "" {
				requestContextValuesDefinedInLimiter = true
				break
			}
		}

		if !requestContextValuesDefinedInLimiter {
			return true
		}

		// ------------------------------
		// If request contains the context key but not the values,
		// skip limiter
		requestContextValuesDefinedInLimiter = false

		for contextKey, contextValues := range lmtContextValues {
			for _, contextValue := range contextValues {
				if r.Header.Get(contextKey) == contextValue {
					requestContextValuesDefinedInLimiter = true
					break
				}
			}
		}

		if !requestContextValuesDefinedInLimiter {
			return true
		}
	}

	// ---------------------------------
	// Filter by basic auth usernames
	lmtBasicAuthUsers := lmt.GetBasicAuthUsers()
	lmtBasicAuthUsersIsSet := len(lmtBasicAuthUsers) > 0

	if lmtBasicAuthUsersIsSet {
		// If request does not contain all of the basic auth users in limiter,
		// skip limiter
		requestAuthUsernameDefinedInLimiter := false

		username, _, ok := r.BasicAuth()
		if ok && libstring.StringInSlice(lmtBasicAuthUsers, username) {
			requestAuthUsernameDefinedInLimiter = true
		}

		if !requestAuthUsernameDefinedInLimiter {
			return true
		}
	}

	return false
}

// BuildKeys generates a slice of keys to rate-limit by given limiter and request structs.
func BuildKeys(lmt *limiter.Limiter, r *http.Request) [][]string {
	remoteIP := libstring.RemoteIP(lmt.GetIPLookups(), lmt.GetForwardedForIndexFromBehind(), r)
	path := r.URL.Path
	sliceKeys := make([][]string, 0)

	lmtMethods := lmt.GetMethods()
	lmtHeaders := lmt.GetHeaders()
	lmtContextValues := lmt.GetContextValues()
	lmtBasicAuthUsers := lmt.GetBasicAuthUsers()

	lmtHeadersIsSet := len(lmtHeaders) > 0
	lmtContextValuesIsSet := len(lmtContextValues) > 0
	lmtBasicAuthUsersIsSet := len(lmtBasicAuthUsers) > 0

	usernameToLimit := ""
	if lmtBasicAuthUsersIsSet {
		username, _, ok := r.BasicAuth()
		if ok && libstring.StringInSlice(lmtBasicAuthUsers, username) {
			usernameToLimit = username
		}
	}

	headerValuesToLimit := [][]string{}
	if lmtHeadersIsSet {
		for headerKey, headerValues := range lmtHeaders {
			reqHeaderValue := r.Header.Get(headerKey)
			if reqHeaderValue == "" {
				continue
			}

			if len(headerValues) == 0 {
				// If header values are empty, rate-limit all request containing headerKey.
				headerValuesToLimit = append(headerValuesToLimit, []string{headerKey, reqHeaderValue})

			} else {
				// If header values are not empty, rate-limit all request with headerKey and headerValues.
				for _, headerValue := range headerValues {
					if r.Header.Get(headerKey) == headerValue {
						headerValuesToLimit = append(headerValuesToLimit, []string{headerKey, headerValue})
						break
					}
				}
			}
		}
	}

	contextValuesToLimit := [][]string{}
	if lmtContextValuesIsSet {
		for contextKey, contextValues := range lmtContextValues {
			reqContextValue := fmt.Sprintf("%v", r.Context().Value(contextKey))
			if reqContextValue == "" {
				continue
			}

			if len(contextValues) == 0 {
				// If context values are empty, rate-limit all request containing contextKey.
				contextValuesToLimit = append(contextValuesToLimit, []string{contextKey, reqContextValue})

			} else {
				// If context values are not empty, rate-limit all request with contextKey and contextValues.
				for _, contextValue := range contextValues {
					if reqContextValue == contextValue {
						contextValuesToLimit = append(contextValuesToLimit, []string{contextKey, contextValue})
						break
					}
				}
			}
		}
	}

	sliceKey := []string{remoteIP, path}

	sliceKey = append(sliceKey, lmtMethods...)

	for _, header := range headerValuesToLimit {
		sliceKey = append(sliceKey, header[0], header[1])
	}

	for _, contextValue := range contextValuesToLimit {
		sliceKey = append(sliceKey, contextValue[0], contextValue[1])
	}

	sliceKey = append(sliceKey, usernameToLimit)

	sliceKeys = append(sliceKeys, sliceKey)

	return sliceKeys
}

// LimitByRequest builds keys based on http.Request struct,
// loops through all the keys, and check if any one of them returns HTTPError.
func LimitByRequest(lmt *limiter.Limiter, w http.ResponseWriter, r *http.Request) *errors.HTTPError {
	setResponseHeaders(lmt, w, r)

	shouldSkip := ShouldSkipLimiter(lmt, r)
	if shouldSkip {
		return nil
	}

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
			if lmt.GetOverrideDefaultResponseWriter() {
				return
			}
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
