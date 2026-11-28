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
	DataStore                  Store
	DisableFancyTextFormatting bool
	Avatars                    AvatarSaver // stores the photos of commenters; without it they are imported without avatars
}

// Credit: https://gitlab.com/commento/commento/-/blob/master/api/domain_import_commento.go#L11-L15
type commentoExport struct {
	Comments   []commentoComment   `json:"comments"`
	Commenters []commentoCommenter `json:"commenters"`
}

// commentoComment holds the fields the import reads.
// Credit: https://gitlab.com/commento/commento/-/blob/master/api/comment.go#L7-L20
type commentoComment struct {
	CommentHex   string    `json:"commentHex"`
	Domain       string    `json:"domain,omitempty"`
	Path         string    `json:"url,omitempty"`
	CommenterHex string    `json:"commenterHex"`
	Markdown     string    `json:"markdown"`
	ParentHex    string    `json:"parentHex"`
	State        string    `json:"state,omitempty"`
	CreationDate time.Time `json:"creationDate"`
	Deleted      bool      `json:"deleted"`
}

// shown reports whether commento shows the comment to readers: only approved comments are, the others wait
// for a moderator or were flagged as spam
func (c commentoComment) shown() bool {
	return c.State == "" || c.State == "approved"
}

// deleted reports whether the comment is deleted. Commento keeps a deleted comment as "[deleted]" by the
// anonymous commenter, and its export leaves out the deleted flag.
func (c commentoComment) deleted() bool {
	return c.Deleted || (c.Markdown == "[deleted]" && c.CommenterHex == "anonymous")
}

// commentoCommenter holds the fields the import reads.
// Credit: https://gitlab.com/commento/commento/-/blob/master/api/commenter.go#L7-L16
type commentoCommenter struct {
	CommenterHex string `json:"commenterHex,omitempty"`
	Name         string `json:"name"`
	Photo        string `json:"photo"`
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
		u := store.User{
			Name: commenter.Name,
			ID:   "commento_" + store.EncodeID(commenter.CommenterHex),
		}
		usersMap[commenter.CommenterHex] = u
	}
	d.setPictures(exportedData, usersMap)

	usersMap["anonymous"] = store.User{
		Name: "Anonymous",
		ID:   "commento_" + store.EncodeID("anonymous"),
	}

	formatter := store.NewCommentFormatter()
	for _, comment := range exportedData.Comments {
		u, ok := usersMap[comment.CommenterHex]
		if !ok {
			continue
		}

		if !comment.shown() {
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
			Text:      formatter.FormatText(comment.Markdown, d.DisableFancyTextFormatting),
			Timestamp: comment.CreationDate,
			ParentID:  parentID,
			Imported:  true,
		}
		// a deleted comment stays as a placeholder for its replies
		if comment.deleted() {
			c.SetDeleted(store.HardDelete)
		}

		if err := emit(c); err != nil {
			return err
		}
	}

	return nil
}

// setPictures stores the photos of the commenters whose comments the import keeps and sets the users'
// pictures to the stored avatars. The import reads the whole export before the site's comments are
// replaced, so they stay available while the photos are fetched. Without an avatar saver, and for a
// photo that can't be fetched or is not an image, the picture stays empty: the import never links
// to the host of a photo.
func (d *Commento) setPictures(export commentoExport, users map[string]store.User) {
	if d.Avatars == nil {
		return
	}
	kept := map[string]bool{} // hex ids of commenters with a comment the import keeps under their name
	for _, c := range export.Comments {
		if c.shown() && !c.deleted() {
			kept[c.CommenterHex] = true
		}
	}
	// a commenter listed twice takes the photo from the same row as the name, the last one
	photoURLs := map[string]string{} // by commenter hex id
	for _, commenter := range export.Commenters {
		photoURLs[commenter.CommenterHex] = commenter.Photo
	}
	var photos []commentoPhoto
	queued := map[string]bool{} // hex ids of commenters whose photo is fetched
	for _, commenter := range export.Commenters {
		hex := commenter.CommenterHex
		u, ok := users[hex]
		if !ok || !kept[hex] || queued[hex] || hex == "anonymous" { // anonymous comments get no picture
			continue
		}
		// commento writes "undefined" for a commenter without a photo
		pic, err := url.Parse(photoURLs[hex])
		if err != nil || (pic.Scheme != "http" && pic.Scheme != "https") || pic.Host == "" {
			continue
		}
		queued[hex] = true
		photos = append(photos, commentoPhoto{userID: u.ID, url: photoURLs[hex]})
	}
	stored := newCommentoAvatars(d.Avatars).fetch(photos)
	for hex := range queued {
		u := users[hex]
		u.Picture = stored[u.ID]
		users[hex] = u
	}
}
