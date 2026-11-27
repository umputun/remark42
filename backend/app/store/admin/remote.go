/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package admin

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

// Key returns the key, same for all sites
func (r *RPC) Key(siteID string) (key string, err error) {
	if key, err = call[string](r, "admin.key", siteID); err != nil {
		return "", err
	}
	// the key signs tokens, and an empty one would let anyone forge them
	if key == "" {
		return "", errors.New("remote admin store returned an empty key")
	}
	return key, nil
}

// Admins returns list of admin's ids for given site
func (r *RPC) Admins(siteID string) (ids []string, err error) {
	resp, err := r.Call("admin.admins", siteID)
	if err != nil {
		return []string{}, err
	}

	// a remote written in Go answers with null for a nil slice, which is no admins
	if resp.Result == nil {
		return []string{}, nil
	}
	if err := json.Unmarshal(*resp.Result, &ids); err != nil {
		return []string{}, err
	}
	return ids, nil
}

// Email gets email address for given site
func (r *RPC) Email(siteID string) (email string, err error) {
	return call[string](r, "admin.email", siteID)
}

// Enabled returns true if allowed
func (r *RPC) Enabled(siteID string) (ok bool, err error) {
	return call[bool](r, "admin.enabled", siteID)
}

// OnEvent reacts (register) events about data modification
func (r *RPC) OnEvent(siteID string, et EventType) error {
	_, err := r.Call("admin.event", siteID, et)
	if err != nil {
		return err
	}
	return nil
}

// errNoResult is the error of a call that has to return a value and got null or no result from the remote
var errNoResult = errors.New("remote admin store returned no result")

// rawResult returns the json of a call's result, errNoResult when the remote sent null or none
func rawResult(resp *jrpc.Response) ([]byte, error) {
	if resp == nil || resp.Result == nil {
		return nil, errNoResult
	}
	return *resp.Result, nil
}

// call makes a remote call that has to return a value and decodes its result into T.
// T is a scalar here, so an unmarshal error leaves res at its zero value
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
