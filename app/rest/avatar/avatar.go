// Package avatar provides cached proxy for user pictures/avatars
// refreshed by login and kept in local store
package avatar

import (
	"crypto/sha1"
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

const imgSfx = ".image"

// Put gets original avatar url from user info and returns proxied url
func (p *Proxy) Put(u store.User) (avatarURL string, err error) {

	if u.Picture == "" {
		if p.DefaultAvatar != "" {
			return p.RoutePath + "/" + p.DefaultAvatar, nil
		}
		return "", errors.Errorf("no picture for %s", u.ID)
	}

	// load avatar from remote location
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

	// get ID and location of locally cached avatar
	encID := p.encodeID(u.ID)
	location := p.location(encID) // location adds partion to path

	if _, err = os.Stat(location); os.IsNotExist(err) {
		if e := os.Mkdir(location, 0700); e != nil {
			return "", errors.Wrapf(e, "failed to mkdir avatar location %s", location)
		}
	}

	avFile := path.Join(location, encID+imgSfx)
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
	return p.RoutePath + "/" + encID + imgSfx, nil
}

// Routes returns auth routes for given provider
func (p *Proxy) Routes() (string, chi.Router) {
	router := chi.NewRouter()

	// GET /123456789.image
	router.Get("/{avatar}", func(w http.ResponseWriter, r *http.Request) {
		avatar := chi.URLParam(r, "avatar")
		location := p.location(strings.TrimSuffix(avatar, imgSfx))
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

	return p.RoutePath, router
}

// encodeID hashes user id to sha1
func (p *Proxy) encodeID(id string) string {
	h := sha1.New()
	_, err := h.Write([]byte(id))
	if err != nil {
		return id
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// get location for user id by adding partion to final path
// the end result is a full path like this - /tmp/avatars.test/92
func (p *Proxy) location(id string) string {
	checksum64 := crc64.Checksum([]byte(id), crc64.MakeTable(crc64.ECMA))
	partition := checksum64 % 100
	return path.Join(p.StorePath, fmt.Sprintf("%02d", partition))
}
