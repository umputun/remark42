// Package routegroup provides a way to group routes and applies middleware to them.
// Works with the standard library's http.ServeMux.
package routegroup

import (
	"net/http"
	"regexp"
	"strings"
)

// Bundle represents a group of routes with associated middleware.
type Bundle struct {
	mux         *http.ServeMux                    // the underlying mux to register the routes to
	basePath    string                            // base path for the group
	middlewares []func(http.Handler) http.Handler // middlewares stack

	// optional custom 404 handler
	notFound http.HandlerFunc

	// root points to the root bundle for global middleware application.
	// for the root bundle, root == nil.
	root *Bundle

	// routesLocked indicates that routes have been registered on the root bundle
	// and no further root-level middlewares may be added.
	routesLocked bool

	// rootCount captures how many root middlewares were present when this bundle
	// was created. Used to avoid double-applying root middlewares for per-route wrapping.
	rootCount int
}

// New creates a new Group.
func New(mux *http.ServeMux) *Bundle {
	return &Bundle{mux: mux}
}

// Mount creates a new group with a specified base path.
func Mount(mux *http.ServeMux, basePath string) *Bundle {
	return &Bundle{mux: mux, basePath: basePath}
}

// ServeHTTP implements the http.Handler interface
func (b *Bundle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// resolve the root bundle (where global middlewares live).
	root := b
	if b.root != nil {
		root = b.root
	}

	// get the handler and pattern for this request
	_, pattern := b.mux.Handler(r)

	// if a pattern was found, create a shallow copy of the request with the pattern set
	// this allows global middlewares to see the pattern before mux.ServeHTTP is called
	if pattern != "" {
		r2 := *r
		r2.Pattern = pattern
		r = &r2
	}

	// create a handler that will let the mux do its routing (including setting path parameters)
	// but intercept 404s to use custom handler if provided
	muxHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if pattern == "" && root.notFound != nil {
			// no route matched, need to check if it's a true 404 or a 405
			// probe the mux to see what status it would return
			probe := &statusRecorder{status: http.StatusOK}
			b.mux.ServeHTTP(probe, r)

			// if mux wants to return 405 (Method Not Allowed), let it handle the request
			// to preserve the proper 405 response and Allow header
			if probe.status == http.StatusMethodNotAllowed {
				b.mux.ServeHTTP(w, r)
				return
			}

			// it's a true 404, use custom handler
			root.notFound.ServeHTTP(w, r)
			return
		}
		// let the mux handle the request normally (this sets path parameters)
		b.mux.ServeHTTP(w, r)
	})

	// apply root (global) middlewares around the mux handler and serve the request.
	root.wrapGlobal(muxHandler).ServeHTTP(w, r)
}

// Group creates a new group with the same middleware stack as the original on top of the existing bundle.
func (b *Bundle) Group() *Bundle {
	return b.clone() // copy the middlewares to avoid modifying the original
}

// Mount creates a new group with a specified base path on top of the existing bundle.
func (b *Bundle) Mount(basePath string) *Bundle {
	g := b.clone() // copy the middlewares to avoid modifying the original
	g.basePath += basePath
	return g
}

// Use adds middleware(s) to the Group.
// Middlewares are executed in the order they are added.
// Note: Root-level middlewares (added to the root bundle) have access to the matched
// route pattern via r.Pattern, but execute before path parameters are parsed.
// Therefore, r.PathValue() will return empty strings in root middlewares.
// Middlewares on mounted groups execute after routing and have full access to path values.
func (b *Bundle) Use(middleware func(http.Handler) http.Handler, more ...func(http.Handler) http.Handler) {
	// disallow adding middlewares after any routes have been registered on this bundle.
	if b.routesLocked {
		panic("routegroup: Use called after routes were registered on this bundle; add middlewares before registering routes or use Group/With for scoped middleware")
	}
	b.middlewares = append(b.middlewares, middleware)
	b.middlewares = append(b.middlewares, more...)
}

// With adds new middleware(s) to the Group and returns a new Group with the updated middleware stack.
// The With method is similar to Use, but instead of modifying the current Group,
// it returns a new Group instance with the added middleware(s).
// This allows for creating chain of middleware without affecting the original Group.
func (b *Bundle) With(middleware func(http.Handler) http.Handler, more ...func(http.Handler) http.Handler) *Bundle {
	newMiddlewares := make([]func(http.Handler) http.Handler, len(b.middlewares), len(b.middlewares)+len(more)+1)
	copy(newMiddlewares, b.middlewares)
	newMiddlewares = append(newMiddlewares, middleware)
	newMiddlewares = append(newMiddlewares, more...)
	// preserve root pointer and rootCount
	nb := &Bundle{mux: b.mux, basePath: b.basePath, middlewares: newMiddlewares, root: b.root, rootCount: b.rootCount}
	if nb.root == nil {
		// b is the root, so all b's middlewares are root middlewares
		nb.root = b
		nb.rootCount = len(b.middlewares)
	}
	return nb
}

// Handle adds a new route to the Group's mux, applying all middlewares to the handler.
func (b *Bundle) Handle(pattern string, handler http.Handler) {
	b.lockRoot() // lock root on first route registration

	// for file server paths (ending with /), preserve the pattern as-is
	if strings.HasSuffix(pattern, "/") {
		fullPath := b.basePath + pattern
		b.mux.Handle(fullPath, b.wrapMiddleware(handler))
		return
	}
	b.register(pattern, handler.ServeHTTP)
}

// HandleFiles is a helper to serve static files from a directory
func (b *Bundle) HandleFiles(pattern string, root http.FileSystem) {
	b.lockRoot() // lock root on first route registration

	// normalize pattern to always have trailing slash
	if !strings.HasSuffix(pattern, "/") {
		pattern += "/"
	}

	// build the full path for registration
	fullPath := b.basePath + pattern

	if pattern == "/" && b.basePath == "" {
		// root case - serve directly without stripping
		b.mux.Handle("/", b.wrapMiddleware(http.FileServer(root)))
		return
	}

	// for both mounted groups and prefixed paths, strip the fullPath
	handler := http.StripPrefix(strings.TrimSuffix(fullPath, "/"), http.FileServer(root))
	b.mux.Handle(fullPath, b.wrapMiddleware(handler))
}

// HandleFunc registers the handler function for the given pattern to the Group's mux.
// The handler is wrapped with the Group's middlewares.
func (b *Bundle) HandleFunc(pattern string, handler http.HandlerFunc) {
	b.register(pattern, handler)
}

// Handler returns the handler and the pattern that matches the request.
// It always returns a non-nil handler, see http.ServeMux.Handler documentation for details.
func (b *Bundle) Handler(r *http.Request) (h http.Handler, pattern string) {
	return b.mux.Handler(r)
}

// DisableNotFoundHandler used to disable auto-registration of a catch-all 404.
// Deprecated: now a no-op retained for API compatibility.
func (b *Bundle) DisableNotFoundHandler() {}

// NotFoundHandler sets a custom handler for any unmatched routes (404 responses).
// Note: This handler is only used for true 404s. Requests to valid paths with
// incorrect HTTP methods will still return 405 Method Not Allowed with Allow header.
func (b *Bundle) NotFoundHandler(handler http.HandlerFunc) {
	// always set on the root bundle so custom 404 works regardless of which bundle serves.
	if b.root != nil {
		b.root.notFound = handler
		return
	}
	b.notFound = handler
}

// matches non-space characters, spaces, then anything, i.e. "GET /path/to/resource"
var reGo122 = regexp.MustCompile(`^(\S+)\s+(.+)$`)

func (b *Bundle) register(pattern string, handler http.HandlerFunc) {
	b.lockRoot() // lock root on first route registration
	matches := reGo122.FindStringSubmatch(pattern)
	var path, method string
	if len(matches) > 2 { // path in the form "GET /path/to/resource"
		method = matches[1]
		path = matches[2]
		pattern = method + " " + b.basePath + path
	} else { // path is just "/path/to/resource"
		path = pattern
		pattern = b.basePath + pattern
		// method is not set intentionally here, the request pattern had no method part
	}
	// if the pattern is the root path on / change it to /{$}
	// this keeps handling the root request without becoming a catch-all
	if pattern == "/" || path == "/" {
		if method != "" { // preserve the method part if it was set
			pattern = method + " " + b.basePath + "/{$}"
		} else {
			pattern = b.basePath + "/{$}" // no method part, just the path
		}
	}
	b.mux.HandleFunc(pattern, b.wrapMiddleware(handler).ServeHTTP)
}

// Route allows for configuring the Group inside the configureFn function.
// When called on the root bundle, it automatically creates a new group to avoid
// accidentally modifying the root bundle's middleware stack.
func (b *Bundle) Route(configureFn func(*Bundle)) {
	// if called on root bundle, auto-create a group for better UX
	if b.root == nil {
		child := b.Group()
		configureFn(child)
		// if child registered routes, lock root too to prevent Use() after routes
		if child.routesLocked {
			b.routesLocked = true
		}
		return
	}
	configureFn(b)
}

// HandleRoot adds a handler for the group's root path without trailing slash.
// This avoids the 301 redirect that would occur with a "/" pattern.
// Method parameter can be empty to register for all HTTP methods.
func (b *Bundle) HandleRoot(method string, handler http.Handler) {
	b.lockRoot() // lock root on first route registration

	// for empty base path, use "/" to match the root
	pattern := b.basePath
	if pattern == "" {
		pattern = "/"
	}

	// add method if specified
	if method != "" {
		pattern = method + " " + pattern
	}

	b.mux.Handle(pattern, b.wrapMiddleware(handler))
}

// HandleRootFunc is like HandleRoot but takes a handler function.
func (b *Bundle) HandleRootFunc(method string, handler http.HandlerFunc) {
	b.lockRoot() // lock root on first route registration

	// for empty base path, use "/" to match the root
	pattern := b.basePath
	if pattern == "" {
		pattern = "/"
	}

	// add method if specified
	if method != "" {
		pattern = method + " " + pattern
	}

	b.mux.HandleFunc(pattern, b.wrapMiddleware(handler).ServeHTTP)
}

// wrapMiddleware applies the registered middlewares to a handler.
func (b *Bundle) wrapMiddleware(handler http.Handler) http.Handler {
	// root bundle: don't apply middlewares here, they're applied globally in ServeHTTP
	if b.root == nil {
		return handler
	}

	// child bundle: apply only middlewares added after mounting (exclude inherited root middlewares)
	start := b.rootCount
	if start > len(b.middlewares) {
		start = len(b.middlewares) // safety: ensure start doesn't exceed bounds
	}

	for i := len(b.middlewares) - 1; i >= start; i-- {
		handler = b.middlewares[i](handler)
	}
	return handler
}

func (b *Bundle) clone() *Bundle {
	middlewares := make([]func(http.Handler) http.Handler, len(b.middlewares))
	copy(middlewares, b.middlewares)
	// preserve root pointer and rootCount
	nb := &Bundle{mux: b.mux, basePath: b.basePath, middlewares: middlewares, root: b.root, rootCount: b.rootCount}
	if nb.root == nil {
		// b is the root, so all b's middlewares are root middlewares
		nb.root = b
		nb.rootCount = len(b.middlewares)
	}
	return nb
}

// Wrap directly wraps the handler with the provided middleware(s).
func Wrap(handler http.Handler, mw1 func(http.Handler) http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	return mw1(handler) // apply the first middleware
}

// wrapGlobal applies only the root bundle's middlewares to the provided handler.
func (b *Bundle) wrapGlobal(handler http.Handler) http.Handler {
	// resolve root bundle
	root := b
	if b.root != nil {
		root = b.root
	}
	for i := len(root.middlewares) - 1; i >= 0; i-- {
		handler = root.middlewares[i](handler)
	}
	return handler
}

// lockRoot marks this bundle as having registered routes.
func (b *Bundle) lockRoot() { b.routesLocked = true }

// statusRecorder is a minimal ResponseWriter that only records the status code.
// Used to probe what status the mux would return without actually writing a response.
type statusRecorder struct {
	status int
}

func (r *statusRecorder) Header() http.Header {
	return make(http.Header)
}

func (r *statusRecorder) Write([]byte) (int, error) {
	return 0, nil
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
}
