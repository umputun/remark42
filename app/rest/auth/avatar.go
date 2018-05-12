package auth

import (
	"fmt"
	"hash/crc64"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/pkg/errors"

	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/store"
)

// AvatarProxy provides avatar store and http handler for avatars
type AvatarProxy struct {
	StorePath string
	RoutePath string
	RemarkURL string

	once     sync.Once
	ctcTable *crc64.Table
}

const imgSfx = ".image"

// Put stores retrieved avatar to StorePath. Gets image from user info. Returns proxied url
func (p *AvatarProxy) Put(u store.User) (avatarURL string, err error) {

	// no picture for user, try default avatar
	if u.Picture == "" {
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
	encID := store.EncodeID(u.ID)
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
	return p.RemarkURL + p.RoutePath + "/" + encID + imgSfx, nil
}

// Routes returns auth routes for given provider
func (p *AvatarProxy) Routes() (string, chi.Router) {
	router := chi.NewRouter()

	// GET /123456789.image
	router.Get("/{avatar}", func(w http.ResponseWriter, r *http.Request) {

		avatar := chi.URLParam(r, "avatar")

		// client-side caching
		etag := `"` + avatar + `"`
		w.Header().Set("Etag", etag)
		w.Header().Set("Cache-Control", "max-age=2592000") // 30 days
		if match := r.Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, etag) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		location := p.location(strings.TrimSuffix(avatar, imgSfx))
		avFile := path.Join(location, avatar)
		fh, err := os.Open(avFile)
		if err != nil {
			rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't load avatar")
			return
		}

		defer func() {
			if e := fh.Close(); e != nil {
				log.Printf("[WARN] can't close avatar file %s, %s", avFile, e)
			}
		}()

		w.Header().Set("Content-Type", "image/*")
		if fi, err := fh.Stat(); err == nil {
			w.Header().Set("Content-Length", strconv.Itoa(int(fi.Size())))
		}

		// write all headers
		if status, ok := r.Context().Value(render.StatusCtxKey).(int); ok {
			w.WriteHeader(status)
		}
		if _, err = io.Copy(w, fh); err != nil {
			log.Printf("[WARN] can't send response to %s, %s", r.RemoteAddr, err)
		}
	})

	return p.RoutePath, router
}

// get location for user id by adding partition to final path in order to keep files
// in different subdirectories and avoid too many files in a single place.
// the end result is a full path like this - /tmp/avatars.test/92
func (p *AvatarProxy) location(id string) string {
	p.once.Do(func() { p.ctcTable = crc64.MakeTable(crc64.ECMA) })
	checksum64 := crc64.Checksum([]byte(id), p.ctcTable)
	partition := checksum64 % 100
	return path.Join(p.StorePath, fmt.Sprintf("%02d", partition))
}
