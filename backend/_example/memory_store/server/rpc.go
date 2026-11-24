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
	"github.com/umputun/remark42/backend/app/store/engine"
	"github.com/umputun/remark42/backend/app/store/image"
)

// RPC handler wraps both engine and remote server and implements all handlers for data store and admin store
// Note: this file can be used as-is in any custom jrpc plugin
type RPC struct {
	*jrpc.Server
	eng engine.Interface
	adm admin.Store
	img image.Store
}

// NewRPC makes RPC instance and register handlers
func NewRPC(e engine.Interface, a admin.Store, i image.Store, r *jrpc.Server) *RPC {
	res := &RPC{eng: e, adm: a, img: i, Server: r}
	res.addHandlers()
	return res
}

func (s *RPC) addHandlers() {
	// data store handlers
	s.Group("store", jrpc.HandlersGroup{
		"create":      handle(s.eng, engine.Interface.Create),
		"find":        handle(s.eng, engine.Interface.Find),
		"get":         handle(s.eng, engine.Interface.Get),
		"update":      handleErr(s.eng, engine.Interface.Update),
		"count":       handle(s.eng, engine.Interface.Count),
		"info":        handle(s.eng, engine.Interface.Info),
		"flag":        handle(s.eng, engine.Interface.Flag),
		"list_flags":  handle(s.eng, engine.Interface.ListFlags),
		"user_detail": handle(s.eng, engine.Interface.UserDetail),
		"delete":      handleErr(s.eng, engine.Interface.Delete),
		"close":       s.closeHndl,
	})

	// admin store handlers
	s.Group("admin", jrpc.HandlersGroup{
		"key":     handle(s.adm, admin.Store.Key),
		"admins":  handle(s.adm, admin.Store.Admins),
		"email":   handle(s.adm, admin.Store.Email),
		"enabled": handle(s.adm, admin.Store.Enabled),
		"event":   s.admEventHndl,
	})

	// image store handlers
	s.Group("image", jrpc.HandlersGroup{
		"save_with_id":        s.imgSaveWithIDHndl,
		"reset_cleanup_timer": handleErr(s.img, image.Store.ResetCleanupTimer),
		"load":                handle(s.img, image.Store.Load),
		"delete":              handleErr(s.img, image.Store.Delete),
		"commit":              handleErr(s.img, image.Store.Commit),
		"cleanup":             handleErr(s.img, imgCleanup),
		"info":                s.imgInfoHndl,
	})
}

// handle makes a handler for a method of store taking one request value, such as engine.Interface.Find:
// it decodes the call's params into Req, calls the method and encodes its result, or its error, as the response.
// The method is called on store only when a request comes, so a store this plugin doesn't serve may be nil.
func handle[Store, Req, Resp any](store Store, method func(Store, Req) (Resp, error)) jrpc.ServerFn {
	return func(id uint64, params json.RawMessage) jrpc.Response {
		var req Req
		if err := json.Unmarshal(params, &req); err != nil {
			return jrpc.Response{Error: err.Error()}
		}
		resp, err := method(store, req)
		return jrpc.EncodeResponse(id, resp, err)
	}
}

// handleErr is handle for a method returning only an error, answered with a null result
func handleErr[Store, Req any](store Store, method func(Store, Req) error) jrpc.ServerFn {
	return handle(store, func(st Store, req Req) (any, error) { return nil, method(st, req) })
}
