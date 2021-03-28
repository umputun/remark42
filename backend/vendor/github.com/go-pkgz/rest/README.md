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

### Deprecation 

Adds rhe HTTP Deprecation response header, see [draft-dalal-deprecation-header-00](https://tools.ietf.org/id/draft-dalal-deprecation-header-00.html
) 
    
## Helpers

- `rest.JSON` - map alias, just for convenience `type JSON map[string]interface{}`
- `rest.RenderJSON` -  renders json response from `interface{}`
- `rest.RenderJSONFromBytes` - renders json response from `[]byte`
- `rest.RenderJSONWithHTML` -  renders json response with html tags and forced `charset=utf-8`
- `rest.SendErrorJSON` - makes `{error: blah, details: blah}` json body and responds with given error code. Also, adds context to the logged message
- `rest.NewErrorLogger(l logger.Backend)` creates a struct providing shorter form of logger call
