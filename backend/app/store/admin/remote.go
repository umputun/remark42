/*
 * Copyright 2019 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package admin

import (
	"encoding/json"

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

	err = json.Unmarshal(*resp.Result, &key)
	return key, err
}

// Admins returns list of admin's ids for given site
func (r *RPC) Admins(siteID string) (ids []string, err error) {
	resp, err := r.Call("admin.admins", siteID)
	if err != nil {
		return []string{}, err
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

	if err := json.Unmarshal(*resp.Result, &email); err != nil {
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

	if err := json.Unmarshal(*resp.Result, &ok); err != nil {
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
