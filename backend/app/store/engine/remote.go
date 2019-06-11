package engine

import (
	"encoding/json"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/remote"
)

// Remote implements remote engine and delegates all Calls to remote http server
type Remote struct {
	remote.Client
}

// Create comment and return ID
func (r *Remote) Create(comment store.Comment) (commentID string, err error) {

	resp, err := r.Call("create", comment)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(*resp.Result, &commentID)
	return commentID, err
}

// Get comment by ID
func (r *Remote) Get(locator store.Locator, commentID string) (comment store.Comment, err error) {
	resp, err := r.Call("get", locator, commentID)
	if err != nil {
		return store.Comment{}, err
	}

	err = json.Unmarshal(*resp.Result, &comment)
	return comment, err
}

// Update comment, mutable parts only
func (r *Remote) Update(locator store.Locator, comment store.Comment) error {
	_, err := r.Call("update", locator, comment)
	return err
}

// Find comments for locator
func (r *Remote) Find(req FindRequest) (comments []store.Comment, err error) {
	resp, err := r.Call("find", req)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(*resp.Result, &comments)
	return comments, err
}

// Info returns post(s) meta info
func (r *Remote) Info(req InfoRequest) (info []store.PostInfo, err error) {
	resp, err := r.Call("info", req)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(*resp.Result, &info)
	return info, err
}

// Flag sets and gets flags
func (r *Remote) Flag(req FlagRequest) (status bool, err error) {
	resp, err := r.Call("flag", req)
	if err != nil {
		return false, err
	}
	err = json.Unmarshal(*resp.Result, &status)
	return status, err
}

// ListFlags get list of flagged keys, like blocked & verified user
func (r *Remote) ListFlags(req FlagRequest) (list []interface{}, err error) {
	resp, err := r.Call("list_flags", req)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(*resp.Result, &list)
	return list, err
}

// Count gets comments count by user or site
func (r *Remote) Count(req FindRequest) (count int, err error) {
	resp, err := r.Call("count", req)
	if err != nil {
		return 0, err
	}
	err = json.Unmarshal(*resp.Result, &count)
	return count, err
}

// Delete post(s) by id or by userID
func (r *Remote) Delete(req DeleteRequest) error {
	_, err := r.Call("delete", req)
	return err
}

// Close storage engine
func (r *Remote) Close() error {
	_, err := r.Call("close")
	return err
}
