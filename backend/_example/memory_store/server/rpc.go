package server

import (
	"encoding/json"

	"github.com/go-pkgz/jrpc"

	"github.com/umputun/remark/backend/app/store"
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
		"create":     s.createHndl,
		"find":       s.findHndl,
		"get":        s.getHndl,
		"update":     s.updateHndl,
		"count":      s.countHndl,
		"info":       s.infoHndl,
		"flag":       s.flagHndl,
		"list_flags": s.listFlagsHndl,
		"delete":     s.deleteHndl,
		"close":      s.closeHndl,
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

// deleteHndl remove comment(s)
func (s *RPC) deleteHndl(id uint64, params json.RawMessage) (rr jrpc.Response) {
	req := engine.DeleteRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	err := s.eng.Delete(req)
	return jrpc.EncodeResponse(id, nil, err)
}

// close store
func (s *RPC) closeHndl(id uint64, _ json.RawMessage) (rr jrpc.Response) {
	if err := s.eng.Close(); err != nil {
		return jrpc.Response{Error: err.Error()}
	}
	return jrpc.Response{}
}

// get admin key
func (s *RPC) admKeyHndl(id uint64, _ json.RawMessage) (rr jrpc.Response) {
	key, err := s.adm.Key()
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
	ps := []interface{}{}
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
