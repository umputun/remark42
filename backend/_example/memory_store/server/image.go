/*
 * Copyright 2020 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/go-pkgz/jrpc"
)

func (s *RPC) imgSaveWithIDHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	var req [2]string
	if err := json.Unmarshal(params, &req); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	img, err := base64.StdEncoding.DecodeString(req[1])
	if err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	err = s.img.SaveWithID(req[0], img)
	return jrpc.EncodeResponse(id, nil, err)
}

func (s *RPC) imgLoadHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	var fileID string
	if err := json.Unmarshal(params, &fileID); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	value, err := s.img.Load(fileID)
	return jrpc.EncodeResponse(id, value, err)
}

func (s *RPC) imgCommitHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	var fileID string
	if err := json.Unmarshal(params, &fileID); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	err := s.img.Commit(fileID)
	return jrpc.EncodeResponse(id, nil, err)
}

func (s *RPC) imgCleanupHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	var ttl time.Duration
	if err := json.Unmarshal(params, &ttl); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	err := s.img.Cleanup(context.TODO(), ttl)
	return jrpc.EncodeResponse(id, nil, err)
}
