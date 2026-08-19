# jrpc - rpc with json [![Build](https://github.com/go-pkgz/jrpc/actions/workflows/ci.yml/badge.svg)](https://github.com/go-pkgz/jrpc/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/jrpc)](https://goreportcard.com/report/github.com/go-pkgz/jrpc) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/jrpc/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/jrpc?branch=master) [![godoc](https://godoc.org/github.com/go-pkgz/jrpc?status.svg)](https://godoc.org/github.com/go-pkgz/jrpc)

jrpc library provides client and server for RPC-like communication over HTTP with json encoded messages.
The protocol is a somewhat simplified version of json-rpc with a single POST call sending `Request` json 
(method name and the list of parameters) moreover, receiving json `Response` with result data and an error string.

## Usage

### Plugin (server)

```go
// Plugin wraps jrpc.Server and adds synced map to store data
type Plugin struct {
	*jrpc.Server
}

// create plugin (jrpc server) with NewServer where required param is a base url for rpc calls
plugin := jrpc.NewServer("/command")

// then add your function to map
plugin.Add("mycommand", func(id uint64, params json.RawMessage) jrpc.Response {
    return jrpc.EncodeResponse(id, "hello, it works", nil)
})

// and run server with port number value
plugin.Run(8080)
```

The constructor `NewServer` accepts two parameters:
* `API` - a base url for rpc calls
* `Options` - optional parameters such as timeouts, logger, limits, middlewares and so on.
  * `Auth` - sets basic auth credentials, accepts `username` and `password`. Auth is enforced only if both of them
    set to non-empty values; setting just one leaves the server serving every request unauthenticated
  * `WithTimeouts` - sets server timeouts, accepts a `Timeouts` struct with `ReadHeaderTimeout`, `WriteTimeout`,
    `IdleTimeout` and `CallTimeout`. `CallTimeout` limits the time allowed for a single call and responds with `503` if
    exceeded, and has to be set below `WriteTimeout`, otherwise the write deadline kills the connection before the
    `503` can be sent
  * `WithLimits` - defines a limit of calls/sec per client, accepts limit value in `float64` type
  * `WithThrottler` - sets throttler middleware limiting the number of parallel calls to the server
  * `WithSignature` - sets server signature, accepts appName, author and version. Disabled by default
  * `WithLogger` - defines custom logger (e.g. [lgr](https://github.com/go-pkgz/lgr))
  * `WithMiddlewares` - sets custom middlewares list to server, accepts list of handlers with idiomatic type `func(http.Handler) http.Handler`

Example with options:
```go
import (
	"time"

	"github.com/go-pkgz/jrpc"
	"github.com/go-pkgz/rest"
)

plugin := jrpc.NewServer("/command",
	jrpc.Auth("user", "password"),
	jrpc.WithTimeouts(jrpc.Timeouts{
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       10 * time.Second,
		CallTimeout:       25 * time.Second,
	}),
	jrpc.WithThrottler(120),
	jrpc.WithLimits(100),
	jrpc.WithSignature("the best plugin ever", "author", "1.0.0"),
	jrpc.WithMiddlewares(rest.Trace),
)
```

### Application (client)

```go
// Client makes jrpc.Client and invoke remote call
rpcClient := jrpc.Client{
    API:        "http://127.0.0.1:8080/command",
    Client:     http.Client{},
    AuthUser:   "user",
    AuthPasswd: "password",
}

resp, err := rpcClient.Call("mycommand")
var message string
if err = json.Unmarshal(*resp.Result, &message); err != nil {
    panic(err)
}
```

### Running the example

[_example](https://github.com/go-pkgz/jrpc/tree/master/_example) has a working pair of a plugin and an application.
Both are separate go modules pointing to the local jrpc with a `replace` directive, so no extra setup is needed
beyond go 1.24 or later and a free local port 8080. Start the plugin first, in one terminal:

```sh
cd _example/plugin
go run .
```

It registers two handlers and listens on port 8080:

```
[INFO] add handler for store.save
[INFO] add handler for store.load
[INFO] listen on [::]:8080
```

Then run the application in another terminal:

```sh
cd _example/application
go run .
```

It calls the plugin three times and prints the results:

```
stored {TS:2025-01-12 12:00:00 +0000 UTC Value:12345} with id=54118548792
loaded {TS:2025-01-12 12:00:00 +0000 UTC Value:12345} from id=54118548792
can't load for id=something, not found
```

The application exits on its own, the plugin keeps listening until stopped with Ctrl-C.

## Technical details
 
 * `jrpc.Server` runs on user-defined port as a regular http server
 * Server accepts a single POST request on user-defined url with [Request](https://github.com/go-pkgz/jrpc/blob/master/jrpc.go#L12) sent as json payload
 <details><summary>request details and an example:</summary>
 
     ```go
     type Request struct {
     	Method string      `json:"method"`
     	Params interface{} `json:"params,omitempty"`
     	ID     uint64      `json:"id"`
     }
     ```
     example: 
     
     ```json
       {
        "method":"test",
        "params":[123,"abc"],
        "id":1
        }
     ```
 </details>
 
* Params can be a struct, primitive type or slice of values, even with different types.
* Server defines `ServerFn` handler function to react on a POST request. The handler provided by the user.
* Communication between the server and the caller can be protected with basic auth. The protection is on only if
  both user and password set with the `Auth` option; with either of them empty the server responds to every request
  without asking for credentials.
* [Client](https://github.com/go-pkgz/jrpc/blob/master/client.go) provides a single method `Call` and return `Response`

 <details><summary>response details:</summary>
 
   ```go
    // Response encloses result and error received from remote server
    type Response struct {
    	Result *json.RawMessage `json:"result,omitempty"`
    	Error  string           `json:"error,omitempty"`
    	ID     uint64           `json:"id"`
    }
   ```
 </details>
 
* User should encode and decode json payloads on the application level, see provided [examples](https://github.com/go-pkgz/jrpc/tree/master/_example)
* `jrpc.Server` doesn't support https internally (yet). If used on exposed or non-private networks, should be proxied with something providing https termination (nginx and others). 

## Status

The code was extracted from [remark42](https://github.com/umputun/remark) and still under development. Until v1.x released the
 API & protocol may change.
 
