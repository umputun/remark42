package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/gorilla/feeds"

	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/rest/cache"
	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/engine"
)

const maxRssItems = 20

// ui uses links like <post-url>#remark42__comment-<comment-id>
const uiNav = "#remark42__comment-"

func (s *Rest) rssRoutes() chi.Router {
	router := chi.NewRouter()
	router.Get("/post", s.rssPostCommentsCtrl)
	router.Get("/site", s.rssSiteCommentsCtrl)
	router.Get("/reply", s.rssRepliesCtrl)
	return router
}

// GET /rss/post?site=siteID&url=post-url
func (s *Rest) rssPostCommentsCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	log.Printf("[DEBUG] get rss for post %+v", locator)

	data, err := s.Cache.Get(cache.Key(cache.URLKey(r), locator.SiteID, locator.URL), func() ([]byte, error) {
		comments, e := s.DataService.Find(locator, "-time")
		if e != nil {
			return nil, e
		}
		comments = s.adminService.alterComments(comments, r)
		rss, e := s.toRssFeed(locator.URL, comments)
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
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(data); err != nil {
		log.Printf("[WARN] failed to send response to %s, %s", r.RemoteAddr, err)
	}
}

// GET /rss/site?site=siteID
func (s *Rest) rssSiteCommentsCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	log.Printf("[DEBUG] get rss for site %s", siteID)

	data, err := s.Cache.Get(cache.Key(cache.URLKey(r), siteID, "last"), func() ([]byte, error) {
		comments, e := s.DataService.Last(siteID, maxRssItems)
		if e != nil {
			return nil, e
		}
		comments = s.adminService.alterComments(comments, r)

		rss, e := s.toRssFeed(r.URL.Query().Get("site"), comments)
		if e != nil {
			return nil, e
		}
		return []byte(rss), e
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get last comments")
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Printf("[WARN] failed to send response to %s, %s", r.RemoteAddr, err)
	}
}

// GET /rss/reply?id=123&site=siteID&url=post-url
func (s *Rest) rssRepliesCtrl(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	siteID := r.URL.Query().Get("site")
	url := r.URL.Query().Get("url")
	locator := store.Locator{SiteID: siteID, URL: url}
	log.Printf("[DEBUG] get rss replies for comment %s and post %+v", id, locator)

	data, err := s.Cache.Get(cache.Key(cache.URLKey(r), locator.SiteID, locator.URL), func() ([]byte, error) {
		// do nothing if comment not found
		_, e := s.DataService.Get(locator, id)
		if e != nil {
			return nil, e
		}

		comments, e := s.DataService.Find(locator, "any")
		if e != nil {
			return nil, e
		}
		comments = s.adminService.alterComments(comments, r)

		replies := findReplies(comments, id)
		replies = engine.SortComments(replies, "-time")
		rss, e := s.toRssFeed(url, replies)
		if e != nil {
			return nil, e
		}
		return []byte(rss), e
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't find replies")
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Printf("[WARN] failed to send reponse to %s, %s", r.RemoteAddr, err)
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
	for i, c := range comments {
		f := feeds.Item{
			Title:       c.User.Name,
			Link:        &feeds.Link{Href: c.Locator.URL + uiNav + c.ID},
			Description: c.Text,
			Created:     c.Timestamp,
			Author:      &feeds.Author{Name: c.User.Name},
		}
		if c.ParentID != "" {
			// add indication to parent comment
			parentComment, err := s.DataService.Get(c.Locator, c.ParentID)
			if err == nil {
				f.Title = fmt.Sprintf("%s > %s", c.User.Name, parentComment.User.Name)
			} else {
				log.Printf("[WARN] failed to get info about parent comment, %s", err)
			}
		}
		feed.Items = append(feed.Items, &f)
		if i > maxRssItems {
			break
		}
	}
	return feed.ToRss()
}

// findReplies find all replies for comment recursively. Skip deleted comments.
func findReplies(comments []store.Comment, parentID string) []store.Comment {
	res := []store.Comment{}
	for _, c := range comments {
		if c.ParentID == parentID {
			if !c.Deleted {
				res = append(res, c)
			}
			rs := findReplies(comments, c.ID)
			res = append(res, rs...)
		}
	}
	return res
}
