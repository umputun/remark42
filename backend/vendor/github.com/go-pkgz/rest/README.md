## REST helpers and middleware [![Build Status](https://travis-ci.org/go-pkgz/rest.svg?branch=master)](https://travis-ci.org/go-pkgz/rest) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/rest)](https://goreportcard.com/report/github.com/go-pkgz/rest) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/rest/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/rest?branch=master) [![godoc](https://godoc.org/github.com/go-pkgz/rest?status.svg)](https://godoc.org/github.com/go-pkgz/rest)


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

Responds with `pong` on `GET /ping`. Also responds to anything with `/ping` suffix, like `/v2/ping` 

example for both:

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

Logs all info about request, including user, method, status code, response size, url, elapsed time, request body (optional).
Can be customized by passing flags - LogNone, LogAll, LogUser and LogBody. Flags can be combined (provided multiple times)

### Recoverer middleware

Recoverer is a middleware that recovers from panics, logs the panic (and a backtrace), 
and returns a HTTP 500 (Internal Server Error) status if possible.

### OnlyFrom middleware

OnlyFrom middleware allows access for limited list of source IPs.
Such IPs can be defined as complete ip (like 192.168.1.12), prefix (129.168.) or CIDR (192.168.0.0/16)

### Metrics middleware

Metrics middleware responds to GET /metrics with list of [expvar](https://golang.org/pkg/expvar/). Optionally allows to restrict list of source ips.

### BlackWords middleware

BlackWords middleware doesn't allow user-defined words in the request body.

## Helpers

- `rest.JSON` - map alias, just for convenience `type JSON map[string]interface{}`
- `rest.RenderJSON` -  renders json response from `interface{}`
- `rest.RenderJSONFromBytes` - renders json response from `[]byte`
- `rest.RenderJSONWithHTML` -  renders json response with html tags and forced `charset=utf-8`
- `rest.SendErrorJSON` - makes `{error: blah, details: blah}` json body and responds with given error code. Also adds context to logged message

## Caching

Cache wrapper provides loading cache for rest/http responses. See [cache readme](https://github.com/go-pkgz/rest/tree/master/cache) for more details and examples.
