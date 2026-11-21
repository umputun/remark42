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
)

// RPC implements remote engine and delegates all Calls to remote http server
type RPC struct {
	jrpc.Client
}

// Key returns the key, same for all sites
func (r *RPC) Key(siteID string) (key string, err error) {
	resp, err := r.Call("admin.key", siteID)
	if err != nil {
		return "", err
	}

	res, err := rawResult(resp)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(res, &key); err != nil {
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
	resp, err := r.Call("admin.email", siteID)
	if err != nil {
		return "", err
	}

	res, err := rawResult(resp)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(res, &email); err != nil {
		return "", err
	}
	return email, nil
}

// Enabled returns true if allowed
func (r *RPC) Enabled(siteID string) (ok bool, err error) {
	resp, err := r.Call("admin.enabled", siteID)
	if err != nil {
		return false, err
	}

	res, err := rawResult(resp)
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(res, &ok); err != nil {
		return false, err
	}
	return ok, nil
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
