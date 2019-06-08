package remote

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/store"
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

// Put updates comment, mutable parts only
func (r *Client) Put(locator store.Locator, comment store.Comment) error {
	_, err := r.call("put", locator, comment)
	return err
}

// Find comments for locator
func (r *Client) Find(locator store.Locator, sort string) (comments []store.Comment, err error) {
	resp, err := r.call("find", locator, sort)
	if err != nil {
		return []store.Comment{}, err
	}
	err = json.Unmarshal(*resp.Result, &comments)
	return comments, err
}

// Last comments for given site, sorted by time
func (r *Client) Last(siteID string, limit int, since time.Time) (comments []store.Comment, err error) {
	resp, err := r.call("last", siteID, limit, since)
	if err != nil {
		return []store.Comment{}, err
	}
	err = json.Unmarshal(*resp.Result, &comments)
	return comments, err
}

// User get comments by user, sorted by time
func (r *Client) User(siteID, userID string, limit, skip int) (comments []store.Comment, err error) {
	resp, err := r.call("user", siteID, userID, limit, skip)
	if err != nil {
		return []store.Comment{}, err
	}
	err = json.Unmarshal(*resp.Result, &comments)
	return comments, err
}

// UserCount gets comments count by user
func (r *Client) UserCount(siteID, userID string) (count int, err error)        {
	resp, err := r.call("user_count", siteID, userID)
	if err != nil {
		return 0, err
	}
	err = json.Unmarshal(*resp.Result, &count)
	return count, err
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
