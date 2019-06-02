package api

import (
	"context"
	"crypto/sha1" // nolint
	"encoding/base64"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	log "github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/cache"
	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/image"
	"github.com/umputun/remark/backend/app/store/service"
)

type public struct {
	dataService      pubStore
	cache            cache.LoadingCache
	readOnlyAge      int
	commentFormatter *store.CommentFormatter
	imageService     *image.Service
	webRoot          string
	streamTimeOut    time.Duration
	streamRefresh    time.Duration
}

type pubStore interface {
	Create(comment store.Comment) (commentID string, err error)
	Get(locator store.Locator, commentID string, user store.User) (store.Comment, error)
	Find(locator store.Locator, sort string, user store.User) ([]store.Comment, error)
	Last(siteID string, limit int, since time.Time, user store.User) ([]store.Comment, error)
	User(siteID, userID string, limit, skip int, user store.User) ([]store.Comment, error)
	UserCount(siteID, userID string) (int, error)
	Count(locator store.Locator) (int, error)
	List(siteID string, limit int, skip int) ([]store.PostInfo, error)
	Info(locator store.Locator, readonlyAge int) (store.PostInfo, error)

	ValidateComment(c *store.Comment) error
	IsReadOnly(locator store.Locator) bool
	Counts(siteID string, postIDs []string) ([]store.PostInfo, error)
}

// GET /find?site=siteID&url=post-url&format=[tree|plain]&sort=[+/-time|+/-score|+/-controversy ]&view=[user|all]
// find comments for given post. Returns in tree or plain formats, sorted
func (s *public) findCommentsCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	sort := r.URL.Query().Get("sort")
	if strings.HasPrefix(sort, " ") { // restore + replaced by " "
		sort = "+" + sort[1:]
	}

	view := r.URL.Query().Get("view")
	log.Printf("[DEBUG] get comments for %+v, sort %s, format %s", locator, sort, r.URL.Query().Get("format"))

	key := cache.NewKey(locator.SiteID).ID(URLKeyWithUser(r)).Scopes(locator.SiteID, locator.URL)
	data, err := s.cache.Get(key, func() ([]byte, error) {
		comments, e := s.dataService.Find(locator, sort, rest.GetUserOrEmpty(r))
		if e != nil {
			comments = []store.Comment{} // error should clear comments and continue for post info
		}
		comments = s.applyView(comments, view)
		var b []byte
		switch r.URL.Query().Get("format") {
		case "tree":
			tree := service.MakeTree(comments, sort, s.readOnlyAge)
			if tree.Nodes == nil { // eliminate json nil serialization
				tree.Nodes = []*service.Node{}
			}
			if s.dataService.IsReadOnly(locator) {
				tree.Info.ReadOnly = true
			}
			b, e = encodeJSONWithHTML(tree)
		default:
			withInfo := commentsWithInfo{Comments: comments}
			if info, ee := s.dataService.Info(locator, s.readOnlyAge); ee == nil {
				withInfo.Info = info
			}
			b, e = encodeJSONWithHTML(withInfo)
		}
		return b, e
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't find comments", rest.ErrCommentNotFound)
		return
	}

	if err = R.RenderJSONFromBytes(w, r, data); err != nil {
		log.Printf("[WARN] can't render comments for post %+v", locator)
	}
}

// POST /preview, body is a comment, returns rendered html
func (s *public) previewCommentCtrl(w http.ResponseWriter, r *http.Request) {
	comment := store.Comment{}
	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't bind comment", rest.ErrDecode)
		return
	}

	user, err := rest.GetUserInfo(r)
	if err != nil { // this not suppose to happen (handled by Auth), just dbl-check
		rest.SendErrorJSON(w, r, http.StatusUnauthorized, err, "can't get user info", rest.ErrNoAccess)
		return
	}
	comment.User = user
	comment.Orig = comment.Text
	if err = s.dataService.ValidateComment(&comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid comment", rest.ErrCommentValidation)
		return
	}

	comment = s.commentFormatter.Format(comment)
	comment.Sanitize()
	render.HTML(w, r, comment.Text)
}

// GET /info?site=siteID&url=post-url - get info about the post
func (s *public) infoCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}

	key := cache.NewKey(locator.SiteID).ID(URLKey(r)).Scopes(locator.SiteID, locator.URL)
	data, err := s.cache.Get(key, func() ([]byte, error) {
		info, e := s.dataService.Info(locator, s.readOnlyAge)
		if e != nil {
			return nil, e
		}
		return encodeJSONWithHTML(info)
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get post info", rest.ErrPostNotFound)
		return
	}

	if err = R.RenderJSONFromBytes(w, r, data); err != nil {
		log.Printf("[WARN] can't render info for post %+v", locator)
	}
}

// GET /stream/info?site=siteID&url=post-url - get info stream about the post
func (s *public) infoStreamCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	log.Printf("[DEBUG] start stream for %+v, timeout=%v, refresh=%v", locator, s.streamTimeOut, s.streamRefresh)

	key := cache.NewKey(locator.SiteID).ID(URLKey(r)).Scopes(locator.SiteID, locator.URL)
	info := func() (data []byte, upd bool, err error) {
		data, err = s.cache.Get(key, func() ([]byte, error) {
			info, e := s.dataService.Info(locator, s.readOnlyAge)
			if e != nil {
				return nil, e
			}
			upd = true // cache update used as indication of post update
			return encodeJSONWithHTML(info)
		})
		if err != nil {
			return data, false, err
		}

		return data, upd, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.streamTimeOut)
	tick := time.NewTicker(s.streamRefresh)
	defer func() {
		tick.Stop()
		cancel()
	}()

	for {
		select {
		case <-r.Context().Done(): // request closes by remote client
			return
		case <-ctx.Done(): // request closed by timeout
			return
		case <-tick.C:
			resp, upd, err := info()
			if err != nil {
				rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get post info", rest.ErrPostNotFound)
				return
			}
			if upd {
				w.Write(resp)
				if fw, ok := w.(http.Flusher); ok {
					fw.Flush()
				}
			}
		}
	}
}

// GET /last/{limit}?site=siteID&since=unix_ts_msec - last comments for the siteID, across all posts, sorted by time, optionally
// limited with "since" param
func (s *public) lastCommentsCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	log.Printf("[DEBUG] get last comments for %s", siteID)

	limit, err := strconv.Atoi(chi.URLParam(r, "limit"))
	if err != nil {
		limit = 0
	}

	sinceTime := time.Time{}
	since := r.URL.Query().Get("since")
	if since != "" {
		unixTS, e := strconv.ParseInt(since, 10, 64)
		if e != nil {
			rest.SendErrorJSON(w, r, http.StatusBadRequest, e, "can't translate since parameter", rest.ErrDecode)
			return
		}
		sinceTime = time.Unix(unixTS/1000, 1000000*(unixTS%1000)) // since param in msec timestamp
	}

	key := cache.NewKey(siteID).ID(URLKey(r)).Scopes(lastCommentsScope)
	data, err := s.cache.Get(key, func() ([]byte, error) {
		comments, e := s.dataService.Last(siteID, limit, sinceTime, rest.GetUserOrEmpty(r))
		if e != nil {
			return nil, e
		}
		// filter deleted from last comments view. Blocked marked as deleted and will sneak in without
		filterDeleted := filterComments(comments, func(c store.Comment) bool { return !c.Deleted })
		return encodeJSONWithHTML(filterDeleted)
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't get last comments", rest.ErrInternal)
		return
	}

	if err = R.RenderJSONFromBytes(w, r, data); err != nil {
		log.Printf("[WARN] can't render last comments for site %s", siteID)
	}
}

// GET /id/{id}?site=siteID&url=post-url - gets a comment by id
func (s *public) commentByIDCtrl(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	siteID := r.URL.Query().Get("site")
	url := r.URL.Query().Get("url")

	log.Printf("[DEBUG] get comments by id %s, %s %s", id, siteID, url)

	comment, err := s.dataService.Get(store.Locator{SiteID: siteID, URL: url}, id, rest.GetUserOrEmpty(r))
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get comment by id", rest.ErrCommentNotFound)
		return
	}
	render.Status(r, http.StatusOK)

	if err = R.RenderJSONWithHTML(w, r, comment); err != nil {
		log.Printf("[WARN] can't render last comments for url=%s, id=%s", url, id)
	}
}

// GET /comments?site=siteID&user=id - returns comments for given userID
func (s *public) findUserCommentsCtrl(w http.ResponseWriter, r *http.Request) {

	userID := r.URL.Query().Get("user")
	siteID := r.URL.Query().Get("site")

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 0
	}

	resp := struct {
		Comments []store.Comment `json:"comments,omitempty"`
		Count    int             `json:"count,omitempty"`
	}{}

	log.Printf("[DEBUG] get comments for userID %s, %s", userID, siteID)

	key := cache.NewKey(siteID).ID(URLKeyWithUser(r)).Scopes(userID, siteID)
	data, err := s.cache.Get(key, func() ([]byte, error) {
		comments, e := s.dataService.User(siteID, userID, limit, 0, rest.GetUserOrEmpty(r))
		if e != nil {
			return nil, e
		}
		comments = filterComments(comments, func(c store.Comment) bool { return !c.Deleted })
		count, e := s.dataService.UserCount(siteID, userID)
		if e != nil {
			return nil, e
		}
		resp.Comments, resp.Count = comments, count
		return encodeJSONWithHTML(resp)
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get comment by user id", rest.ErrCommentNotFound)
		return
	}

	if err = R.RenderJSONFromBytes(w, r, data); err != nil {
		log.Printf("[WARN] can't render found comments for user %s", userID)
	}
}

// GET /count?site=siteID&url=post-url - get number of comments for given post
func (s *public) countCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	count, err := s.dataService.Count(locator)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get count", rest.ErrPostNotFound)
		return
	}
	render.JSON(w, r, R.JSON{"count": count, "locator": locator})
}

// POST /counts?site=siteID - get number of comments for posts from post body
func (s *public) countMultiCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	posts := []string{}
	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &posts); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get list of posts from request", rest.ErrSiteNotFound)
		return
	}

	// key could be long for multiple posts, make it sha1
	k := URLKey(r) + strings.Join(posts, ",")
	h := sha1.Sum([]byte(k)) // nolint
	sha := base64.URLEncoding.EncodeToString(h[:])

	key := cache.NewKey(siteID).ID(sha).Scopes(siteID)
	data, err := s.cache.Get(key, func() ([]byte, error) {
		counts, e := s.dataService.Counts(siteID, posts)
		if e != nil {
			return nil, e
		}
		return encodeJSONWithHTML(counts)
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get counts for "+siteID, rest.ErrSiteNotFound)
		return
	}

	if err = R.RenderJSONFromBytes(w, r, data); err != nil {
		log.Printf("[WARN] can't render comments counters site %s", siteID)
	}
}

// GET /list?site=siteID&limit=50&skip=10 - list posts with comments
func (s *public) listCtrl(w http.ResponseWriter, r *http.Request) {

	siteID := r.URL.Query().Get("site")
	limit, skip := 0, 0

	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
		limit = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("skip")); err == nil {
		skip = v
	}

	key := cache.NewKey(siteID).ID(URLKey(r)).Scopes(siteID)
	data, err := s.cache.Get(key, func() ([]byte, error) {
		posts, e := s.dataService.List(siteID, limit, skip)
		if e != nil {
			return nil, e
		}
		return encodeJSONWithHTML(posts)
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get list of comments for "+siteID, rest.ErrSiteNotFound)
		return
	}

	if err = R.RenderJSONFromBytes(w, r, data); err != nil {
		log.Printf("[WARN] can't render posts lits for site %s", siteID)
	}
}

// GET /picture/{user}/{id} - get picture
func (s *public) loadPictureCtrl(w http.ResponseWriter, r *http.Request) {

	imgContentType := func(img string) string {
		img = strings.ToLower(img)
		switch {
		case strings.HasSuffix(img, ".png"):
			return "image/png"
		case strings.HasSuffix(img, ".jpg") || strings.HasSuffix(img, ".jpeg"):
			return "image/jpeg"
		case strings.HasSuffix(img, ".gif"):
			return "image/gif"
		}
		return "image/*"
	}

	id := chi.URLParam(r, "user") + "/" + chi.URLParam(r, "id")
	imgRdr, size, err := s.imageService.Load(id)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get image "+id, rest.ErrAssetNotFound)
		return
	}
	// enforce client-side caching
	etag := `"` + id + `"`
	w.Header().Set("Etag", etag)
	w.Header().Set("Cache-Control", "max-age=604800") // 7 days
	if match := r.Header.Get("If-None-Match"); match != "" {
		if strings.Contains(match, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	defer func() {
		if e := imgRdr.Close(); e != nil {
			log.Printf("[WARN] failed to close reader for picture %s, %v", id, e)
		}
	}()

	w.Header().Set("Content-Type", imgContentType(id))
	w.Header().Set("Content-Length", strconv.Itoa(int(size)))
	w.WriteHeader(http.StatusOK)
	if _, err = io.Copy(w, imgRdr); err != nil {
		log.Printf("[WARN] can't send response to %s, %s", r.RemoteAddr, err)
	}
}

// GET /index.html - respond to /index.html with the content of getstarted.html under /web root
func (s *public) getStartedCtrl(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile(path.Join(s.webRoot, "getstarted.html"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	render.HTML(w, r, string(data))
}

// GET /robots.txt
func (s *public) robotsCtrl(w http.ResponseWriter, r *http.Request) {
	allowed := []string{"/find", "/last", "/id", "/count", "/counts", "/list", "/config",
		"/img", "/avatar", "/picture"}
	for i := range allowed {
		allowed[i] = "Allow: /api/v1" + allowed[i]
	}
	render.PlainText(w, r, "User-agent: *\nDisallow: /auth/\nDisallow: /api/\n"+strings.Join(allowed, "\n")+"\n")
}

func (s *public) applyView(comments []store.Comment, view string) []store.Comment {
	if strings.EqualFold(view, "user") {
		projection := make([]store.Comment, len(comments))
		for i, c := range comments {
			p := store.Comment{
				ID:   c.ID,
				User: c.User,
			}
			projection[i] = p
		}
		return projection
	}
	return comments
}
