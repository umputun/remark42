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

	"github.com/umputun/remark42/backend/app/store/image"
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
	err = s.img.Save(req[0], img)
	return jrpc.EncodeResponse(id, nil, err)
}

// imgCleanup is image.Store's Cleanup without the context, which an RPC call doesn't carry
func imgCleanup(img image.Store, ttl time.Duration) error { return img.Cleanup(context.TODO(), ttl) }

func (s *RPC) imgInfoHndl(id uint64, _ json.RawMessage) (rr jrpc.Response) {
	info, err := s.img.Info()
	return jrpc.EncodeResponse(id, info, err)
}
