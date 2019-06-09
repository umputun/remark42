package remote

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

// Client implements remote engine and delegates all calls to remote http server
type Client struct {
	API        string
	Client     http.Client
	AuthUser   string
	AuthPasswd string
}

// Call remote server with given method and arguments
func (r *Client) Call(method string, args ...interface{}) (*Response, error) {

	b, err := json.Marshal(Request{Method: method, Params: args})
	if err != nil {
		return nil, errors.Wrapf(err, "marshaling failed for %s", method)
	}

	req, err := http.NewRequest("POST", r.API, bytes.NewBuffer(b))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make request for %s", method)
	}

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
