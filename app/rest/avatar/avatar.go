// Package avatar provides cached proxy for user pictures/avatars
// refreshed by login and kept in local store
package avatar

import (
	"net/http"
	"time"

	"os"
	"path"

	"io"

	"log"

	"bytes"
	"image"
	"image/png"

	"fmt"
	"hash/crc64"

	"strings"

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

// Put gets original avatar url from user info and returns proxied
func (p *Proxy) Put(u store.User) (avatarURL string, err error) {
	if u.Picture == "" {
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

	pngWr := &bytes.Buffer{}
	if err = p.convertToPng(resp.Body, pngWr); err != nil {
		return "", err
	}

	location := p.location(u.ID)
	if err = os.Mkdir(location, 0600); err != nil && !strings.Contains(err.Error(), "file exists") {
		return "", errors.Wrapf(err, "failed to make avatar location %s", location)
	}

	avFile := path.Join(location, u.ID+".png")
	fh, err := os.Create(avFile)
	if err != nil {
		return "", errors.Wrapf(err, "can't create file %s", avFile)
	}
	defer func() {
		if e := fh.Close(); e != nil {
			log.Printf("[WARN] can't close avatar file %s, %s", avFile, e)
		}
	}()

	if _, err = io.Copy(fh, pngWr); err != nil {
		return "", errors.Wrapf(err, "can't save file %s", avFile)
	}
	return p.RoutePath + "/" + u.ID + ".png", nil
}

// Routes returns auth routes for given provider
func (p *Proxy) Routes() chi.Router {
	router := chi.NewRouter()
	router.Get("/{avatar}", func(w http.ResponseWriter, r *http.Request) {
		avatar := chi.URLParam(r, "avatar")
		location := p.location(strings.TrimSuffix(avatar, ".png"))
		avFile := path.Join(location, avatar)
		fh, err := os.Open(avFile)
		if err != nil {
			common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't load avatar")
			return
		}

		defer func() {
			if e := fh.Close(); e != nil {
				log.Printf("[WARN] can't close avatar file %s, %s", avFile, e)
			}
		}()

		w.Header().Set("Content-Type", "image/png")
		if status, ok := r.Context().Value(render.StatusCtxKey).(int); ok {
			w.WriteHeader(status)
		}
		if _, err = io.Copy(w, fh); err != nil {
			log.Printf("[WARN] can't send response to %s, %s", r.RemoteAddr, err)
		}
	})

	return router
}

func (p *Proxy) convertToPng(r io.Reader, w io.Writer) error {
	imageData, _, err := image.Decode(r)
	if err != nil {
		return errors.Wrap(err, "can't decode image")
	}

	if err = png.Encode(w, imageData); err != nil {
		return errors.Wrap(err, "can't encode png image")
	}
	return nil
}

func (p *Proxy) location(id string) string {
	checksum64 := crc64.Checksum([]byte(id), crc64.MakeTable(crc64.ECMA))
	partition := checksum64 % 1000
	return path.Join(p.StorePath, fmt.Sprintf("%03d", partition))
}
