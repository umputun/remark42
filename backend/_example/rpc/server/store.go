package server

import (
	"encoding/json"

	"github.com/umputun/remark/backend/app/rpc"
	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/admin"
	"github.com/umputun/remark/backend/app/store/engine"
)

// Store handler wraps both engine and remote server and implements all handlers
// Note: this file can be used as-is in any custom rpc plugin
type Store struct {
	*rpc.Server
	eng engine.Interface
	adm admin.Store
}

// NewStore makes Store instance and register handlers
func NewStore(e engine.Interface, a admin.Store, r *rpc.Server) *Store {
	res := &Store{eng: e, adm: a, Server: r}
	res.addHandlers()
	return res
}


func (s *Store) addHandlers() {
	// data store handlers
	s.Group("store", rpc.HandlersGroup{
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
	s.Group("admin", rpc.HandlersGroup{
		"key":    s.admKeyHndl,
		"admins": s.admAdminsHndl,
		"email":  s.admEmailHndl,
	})
}

func (s *Store) createHndl(id uint64, params json.RawMessage) (rr rpc.Response) {
	comment := store.Comment{}
	if err := json.Unmarshal(params, &comment); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	commentID, err := s.eng.Create(comment)
	if rr, err = s.EncodeResponse(id, commentID, err); err != nil {
		return rpc.Response{Error: err.Error()}

	}
	return rr
}

// Find comments
func (s *Store) findHndl(id uint64, params json.RawMessage) (rr rpc.Response) {
	req := engine.FindRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	resp, err := s.eng.Find(req)
	if rr, err = s.EncodeResponse(id, resp, err); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	return rr
}

// Get comment
func (s *Store) getHndl(id uint64, params json.RawMessage) (rr rpc.Response) {
	req := engine.GetRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	comment, err := s.eng.Get(req)
	if rr, err = s.EncodeResponse(id, comment, err); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	return rr
}

// Update comment
func (s *Store) updateHndl(id uint64, params json.RawMessage) (rr rpc.Response) {
	comment := store.Comment{}
	if err := json.Unmarshal(params, &comment); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	err := s.eng.Update(comment)
	if rr, err = s.EncodeResponse(id, nil, err); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	return rr
}

// counts for site and users
func (s *Store) countHndl(id uint64, params json.RawMessage) (rr rpc.Response) {
	req := engine.FindRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	count, err := s.eng.Count(req)
	if rr, err = s.EncodeResponse(id, count, err); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	return rr
}

// info get post meta info
func (s *Store) infoHndl(id uint64, params json.RawMessage) (rr rpc.Response) {
	req := engine.InfoRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	info, err := s.eng.Info(req)
	if rr, err = s.EncodeResponse(id, info, err); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	return rr
}

// flagHndl get and sets flag value
func (s *Store) flagHndl(id uint64, params json.RawMessage) (rr rpc.Response) {
	req := engine.FlagRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	status, err := s.eng.Flag(req)
	if rr, err = s.EncodeResponse(id, status, err); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	return rr
}

// listFlagsHndl list flags for given request
func (s *Store) listFlagsHndl(id uint64, params json.RawMessage) (rr rpc.Response) {
	req := engine.FlagRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	flags, err := s.eng.ListFlags(req)
	if rr, err = s.EncodeResponse(id, flags, err); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	return rr
}

// deleteHndl remove comment(s)
func (s *Store) deleteHndl(id uint64, params json.RawMessage) (rr rpc.Response) {
	req := engine.DeleteRequest{}
	if err := json.Unmarshal(params, &req); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	err := s.eng.Delete(req)
	if rr, err = s.EncodeResponse(id, nil, err); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	return rr
}

// close store
func (s *Store) closeHndl(id uint64, _ json.RawMessage) (rr rpc.Response) {
	if err := s.eng.Close(); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	return rpc.Response{}
}

// get admin key
func (s *Store) admKeyHndl(id uint64, _ json.RawMessage) (rr rpc.Response) {
	key, err := s.adm.Key()
	if err != nil {
		return rpc.Response{Error: err.Error()}
	}
	if rr, err = s.EncodeResponse(id, key, err); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	return rr
}

// get admins list
func (s *Store) admAdminsHndl(id uint64, params json.RawMessage) (rr rpc.Response) {
	args := []interface{}{}
	if err := json.Unmarshal(params, &args); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	siteID, ok := args[0].(string)
	if !ok {
		return rpc.Response{Error: "incompatible argument"}
	}

	admins, err := s.adm.Admins(siteID)
	if err != nil {
		return rpc.Response{Error: err.Error()}
	}

	if rr, err = s.EncodeResponse(id, admins, err); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	return rr
}

// get admin email
func (s *Store) admEmailHndl(id uint64, params json.RawMessage) (rr rpc.Response) {
	args := []interface{}{}
	if err := json.Unmarshal(params, &args); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	siteID, ok := args[0].(string)
	if !ok {
		return rpc.Response{Error: "incompatible argument"}
	}

	email, err := s.adm.Email(siteID)
	if err != nil {
		return rpc.Response{Error: err.Error()}
	}

	if rr, err = s.EncodeResponse(id, email, err); err != nil {
		return rpc.Response{Error: err.Error()}
	}
	return rr
}
