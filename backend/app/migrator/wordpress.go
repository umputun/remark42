package migrator

import (
	"encoding/xml"
	"html"
	"io"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"

	"github.com/umputun/remark42/backend/app/store"
)

const wpTimeLayout = "2006-01-02 15:04:05"

// WordPress implements Importer from WP xml
type WordPress struct {
	DataStore Store
}

type wpItem struct {
	Link     string      `xml:"link"`
	Comments []wpComment `xml:"comment"`
}

type wpComment struct {
	ID          string `xml:"comment_id"`
	Author      string `xml:"comment_author"`
	AuthorEmail string `xml:"comment_author_email"`
	AuthorIP    string `xml:"comment_author_IP"`
	Date        wpTime `xml:"comment_date_gmt"`
	Content     string `xml:"comment_content"`
	Approved    string `xml:"comment_approved"`
	PID         string `xml:"comment_parent"`
}

type wpTime struct {
	time time.Time
}

// UnmarshalXML decoding xml with time in WP format
func (w *wpTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var v string
	if err := d.DecodeElement(&v, &start); err != nil {
		return err
	}
	t, err := time.Parse(wpTimeLayout, v)
	if err != nil {
		return err
	}
	w.time = t
	return err
}

// Convert satisfies formatter.CommentConverter
func (w *WordPress) Convert(text string) string {
	return html.UnescapeString(text) // sanitize remains on comment create
}

// Import comments from WP and save to store
func (w *WordPress) Import(r io.Reader, siteID string) (size int, err error) {

	if e := w.DataStore.DeleteAll(siteID); e != nil {
		return 0, e
	}

	commentsCh := w.convert(r, siteID)
	failed, passed := 0, 0
	for c := range commentsCh {
		if _, err = w.DataStore.Create(c); err != nil {
			failed++
			continue
		}
		passed++
	}

	if failed > 0 {
		err = errors.Errorf("failed to save %d comments", failed)
		if passed == 0 {
			err = errors.New("import failed")
		}
	}

	log.Printf("[DEBUG] imported %d comments to site %s", passed, siteID)

	return passed, err
}

func (w *WordPress) convert(r io.Reader, siteID string) chan store.Comment {

	decoder := xml.NewDecoder(r)
	commentsCh := make(chan store.Comment)

	stats := struct {
		inpItems, failedItems       int
		inpComments, failedComments int
		rejectedComments            int // not approved
	}{}

	commentFormatter := store.NewCommentFormatter(w)

	go func() {
		for {
			t, err := decoder.Token()
			if t == nil || err != nil {
				break
			}

			switch el := t.(type) {
			case xml.StartElement:
				if el.Name.Local == "item" {
					stats.inpItems++
					item := wpItem{}
					if err = decoder.DecodeElement(&item, &el); err != nil {
						log.Printf("[WARN] Can't decode item, %s", err)
						stats.failedItems++
						continue
					}
					if item.Comments != nil {
						for _, comment := range item.Comments {
							if comment.Approved != "1" {
								stats.rejectedComments++
								continue
							}

							if comment.PID == "0" {
								comment.PID = ""
							}

							c := store.Comment{
								ID:      comment.ID,
								Locator: store.Locator{URL: item.Link, SiteID: siteID},
								User: store.User{
									ID:   "wordpress_" + store.EncodeID(comment.Author),
									Name: comment.Author,
									IP:   comment.AuthorIP,
								},
								Text:      comment.Content,
								Timestamp: comment.Date.time,
								ParentID:  comment.PID,
								Imported:  true,
							}
							commentsCh <- commentFormatter.Format(c)
							stats.inpComments++
							if stats.inpComments%1000 == 0 {
								log.Printf("[DEBUG] processed %d comments", stats.inpComments)
							}
						}
					}
				}
			}
		}
		close(commentsCh)
		log.Printf("[INFO] converted %d comments, %+v", stats.inpComments-stats.failedComments, stats)
	}()
	return commentsCh
}
