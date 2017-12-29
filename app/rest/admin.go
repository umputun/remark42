package rest

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	cache "github.com/patrickmn/go-cache"

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/store"
)

// admin provides router for all requests available for admin only
type admin struct {
	dataService store.Service
	exporter    migrator.Exporter
	importer    migrator.Importer
	respCache   *cache.Cache
}

func (a *admin) routes() chi.Router {
	router := chi.NewRouter()
	router.Use(auth.AdminOnly)
	router.Delete("/comment/{id}", a.deleteCommentCtrl)
	router.Put("/user/{userid}", a.setBlockCtrl)
	router.Get("/export", a.exportCtrl)
	router.Post("/import", a.importCtrl)
	router.Put("/pin/{id}", a.setPinCtrl)
	return router
}

// DELETE /comment/{id}?site=siteID&url=post-url
func (a *admin) deleteCommentCtrl(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	log.Printf("[INFO] delete comment %s", id)

	err := a.dataService.Delete(locator, id)
	if err != nil {
		log.Printf("[WARN] can't delete comment, %s", err)
		httpError(w, r, http.StatusInternalServerError, err, "can't delete comment")
		return
	}
	a.respCache.Flush()
	render.Status(r, http.StatusOK)
	render.JSON(w, r, JSON{"id": id, "loc": locator})
}

// PUT /user/{userid}?site=side-id&block=1
func (a *admin) setBlockCtrl(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userid")
	siteID := r.URL.Query().Get("site")
	blockStatus := r.URL.Query().Get("block") == "1"

	if err := a.dataService.SetBlock(store.Locator{SiteID: siteID}, userID, blockStatus); err != nil {
		httpError(w, r, http.StatusBadRequest, err, "can't set blocking status")
		return
	}
	a.respCache.Flush()
	render.JSON(w, r, JSON{"user_id": userID, "site_id": siteID, "block": blockStatus})
}

// PUT /pin/{id}?site=siteID&url=post-url&pin=1
func (a *admin) setPinCtrl(w http.ResponseWriter, r *http.Request) {
	commentID := chi.URLParam(r, "id")
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	pinStatus := r.URL.Query().Get("pin") == "1"

	if err := a.dataService.SetPin(locator, commentID, pinStatus); err != nil {
		httpError(w, r, http.StatusBadRequest, err, "can't set pin status")
		return
	}
	a.respCache.Flush()
	render.JSON(w, r, JSON{"id": commentID, "loc": locator, "pin": pinStatus})
}

// GET /export?site=site-id?mode=file|stream
func (a *admin) exportCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	var writer io.Writer = w
	if r.URL.Query().Get("mode") == "file" {
		exportFile := fmt.Sprintf("%s-%s.json.gz", siteID, time.Now().Format("20060102"))
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", "attachment;filename="+exportFile)
		w.WriteHeader(http.StatusOK)
		writer = gzip.NewWriter(w)
	}

	if err := a.exporter.Export(writer, siteID); err != nil {
		httpError(w, r, http.StatusInternalServerError, err, "export failed")
	}
}

// POST /import?site=site-id
func (a *admin) importCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	if err := a.importer.Import(r.Body, siteID); err != nil {
		httpError(w, r, http.StatusBadRequest, err, "import failed")
	}
}

func (a *admin) checkBlocked(locator store.Locator, user store.User) bool {
	return a.dataService.IsBlocked(locator, user.ID)
}
