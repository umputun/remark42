package remote

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"sync/atomic"

	"github.com/pkg/errors"
)

// Client implements remote engine and delegates all calls to remote http server
type Client struct {
	API        string
	Client     http.Client
	AuthUser   string
	AuthPasswd string

	id uint64
}

// Call remote server with given method and arguments
func (r *Client) Call(method string, args ...interface{}) (*Response, error) {

	var b []byte
	var err error
	if len(args) == 1 && reflect.TypeOf(args[0]).Kind() == reflect.Struct {
		b, err = json.Marshal(Request{Method: method, Params: args[0], ID: atomic.AddUint64(&r.id, 1)})
		if err != nil {
			return nil, errors.Wrapf(err, "marshaling failed for %s", method)
		}
	} else {
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
		return nil, errors.Errorf("bad status %d for %s", resp.StatusCode, method)
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
