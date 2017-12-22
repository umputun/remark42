package auth

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/gorilla/sessions"
	"github.com/umputun/remark/app/store"
)

type Params struct {
	Cid          string
	Csecret      string
	SessionStore *sessions.FilesystemStore
	Admins       []string
}

type SessionStore struct {
	StorePath string
	StoreKey  string

	store *sessions.FilesystemStore
	once  sync.Once
}

func (s *SessionStore) GetSession(name string) {

}

func randToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	s := sha1.New()
	s.Write(b)
	return fmt.Sprintf("%x", s.Sum(nil))
}

func init() {
	gob.Register(store.User{})
}
