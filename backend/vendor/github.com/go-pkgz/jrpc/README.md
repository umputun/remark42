# jrpc - rpc with json [![Build Status](https://travis-ci.org/go-pkgz/jrpc.svg?branch=master)](https://travis-ci.org/go-pkgz/jrpc) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/jrpc)](https://goreportcard.com/report/github.com/go-pkgz/jrpc) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/jrpc/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/jrpc?branch=master) [![godoc](https://godoc.org/github.com/go-pkgz/jrpc?status.svg)](https://godoc.org/github.com/go-pkgz/jrpc)

jrpc library provides client and server for RPC-like communication over HTTP with json encoded messages.
The protocol is a somewhat simplified version of json-rpc with a single POST call sending Request json 
(method name and the list of parameters) moreover, receiving json Response with result data and an error string.

## Usage

### Plugin (server)

```go
// Server wraps jrpc.Server and adds synced map to store data
type Puglin struct {
	*jrpc.Server
}

// create plugin (jrpc server)
plugin := jrpcServer{
    Server: &jrpc.Server{
        API:        "/command",     // base url for rpc calls
        AuthUser:   "user",         // basic auth user name
        AuthPasswd: "password",     // basic auth password
        AppName:    "jrpc-example", // plugin name for headers
        Logger:     logger,
    },
}

plugin.Add("mycommand", func(id uint64, params json.RawMessage) Response {
    return jrpc.EncodeResponse(id, "hello, it works", nil)
})
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

*for functional examples for both plugin and application see [_example](https://github.com/go-pkgz/jrpc/tree/master/_example)*
 
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
* Communication between the server and the caller can be protected with basic auth.
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
 