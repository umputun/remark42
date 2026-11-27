package engine

import (
	"encoding/json"
	"errors"

	"github.com/go-pkgz/jrpc"

	"github.com/umputun/remark42/backend/app/store"
)

// RPC implements remote engine and delegates all Calls to remote http server
type RPC struct {
	jrpc.Client
}

// Call makes the remote call, see store.HideRemoteAddress about its errors
func (r *RPC) Call(method string, args ...any) (*jrpc.Response, error) {
	resp, err := r.Client.Call(method, args...)
	return resp, store.HideRemoteAddress(method, err)
}

// Create comment and return ID
func (r *RPC) Create(comment store.Comment) (commentID string, err error) {
	return call[string](r, "store.create", comment)
}

// Get comment by ID
func (r *RPC) Get(req GetRequest) (comment store.Comment, err error) {
	return call[store.Comment](r, "store.get", req)
}

// Update comment, mutable parts only
func (r *RPC) Update(comment store.Comment) error {
	_, err := r.Call("store.update", comment)
	return err
}

// Find comments for locator
func (r *RPC) Find(req FindRequest) (comments []store.Comment, err error) {
	return callList[store.Comment](r, "store.find", req)
}

// Info returns post(s) meta info
func (r *RPC) Info(req InfoRequest) (info []store.PostInfo, err error) {
	return callList[store.PostInfo](r, "store.info", req)
}

// Flag sets and gets flags
func (r *RPC) Flag(req FlagRequest) (status bool, err error) {
	return call[bool](r, "store.flag", req)
}

// ListFlags get list of flagged keys, like blocked & verified user
func (r *RPC) ListFlags(req FlagRequest) ([]any, error) {
	resp, err := r.Call("store.list_flags", req)
	if err != nil {
		return nil, err
	}
	// try []store.BlockedUser
	list, err := unmarshalList[store.BlockedUser](listResult(resp))
	if err == nil {
		return list, nil
	}

	// try []strings
	return unmarshalList[string](listResult(resp))
}

// UserDetail sets or gets single detail value, or gets all details for requested site.
// UserDetail returns list even for single entry request is a compromise in order to have both single detail getting and setting
// and all site's details listing under the same function (and not to extend interface by two separate functions).
func (r *RPC) UserDetail(req UserDetailRequest) (result []UserDetailEntry, err error) {
	return callList[UserDetailEntry](r, "store.user_detail", req)
}

// Count gets comments count by user or site
func (r *RPC) Count(req FindRequest) (count int, err error) {
	return call[int](r, "store.count", req)
}

// Delete post(s), user, comment, user details, or everything
func (r *RPC) Delete(req DeleteRequest) error {
	_, err := r.Call("store.delete", req)
	return err
}

// Close storage engine
func (r *RPC) Close() error {
	_, err := r.Call("store.close")
	r.Client.Client.CloseIdleConnections()
	return err
}

// errNoResult is the error of a call that has to return a value and got null or no result from the remote
var errNoResult = errors.New("remote store returned no result")

// rawResult returns the json of a call's result, errNoResult when the remote sent null or none
func rawResult(resp *jrpc.Response) ([]byte, error) {
	if resp == nil || resp.Result == nil {
		return nil, errNoResult
	}
	return *resp.Result, nil
}

// listResult returns the json of a call that returns a list. A remote written in Go answers with null for a
// nil slice, which leaves Result unset; it is read as null here and unmarshals to an empty list
func listResult(resp *jrpc.Response) []byte {
	if res, err := rawResult(resp); err == nil {
		return res
	}
	return []byte("null")
}

// call makes a remote call that has to return a value and decodes its result into T
func call[T any](r *RPC, method string, args ...any) (res T, err error) {
	resp, err := r.Call(method, args...)
	if err != nil {
		return res, err
	}
	raw, err := rawResult(resp)
	if err != nil {
		return res, err
	}
	err = json.Unmarshal(raw, &res)
	return res, err
}

// callList makes a remote call that returns a list and decodes its result, null included, into []T
func callList[T any](r *RPC, method string, args ...any) (res []T, err error) {
	resp, err := r.Call(method, args...)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(listResult(resp), &res)
	return res, err
}

// unmarshalList decodes a json list of T into a list of any
func unmarshalList[T any](data []byte) ([]any, error) {
	var items []T
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	list := make([]any, 0, len(items))
	for _, item := range items {
		list = append(list, item)
	}
	return list, nil
}
