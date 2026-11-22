/*
 * Copyright 2020 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package server

import (
	"encoding/json"

	"github.com/go-pkgz/jrpc"
)

// close store
func (s *RPC) closeHndl(_ uint64, _ json.RawMessage) (rr jrpc.Response) {
	if err := s.eng.Close(); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	return jrpc.Response{}
}
