package migrator

import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/umputun/remark/app/store"
)

// Disqus implements Importer from disqus xml
type Disqus struct {
	DataStore store.Interface
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
	AuthorNmae  string    `xml:"author>name"`
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
func (d Disqus) Import(r io.Reader) (err error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return errors.Wrap(err, "failed to read data")
	}
	dxml := disqusXML{}
	if err = xml.Unmarshal(data, &dxml); err != nil {
		return errors.Wrap(err, "can't unmarshal disqus xml")
	}

	failed := 0
	for _, c := range d.convert(dxml) {
		if _, err = d.DataStore.Create(c); err != nil {
			failed++
		}
	}

	if failed > 0 {
		return errors.Errorf("failed to save %d comments", failed)
	}
	return nil
}

func (d Disqus) convert(dxml disqusXML) (comments []store.Comment) {
	postsMap := map[string]string{} // tid:url
	log.Printf("[DEBUG] convert %d posts, %d comments", len(dxml.Threads), len(dxml.Comments))
	for _, thread := range dxml.Threads {
		postsMap[thread.UID] = thread.Link
	}
	for _, comment := range dxml.Comments {
		c := store.Comment{
			ID:        comment.ID,
			Locator:   store.Locator{URL: postsMap[comment.Tid.Val]},
			User:      store.User{ID: comment.AuthorUserName, Name: comment.AuthorName, IP: comment.IP},
			Text:      comment.Message,
			Timestamp: comment.CreatedAt,
			ParentID:  comment.Pid.Val,
		}
		comments = append(comments, c)
	}
	return comments
}
