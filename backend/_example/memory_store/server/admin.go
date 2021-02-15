/*
 * Copyright 2020 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package server

import (
	"encoding/json"

	"github.com/go-pkgz/jrpc"

	"github.com/umputun/remark42/backend/app/store/admin"
)

// get admin key
func (s *RPC) admKeyHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	var siteID string
	if err := json.Unmarshal(params, &siteID); err != nil {
		return jrpc.Response{Error: err.Error()}
	}

	key, err := s.adm.Key(siteID)
	if err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	return jrpc.EncodeResponse(id, key, err)
}

// get admins list
func (s *RPC) admAdminsHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	var siteID string
	if err := json.Unmarshal(params, &siteID); err != nil {
		return jrpc.Response{Error: err.Error()}
	}

	admins, err := s.adm.Admins(siteID)
	if err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	return jrpc.EncodeResponse(id, admins, err)
}

// get admin email
func (s *RPC) admEmailHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	var siteID string
	if err := json.Unmarshal(params, &siteID); err != nil {
		return jrpc.Response{Error: err.Error()}
	}

	email, err := s.adm.Email(siteID)
	if err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	return jrpc.EncodeResponse(id, email, err)
}

// return site enabled status
func (s *RPC) admEnabledHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	var siteID string
	if err := json.Unmarshal(params, &siteID); err != nil {
		return jrpc.Response{Error: err.Error()}
	}

	ok, err := s.adm.Enabled(siteID)
	if err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	return jrpc.EncodeResponse(id, ok, err)
}

// onEvent returns nothing, callback to OnEvent
func (s *RPC) admEventHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	var siteID string
	var ps []interface{}
	if err := json.Unmarshal(params, &ps); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	siteID, ok := ps[0].(string)
	if !ok {
		return jrpc.Response{Error: "wrong siteID type"}
	}
	evType, ok := ps[1].(float64)
	if !ok {
		return jrpc.Response{Error: "wrong event type"}
	}
	err := s.adm.OnEvent(siteID, admin.EventType(evType))
	if err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	return jrpc.EncodeResponse(id, nil, err)
}
