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

// Save saves image with given id to staging.
func (r *RPC) Save(id string, img []byte) error {
	_, err := r.Call("image.save_with_id", id, img)
	return err
}

// Load image with given id
func (r *RPC) Load(id string) ([]byte, error) {
	resp, err := r.Call("image.load", id)
	if err != nil {
		return nil, err
	}
	var rawImg string
	if err := json.Unmarshal(*resp.Result, &rawImg); err != nil {
		return nil, err
	}
	return ioutil.ReadAll(base64.NewDecoder(base64.StdEncoding, strings.NewReader(rawImg)))
}

// Commit file stored in staging location by moving it to permanent location
func (r *RPC) Commit(id string) error {
	_, err := r.Call("image.commit", id)
	return err
}

// Cleanup runs scan of staging and removes old files based on ttl
func (r *RPC) Cleanup(_ context.Context, ttl time.Duration) error {
	_, err := r.Call("image.cleanup", ttl)
	return err
}

// Info returns meta information about storage
func (r *RPC) Info() (StoreInfo, error) {
	resp, err := r.Call("image.info")
	if err != nil {
		return StoreInfo{}, err
	}
	info := StoreInfo{}
	if e := json.Unmarshal(*resp.Result, &info); e != nil {
		return StoreInfo{}, e
	}
	return info, err
}
