package api

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/gorilla/feeds"
	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/store"
)

// GET /rss/post?site=siteID&url=post-url
func (s *Rest) rssPostCommentsCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	sort := "-time"
	log.Printf("[DEBUG] get rss for post %+v", locator)

	data, err := s.Cache.Get(rest.URLKey(r), time.Hour, func() ([]byte, error) {
		comments, e := s.DataService.Find(locator, sort)
		if e != nil {
			return nil, e
		}
		maskedComments := s.adminService.alterComments(comments, r)

		rss, e := s.toRssFeed(locator.URL, maskedComments)
		if e != nil {
			return nil, e
		}
		return []byte(rss), e
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't find comments")
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	if status, ok := r.Context().Value(render.StatusCtxKey).(int); ok {
		w.WriteHeader(status)
	}
	if _, err := w.Write(data); err != nil {
		log.Printf("[WARN] failed to send response to %s, %s", r.RemoteAddr, err)
	}
}

func (s *Rest) toRssFeed(url string, comments []store.Comment) (string, error) {

	lastCommentTS := time.Unix(0, 0)
	if len(comments) > 0 {
		lastCommentTS = comments[0].Timestamp
	}

	feed := &feeds.Feed{
		Title:       "Remark42 comments",
		Link:        &feeds.Link{Href: url},
		Description: "comment updates",
		Created:     lastCommentTS,
	}

	feed.Items = []*feeds.Item{}
	for _, c := range comments {
		f := feeds.Item{
			Title:       "update",
			Link:        &feeds.Link{Href: url},
			Description: c.Text,
			Created:     c.Timestamp,
			Author:      &feeds.Author{Name: c.User.Name},
		}
		feed.Items = append(feed.Items, &f)
	}
	return feed.ToRss()
}
