package rest

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/umputun/remark/app/store"
)

type moderator struct {
	dataStore store.Interface
}

func (m *moderator) routes() chi.Router {
	router := chi.NewRouter()
	router.Use(AdminOnly)
	router.Delete("/comment/{id}", m.deleteCommentCtrl)
	return router
}

// DELETE /comment/{id}?url=post-url
func (m *moderator) deleteCommentCtrl(w http.ResponseWriter, r *http.Request) {

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		log.Printf("[WARN] bad id %s", chi.URLParam(r, "id"))
		httpError(w, r, http.StatusBadRequest, err, "can't parse id")
	}

	log.Printf("[INFO] delete comment %d", id)

	url := r.URL.Query().Get("url")
	err = m.dataStore.Delete(store.Locator{URL: url}, id)
	if err != nil {
		log.Printf("[WARN] can't delete comment, %s", err)
		httpError(w, r, http.StatusInternalServerError, err, "can't delete comment")
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, JSON{"id": id, "url": url})
}
