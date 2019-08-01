// Package jrpc implements client and server for RPC-like communication over HTTP with json encoded messages.
// The protocol is somewhat simplified version of json-rpc with a single POST call sending Request json
// (method name and the list of parameters) and receiving back json Response with "result" json
// and error string
package jrpc

import (
	"encoding/json"
)

// Request encloses method name and all params
type Request struct {
	Method string      `json:"method"`           // method (function) name
	Params interface{} `json:"params,omitempty"` // function arguments
	ID     uint64      `json:"id"`               // unique call id
}

// Response encloses result and error received from remote server
type Response struct {
	Result *json.RawMessage `json:"result,omitempty"` // response json
	Error  string           `json:"error,omitempty"`  // optional remote (server side / plugin side) error
	ID     uint64           `json:"id"`               // unique call id, echoed Request.ID to allow calls tracing
}

// EncodeResponse convert anything (type interface{}) and incoming error (if any) to Response
func EncodeResponse(id uint64, resp interface{}, e error) Response {
	v, err := json.Marshal(&resp)
	if err != nil {
		return Response{Error: err.Error()}
	}
	if e != nil {
		return Response{ID: id, Result: nil, Error: e.Error()} // pass input error
	}
	raw := json.RawMessage{}
	if err := raw.UnmarshalJSON(v); err != nil {
		return Response{Error: err.Error()}
	}

	return Response{ID: id, Result: &raw}
}
