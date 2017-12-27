package rest

import (
	"log"
	"net/http"

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
	respCache   *cache.Cache
}

func (a *admin) routes() chi.Router {
	router := chi.NewRouter()
	router.Use(auth.AdminOnly)
	router.Delete("/comment/{id}", a.deleteCommentCtrl)
	router.Put("/user/{userid}", a.setBlockCtrl)
	router.Get("/export", a.exportCtrl)
	router.Put("/pin/{id}", a.setPinCtrl)
	return router
}

// DELETE /comment/{id}?url=post-url
func (a *admin) deleteCommentCtrl(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	log.Printf("[INFO] delete comment %s", id)

	url := r.URL.Query().Get("url")
	err := a.dataService.Delete(store.Locator{URL: url}, id)
	if err != nil {
		log.Printf("[WARN] can't delete comment, %s", err)
		httpError(w, r, http.StatusInternalServerError, err, "can't delete comment")
		return
	}
	a.respCache.Flush()
	render.Status(r, http.StatusOK)
	render.JSON(w, r, JSON{"id": id, "url": url})
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

// PUT /pin/{id}?url=post-url&pin=1
func (a *admin) setPinCtrl(w http.ResponseWriter, r *http.Request) {
	commentID := chi.URLParam(r, "id")
	url := r.URL.Query().Get("url")
	pinStatus := r.URL.Query().Get("pin") == "1"

	if err := a.dataService.SetPin(store.Locator{URL: url}, commentID, pinStatus); err != nil {
		httpError(w, r, http.StatusBadRequest, err, "can't set pin status")
		return
	}
	a.respCache.Flush()
	render.JSON(w, r, JSON{"id": commentID, "url": url, "pin": pinStatus})
}

// GET /export?site=site-id
func (a *admin) exportCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	if err := a.exporter.Export(w, siteID); err != nil {
		httpError(w, r, http.StatusInternalServerError, err, "export failed")
	}
}
func (a *admin) checkBlocked(locator store.Locator, user store.User) bool {
	return a.dataService.IsBlocked(store.Locator{}, user.ID)
}
