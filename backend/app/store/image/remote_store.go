package image

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"strings"
	"time"

	"github.com/go-pkgz/jrpc"
)

// RPC implements remote engine and delegates all Calls to remote http server
type RPC struct {
	jrpc.Client
}

func (r *RPC) SaveWithID(id string, img []byte) error {
	_, err := r.Call("image.save_with_id", id, img)
	return err
}

func (r *RPC) Load(id string) ([]byte, error) {
	resp, err := r.Call("image.load", id)
	if err != nil {
		return nil, err
	}
	var rawImg string
	if err = json.Unmarshal(*resp.Result, &rawImg); err != nil {
		return nil, err
	}
	return ioutil.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(rawImg)))
}

func (r *RPC) Commit(id string) error {
	_, err := r.Call("image.commit", id)
	return err
}

func (r *RPC) Cleanup(_ context.Context, ttl time.Duration) error {
	_, err := r.Call("image.cleanup", ttl)
	return err
}
