package proxy

import (
	"hash/crc64"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"

	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/store"
)

// Avatar provides file-system store and http handler for avatars
// On user login auth will call Put and it will retrieve and save picture locally.
type Avatar struct {
	Store     AvatarStore
	RoutePath string
	RemarkURL string

	once     sync.Once
	ctcTable *crc64.Table
}

const imgSfx = ".image"

// Put stores retrieved avatar to StorePath. Gets image from user info. Returns proxied url
func (p *Avatar) Put(u store.User) (avatarURL string, err error) {

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

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("failed to get avatar from the orig, status %s", resp.Status)
	}

	avatar, err := p.Store.Put(u.ID, resp.Body)
	if err != nil {
		return "", err
	}

	log.Printf("[DEBUG] saved avatar from %s to %s, user %q", u.Picture, avatar, u.Name)
	return p.RemarkURL + p.RoutePath + "/" + avatar, nil
}

// Routes returns auth routes for given provider
func (p *Avatar) Routes(middlewares ...func(http.Handler) http.Handler) (string, chi.Router) {
	router := chi.NewRouter()
	router.Use(middlewares...)

	// GET /123456789.image
	router.Get("/{avatar}", func(w http.ResponseWriter, r *http.Request) {

		avatar := chi.URLParam(r, "avatar")

		// enforce client-side caching
		etag := `"` + avatar + `"`
		w.Header().Set("Etag", etag)
		w.Header().Set("Cache-Control", "max-age=2592000") // 30 days
		if match := r.Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, etag) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		avReader, size, err := p.Store.Get(avatar)
		if err != nil {
			rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't load avatar")
			return
		}

		defer func() {
			if e := avReader.Close(); e != nil {
				log.Printf("[WARN] can't close avatar reader for %s, %s", avatar, e)
			}
		}()

		w.Header().Set("Content-Type", "image/*")
		w.Header().Set("Content-Length", strconv.Itoa(size))
		w.WriteHeader(http.StatusOK)
		if _, err = io.Copy(w, avReader); err != nil {
			log.Printf("[WARN] can't send response to %s, %s", r.RemoteAddr, err)
		}
	})

	return p.RoutePath, router
}
