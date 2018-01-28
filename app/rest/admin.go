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

	"github.com/umputun/remark/app/migrator"
	"github.com/umputun/remark/app/rest/auth"
	"github.com/umputun/remark/app/rest/common"
	"github.com/umputun/remark/app/store"
)

// admin provides router for all requests available for admin users only
type admin struct {
	dataService store.Service
	exporter    migrator.Exporter
	importer    migrator.Importer
	cache       common.LoadingCache
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

// DELETE /comment/{id}?site=siteID&url=post-url - removes comment
func (a *admin) deleteCommentCtrl(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	log.Printf("[INFO] delete comment %s", id)

	err := a.dataService.Delete(locator, id)
	if err != nil {
		common.SendErrorJSON(w, r, http.StatusInternalServerError, err, "can't delete comment")
		return
	}
	a.cache.Flush()
	render.Status(r, http.StatusOK)
	render.JSON(w, r, JSON{"id": id, "loc": locator})
}

// PUT /user/{userid}?site=side-id&block=1 - block or unblock user
func (a *admin) setBlockCtrl(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userid")
	siteID := r.URL.Query().Get("site")
	blockStatus := r.URL.Query().Get("block") == "1"

	if err := a.dataService.SetBlock(siteID, userID, blockStatus); err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't set blocking status")
		return
	}
	a.cache.Flush()
	render.JSON(w, r, JSON{"user_id": userID, "site_id": siteID, "block": blockStatus})
}

// PUT /pin/{id}?site=siteID&url=post-url&pin=1
// mark/unmark comment as a special
func (a *admin) setPinCtrl(w http.ResponseWriter, r *http.Request) {
	commentID := chi.URLParam(r, "id")
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	pinStatus := r.URL.Query().Get("pin") == "1"

	if err := a.dataService.SetPin(locator, commentID, pinStatus); err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't set pin status")
		return
	}
	a.cache.Flush()
	render.JSON(w, r, JSON{"id": commentID, "locator": locator, "pin": pinStatus})
}

// GET /export?site=site-id?mode=file|stream
// exports all comments for siteID as json stream or gz file
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
		common.SendErrorJSON(w, r, http.StatusInternalServerError, err, "export failed")
	}
}

// POST /import?site=site-id
// imports comments from post body.
func (a *admin) importCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	if err := a.importer.Import(r.Body, siteID); err != nil {
		common.SendErrorJSON(w, r, http.StatusBadRequest, err, "import failed")
	}
	a.cache.Flush()
}

func (a *admin) checkBlocked(siteID string, user store.User) bool {
	return a.dataService.IsBlocked(siteID, user.ID)
}

// processes comments and hides text of all comments for blocked users.
// resets score and votes too
func (a *admin) maskBlockedUsers(comments []store.Comment) (res []store.Comment) {
	res = make([]store.Comment, len(comments))
	for i, c := range comments {
		if a.dataService.IsBlocked(c.Locator.SiteID, c.User.ID) {
			c.User.Blocked = true
			c.Text = "this comment was deleted"
			c.Score = 0
			c.Votes = map[string]bool{}
			c.Edit = nil
		}
		res[i] = c
	}
	return res
}
