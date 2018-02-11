package migrator

import (
	"encoding/xml"
	"io"
	"log"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/umputun/remark/app/store"
)

// Disqus implements Importer from disqus xml
type Disqus struct {
	DataStore store.Interface
}

type disqusThread struct {
	UID         string    `xml:"id,attr"`
	Forum       string    `xml:"forum"`
	Link        string    `xml:"link"`
	Title       string    `xml:"title"`
	Message     string    `xml:"message"`
	CreateAt    time.Time `xml:"createdAt"`
	AuthorName  string    `xml:"author>name"`
	AuthorEmail string    `xml:"author>email"`
	Anonymous   bool      `xml:"author>isAnonymous"`
	IP          string    `xml:"ipAddress"`
	Closed      bool      `xml:"isClosed"`
	Deleted     bool      `xml:"isDeleted"`
}

type disqusComment struct {
	UID            string    `xml:"id,attr"`
	ID             string    `xml:"id"`
	Message        string    `xml:"message"`
	CreatedAt      time.Time `xml:"createdAt"`
	IsSpam         bool      `xml:"isSpam"`
	AuthorEmail    string    `xml:"author>email"`
	AuthorName     string    `xml:"author>name"`
	AuthorUserName string    `xml:"author>username"`
	IP             string    `xml:"ipAddress"`
	Tid            uid       `xml:"thread"`
	Pid            uid       `xml:"parent"`
}

type uid struct {
	Val string `xml:"id,attr"`
}

// Import from disqus and save to store
func (d *Disqus) Import(r io.Reader, siteID string) (err error) {

	commentsCh := d.convert(r, siteID)
	failed, passed := 0, 0
	for c := range commentsCh {
		if _, err = d.DataStore.Create(c); err != nil {
			failed++
			continue
		}
		passed++
	}

	if failed > 0 {
		return errors.Errorf("failed to save %d comments", failed)
	}

	log.Printf("[DEBUG] imported %d comments to site %s", passed, siteID)
	return nil
}

// convert disqus stream (xml) from reader and fill channel of comments.
// runs async and closes channel on completion.
func (d *Disqus) convert(r io.Reader, siteID string) (ch chan store.Comment) {

	postsMap := map[string]string{} // tid:url
	decoder := xml.NewDecoder(r)
	commentsCh := make(chan store.Comment)

	stats := struct {
		inpThreads, inpComments     int
		commentsCount, spamComments int
		failedThreads, failedPosts  int
	}{}

	go func() {
		for {
			t, err := decoder.Token()
			if t == nil || err != nil {
				break
			}

			switch se := t.(type) {
			case xml.StartElement:
				if se.Name.Local == "thread" {
					stats.inpThreads++
					thread := disqusThread{}
					if err := decoder.DecodeElement(&thread, &se); err != nil {
						log.Printf("[WARN] can't decode disqus thread, %s", err)
						stats.failedThreads++
						continue
					}
					postsMap[thread.UID] = thread.Link
					continue
				}
				if se.Name.Local == "post" {
					stats.inpComments++
					comment := disqusComment{}
					if err := decoder.DecodeElement(&comment, &se); err != nil {
						log.Printf("[WARN] can't decode disqus comment, %s", err)
						stats.failedPosts++
						continue
					}
					if comment.IsSpam {
						stats.spamComments++
						continue
					}
					c := store.Comment{
						ID:        comment.UID,
						Locator:   store.Locator{URL: postsMap[comment.Tid.Val], SiteID: siteID},
						User:      store.User{ID: "disqus_" + comment.AuthorUserName, Name: comment.AuthorName, IP: comment.IP},
						Text:      d.cleanText(comment.Message),
						Timestamp: comment.CreatedAt,
						ParentID:  comment.Pid.Val,
					}
					if c.User.ID == "disqus_" { // empty comment.AuthorUserName from disqus
						c.User.ID = "disqus_" + c.User.Name
					}
					if c.ID == "" { // no comment.UID
						c.ID = comment.ID
					}
					commentsCh <- c
					stats.commentsCount++
					if stats.commentsCount%1000 == 0 {
						log.Printf("[DEBUG] processed %d comments", stats.commentsCount)
					}
				}

			}
		}
		close(commentsCh)
		log.Printf("[INFO] converted %d posts, %+v", len(postsMap), stats)
	}()

	return commentsCh
}

func (d *Disqus) cleanText(text string) string {
	text = strings.Replace(text, "\n", "", -1)
	text = strings.Replace(text, "\t", "", -1)
	return text
}
