## REST helpers and middleware [![Build Status](https://github.com/go-pkgz/rest/workflows/build/badge.svg)](https://github.com/go-pkgz/rest/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/rest)](https://goreportcard.com/report/github.com/go-pkgz/rest) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/rest/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/rest?branch=master) [![godoc](https://godoc.org/github.com/go-pkgz/rest?status.svg)](https://godoc.org/github.com/go-pkgz/rest)


## Install and update

`go get -u github.com/go-pkgz/rest`

## Middlewares 

### AppInfo middleware

Adds info to every response header:
- App-Name - application name
- App-Version - application version
- Org - organization
- M-Host - host name from instance-level `$MHOST` env

### Ping-Pong middleware

Responds with `pong` on `GET /ping`. Also, responds to anything with `/ping` suffix, like `/v2/ping`.

Example for both:

```
> http GET https://remark42.radio-t.com/ping

HTTP/1.1 200 OK
Date: Sun, 15 Jul 2018 19:40:31 GMT
Content-Type: text/plain
Content-Length: 4
Connection: keep-alive
App-Name: remark42
App-Version: master-ed92a0b-20180630-15:59:56
Org: Umputun

pong
```

### Health middleware

Responds with the status 200 if all health checks passed, 503 if any failed. Both health path and check functions passed by consumer.
For production usage this middleware should be used with throttler/limiter and, optionally, with some auth middlewares

Example of usage:

```go
    check1 := func(ctx context.Context) (name string, err error) {
        // do some check, for example check DB connection		
		return "check1", nil // all good, passed
    }
    check2 := func(ctx context.Context) (name string, err error) {
        // do some other check, for example ping an external service		
		return "check2", errors.New("some error") // check failed
    }

    router := chi.NewRouter()
	router.Use(rest.Health("/health", check1, check2))
```

example of the actual call and response:

```
> http GET https://example.com/health

HTTP/1.1 503 Service Unavailable
Date: Sun, 15 Jul 2018 19:40:31 GMT
Content-Type: application/json; charset=utf-8
Content-Length: 36

[
    {"name":"check1","status":"ok"},
    {"name":"check2","status":"failed","error":"some error"}
]
```

_this middleware is pretty basic, but can be used for simple health checks. For more complex cases, like async/cached health checks see [alexliesenfeld/health](https://github.com/alexliesenfeld/health)_

### Logger middleware

Logs request, request handling time and response. Log record fields in order of occurrence:

- Request's HTTP method
- Requested URL (with sanitized query)
- Remote IP
- Response's HTTP status code
- Response body size
- Request handling time
- Userinfo associated with the request (optional)
- Request subject (optional)
- Request ID (if `X-Request-ID` present)
- Request body (optional)

_remote IP can be masked with user defined function_

example: `019/03/05 17:26:12.976 [INFO] GET - /api/v1/find?site=remark - 8e228e9cfece - 200 (115) - 4.47784618s`

### Recoverer middleware

Recoverer is a middleware that recovers from panics, logs the panic (and a backtrace), 
and returns an HTTP 500 (Internal Server Error) status if possible.

### OnlyFrom middleware

OnlyFrom middleware allows access for limited list of source IPs.
Such IPs can be defined as complete ip (like 192.168.1.12), prefix (129.168.) or CIDR (192.168.0.0/16)

### Metrics middleware

Metrics middleware responds to GET /metrics with list of [expvar](https://golang.org/pkg/expvar/). Optionally allows restricting list of source ips.

### BlackWords middleware

BlackWords middleware doesn't allow user-defined words in the request body.

### SizeLimit middleware

SizeLimit middleware checks if body size is above the limit and returns `StatusRequestEntityTooLarge` (413) 

### Trace middleware

It looks for `X-Request-ID` header and makes it as a random id
 (if not found), then populates it to the result's header
    and to the request's context.

### Deprecation middleware

Adds the HTTP Deprecation response header, see [draft-dalal-deprecation-header-00](https://tools.ietf.org/id/draft-dalal-deprecation-header-00.html
) 

### BasicAuth middleware

BasicAuth middleware requires basic auth and matches user & passwd with client-provided checker. In case if no basic auth headers returns
`StatusUnauthorized`, in case if checker failed - `StatusForbidden`

### Rewrite middleware

Rewrites requests with from->to rule. Supports regex (like nginx) and prevents multiple rewrites. For example `Rewrite("^/sites/(.*)/settings/$", "/sites/settings/$1")` will change request's URL from `/sites/id1/settings/` to `/sites/settings/id1`

### NoCache middleware

Sets a number of HTTP headers to prevent a router (handler's) response from being cached by an upstream proxy and/or client.

### Headers middleware

Sets headers (passed as key:value) to requests. I.e. `rest.Headers("Server:MyServer", "X-Blah:Foo")`

### Gzip middleware

Compresses response with gzip.

### RealIP middleware

RealIP is a middleware that sets a http.Request's RemoteAddr to the results of parsing either the X-Forwarded-For or X-Real-IP headers.

### Maybe middleware

Maybe middleware will allow you to change the flow of the middleware stack execution depending on return
value of maybeFn(request). This is useful for example if you'd like to skip a middleware handler if
a request does not satisfy the maybeFn logic.

### Reject middleware

Reject is a middleware that rejects requests with a given status code and message based on a user-defined function.
This is useful for example if you'd like to reject requests to a particular resource based on a request header, or want to implement a conditional request handler based on service parameters.

example with chi router:

```go
    router := chi.NewRouter()
	
	rejectFn := func(r *http.Request) (bool) {
        return r.Header.Get("X-Request-Id") == "" // reject if no X-Request-Id header
    }
	
	router.Use(rest.Reject(http.StatusBadRequest, "X-Request-Id header is required", rejectFn))
```

### Benchmarks middleware

Benchmarks middleware allows to measure the time of request handling, number of request per second and report aggregated metrics. This middleware keeps track of the request in the memory and keep up to 900 points (15 minutes, data-point per second).

In order to retrieve the data user should call `Stats(d duration)` method. duration is the time window for which the benchmark data should be returned. It can be any duration from 1s to 15m. Note: all the time data is in microseconds.

example with chi router:

```go
    router := chi.NewRouter()
	bench = rest.NewBenchmarks()
	router.Use(bench.Middleware)
	...
	router.Get("/bench", func(w http.ResponseWriter, r *http.Request) {
        resp := struct {
            OneMin     rest.BenchmarkStats `json:"1min"`
            FiveMin    rest.BenchmarkStats `json:"5min"`
            FifteenMin rest.BenchmarkStats `json:"15min"`
        }{
            bench.Stats(time.Minute),
            bench.Stats(time.Minute * 5),
            bench.Stats(time.Minute * 15),
        }
        render.JSON(w, r, resp) 		
    })
```

## Helpers

- `rest.Wrap` - converts a list of middlewares to nested handlers calls (in reverse order)
- `rest.JSON` - map alias, just for convenience `type JSON map[string]interface{}`
- `rest.RenderJSON` -  renders json response from `interface{}`
- `rest.RenderJSONFromBytes` - renders json response from `[]byte`
- `rest.RenderJSONWithHTML` -  renders json response with html tags and forced `charset=utf-8`
- `rest.SendErrorJSON` - makes `{error: blah, details: blah}` json body and responds with given error code. Also, adds context to the logged message
- `rest.NewErrorLogger` - creates a struct providing shorter form of logger call
- `rest.FileServer` - creates a file server for static assets with directory listing disabled
- `realip.Get` - returns client's IP address
- `rest.ParseFromTo` - parses "from" and "to" request's query params with various formats

## Profiler

Profiler is a convenient subrouter used for mounting net/http/pprof, i.e.

```go
 func MyService() http.Handler {
   r := chi.NewRouter()
   // ..middlewares
   r.Mount("/debug", middleware.Profiler())
   // ..routes
   return r
 }
```

It exposes a bunch of `/pprof/*` endpoints as well as `/vars`. Builtin support for `onlyIps` allows restricting access, which is important if it runs on a publicly exposed port. However, counting on IP check only is not that reliable way to limit request and for production use it would be better to add some sort of auth (for example provided `BasicAuth` middleware) or run with a separate http server, exposed to internal ip/port only.

