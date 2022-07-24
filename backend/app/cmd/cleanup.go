package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/remark42/backend/app/store"
)

// CleanupCommand set of flags and command for cleanup
type CleanupCommand struct {
	Dry      bool     `long:"dry" description:"dry mode, will not remove comments"`
	From     string   `long:"from" description:"from yyyymmdd"`
	To       string   `long:"to" description:"from yyyymmdd"`
	BadWords []string `short:"w" long:"bword" description:"bad word(s)"`
	BadUsers []string `short:"u" long:"buser" description:"bad user(s)"`
	SetTitle bool     `long:"title" description:"title mode, will not remove comments, but reset titles to page's title'"`

	SupportCmdOpts
	CommonOpts
}

var (
	defaultFrom = time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local)
	defaultTo   = time.Date(2999, 1, 1, 0, 0, 0, 0, time.Local)
)

// Execute runs cleanup with CleanupCommand parameters, entry point for "cleanup" command
// This command uses provided flags to detect and remove junk comments
func (cc *CleanupCommand) Execute(_ []string) error {
	log.Printf("[INFO] cleanup for site %s", cc.Site)

	posts, err := cc.postsInRange(cc.From, cc.To)
	if err != nil {
		return fmt.Errorf("can't get posts: %w", err)
	}
	log.Printf("[DEBUG] got %d posts", len(posts))

	totalComments, spamComments := 0, 0
	for _, post := range posts {
		comments, e := cc.listComments(post.URL)
		if e != nil {
			continue
		}
		totalComments += len(comments)

		if cc.SetTitle {
			cc.procTitles(comments)
		} else {
			spamComments += cc.procSpam(comments)
		}
	}

	msg := fmt.Sprintf("comments=%d, spam=%d", totalComments, spamComments)
	if cc.SetTitle {
		msg = fmt.Sprintf("comments=%d", totalComments)
	}

	log.Printf("[INFO] completed, %s", msg)
	return err
}

func (cc *CleanupCommand) procSpam(comments []store.Comment) int {
	spamComments := 0
	for _, comment := range comments {
		spam, score := cc.isSpam(comment)
		if spam {
			spamComments++
			if !cc.Dry {
				if err := cc.deleteComment(comment); err != nil {
					log.Printf("[WARN] can't remove comment, %v", err)
				}
			}
			comment.Text = strings.Replace(comment.Text, "\n", " ", -1)
			log.Printf("[SPAM] %+v [%.0f%%]", comment, score)
		}
	}
	return spamComments
}

func (cc *CleanupCommand) procTitles(comments []store.Comment) {
	for _, comment := range comments {
		if !cc.Dry {
			if err := cc.setTitle(comment); err != nil {
				log.Printf("[WARN] can't set title for comment, %v", err)
			}
		}
	}
}

// get list of posts in from/to represented as yyyymmdd. this is [from-to] inclusive
func (cc *CleanupCommand) postsInRange(fromS, toS string) ([]store.PostInfo, error) {
	posts, err := cc.listPosts()
	if err != nil {
		return nil, fmt.Errorf("can't list posts for %s: %w", cc.Site, err)
	}

	from, to := defaultFrom, defaultTo

	if fromS != "" {
		from, err = time.ParseInLocation("20060102", fromS, time.Local)
		if err != nil {
			return nil, fmt.Errorf("can't parse --from: %w", err)
		}
	}

	if toS != "" {
		to, err = time.ParseInLocation("20060102", toS, time.Local)
		if err != nil {
			return nil, fmt.Errorf("can't parse --to: %w", err)
		}
	}

	var filteredList []store.PostInfo
	for _, postInfo := range posts {
		if postInfo.FirstTS.After(from) && postInfo.LastTS.Before(to.AddDate(0, 0, 1)) {
			filteredList = append(filteredList, postInfo)
		}
	}
	return filteredList, nil
}

// get all posts via GET /list?site=siteID&limit=50&skip=10
func (cc *CleanupCommand) listPosts() ([]store.PostInfo, error) {
	listURL := fmt.Sprintf("%s/api/v1/list?site=%s&limit=10000", cc.RemarkURL, cc.Site)
	client := http.Client{Timeout: 30 * time.Second}
	defer client.CloseIdleConnections()
	r, err := client.Get(listURL)
	if err != nil {
		return nil, fmt.Errorf("get request failed for list of posts, site %s: %w", cc.Site, err)
	}
	defer func() { _ = r.Body.Close() }()

	if r.StatusCode != 200 {
		return nil, fmt.Errorf("request %s failed with status %d", listURL, r.StatusCode)
	}

	list := []store.PostInfo{}
	if err = json.NewDecoder(r.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("can't decode list of posts for site %s: %w", cc.Site, err)
	}
	return list, nil
}

// get all comments for post url via /find?site=siteID&url=post-url&format=[tree|plain]
func (cc *CleanupCommand) listComments(postURL string) ([]store.Comment, error) {
	commentsURL := fmt.Sprintf("%s/api/v1/find?site=%s&url=%s&format=plain", cc.RemarkURL, cc.Site, postURL)

	var r *http.Response
	var err error

	// handle 429 error from limiter
	client := http.Client{Timeout: 30 * time.Second}
	defer client.CloseIdleConnections()
	for {
		r, err = client.Get(commentsURL)
		if err != nil {
			return nil, fmt.Errorf("get request failed for comments, %s: %w", postURL, err)
		}
		if r.StatusCode == http.StatusTooManyRequests {
			_ = r.Body.Close()
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}

	defer func() { _ = r.Body.Close() }()

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request %s failed with status %d", commentsURL, r.StatusCode)
	}

	commentsWithInfo := struct {
		Comments []store.Comment `json:"comments"`
		Info     store.PostInfo  `json:"info,omitempty"`
	}{}

	if err = json.NewDecoder(r.Body).Decode(&commentsWithInfo); err != nil {
		return nil, fmt.Errorf("can't decode list of comments for %s: %w", postURL, err)
	}
	return commentsWithInfo.Comments, nil
}

// deleteComment with DELETE /admin/comment/{id}?site=siteID&url=post-url
func (cc *CleanupCommand) deleteComment(c store.Comment) error { //nolint:dupl // not worth combining
	deleteURL := fmt.Sprintf("%s/api/v1/admin/comment/%s?site=%s&url=%s&format=plain", cc.RemarkURL, c.ID, cc.Site, c.Locator.URL)
	req, err := http.NewRequest("DELETE", deleteURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to make delete request for comment %s, %s: %w", c.ID, c.Locator.URL, err)
	}
	req.SetBasicAuth("admin", cc.AdminPasswd)

	client := http.Client{}
	defer client.CloseIdleConnections()
	r, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed for comment %s, %s: %w", c.ID, c.Locator.URL, err)
	}
	defer func() { _ = r.Body.Close() }()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("delete request failed with status %s", r.Status)
	}
	return nil
}

// setTitle with PUT /admin/title/{id}?site=siteID&url=post-url
func (cc *CleanupCommand) setTitle(c store.Comment) error { //nolint:dupl // not worth combining
	titleURL := fmt.Sprintf("%s/api/v1/admin/title/%s?site=%s&url=%s&format=plain", cc.RemarkURL, c.ID, cc.Site, c.Locator.URL)
	req, err := http.NewRequest("PUT", titleURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to make title request for comment %s, %s: %w", c.ID, c.Locator.URL, err)
	}
	req.SetBasicAuth("admin", cc.AdminPasswd)

	client := http.Client{}
	defer client.CloseIdleConnections()
	r, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("title request failed for comment %s, %s: %w", c.ID, c.Locator.URL, err)
	}
	defer func() { _ = r.Body.Close() }()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("title request failed with status %s", r.Status)
	}
	return nil
}

// isSpam calculates spam's probability as a score
func (cc *CleanupCommand) isSpam(comment store.Comment) (isSpam bool, spamScore float64) {
	badWord := func(txt string) float64 {
		res := 0.0
		for _, w := range cc.BadWords {
			if strings.Contains(txt, w) {
				res += 0.25
			}
			if res > 1 {
				return 1
			}
		}
		return res
	}

	hasBadUser := func(txt string) bool {
		for _, w := range cc.BadUsers {
			if strings.Contains(txt, w) {
				return true
			}
		}
		return false
	}

	score := 0.0

	// don't mark deleted as spam
	if comment.Deleted {
		return false, 0
	}

	score += 50 * badWord(comment.Text) // up to 50, 4 bad words will reach max

	if hasBadUser(comment.User.ID) { // predefined list of bad user substrings
		score += 10
	}

	if comment.Score == 0 { // most of spam comments with 0 score
		score += 20
	}

	// any link inside
	if strings.Contains(comment.Text, "http:") || strings.Contains(comment.Text, "https:") {
		score += 10
	}

	// 5 or more links
	if strings.Count(comment.Text, "href") >= 5 {
		score += 10
	}

	score = math.Max(score, 0)
	score = math.Min(score, 100)

	return score > 50, score
}
