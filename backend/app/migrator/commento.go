package migrator

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

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
	JoinDate     time.Time `json:"joinDate"`
	IsModerator  bool      `json:"isModerator"`
}

// Import comments from Commento and save to store
func (d *Commento) Import(r io.Reader, siteID string) (size int, err error) {
	return importComments(d.DataStore, r, siteID, d.convert)
}

// convert reads Commento export (json) from r and passes each comment to emit
func (d *Commento) convert(r io.Reader, siteID string, emit func(store.Comment) error) error {
	// any JSON object decodes into commentoExport, so check for the lists a Commento export always has
	var raw struct {
		Comments   json.RawMessage `json:"comments"`
		Commenters json.RawMessage `json:"commenters"`
	}
	dec := json.NewDecoder(r)
	if err := dec.Decode(&raw); err != nil {
		return fmt.Errorf("can't read commento export: %w", err)
	}
	// the export is a single object, anything after it means the file is not what it claims to be
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("can't read commento export: unexpected data after the export object")
	}
	if raw.Comments == nil || raw.Commenters == nil {
		return fmt.Errorf("not a commento export, no comments and commenters lists found")
	}
	var exportedData commentoExport
	if err := json.Unmarshal(raw.Comments, &exportedData.Comments); err != nil {
		return fmt.Errorf("can't read commento comments: %w", err)
	}
	if err := json.Unmarshal(raw.Commenters, &exportedData.Commenters); err != nil {
		return fmt.Errorf("can't read commento commenters: %w", err)
	}

	usersMap := map[string]store.User{}
	for _, commenter := range exportedData.Commenters {
		usersMap[commenter.CommenterHex] = store.User{
			Name:    commenter.Name,
			ID:      "commento_" + store.EncodeID(commenter.CommenterHex),
			Picture: commenter.Photo,
		}
	}

	usersMap["anonymous"] = store.User{
		Name: "Anonymous",
		ID:   "commento_" + store.EncodeID("anonymous"),
	}

	for _, comment := range exportedData.Comments {
		u, ok := usersMap[comment.CommenterHex]
		if !ok {
			continue
		}

		if comment.Deleted {
			continue
		}

		parentID := comment.ParentHex
		// comments with ParentHex == "root" are top-level comments
		if parentID == "root" {
			parentID = ""
		}

		commentURL, err := url.JoinPath("https://", comment.Domain, comment.Path)
		if err != nil {
			log.Printf("[WARN] can't construct comment URL in commento import, %s", err)
		}

		c := store.Comment{
			ID: comment.CommentHex,
			Locator: store.Locator{
				URL:    commentURL,
				SiteID: siteID,
			},
			User:      u,
			Text:      comment.Markdown,
			Timestamp: comment.CreationDate,
			ParentID:  parentID,
			Imported:  true,
		}

		if err := emit(c); err != nil {
			return err
		}
	}

	return nil
}
