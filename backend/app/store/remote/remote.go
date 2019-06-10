// Package remote implements client ans server for RPC-like communication with remote storage.
// The protocol is somewhat simplified version of json-rpc with a single POST call sending
// Request json (method name and the list of parameters) and receiving back json Response with "result" json
// and error string
package remote

import (
	"encoding/json"
)

// Request encloses method name and all params
type Request struct {
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
	ID     uint64      `json:"id"`
}

// Response encloses result and error received from remote server
type Response struct {
	Result *json.RawMessage `json:"result,omitempty"`
	Error  string           `json:"error,omitempty"`
	ID     uint64           `json:"id"`
}
