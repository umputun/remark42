// Package avatar provides cached proxy for user pictures/avatars
// refreshed by login and kept in local store
package avatar

import (
	"fmt"
	"hash/crc64"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/pkg/errors"

	"github.com/umputun/remark/app/rest/common"
	"github.com/umputun/remark/app/store"
)

// Proxy provides avatar store and http handler for avatars
type Proxy struct {
	StorePath     string
	DefaultAvatar string
	RoutePath     string
}

// Put gets original avatar url from user info and returns proxied url
func (p *Proxy) Put(u store.User) (avatarURL string, err error) {

	if u.Picture == "" {
		if p.DefaultAvatar != "" {
			return p.RoutePath + "/" + p.DefaultAvatar, nil
		}
		return "", errors.Errorf("no picture for %s", u.ID)
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(u.Picture)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get avatar for user %s from %s", u.ID, u.Picture)
	}
	defer func() {
		if e := resp.Body.Close(); e != nil {
			log.Printf("[WARN] can't close response body, %s", e)
		}
	}()

	location := p.location(u.ID)
	if err = os.Mkdir(location, 0700); err != nil && !strings.Contains(err.Error(), "file exists") {
		return "", errors.Wrapf(err, "failed to make avatar location %s", location)
	}

	avFile := path.Join(location, u.ID+".image")
	fh, err := os.Create(avFile)
	if err != nil {
		return "", errors.Wrapf(err, "can't create file %s", avFile)
	}
	defer func() {
		if e := fh.Close(); e != nil {
			log.Printf("[WARN] can't close avatar file %s, %s", avFile, e)
		}
	}()

	if _, err = io.Copy(fh, resp.Body); err != nil {
		return "", errors.Wrapf(err, "can't save file %s", avFile)
	}

	log.Printf("[DEBUG] saved avatar from %s to %s, user %q", u.Picture, avFile, u.Name)
	return p.RoutePath + "/" + u.ID + ".image", nil
}

// Routes returns auth routes for given provider
func (p *Proxy) Routes() chi.Router {
	router := chi.NewRouter()
	router.Get("/{avatar}", func(w http.ResponseWriter, r *http.Request) {
		avatar := chi.URLParam(r, "avatar")
		location := p.location(strings.TrimSuffix(avatar, ".image"))
		avFile := path.Join(location, avatar)
		fh, err := os.Open(avFile)
		if err != nil {
			if p.DefaultAvatar == "" {
				common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't load avatar")
				return
			}
			if fh, err = os.Open(path.Join(p.StorePath, p.DefaultAvatar)); err != nil {
				common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't load default avatar")
				return
			}
		}

		defer func() {
			if e := fh.Close(); e != nil {
				log.Printf("[WARN] can't close avatar file %s, %s", avFile, e)
			}
		}()

		w.Header().Set("Content-Type", "image/*")
		if status, ok := r.Context().Value(render.StatusCtxKey).(int); ok {
			w.WriteHeader(status)
		}
		if _, err = io.Copy(w, fh); err != nil {
			log.Printf("[WARN] can't send response to %s, %s", r.RemoteAddr, err)
		}
	})

	return router
}

// get location for user id by adding partion to final path
// the end result is a full path like this - /tmp/avatars.test/992
func (p *Proxy) location(id string) string {
	checksum64 := crc64.Checksum([]byte(id), crc64.MakeTable(crc64.ECMA))
	partition := checksum64 % 100
	return path.Join(p.StorePath, fmt.Sprintf("%02d", partition))
}
