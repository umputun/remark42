package api

import (
	"fmt"
	"net/http"
	"time"

	cache "github.com/go-pkgz/lcw"
	log "github.com/go-pkgz/lgr"
	"github.com/gorilla/feeds"
	"github.com/pkg/errors"

	"github.com/umputun/remark/backend/app/rest"
	"github.com/umputun/remark/backend/app/store"
)

type rss struct {
	dataService rssStore
	cache       LoadingCache
}

type rssStore interface {
	Find(locator store.Locator, sort string, user store.User) ([]store.Comment, error)
	Last(siteID string, limit int, since time.Time, user store.User) ([]store.Comment, error)
	Get(locator store.Locator, commentID string, user store.User) (store.Comment, error)
	UserReplies(siteID, userID string, limit int, duration time.Duration) ([]store.Comment, string, error)
}

const maxRssItems = 20
const maxReplyDuration = 31 * 24 * time.Hour

// ui uses links like <post-url>#remark42__comment-<comment-id>
const uiNav = "#remark42__comment-"

// GET /rss/post?site=siteID&url=post-url
func (s *rss) postCommentsCtrl(w http.ResponseWriter, r *http.Request) {
	locator := store.Locator{SiteID: r.URL.Query().Get("site"), URL: r.URL.Query().Get("url")}
	log.Printf("[DEBUG] get rss for post %+v", locator)

	key := cache.NewKey(locator.SiteID).ID(URLKey(r)).Scopes(locator.SiteID, locator.URL)
	data, err := s.cache.Get(key, func() ([]byte, error) {
		comments, e := s.dataService.Find(locator, "-time", rest.GetUserOrEmpty(r))
		if e != nil {
			return nil, e
		}
		feed, e := s.toRssFeed(locator.URL, comments, "post comments for "+r.URL.Query().Get("url"))
		if e != nil {
			return nil, e
		}
		return []byte(feed), e
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't find comments", rest.ErrPostNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if _, err = w.Write(data); err != nil {
		log.Printf("[WARN] failed to send response to %s, %s", r.RemoteAddr, err)
	}
}

// GET /rss/site?site=siteID
func (s *rss) siteCommentsCtrl(w http.ResponseWriter, r *http.Request) {
	siteID := r.URL.Query().Get("site")
	log.Printf("[DEBUG] get rss for site %s", siteID)

	key := cache.NewKey(siteID).ID(URLKey(r)).Scopes(siteID, lastCommentsScope)
	data, err := s.cache.Get(key, func() ([]byte, error) {
		comments, e := s.dataService.Last(siteID, maxRssItems, time.Time{}, rest.GetUserOrEmpty(r))
		if e != nil {
			return nil, e
		}

		feed, e := s.toRssFeed(r.URL.Query().Get("site"), comments, "site comment for "+siteID)
		if e != nil {
			return nil, e
		}
		return []byte(feed), e
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get last comments", rest.ErrSiteNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(data); err != nil {
		log.Printf("[WARN] failed to send response to %s, %s", r.RemoteAddr, err)
	}
}

// GET /rss/reply?user=userID&site=siteID
func (s *rss) repliesCtrl(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user")
	siteID := r.URL.Query().Get("site")
	log.Printf("[DEBUG] get rss replies to user %s for site %s", userID, siteID)

	key := cache.NewKey(siteID).ID(URLKey(r)).Scopes(siteID, lastCommentsScope)
	data, err := s.cache.Get(key, func() (res []byte, e error) {

		replies, userName, e := s.dataService.UserReplies(siteID, userID, maxRssItems, maxReplyDuration)
		if e != nil {
			return nil, errors.Wrap(e, "can't get last comments")
		}

		feed, e := s.toRssFeed(siteID, replies, "replies to "+userName)
		if e != nil {
			return nil, e
		}
		return []byte(feed), e
	})

	if err != nil {
		rest.SendErrorJSON(w, r, http.StatusBadRequest, err, "can't get replies", rest.ErrSiteNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(data); err != nil {
		log.Printf("[WARN] failed to send response to %s, %s", r.RemoteAddr, err)
	}
}

func (s *rss) toRssFeed(url string, comments []store.Comment, description string) (string, error) {

	if description == "" {
		description = "comment updates"
	}
	lastCommentTS := time.Unix(0, 0)
	if len(comments) > 0 {
		lastCommentTS = comments[0].Timestamp
	}

	feed := &feeds.Feed{
		Title:       "Remark42 comments",
		Link:        &feeds.Link{Href: url},
		Description: description,
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
			Id:          c.ID,
		}
		if c.ParentID != "" {
			// add indication to parent comment
			parentComment, err := s.dataService.Get(c.Locator, c.ParentID, store.User{})
			if err == nil {
				f.Title = fmt.Sprintf("%s > %s", c.User.Name, parentComment.User.Name)
				f.Description = f.Description + "<blockquote><p>" + parentComment.Snippet(300) + "</p></blockquote>"
			} else {
				log.Printf("[WARN] failed to get info about parent comment, %s", err)
			}
		}
		if c.PostTitle != "" {
			f.Title = f.Title + ", " + c.PostTitle
		}

		feed.Items = append(feed.Items, &f)
		if i > maxRssItems {
			break
		}
	}
	return feed.ToRss()
}
