package migrator

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/remark42/backend/app/store"
)

// Disqus implements Importer from disqus xml
type Disqus struct {
	DataStore Store
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
	IP          string    `xml:"ipAddress"`
	Anonymous   bool      `xml:"author>isAnonymous"`
	Closed      bool      `xml:"isClosed"`
	Deleted     bool      `xml:"isDeleted"`
}

type disqusComment struct {
	UID            string    `xml:"id,attr"`
	ID             string    `xml:"id"`
	Message        string    `xml:"message"`
	CreatedAt      time.Time `xml:"createdAt"`
	AuthorEmail    string    `xml:"author>email"`
	AuthorName     string    `xml:"author>name"`
	AuthorUserName string    `xml:"author>username"`
	IP             string    `xml:"ipAddress"`
	Tid            uid       `xml:"thread"`
	Pid            uid       `xml:"parent"`
	IsSpam         bool      `xml:"isSpam"`
	Deleted        bool      `xml:"isDeleted"`
}

type uid struct {
	Val string `xml:"id,attr"`
}

// Import from disqus and save to store
func (d *Disqus) Import(r io.Reader, siteID string) (size int, err error) {
	return importComments(d.DataStore, r, siteID, d.convert)
}

// convert reads disqus export (xml) from r and passes each comment to emit
func (d *Disqus) convert(r io.Reader, siteID string, emit func(store.Comment) error) error {
	postsMap := map[string]string{} // tid:url
	decoder := xml.NewDecoder(r)

	stats := struct {
		inpThreads, inpComments          int
		commentsCount, spamComments      int
		failedThreads, failedPosts       int
		deletedComments, skippedComments int
	}{}

	rootSeen := false
	for {
		t, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("can't read disqus export: %w", err)
		}
		if se, ok := t.(xml.StartElement); ok && !rootSeen {
			if se.Name.Local != "disqus" {
				return fmt.Errorf("not a disqus export, the root element is <%s>", se.Name.Local)
			}
			rootSeen = true
			continue
		}

		if se, ok := t.(xml.StartElement); ok {
			if se.Name.Local == "thread" {
				stats.inpThreads++
				thread := disqusThread{}
				if err = decoder.DecodeElement(&thread, &se); err != nil {
					log.Printf("[WARN] can't decode disqus thread, %s", err)
					stats.failedThreads++
					continue
				}
				if thread.Deleted {
					continue
				}
				postsMap[thread.UID] = thread.Link
				continue
			}

			if se.Name.Local == "post" {
				stats.inpComments++
				comment := disqusComment{}
				if err = decoder.DecodeElement(&comment, &se); err != nil {
					log.Printf("[WARN] can't decode disqus comment, %s", err)
					stats.failedPosts++
					continue
				}
				if comment.Deleted {
					stats.deletedComments++
					continue
				}
				if comment.IsSpam {
					stats.spamComments++
					continue
				}

				url, ok := postsMap[comment.Tid.Val]
				if !ok {
					stats.skippedComments++
					continue
				}

				c := store.Comment{
					ID:      comment.UID,
					Locator: store.Locator{URL: url, SiteID: siteID},
					User: store.User{
						ID:   "disqus_" + store.EncodeID(comment.AuthorUserName),
						Name: comment.AuthorName,
						IP:   comment.IP,
					},
					Text:      d.cleanText(comment.Message),
					Timestamp: comment.CreatedAt,
					ParentID:  comment.Pid.Val,
					Imported:  true,
				}
				if comment.AuthorUserName == "" { // empty comment.AuthorUserName from disqus
					c.User.ID = "disqus_" + store.EncodeID(c.User.Name)
				}
				if c.ID == "" { // no comment.UID
					c.ID = comment.ID
				}
				if err := emit(c); err != nil {
					return err
				}
				stats.commentsCount++
				if stats.commentsCount%1000 == 0 {
					log.Printf("[DEBUG] processed %d comments", stats.commentsCount)
				}
			}
		}
	}
	if !rootSeen {
		return fmt.Errorf("not a disqus export, no <disqus> element found")
	}
	if stats.commentsCount == 0 && (stats.failedPosts > 0 || stats.failedThreads > 0) {
		return fmt.Errorf("no comments could be read from the disqus export, %d threads and %d posts failed to decode",
			stats.failedThreads, stats.failedPosts)
	}
	log.Printf("[INFO] converted %d posts, %+v", len(postsMap), stats)
	return nil
}

func (*Disqus) cleanText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", "")
	text = strings.ReplaceAll(text, "\t", "")
	return text
}
