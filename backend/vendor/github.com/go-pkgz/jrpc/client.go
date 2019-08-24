package jrpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync/atomic"

	"github.com/pkg/errors"
)

// Client implements remote engine and delegates all calls to remote http server
// if AuthUser and AuthPasswd defined will be used for basic auth in each call to server
type Client struct {
	API        string      // URL to jrpc server with entrypoint, i.e. http://127.0.0.1:8080/command
	Client     http.Client // http client injected by user
	AuthUser   string      // basic auth user name, should match Server.AuthUser, optional
	AuthPasswd string      // basic auth password, should match Server.AuthPasswd, optional

	id uint64 // used with atomic to populate unique id to Request.ID
}

// Call remote server with given method and arguments.
// Empty args will be ignored, single arg will be marshaled as-us and multiple args marshaled as []interface{}.
// Returns Response and error. Note: Response has it's own Error field, but that onw controlled by server.
// Returned error represent client-level errors, like failed http call, failed marshaling and so on.
func (r *Client) Call(method string, args ...interface{}) (*Response, error) {

	var b []byte
	var err error

	switch {
	case len(args) == 0:
		b, err = json.Marshal(Request{Method: method, ID: atomic.AddUint64(&r.id, 1)})
		if err != nil {
			return nil, errors.Wrapf(err, "marshaling failed for %s", method)
		}
	case len(args) == 1:
		b, err = json.Marshal(Request{Method: method, Params: args[0], ID: atomic.AddUint64(&r.id, 1)})
		if err != nil {
			return nil, errors.Wrapf(err, "marshaling failed for %s", method)
		}
	default:
		b, err = json.Marshal(Request{Method: method, Params: args, ID: atomic.AddUint64(&r.id, 1)})
		if err != nil {
			return nil, errors.Wrapf(err, "marshaling failed for %s", method)
		}
	}

	req, err := http.NewRequest("POST", r.API, bytes.NewBuffer(b))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make request for %s", method)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	if r.AuthUser != "" && r.AuthPasswd != "" {
		req.SetBasicAuth(r.AuthUser, r.AuthPasswd)
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "remote call failed for %s", method)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.Errorf("bad status %s for %s", resp.Status, method)
	}

	cr := Response{}
	if err = json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, errors.Wrapf(err, "failed to decode response for %s", method)
	}

	if cr.Error != "" {
		return nil, errors.New(cr.Error)
	}
	return &cr, nil
}
