package rest

import (
	"log"
	"net/http"

	"github.com/umputun/remark/app/migrator"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"

	"github.com/umputun/remark/app/store"
)

type admin struct {
	dataStore store.Interface
	exporter  migrator.Exporter
}

func (a *admin) routes() chi.Router {
	router := chi.NewRouter()
	router.Use(AdminOnly)
	router.Delete("/comment/{id}", a.deleteCommentCtrl)
	router.Put("/user/{userid}", a.setBlockCtrl)
	router.Get("/export", a.exportCtrl)
	return router
}

// DELETE /comment/{id}?url=post-url
func (a *admin) deleteCommentCtrl(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	log.Printf("[INFO] delete comment %s", id)

	url := r.URL.Query().Get("url")
	err := a.dataStore.Delete(store.Locator{URL: url}, id)
	if err != nil {
		log.Printf("[WARN] can't delete comment, %s", err)
		httpError(w, r, http.StatusInternalServerError, err, "can't delete comment")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, JSON{"id": id, "url": url})
}

// PUT /user/{userid}?site=side-id&block=1
func (a *admin) setBlockCtrl(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userid")
	siteID := r.URL.Query().Get("site")
	blockStatus := r.URL.Query().Get("block") == "1"

	if err := a.dataStore.SetBlock(store.Locator{SiteID: siteID}, userID, blockStatus); err != nil {
		httpError(w, r, http.StatusBadRequest, err, "can't set blocking status")
		return
	}

	render.JSON(w, r, JSON{"user_id": userID, "site_id": siteID, "block": blockStatus})
}

// GET /export?site=site-id
func (a *admin) exportCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	if err := a.exporter.Export(w, siteID); err != nil {
		httpError(w, r, http.StatusInternalServerError, err, "export failed")
	}
}
func (a *admin) checkBlocked(locator store.Locator, user store.User) bool {
	return a.dataStore.IsBlocked(store.Locator{}, user.ID)
}
