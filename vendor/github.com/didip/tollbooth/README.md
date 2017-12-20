[![GoDoc](https://godoc.org/github.com/didip/tollbooth?status.svg)](http://godoc.org/github.com/didip/tollbooth)
[![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/didip/tollbooth/master/LICENSE)

## Tollbooth

This is a generic middleware to rate-limit HTTP requests.

**NOTE:** This library is considered finished, any new activities are probably centered around `thirdparty` modules.


## Five Minutes Tutorial
```
package main

import (
    "github.com/didip/tollbooth"
    "net/http"
    "time"
)

func HelloHandler(w http.ResponseWriter, req *http.Request) {
    w.Write([]byte("Hello, World!"))
}

func main() {
    // Create a request limiter per handler.
    http.Handle("/", tollbooth.LimitFuncHandler(tollbooth.NewLimiter(1, time.Second), HelloHandler))
    http.ListenAndServe(":12345", nil)
}
```

## Features

1. Rate-limit by request's remote IP, path, methods, custom headers, & basic auth usernames.
    ```
    limiter := tollbooth.NewLimiter(1, time.Second)

    // Configure list of places to look for IP address.
    // By default it's: "RemoteAddr", "X-Forwarded-For", "X-Real-IP"
    // If your application is behind a proxy, set "X-Forwarded-For" first.
    limiter.IPLookups = []string{"RemoteAddr", "X-Forwarded-For", "X-Real-IP"}

    // Limit only GET and POST requests.
    limiter.Methods = []string{"GET", "POST"}

    // Limit request headers containing certain values.
    // Typically, you prefetched these values from the database.
    limiter.Headers = make(map[string][]string)
    limiter.Headers["X-Access-Token"] = []string{"abc123", "xyz098"}

    // Limit based on basic auth usernames.
    // Typically, you prefetched these values from the database.
    limiter.BasicAuthUsers = []string{"bob", "joe", "didip"}
    ```

2. Each request handler can be rate-limited individually.

3. Compose your own middleware by using `LimitByKeys()`.

4. Tollbooth does not require external storage since it uses an algorithm called [Token Bucket](http://en.wikipedia.org/wiki/Token_bucket) [(Go library: ratelimit)](https://github.com/juju/ratelimit).


# Other Web Frameworks

Support for other web frameworks are defined under `/thirdparty` directory.
