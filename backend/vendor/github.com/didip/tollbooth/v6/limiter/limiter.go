// Package limiter provides data structure to configure rate-limiter.
package limiter

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-pkgz/expirable-cache"
	"golang.org/x/time/rate"
)

// New is a constructor for Limiter.
func New(generalExpirableOptions *ExpirableOptions) *Limiter {
	lmt := &Limiter{}

	lmt.SetMessageContentType("text/plain; charset=utf-8").
		SetMessage("You have reached maximum request limit.").
		SetStatusCode(429).
		SetOnLimitReached(nil).
		SetIPLookups([]string{"RemoteAddr", "X-Forwarded-For", "X-Real-IP"}).
		SetForwardedForIndexFromBehind(0).
		SetHeaders(make(map[string][]string)).
		SetContextValues(make(map[string][]string))

	if generalExpirableOptions != nil {
		lmt.generalExpirableOptions = generalExpirableOptions
	} else {
		lmt.generalExpirableOptions = &ExpirableOptions{}
	}

	// Default for DefaultExpirationTTL is 10 years.
	if lmt.generalExpirableOptions.DefaultExpirationTTL <= 0 {
		lmt.generalExpirableOptions.DefaultExpirationTTL = 87600 * time.Hour
	}

	lmt.tokenBuckets, _ = cache.NewCache(cache.TTL(lmt.generalExpirableOptions.DefaultExpirationTTL))

	lmt.basicAuthUsers, _ = cache.NewCache(cache.TTL(lmt.generalExpirableOptions.DefaultExpirationTTL))

	return lmt
}

// Limiter is a config struct to limit a particular request handler.
type Limiter struct {
	// Maximum number of requests to limit per second.
	max float64

	// Limiter burst size
	burst int

	// HTTP message when limit is reached.
	message string

	// Content-Type for Message
	messageContentType string

	// HTTP status code when limit is reached.
	statusCode int

	// A function to call when a request is rejected.
	onLimitReached func(w http.ResponseWriter, r *http.Request)

	// An option to write back what you want upon reaching a limit.
	overrideDefaultResponseWriter bool

	// List of places to look up IP address.
	// Default is "RemoteAddr", "X-Forwarded-For", "X-Real-IP".
	// You can rearrange the order as you like.
	ipLookups []string

	forwardedForIndex int

	// List of HTTP Methods to limit (GET, POST, PUT, etc.).
	// Empty means limit all methods.
	methods []string

	// Able to configure token bucket expirations.
	generalExpirableOptions *ExpirableOptions

	// List of basic auth usernames to limit.
	basicAuthUsers cache.Cache

	// Map of HTTP headers to limit.
	// Empty means skip headers checking.
	headers map[string]cache.Cache

	// Map of Context values to limit.
	contextValues map[string]cache.Cache

	// Map of limiters with TTL
	tokenBuckets cache.Cache

	tokenBucketExpirationTTL  time.Duration
	basicAuthExpirationTTL    time.Duration
	headerEntryExpirationTTL  time.Duration
	contextEntryExpirationTTL time.Duration

	sync.RWMutex
}

// SetTokenBucketExpirationTTL is thread-safe way of setting custom token bucket expiration TTL.
func (l *Limiter) SetTokenBucketExpirationTTL(ttl time.Duration) *Limiter {
	l.Lock()
	l.tokenBucketExpirationTTL = ttl
	l.Unlock()

	return l
}

// GetTokenBucketExpirationTTL is thread-safe way of getting custom token bucket expiration TTL.
func (l *Limiter) GetTokenBucketExpirationTTL() time.Duration {
	l.RLock()
	defer l.RUnlock()
	return l.tokenBucketExpirationTTL
}

// SetBasicAuthExpirationTTL is thread-safe way of setting custom basic auth expiration TTL.
func (l *Limiter) SetBasicAuthExpirationTTL(ttl time.Duration) *Limiter {
	l.Lock()
	l.basicAuthExpirationTTL = ttl
	l.Unlock()

	return l
}

// GetBasicAuthExpirationTTL is thread-safe way of getting custom basic auth expiration TTL.
func (l *Limiter) GetBasicAuthExpirationTTL() time.Duration {
	l.RLock()
	defer l.RUnlock()
	return l.basicAuthExpirationTTL
}

// SetHeaderEntryExpirationTTL is thread-safe way of setting custom basic auth expiration TTL.
func (l *Limiter) SetHeaderEntryExpirationTTL(ttl time.Duration) *Limiter {
	l.Lock()
	l.headerEntryExpirationTTL = ttl
	l.Unlock()

	return l
}

// GetHeaderEntryExpirationTTL is thread-safe way of getting custom basic auth expiration TTL.
func (l *Limiter) GetHeaderEntryExpirationTTL() time.Duration {
	l.RLock()
	defer l.RUnlock()
	return l.headerEntryExpirationTTL
}

// SetContextValueEntryExpirationTTL is thread-safe way of setting custom Context value expiration TTL.
func (l *Limiter) SetContextValueEntryExpirationTTL(ttl time.Duration) *Limiter {
	l.Lock()
	l.contextEntryExpirationTTL = ttl
	l.Unlock()

	return l
}

// GetContextValueEntryExpirationTTL is thread-safe way of getting custom Context value expiration TTL.
func (l *Limiter) GetContextValueEntryExpirationTTL() time.Duration {
	l.RLock()
	defer l.RUnlock()
	return l.contextEntryExpirationTTL
}

// SetMax is thread-safe way of setting maximum number of requests to limit per second.
func (l *Limiter) SetMax(max float64) *Limiter {
	l.Lock()
	l.max = max
	l.Unlock()

	return l
}

// GetMax is thread-safe way of getting maximum number of requests to limit per second.
func (l *Limiter) GetMax() float64 {
	l.RLock()
	defer l.RUnlock()
	return l.max
}

// SetBurst is thread-safe way of setting maximum burst size.
func (l *Limiter) SetBurst(burst int) *Limiter {
	l.Lock()
	l.burst = burst
	l.Unlock()

	return l
}

// GetBurst is thread-safe way of setting maximum burst size.
func (l *Limiter) GetBurst() int {
	l.RLock()
	defer l.RUnlock()

	return l.burst
}

// SetMessage is thread-safe way of setting HTTP message when limit is reached.
func (l *Limiter) SetMessage(msg string) *Limiter {
	l.Lock()
	l.message = msg
	l.Unlock()

	return l
}

// GetMessage is thread-safe way of getting HTTP message when limit is reached.
func (l *Limiter) GetMessage() string {
	l.RLock()
	defer l.RUnlock()
	return l.message
}

// SetMessageContentType is thread-safe way of setting HTTP message Content-Type when limit is reached.
func (l *Limiter) SetMessageContentType(contentType string) *Limiter {
	l.Lock()
	l.messageContentType = contentType
	l.Unlock()

	return l
}

// GetMessageContentType is thread-safe way of getting HTTP message Content-Type when limit is reached.
func (l *Limiter) GetMessageContentType() string {
	l.RLock()
	defer l.RUnlock()
	return l.messageContentType
}

// SetStatusCode is thread-safe way of setting HTTP status code when limit is reached.
func (l *Limiter) SetStatusCode(statusCode int) *Limiter {
	l.Lock()
	l.statusCode = statusCode
	l.Unlock()

	return l
}

// GetStatusCode is thread-safe way of getting HTTP status code when limit is reached.
func (l *Limiter) GetStatusCode() int {
	l.RLock()
	defer l.RUnlock()
	return l.statusCode
}

// SetOnLimitReached is thread-safe way of setting after-rejection function when limit is reached.
func (l *Limiter) SetOnLimitReached(fn func(w http.ResponseWriter, r *http.Request)) *Limiter {
	l.Lock()
	l.onLimitReached = fn
	l.Unlock()

	return l
}

// ExecOnLimitReached is thread-safe way of executing after-rejection function when limit is reached.
func (l *Limiter) ExecOnLimitReached(w http.ResponseWriter, r *http.Request) {
	l.RLock()
	defer l.RUnlock()

	fn := l.onLimitReached
	if fn != nil {
		fn(w, r)
	}
}

// SetOverrideDefaultResponseWriter is a thread-safe way of setting the response writer override variable.
func (l *Limiter) SetOverrideDefaultResponseWriter(override bool) {
	l.Lock()
	l.overrideDefaultResponseWriter = override
	l.Unlock()
}

// GetOverrideDefaultResponseWriter is a thread-safe way of getting the response writer override variable.
func (l *Limiter) GetOverrideDefaultResponseWriter() bool {
	l.RLock()
	defer l.RUnlock()
	return l.overrideDefaultResponseWriter
}

// SetIPLookups is thread-safe way of setting list of places to look up IP address.
func (l *Limiter) SetIPLookups(ipLookups []string) *Limiter {
	l.Lock()
	l.ipLookups = ipLookups
	l.Unlock()

	return l
}

// GetIPLookups is thread-safe way of getting list of places to look up IP address.
func (l *Limiter) GetIPLookups() []string {
	l.RLock()
	defer l.RUnlock()
	return l.ipLookups
}

// SetForwardedForIndexFromBehind is thread-safe way of setting which X-Forwarded-For index to choose.
func (l *Limiter) SetForwardedForIndexFromBehind(forwardedForIndex int) *Limiter {
	l.Lock()
	l.forwardedForIndex = forwardedForIndex
	l.Unlock()

	return l
}

// GetForwardedForIndexFromBehind is thread-safe way of getting which X-Forwarded-For index to choose.
func (l *Limiter) GetForwardedForIndexFromBehind() int {
	l.RLock()
	defer l.RUnlock()
	return l.forwardedForIndex
}

// SetMethods is thread-safe way of setting list of HTTP Methods to limit (GET, POST, PUT, etc.).
func (l *Limiter) SetMethods(methods []string) *Limiter {
	l.Lock()
	l.methods = methods
	l.Unlock()

	return l
}

// GetMethods is thread-safe way of getting list of HTTP Methods to limit (GET, POST, PUT, etc.).
func (l *Limiter) GetMethods() []string {
	l.RLock()
	defer l.RUnlock()
	return l.methods
}

// SetBasicAuthUsers is thread-safe way of setting list of basic auth usernames to limit.
func (l *Limiter) SetBasicAuthUsers(basicAuthUsers []string) *Limiter {
	ttl := l.GetBasicAuthExpirationTTL()
	if ttl <= 0 {
		ttl = l.generalExpirableOptions.DefaultExpirationTTL
	}

	for _, basicAuthUser := range basicAuthUsers {
		l.basicAuthUsers.Set(basicAuthUser, true, ttl)
	}

	return l
}

// GetBasicAuthUsers is thread-safe way of getting list of basic auth usernames to limit.
func (l *Limiter) GetBasicAuthUsers() []string {
	return l.basicAuthUsers.Keys()
}

// RemoveBasicAuthUsers is thread-safe way of removing basic auth usernames from existing list.
func (l *Limiter) RemoveBasicAuthUsers(basicAuthUsers []string) *Limiter {
	for _, toBeRemoved := range basicAuthUsers {
		l.basicAuthUsers.Invalidate(toBeRemoved)
	}

	return l
}

// SetHeaders is thread-safe way of setting map of HTTP headers to limit.
func (l *Limiter) SetHeaders(headers map[string][]string) *Limiter {
	if l.headers == nil {
		l.headers = make(map[string]cache.Cache)
	}

	for header, entries := range headers {
		l.SetHeader(header, entries)
	}

	return l
}

// GetHeaders is thread-safe way of getting map of HTTP headers to limit.
func (l *Limiter) GetHeaders() map[string][]string {
	results := make(map[string][]string)

	l.RLock()
	defer l.RUnlock()

	for header, entriesAsGoCache := range l.headers {
		results[header] = entriesAsGoCache.Keys()
	}

	return results
}

// SetHeader is thread-safe way of setting entries of 1 HTTP header.
func (l *Limiter) SetHeader(header string, entries []string) *Limiter {
	l.RLock()
	existing, found := l.headers[header]
	l.RUnlock()

	ttl := l.GetHeaderEntryExpirationTTL()
	if ttl <= 0 {
		ttl = l.generalExpirableOptions.DefaultExpirationTTL
	}

	if !found {
		existing, _ = cache.NewCache(cache.TTL(ttl))
	}

	for _, entry := range entries {
		existing.Set(entry, true, ttl)
	}

	l.Lock()
	l.headers[header] = existing
	l.Unlock()

	return l
}

// GetHeader is thread-safe way of getting entries of 1 HTTP header.
func (l *Limiter) GetHeader(header string) []string {
	l.RLock()
	entriesAsGoCache := l.headers[header]
	l.RUnlock()

	return entriesAsGoCache.Keys()
}

// RemoveHeader is thread-safe way of removing entries of 1 HTTP header.
func (l *Limiter) RemoveHeader(header string) *Limiter {
	ttl := l.GetHeaderEntryExpirationTTL()
	if ttl <= 0 {
		ttl = l.generalExpirableOptions.DefaultExpirationTTL
	}

	l.Lock()
	l.headers[header], _ = cache.NewCache(cache.TTL(ttl))
	l.Unlock()

	return l
}

// RemoveHeaderEntries is thread-safe way of removing new entries to 1 HTTP header rule.
func (l *Limiter) RemoveHeaderEntries(header string, entriesForRemoval []string) *Limiter {
	l.RLock()
	entries, found := l.headers[header]
	l.RUnlock()

	if !found {
		return l
	}

	for _, toBeRemoved := range entriesForRemoval {
		entries.Invalidate(toBeRemoved)
	}

	return l
}

// SetContextValues is thread-safe way of setting map of HTTP headers to limit.
func (l *Limiter) SetContextValues(contextValues map[string][]string) *Limiter {
	if l.contextValues == nil {
		l.contextValues = make(map[string]cache.Cache)
	}

	for contextValue, entries := range contextValues {
		l.SetContextValue(contextValue, entries)
	}

	return l
}

// GetContextValues is thread-safe way of getting a map of Context values to limit.
func (l *Limiter) GetContextValues() map[string][]string {
	results := make(map[string][]string)

	l.RLock()
	defer l.RUnlock()

	for contextValue, entriesAsGoCache := range l.contextValues {
		results[contextValue] = entriesAsGoCache.Keys()
	}

	return results
}

// SetContextValue is thread-safe way of setting entries of 1 Context value.
func (l *Limiter) SetContextValue(contextValue string, entries []string) *Limiter {
	l.RLock()
	existing, found := l.contextValues[contextValue]
	l.RUnlock()

	ttl := l.GetContextValueEntryExpirationTTL()
	if ttl <= 0 {
		ttl = l.generalExpirableOptions.DefaultExpirationTTL
	}

	if !found {
		existing, _ = cache.NewCache(cache.TTL(ttl))
	}

	for _, entry := range entries {
		existing.Set(entry, true, ttl)
	}

	l.Lock()
	l.contextValues[contextValue] = existing
	l.Unlock()

	return l
}

// GetContextValue is thread-safe way of getting 1 Context value entry.
func (l *Limiter) GetContextValue(contextValue string) []string {
	l.RLock()
	entriesAsGoCache := l.contextValues[contextValue]
	l.RUnlock()

	return entriesAsGoCache.Keys()
}

// RemoveContextValue is thread-safe way of removing entries of 1 Context value.
func (l *Limiter) RemoveContextValue(contextValue string) *Limiter {
	ttl := l.GetContextValueEntryExpirationTTL()
	if ttl <= 0 {
		ttl = l.generalExpirableOptions.DefaultExpirationTTL
	}

	l.Lock()
	l.contextValues[contextValue], _ = cache.NewCache(cache.TTL(ttl))
	l.Unlock()

	return l
}

// RemoveContextValuesEntries is thread-safe way of removing entries to a ContextValue.
func (l *Limiter) RemoveContextValuesEntries(contextValue string, entriesForRemoval []string) *Limiter {
	l.RLock()
	entries, found := l.contextValues[contextValue]
	l.RUnlock()

	if !found {
		return l
	}

	for _, toBeRemoved := range entriesForRemoval {
		entries.Invalidate(toBeRemoved)
	}

	return l
}

func (l *Limiter) limitReachedWithTokenBucketTTL(key string, tokenBucketTTL time.Duration) bool {
	lmtMax := l.GetMax()
	lmtBurst := l.GetBurst()
	l.Lock()
	defer l.Unlock()

	if _, found := l.tokenBuckets.Get(key); !found {
		l.tokenBuckets.Set(
			key,
			rate.NewLimiter(rate.Limit(lmtMax), lmtBurst),
			tokenBucketTTL,
		)
	}

	expiringMap, found := l.tokenBuckets.Get(key)
	if !found {
		return false
	}

	return !expiringMap.(*rate.Limiter).Allow()
}

// LimitReached returns a bool indicating if the Bucket identified by key ran out of tokens.
func (l *Limiter) LimitReached(key string) bool {
	ttl := l.GetTokenBucketExpirationTTL()

	if ttl <= 0 {
		ttl = l.generalExpirableOptions.DefaultExpirationTTL
	}

	return l.limitReachedWithTokenBucketTTL(key, ttl)
}
