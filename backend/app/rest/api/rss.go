package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/gorilla/feeds"
	"github.com/pkg/errors"

	"github.com/go-pkgz/rest/cache"

	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/store"
)

const maxRssItems = 20
const maxLastCommentsReply = 1000
const maxReplyDuration = 30 * time.Minute

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

	key := cache.NewKey(locator.SiteID).ID(URLKey(r)).Scopes(locator.SiteID, locator.URL)
	data, err := s.Cache.Get(key, func() ([]byte, error) {
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

	key := cache.NewKey(siteID).ID(URLKey(r)).Scopes(siteID, lastCommentsScope)
	data, err := s.Cache.Get(key, func() ([]byte, error) {
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

// GET /rss/reply?user=userID&site=siteID
func (s *Rest) rssRepliesCtrl(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user")
	siteID := r.URL.Query().Get("site")
	log.Printf("[DEBUG] get rss replies to user %s for site %s", userID, siteID)

	key := cache.NewKey(siteID).ID(URLKey(r)).Scopes(siteID, lastCommentsScope)
	data, err := s.Cache.Get(key, func() (res []byte, e error) {
		comments, e := s.DataService.Last(siteID, maxLastCommentsReply)
		if e != nil {
			return nil, errors.Wrap(e, "can't get last comments")
		}
		comments = s.adminService.alterComments(comments, r)
		replies := []store.Comment{}
		for _, c := range comments {
			if len(replies) > maxRssItems || c.Timestamp.Add(maxReplyDuration).Before(time.Now()) {
				break
			}
			if c.ParentID != "" && !c.Deleted && c.User.ID != userID { // not interested replies to yourself
				var pc store.Comment
				if pc, e = s.DataService.Get(c.Locator, c.ParentID); e != nil {
					return nil, errors.Wrap(e, "can't get parent comment")
				}
				if pc.User.ID == userID {
					replies = append(replies, c)
				}
			}
		}

		rss, e := s.toRssFeed(siteID, replies)
		if e != nil {
			return nil, e
		}
		return []byte(rss), e
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get replies")
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
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
