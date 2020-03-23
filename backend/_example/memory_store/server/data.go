/*
 * Copyright 2020 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package server

import (
	"encoding/json"

	"github.com/go-pkgz/jrpc"
	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/engine"
)

func (s *RPC) createHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	comment := store.Comment{}
	if err := json.Unmarshal(params, &comment); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	commentID, err := s.eng.Create(comment)
	return jrpc.EncodeResponse(id, commentID, err)
}

// Find comments
func (s *RPC) findHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	req := engine.FindRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	resp, err := s.eng.Find(req)
	return jrpc.EncodeResponse(id, resp, err)
}

// Get comment
func (s *RPC) getHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	req := engine.GetRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	comment, err := s.eng.Get(req)
	return jrpc.EncodeResponse(id, comment, err)
}

// Update comment
func (s *RPC) updateHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	comment := store.Comment{}
	if err := json.Unmarshal(params, &comment); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	err := s.eng.Update(comment)
	return jrpc.EncodeResponse(id, nil, err)
}

// counts for site and users
func (s *RPC) countHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	req := engine.FindRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	count, err := s.eng.Count(req)
	return jrpc.EncodeResponse(id, count, err)
}

// info get post meta info
func (s *RPC) infoHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	req := engine.InfoRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	info, err := s.eng.Info(req)
	return jrpc.EncodeResponse(id, info, err)
}

// flagHndl get and sets flag value
func (s *RPC) flagHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	req := engine.FlagRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	status, err := s.eng.Flag(req)
	return jrpc.EncodeResponse(id, status, err)
}

// listFlagsHndl list flags for given request
func (s *RPC) listFlagsHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	req := engine.FlagRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	flags, err := s.eng.ListFlags(req)
	return jrpc.EncodeResponse(id, flags, err)
}

// userDetailHndl sets or gets single detail value, or gets all details for requested site.
// userDetailHndl returns list even for single entry request is a compromise in order to have both single detail getting and setting
// and all site's details listing under the same function (and not to extend engine interface by two separate functions).
func (s *RPC) userDetailHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	req := engine.UserDetailRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	value, err := s.eng.UserDetail(req)
	return jrpc.EncodeResponse(id, value, err)
}

// deleteHndl delete post(s), user, comment, user details, or everything
func (s *RPC) deleteHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	req := engine.DeleteRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	err := s.eng.Delete(req)
	return jrpc.EncodeResponse(id, nil, err)
}

// close store
func (s *RPC) closeHndl(_ uint64, _ json.RawMessage) (rr jrpc.Response) {
	if err := s.eng.Close(); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	return jrpc.Response{}
}
