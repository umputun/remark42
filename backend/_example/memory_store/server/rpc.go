/*
 * Copyright 2020 Umputun. All rights reserved.
 * Use of this source code is governed by a MIT-style
 * license that can be found in the LICENSE file.
 */

package server

import (
	"github.com/go-pkgz/jrpc"

	"github.com/umputun/remark/backend/app/store/admin"
	"github.com/umputun/remark/backend/app/store/engine"
)

// RPC handler wraps both engine and remote server and implements all handlers for data store and admin store
// Note: this file can be used as-is in any custom jrpc plugin
type RPC struct {
	*jrpc.Server
	eng engine.Interface
	adm admin.Store
}

// NewRPC makes RPC instance and register handlers
func NewRPC(e engine.Interface, a admin.Store, r *jrpc.Server) *RPC {
	res := &RPC{eng: e, adm: a, Server: r}
	res.addHandlers()
	return res
}

func (s *RPC) addHandlers() {
	// data store handlers
	s.Group("store", jrpc.HandlersGroup{
		"create":      s.createHndl,
		"find":        s.findHndl,
		"get":         s.getHndl,
		"update":      s.updateHndl,
		"count":       s.countHndl,
		"info":        s.infoHndl,
		"flag":        s.flagHndl,
		"list_flags":  s.listFlagsHndl,
		"user_detail": s.userDetailHndl,
		"delete":      s.deleteHndl,
		"close":       s.closeHndl,
	})

	// admin store handlers
	s.Group("admin", jrpc.HandlersGroup{
		"key":     s.admKeyHndl,
		"admins":  s.admAdminsHndl,
		"email":   s.admEmailHndl,
		"enabled": s.admEnabledHndl,
		"event":   s.admEventHndl,
	})
}
