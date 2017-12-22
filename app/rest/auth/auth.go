package auth

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/gob"
	"fmt"

	"github.com/gorilla/sessions"

	"github.com/umputun/remark/app/store"
)

type Params struct {
	Cid          string
	Csecret      string
	SessionStore *sessions.FilesystemStore
	Admins       []string
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
