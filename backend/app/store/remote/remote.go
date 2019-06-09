package remote

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/engine"
)

// Client implements remote engine and delegates all calls to remote http server
type Client struct {
	API        string
	Client     http.Client
	AuthUser   string
	AuthPasswd string
}

// Request encloses method name and all params
type Request struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

// Response encloses result and error received from remote server
type Response struct {
	Result *json.RawMessage `json:"result,omitempty"`
	Error  string           `json:"error,omitempty"`
}

// Create comment and return ID
func (r *Client) Create(comment store.Comment) (commentID string, err error) {

	resp, err := r.call("create", comment)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(*resp.Result, &commentID)
	return commentID, err
}

// Get comment by ID
func (r *Client) Get(locator store.Locator, commentID string) (comment store.Comment, err error) {
	resp, err := r.call("get", locator, commentID)
	if err != nil {
		return store.Comment{}, err
	}

	err = json.Unmarshal(*resp.Result, &comment)
	return comment, err
}

// Update comment, mutable parts only
func (r *Client) Update(locator store.Locator, comment store.Comment) error {
	_, err := r.call("update", locator, comment)
	return err
}

// Find comments for locator
func (r *Client) Find(req engine.FindRequest) (comments []store.Comment, err error) {
	resp, err := r.call("find", req)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(*resp.Result, &comments)
	return comments, err
}

// Info returns post(s) meta info
func (r *Client) Info(req engine.InfoRequest) (info []store.PostInfo, err error) {
	resp, err := r.call("info", req)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(*resp.Result, &info)
	return info, err
}

// Flag sets and gets flags
func (r *Client) Flag(req engine.FlagRequest) (status bool, err error) {
	resp, err := r.call("flag", req)
	if err != nil {
		return false, err
	}
	err = json.Unmarshal(*resp.Result, &status)
	return status, err
}

// ListFlags get list of flagged keys, like blocked & verified user
func (r *Client) ListFlags(siteID string, flag engine.Flag) (list []interface{}, err error) {
	resp, err := r.call("list_flags", siteID, flag)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(*resp.Result, &list)
	return list, err
}

// Count gets comments count by user or site
func (r *Client) Count(req engine.FindRequest) (count int, err error) {
	resp, err := r.call("count", req)
	if err != nil {
		return 0, err
	}
	err = json.Unmarshal(*resp.Result, &count)
	return count, err
}

// Delete post(s) by id or by userID
func (r *Client) Delete(req engine.DeleteRequest) error {
	_, err := r.call("delete", req)
	return err
}

// Close storage engine
func (r *Client) Close() error {
	_, err := r.call("close")
	return err
}

func (r *Client) call(method string, args ...interface{}) (*Response, error) {

	b, err := json.Marshal(Request{Method: method, Params: args})
	if err != nil {
		return nil, errors.Wrapf(err, "marshaling failed for %s", method)
	}

	req, err := http.NewRequest("POST", r.API, bytes.NewBuffer(b))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make request for %s", method)
	}

	req.SetBasicAuth(r.AuthUser, r.AuthPasswd)
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
