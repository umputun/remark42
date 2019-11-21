package engine

import (
	"encoding/json"

	"github.com/go-pkgz/jrpc"

	"github.com/umputun/remark/backend/app/store"
)

// RPC implements remote engine and delegates all Calls to remote http server
type RPC struct {
	jrpc.Client
}

// Create comment and return ID
func (r *RPC) Create(comment store.Comment) (commentID string, err error) {

	resp, err := r.Call("store.create", comment)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(*resp.Result, &commentID)
	return commentID, err
}

// Get comment by ID
func (r *RPC) Get(req GetRequest) (comment store.Comment, err error) {
	resp, err := r.Call("store.get", req)
	if err != nil {
		return store.Comment{}, err
	}

	err = json.Unmarshal(*resp.Result, &comment)
	return comment, err
}

// Update comment, mutable parts only
func (r *RPC) Update(comment store.Comment) error {
	_, err := r.Call("store.update", comment)
	return err
}

// Find comments for locator
func (r *RPC) Find(req FindRequest) (comments []store.Comment, err error) {
	resp, err := r.Call("store.find", req)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(*resp.Result, &comments)
	return comments, err
}

// Info returns post(s) meta info
func (r *RPC) Info(req InfoRequest) (info []store.PostInfo, err error) {
	resp, err := r.Call("store.info", req)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(*resp.Result, &info)
	return info, err
}

// Flag sets and gets flags
func (r *RPC) Flag(req FlagRequest) (status bool, err error) {
	resp, err := r.Call("store.flag", req)
	if err != nil {
		return false, err
	}
	err = json.Unmarshal(*resp.Result, &status)
	return status, err
}

// ListFlags get list of flagged keys, like blocked & verified user
func (r *RPC) ListFlags(req FlagRequest) (list []interface{}, err error) {
	resp, err := r.Call("store.list_flags", req)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(*resp.Result, &list)
	return list, err
}

// UserDetail sets or gets single detail value, or gets all details for requested site
func (r *RPC) UserDetail(req UserDetailRequest) (result []UserDetailEntry, err error) {
	resp, err := r.Call("store.user_detail", req)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(*resp.Result, &result)
	return result, err
}

// Count gets comments count by user or site
func (r *RPC) Count(req FindRequest) (count int, err error) {
	resp, err := r.Call("store.count", req)
	if err != nil {
		return 0, err
	}
	err = json.Unmarshal(*resp.Result, &count)
	return count, err
}

// Delete post(s), user, comment, user details, or everything
func (r *RPC) Delete(req DeleteRequest) error {
	_, err := r.Call("store.delete", req)
	return err
}

// Close storage engine
func (r *RPC) Close() error {
	_, err := r.Call("store.close")
	return err
}
