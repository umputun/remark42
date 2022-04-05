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

## Rewrite middleware

Rewrites requests with from->to rule. Supports regex (like nginx) and prevents multiple rewrites. For example `Rewrite("^/sites/(.*)/settings/$", "/sites/settings/$1")` will change request's URL from `/sites/id1/settings/` to `/sites/settings/id1`

## NoCache middleware

Sets a number of HTTP headers to prevent a router (handler's) response from being cached by an upstream proxy and/or client.

## Headers middleware

Sets headers (passed as key:value) to requests. I.e. `rest.Headers("Server:MyServer", "X-Blah:Foo")`

## Gzip middleware

Compresses response with gzip.

## RealIP middleware

RealIP is a middleware that sets a http.Request's RemoteAddr to the results of parsing either the X-Forwarded-For or X-Real-IP headers.

## Maybe middleware

Maybe middleware will allow you to change the flow of the middleware stack execution depending on return
value of maybeFn(request). This is useful for example if you'd like to skip a middleware handler if
a request does not satisfy the maybeFn logic.

## Headers middleware

Headers middleware adds headers to request


## Helpers

- `rest.Wrap` - converts a list of middlewares to nested handlers calls (in reverse order)
- `rest.JSON` - map alias, just for convenience `type JSON map[string]interface{}`
- `rest.RenderJSON` -  renders json response from `interface{}`
- `rest.RenderJSONFromBytes` - renders json response from `[]byte`
- `rest.RenderJSONWithHTML` -  renders json response with html tags and forced `charset=utf-8`
- `rest.SendErrorJSON` - makes `{error: blah, details: blah}` json body and responds with given error code. Also, adds context to the logged message
- `rest.NewErrorLogger` - creates a struct providing shorter form of logger call
- `rest.FileServer` - creates a file server for static assets with directory listing disabled

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

