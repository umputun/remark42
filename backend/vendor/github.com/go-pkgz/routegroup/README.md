## routegroup [![Build Status](https://github.com/go-pkgz/routegroup/workflows/build/badge.svg)](https://github.com/go-pkgz/routegroup/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/routegroup)](https://goreportcard.com/report/github.com/go-pkgz/routegroup) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/routegroup/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/routegroup?branch=master) [![godoc](https://godoc.org/github.com/go-pkgz/routegroup?status.svg)](https://godoc.org/github.com/go-pkgz/routegroup)


`routegroup` is a tiny Go package providing a lightweight wrapper for efficient route grouping and middleware integration with the standard `http.ServeMux`.

## Features

- Simple and intuitive API for route grouping and route mounting.
- Lightweight, just about 100 LOC
- Easy middleware integration for individual routes or groups of routes.
- Seamless integration with Go's standard `http.ServeMux`.
- Fully compatible with the `http.Handler` interface and can be used as a drop-in replacement for `http.ServeMux`.
- No external dependencies.

## Requirements

- Go 1.23 or higher
  *(This library uses `http.Request.Pattern` to make route patterns available to global middlewares and relies on the enhanced `http.ServeMux` routing behavior introduced in Go 1.22/1.23)*

## Install and update

`go get -u github.com/go-pkgz/routegroup`

## Usage

**Creating a New Route Group**

To start, create a new route group without a base path:

```go
func main() {
    mux := http.NewServeMux()
    group := routegroup.New(mux)
}
```

**Adding Routes with Middleware**

Add routes to your group, optionally with middleware:

```go
    group.Use(loggingMiddleware, corsMiddleware)
    group.Handle("/hello", helloHandler)
    group.Handle("/bye", byeHandler)
```
**Creating a Nested Route Group**

For routes under a specific path prefix `Mount` method can be used to create a nested group:

```go
    apiGroup := routegroup.Mount(mux, "/api")
    apiGroup.Use(loggingMiddleware, corsMiddleware)
    apiGroup.Handle("/v1", apiV1Handler)
    apiGroup.Handle("/v2", apiV2Handler)

```

**Complete Example**

Here's a complete example demonstrating route grouping and middleware usage:

```go
package main

import (
	"net/http"

	"github.com/go-pkgz/routegroup"
)

func main() {
	router := routegroup.New(http.NewServeMux())
	router.Use(loggingMiddleware)

	// handle the /hello route
	router.Handle("GET /hello", helloHandler)
	
	// create a new group for the /api path
	apiRouter := router.Mount("/api")
	// add middleware
	apiRouter.Use(loggingMiddleware, corsMiddleware)

	// route handling
	apiRouter.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, API!"))
	})

	// add another group with its own set of middlewares
	protectedGroup := router.Group()
	protectedGroup.Use(authMiddleware)
	protectedGroup.HandleFunc("GET /protected", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Protected API!"))
	})

	http.ListenAndServe(":8080", router)
}
```

**Applying Middleware to Specific Routes**

You can also apply middleware to specific routes inside the group without modifying the group's middleware stack:

```go
apiGroup.With(corsMiddleware, apiMiddleware).Handle("GET /hello", helloHandler)
```

**Alternative Usage with `Route`**

You can also use the `Route` method to add routes and middleware in a single function call:

```go
router := routegroup.New(http.NewServeMux())
router.Route(func(b *routegroup.Bundle) {
    b.Use(loggingMiddleware, corsMiddleware)
    b.Handle("GET /hello", helloHandler)
    b.Handle("GET /bye", byeHandler)
})
http.ListenAndServe(":8080", router)
```

When called on the root bundle, `Route` automatically creates a new group to avoid accidentally modifying the root bundle's middleware stack. This means the middleware and routes defined inside the `Route` function are isolated from other routes on the root bundle.

The `Route` method can also be chained after `Mount` or `Group` for a more functional style:

```go
router := routegroup.New(http.NewServeMux())
router.Group().Route(func(b *routegroup.Bundle) {
    b.Use(loggingMiddleware, corsMiddleware)
    b.Handle("GET /hello", helloHandler)
    b.Handle("GET /bye", byeHandler)
})
```

**Setting optional `NotFoundHandler`**

It is possible to set a custom `NotFoundHandler` for the group. This handler will be called when no other route matches the request:

```go
group.NotFoundHandler(func(w http.ResponseWriter, _ *http.Request) {
    http.Error(w, "404 page not found, something is wrong!", http.StatusNotFound)
}
```

If a custom `NotFoundHandler` is not configured, `routegroup` will default to using the standard library behavior.

Note on 405: In the current design, `routegroup` applies root-level middlewares to all requests at the top level without installing a catch‑all route. This preserves native `405 Method Not Allowed` responses from `http.ServeMux` when a path exists but a wrong method is used. A configured `NotFoundHandler` is only invoked when no route matches; it does not interfere with 405 handling. The custom `NotFoundHandler` will have the root bundle's global middlewares applied to it.

Legacy note: `DisableNotFoundHandler()` is now a no‑op and preserved only for API compatibility.

### Middleware Ordering

- Call `Use(...)` before registering routes on the same bundle. Calling `Use` after any handler has been registered on that bundle will panic with a descriptive error.
- Root bundle middlewares (added via `router.Use(...)`) are applied globally to all requests at serve time.
- Group/bundle middlewares (added via `group.Use(...)`) apply to the routes registered on that bundle and its descendants, provided they are added before those routes.
- `With(...)` returns a new bundle; you can add middlewares there first, then register routes. This is the preferred way to add scoped middlewares without affecting previously defined routes.

**Important**: Route registration (HandleFunc, Handle, HandleFiles, etc.) should be done during initialization and not performed concurrently. The library is designed for typical usage where routes are registered at startup time in a single goroutine.

Examples

Incorrect: calling `Use` after routes on the same bundle (will panic)

```go
mux := http.NewServeMux()
router := routegroup.New(mux)

router.HandleFunc("/r", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

// This will panic: Use called after routes were registered on this bundle
router.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // global header
        w.Header().Set("X-Global", "true")
        next.ServeHTTP(w, r)
    })
})
```

Allowed: parent/root `Use` after child bundle routes

```go
mux := http.NewServeMux()
router := routegroup.New(mux)

child := router.Group()
child.HandleFunc("/child", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })

// Parent has not registered its own routes yet; this is allowed and will apply globally
router.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Parent", "true")
        next.ServeHTTP(w, r)
    })
})
```

Preferred: use `With` (or `Group`+`Use`) to attach scoped middleware before routes

```go
mux := http.NewServeMux()
router := routegroup.New(mux)

// Global middleware (optional), add before any root routes
router.Use(loggingMiddleware)

// Scoped middleware using With: returns a new bundle on which we can add routes
api := router.With(authMiddleware)
api.HandleFunc("GET /items", itemsHandler)
api.HandleFunc("POST /items", createItem)

// Or using Group + Use before routes
admin := router.Group()
admin.Use(adminOnly)
admin.HandleFunc("GET /dashboard", dashboardHandler)
```


**Handling Root Paths Without Trailing Slashes**

When working with mounted groups, you often need to handle requests to the group's root path without a trailing slash. For this purpose, `routegroup` provides the `HandleRoot` or `HandleRootFunc` methods:

```go
// Create mounted groups
apiGroup := router.Mount("/api")
v1Group := apiGroup.Mount("/v1")
usersGroup := v1Group.Mount("/users")

// Handle the root paths (no trailing slashes)
apiGroup.HandleRoot("GET", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // This handles requests to "/api" (without trailing slash)
    w.Write([]byte("API Documentation"))
}))

usersGroup.HandleRoot("GET", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // This handles requests to "/api/v1/users" (without trailing slash)
    w.Write([]byte("List users"))
}))

// Different HTTP methods can be handled separately
usersGroup.HandleRoot("POST", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // This handles POST requests to "/api/v1/users"
    w.Write([]byte("Create user"))
}))
```

While it's also possible to handle such paths using a trailing slash pattern (`"/"`) with the regular `Handle` or `HandleFunc` methods, that approach results in a redirect from non-trailing slash URLs (e.g., `/api`) to the trailing slash version (e.g., `/api/`). The `HandleRoot` method avoids this redirect, providing a more direct response and avoiding an extra round-trip, which is especially important for non-GET requests or when clients don't automatically follow redirects.

### Using derived groups

In some instances, it's practical to create an initial group that includes a set of middlewares, and then derive all other groups from it. This approach guarantees that every group incorporates a common set of middlewares as a foundation, allowing each to add its specific middlewares. To facilitate this scenario, `routegroup` offers both `Bundle.Group` and `Bundle.Mount` methods, and it also implements the `http.Handler` interface. The following example illustrates how to use derived groups:

```go
// create a new bundle with a base set of middlewares
// note: the bundle is also http.Handler and can be passed to http.ListenAndServe
router := routegroup.New(http.NewServeMux()) 
router.Use(loggingMiddleware, corsMiddleware)

// add a new, derived group with its own set of middlewares
// this group will inherit the middlewares from the base group
apiGroup := router.Group()
apiGroup.Use(apiMiddleware)
apiGroup.Handle("GET /hello", helloHandler)
apiGroup.Handle("GET /bye", byeHandler)

// mount another group for the /admin path with its own set of middlewares, 
// using `Route` method to show the alternative usage.
// this group will inherit the middlewares from the base group as well
router.Mount("/admin").Route(func(b *routegroup.Bundle) {
    b.Use(adminMiddleware)
    b.Handle("POST /do", doHandler)
})

// start the server, passing the wrapped mux as the handler
http.ListenAndServe(":8080", router)
```
### Wrap Function

Sometimes route's group is not necessary, and all you need is to apply middleware(s) directly to a single route. In this case, `routegroup` provides a `Wrap` function that can be used to wrap a single `http.Handler` with one or more middlewares. Here's an example:

```go
mux := http.NewServeMux()
mux.HandleFunc("/hello", routegroup.Wrap(helloHandler, loggingMiddleware, corsMiddleware))
http.ListenAndServe(":8080", mux)
```

### 404 and 405 behavior

`routegroup` applies the root bundle's middlewares to all requests at the top level. This keeps the standard library's matching logic intact:
- Wrong method on an existing path returns `405 Method Not Allowed` (with an `Allow` header).
- Unknown path returns `404 Not Found`.

You can optionally configure a custom 404 handler with `NotFoundHandler(fn)`. It will run only when no route matches and does not affect 405 handling. The custom handler will have global middlewares applied to it. The legacy `DisableNotFoundHandler()` is now a no‑op and kept only for compatibility.

### HandleFiles helper

`routegroup` provides a helper function `HandleFiles` that can be used to serve static files from a directory. The function is a thin wrapper around the standard `http.FileServer` and can be used to serve files from a specific directory. Here's an example:

```go
// serve static files from the "assets/static" directory
router.HandleFiles("/static/", http.Dir("assets/static"))
```

## Real-world example

Here's an example of how `routegroup` can be used in a real-world application. The following code snippet is taken from a web service that provides a set of routes for user authentication, session management, and user management. The service also serves static files from the "assets/static" embedded file system.

```go

// Routes returns http.Handler that handles all the routes for the Service.
// It also serves static files from the "assets/static" directory.
// The rootURL option sets prefix for the routes.
func (s *Service) Routes() http.Handler {
	router := routegroup.Mount(http.NewServeMux(), s.rootURL) // make a bundle with the rootURL base path
	// add common middlewares
	router.Use(rest.Maybe(handlers.CompressHandler, func(*http.Request) bool { return !s.skipGZ }))
	router.Use(rest.Throttle(s.limitActiveReqs))
	router.Use(s.middleware.securityHeaders(s.skipSecurityHeaders))

	// prepare csrf middleware
	csrfMiddleware := s.middleware.csrf(s.skipCSRFCheck)

	// add open routes
	router.HandleFunc("GET /login", s.loginPageHandler)
	router.HandleFunc("POST /login", s.loginCheckHandler)
	router.HandleFunc("GET /logout", s.logoutHandler)

	// add routes with auth middleware
	router.Group().Route(func(auth *routegroup.Bundle) {
		auth.Use(s.middleware.Auth())
		auth.HandleFunc("GET /update", s.pwdUpdateHandler)
		auth.With(csrfMiddleware).HandleFunc("PUT /update", s.pwdUpdateHandler)
	})

	// add admin routes
	router.Mount("/admin").Route(func(admin *routegroup.Bundle) {
		admin.Use(s.middleware.Auth("admin"))
		admin.Use(s.middleware.AdminOnly)
		admin.HandleFunc("GET /", s.admin.renderHandler)
		admin.With(csrfMiddleware).Route(func(csrf *routegroup.Bundle) {
			csrf.HandleFunc("DELETE /sessions", s.admin.deleteSessionsHandler)
			csrf.HandleFunc("POST /user", s.admin.addUserHandler)
			csrf.HandleFunc("DELETE /user", s.admin.deleteUserHandler)
		})
	})

	router.HandleFunc("GET /static/*", s.fileServerHandlerFunc()) // serve static files
	return router
}

// fileServerHandlerFunc returns http.HandlerFunc that serves static files from the "assets/static" directory.
// prefix is set by the rootURL option.
func (s *Service) fileServerHandlerFunc() http.HandlerFunc {
    staticFS, err := fs.Sub(assets, "assets/static") // error is always nil
    if err != nil {
        panic(err) // should never happen we load from embedded FS
    }
    return func(w http.ResponseWriter, r *http.Request) {
        webFS := http.StripPrefix(s.rootURL+"/static/", http.FileServer(http.FS(staticFS)))
        webFS.ServeHTTP(w, r)
    }
}
```

## Contributing

Contributions to `routegroup` are welcome! Please submit a pull request or open an issue for any bugs or feature requests.

## License

`routegroup` is available under the MIT license. See the [LICENSE](https://github.com/go-pkgz/routegroup/blob/master/LICENSE) file for more info.
