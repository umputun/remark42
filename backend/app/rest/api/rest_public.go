package api

import (
	"crypto/sha1" //nolint
	"encoding/base64"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	log "github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/cache"

	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/store"
)

// GET /find?site=siteID&url=post-url&format=[tree|plain]&sort=[+/-time|+/-score|+/-controversy ]
// find comments for given post. Returns in tree or plain formats, sorted
func (s *Rest) findCommentsCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	sort := r.URL.Query().Get("sort")
	if strings.HasPrefix(sort, " ") { // restore + replaced by " "
		sort = "+" + sort[1:]
	}
	log.Printf("[DEBUG] get comments for %+v, sort %s, format %s", locator, sort, r.URL.Query().Get("format"))

	key := cache.NewKey(locator.SiteID).ID(URLKey(r)).Scopes(locator.SiteID, locator.URL)
	data, err := s.Cache.Get(key, func() ([]byte, error) {
		comments, e := s.DataService.Find(locator, sort)
		if e != nil {
			comments = []store.Comment{} // error should clear comments and continue for post info
		}
		maskedComments := s.adminService.alterComments(comments, r)
		var b []byte
		switch r.URL.Query().Get("format") {
		case "tree":
			tree := rest.MakeTree(maskedComments, sort, s.ReadOnlyAge)
			if tree.Nodes == nil { // eliminate json nil serialization
				tree.Nodes = []*rest.Node{}
			}
			if s.DataService.IsReadOnly(locator) {
				tree.Info.ReadOnly = true
			}
			b, e = encodeJSONWithHTML(tree)
		default:
			withInfo := commentsWithInfo{Comments: maskedComments}
			if info, ee := s.DataService.Info(locator, s.ReadOnlyAge); ee == nil {
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
func (s *Rest) previewCommentCtrl(w http.ResponseWriter, r *http.Request) {
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
	if err = s.DataService.ValidateComment(&comment); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "invalid comment", rest.ErrCommentValidation)
		return
	}

	comment = s.CommentFormatter.Format(comment)
	comment.Sanitize()
	render.HTML(w, r, comment.Text)
}

// GET /info?site=siteID&url=post-url - get info about the post
func (s *Rest) infoCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}

	key := cache.NewKey(locator.SiteID).ID(URLKey(r)).Scopes(locator.SiteID, locator.URL)
	data, err := s.Cache.Get(key, func() ([]byte, error) {
		info, e := s.DataService.Info(locator, s.ReadOnlyAge)
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

// GET /last/{limit}?site=siteID - last comments for the siteID, across all posts, sorted by time
func (s *Rest) lastCommentsCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	log.Printf("[DEBUG] get last comments for %s", siteID)

	limit, err := strconv.Atoi(chi.URLParam(r, "limit"))
	if err != nil {
		limit = 0
	}

	key := cache.NewKey(siteID).ID(URLKey(r)).Scopes(lastCommentsScope)
	data, err := s.Cache.Get(key, func() ([]byte, error) {
		comments, e := s.DataService.Last(siteID, limit)
		if e != nil {
			return nil, e
		}
		comments = s.adminService.alterComments(comments, r)
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
func (s *Rest) commentByIDCtrl(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	siteID := r.URL.Query().Get("site")
	url := r.URL.Query().Get("url")

	log.Printf("[DEBUG] get comments by id %s, %s %s", id, siteID, url)

	comment, err := s.DataService.Get(store.Locator{SiteID: siteID, URL: url}, id)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get comment by id", rest.ErrCommentNotFound)
		return
	}
	comment = s.adminService.alterComments([]store.Comment{comment}, r)[0]
	render.Status(r, http.StatusOK)

	if err = R.RenderJSONWithHTML(w, r, comment); err != nil {
		log.Printf("[WARN] can't render last comments for url=%s, id=%s", url, id)
	}
}

// GET /comments?site=siteID&user=id - returns comments for given userID
func (s *Rest) findUserCommentsCtrl(w http.ResponseWriter, r *http.Request) {

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

	key := cache.NewKey(siteID).ID(URLKey(r)).Scopes(userID, siteID)
	data, err := s.Cache.Get(key, func() ([]byte, error) {
		comments, e := s.DataService.User(siteID, userID, limit, 0)
		if e != nil {
			return nil, e
		}
		comments = s.adminService.alterComments(comments, r)
		comments = filterComments(comments, func(c store.Comment) bool { return !c.Deleted })
		count, e := s.DataService.UserCount(siteID, userID)
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

// GET /config?site=siteID - returns configuration
func (s *Rest) configCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")

	type config struct {
		Version        string   `json:"version"`
		EditDuration   int      `json:"edit_duration"`
		MaxCommentSize int      `json:"max_comment_size"`
		Admins         []string `json:"admins"`
		AdminEmail     string   `json:"admin_email"`
		Auth           []string `json:"auth_providers"`
		LowScore       int      `json:"low_score"`
		CriticalScore  int      `json:"critical_score"`
		PositiveScore  bool     `json:"positive_score"`
		ReadOnlyAge    int      `json:"readonly_age"`
	}

	cnf := config{
		Version:        s.Version,
		EditDuration:   int(s.DataService.EditDuration.Seconds()),
		MaxCommentSize: s.DataService.MaxCommentSize,
		Admins:         s.DataService.AdminStore.Admins(siteID),
		AdminEmail:     s.DataService.AdminStore.Email(siteID),
		LowScore:       s.ScoreThresholds.Low,
		CriticalScore:  s.ScoreThresholds.Critical,
		PositiveScore:  s.DataService.PositiveScore,
		ReadOnlyAge:    s.ReadOnlyAge,
	}

	cnf.Auth = []string{}
	for _, ap := range s.Authenticator.Providers() {
		cnf.Auth = append(cnf.Auth, ap.Name())
	}

	if cnf.Admins == nil { // prevent json serialization to nil
		cnf.Admins = []string{}
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, cnf)
}

// GET /count?site=siteID&url=post-url - get number of comments for given post
func (s *Rest) countCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	count, err := s.DataService.Count(locator)
	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get count", rest.ErrPostNotFound)
		return
	}
	render.JSON(w, r, R.JSON{"count": count, "locator": locator})
}

// POST /counts?site=siteID - get number of comments for posts from post body
func (s *Rest) countMultiCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	posts := []string{}
	if err := render.DecodeJSON(http.MaxBytesReader(w, r.Body, hardBodyLimit), &posts); err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get list of posts from request", rest.ErrSiteNotFound)
		return
	}

	// key could be long for multiple posts, make it sha1
	k := URLKey(r) + strings.Join(posts, ",")
	hasher := sha1.New() //nolint
	if _, err := hasher.Write([]byte(k)); err != nil {
		rest.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't make sha1 for list of urls", rest.ErrInternal)
		return
	}
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	key := cache.NewKey(siteID).ID(sha).Scopes(siteID)
	data, err := s.Cache.Get(key, func() ([]byte, error) {
		counts, e := s.DataService.Counts(siteID, posts)
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
func (s *Rest) listCtrl(w http.ResponseWriter, r *http.Request) {

	siteID := r.URL.Query().Get("site")
	limit, skip := 0, 0

	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
		limit = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("skip")); err == nil {
		skip = v
	}

	key := cache.NewKey(siteID).ID(URLKey(r)).Scopes(siteID)
	data, err := s.Cache.Get(key, func() ([]byte, error) {
		posts, e := s.DataService.List(siteID, limit, skip)
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
func (s *Rest) loadPictureCtrl(w http.ResponseWriter, r *http.Request) {

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
	imgRdr, size, err := s.ImageService.Load(id)
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

	defer imgRdr.Close()

	w.Header().Set("Content-Type", imgContentType(id))
	w.Header().Set("Content-Length", strconv.Itoa(int(size)))
	w.WriteHeader(http.StatusOK)
	if _, err = io.Copy(w, imgRdr); err != nil {
		log.Printf("[WARN] can't send response to %s, %s", r.RemoteAddr, err)
	}
}
