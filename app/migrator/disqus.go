package migrator

import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"log"
	"time"

	"sync"

	"strings"

	"github.com/pkg/errors"
	"github.com/umputun/remark/app/store"
)

// Disqus implements Importer from disqus xml
type Disqus struct {
	DataStore store.Interface

	ch   chan store.Comment
	once sync.Once
}

type disqusXML struct {
	Threads  []disqusThread  `xml:"thread"`
	Comments []disqusComment `xml:"post"`
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

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return errors.Wrap(err, "failed to read data")
	}
	dxml := disqusXML{}
	if err = xml.Unmarshal(data, &dxml); err != nil {
		return errors.Wrap(err, "can't unmarshal disqus xml")
	}

	commentsCh := d.convert(r, siteID)
	failed := 0
	for c := range commentsCh {
		if _, err = d.DataStore.Create(c); err != nil {
			failed++
		}
	}

	if failed > 0 {
		return errors.Errorf("failed to save %d comments", failed)
	}

	return nil
}

func (d *Disqus) convert(r io.Reader, siteID string) (ch chan store.Comment) {

	postsMap := map[string]string{} // tid:url
	decoder := xml.NewDecoder(r)
	commentsCh := make(chan store.Comment)

	go func() {
		commentsCount := 0
		for {
			t, err := decoder.Token()
			if t == nil || err != nil {
				break
			}

			switch se := t.(type) {
			case xml.StartElement:
				if se.Name.Local == "thread" {
					thread := disqusThread{}
					if err := decoder.DecodeElement(&thread, &se); err == nil {
						postsMap[thread.UID] = thread.Link
					}
				}
				if se.Name.Local == "post" {
					comment := disqusComment{}
					if err := decoder.DecodeElement(&comment, &se); err != nil {
						continue
					}
					c := store.Comment{
						ID:        comment.ID,
						Locator:   store.Locator{URL: postsMap[comment.Tid.Val], SiteID: siteID},
						User:      store.User{ID: comment.AuthorUserName, Name: comment.AuthorName, IP: comment.IP},
						Text:      d.cleanText(comment.Message),
						Timestamp: comment.CreatedAt,
						ParentID:  comment.Pid.Val,
					}
					commentsCh <- c
					commentsCount++
				}
			}
		}
		close(commentsCh)
		log.Printf("[DEBUG] converted %d posts, %d comments", len(postsMap), commentsCount)
	}()

	return commentsCh
}

func (d *Disqus) cleanText(text string) string {
	text = strings.Replace(text, "\n", "", -1)
	text = strings.Replace(text, "\t", "", -1)
	return text
}
