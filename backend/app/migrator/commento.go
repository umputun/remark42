package migrator

import (
	"encoding/json"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/umputun/remark42/backend/app/store"

	log "github.com/go-pkgz/lgr"
)

// Commento implements Importer from commento export json
type Commento struct {
	DataStore Store
}

// Credit: https://gitlab.com/commento/commento/-/blob/master/api/domain_import_commento.go#L11-L15
type commentoExport struct {
	Version    int                 `json:"version"`
	Comments   []commentoComment   `json:"comments"`
	Commenters []commentoCommenter `json:"commenters"`
}

// Credit: https://gitlab.com/commento/commento/-/blob/master/api/comment.go#L7-L20
type commentoComment struct {
	CommentHex   string    `json:"commentHex"`
	Domain       string    `json:"domain,omitempty"`
	Path         string    `json:"url,omitempty"`
	CommenterHex string    `json:"commenterHex"`
	Markdown     string    `json:"markdown"`
	HTML         string    `json:"html"`
	ParentHex    string    `json:"parentHex"`
	Score        int       `json:"score"`
	State        string    `json:"state,omitempty"`
	CreationDate time.Time `json:"creationDate"`
	Direction    int       `json:"direction"`
	Deleted      bool      `json:"deleted"`
}

// Credit: https://gitlab.com/commento/commento/-/blob/master/api/commenter.go#L7-L16
type commentoCommenter struct {
	CommenterHex string    `json:"commenterHex,omitempty"`
	Email        string    `json:"email,omitempty"`
	Name         string    `json:"name"`
	Link         string    `json:"link"`
	Photo        string    `json:"photo"`
	Provider     string    `json:"provider,omitempty"`
	JoinDate     time.Time `json:"joinDate,omitempty"`
	IsModerator  bool      `json:"isModerator"`
}

// Import comments from Commento and save to store
func (d *Commento) Import(r io.Reader, siteID string) (size int, err error) {
	if e := d.DataStore.DeleteAll(siteID); e != nil {
		return 0, e
	}

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
		err = errors.Errorf("failed to save %d comments", failed)
		if passed == 0 {
			err = errors.New("import failed")
		}
	}

	log.Printf("[DEBUG] imported %d comments to site %s", passed, siteID)

	return passed, err
}

func (d *Commento) convert(r io.Reader, siteID string) (ch chan store.Comment) {
	commentsCh := make(chan store.Comment)

	decoder := json.NewDecoder(r)

	go func() {

		var exportedData commentoExport
		err := decoder.Decode(&exportedData)
		if err != nil {
			log.Printf("[WARN] can't decode commento export json, %s", err.Error())
		}

		usersMap := map[string]store.User{}
		for _, commenter := range exportedData.Commenters {
			usersMap[commenter.CommenterHex] = store.User{
				Name:    commenter.Name,
				ID:      "commento_" + store.EncodeID(commenter.CommenterHex),
				Picture: commenter.Photo,
			}
		}

		for _, comment := range exportedData.Comments {
			u, ok := usersMap[comment.CommenterHex]
			if !ok {
				continue
			}

			if comment.Deleted {
				continue
			}

			c := store.Comment{
				ID: comment.CommentHex,
				Locator: store.Locator{
					URL:    comment.Path,
					SiteID: siteID,
				},
				User:      u,
				Text:      comment.Markdown,
				Timestamp: comment.CreationDate,
				ParentID:  comment.ParentHex,
				Imported:  true,
			}

			commentsCh <- c
		}

		close(commentsCh)
	}()

	return commentsCh
}
