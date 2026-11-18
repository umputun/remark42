package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
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
	resetEnv("SECRET", "ADMIN_PASSWD")

	ctx, cancel := context.WithTimeout(context.Background(), cc.Timeout)
	defer cancel()

	posts, err := cc.postsInRange(ctx, cc.From, cc.To)
	if err != nil {
		return fmt.Errorf("can't get posts: %w", err)
	}
	log.Printf("[DEBUG] got %d posts", len(posts))

	totalComments, spamComments, failedPosts := 0, 0, 0
	for _, post := range posts {
		comments, e := cc.listComments(ctx, post.URL)
		if e != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("can't get comments for %s: %w", post.URL, e)
			}
			log.Printf("[WARN] can't get comments for %s, %v", post.URL, e)
			failedPosts++
			continue
		}
		totalComments += len(comments)

		if cc.SetTitle {
			cc.procTitles(ctx, comments)
		} else {
			spamComments += cc.procSpam(ctx, comments)
		}
	}

	msg := fmt.Sprintf("comments=%d, spam=%d", totalComments, spamComments)
	if cc.SetTitle {
		msg = fmt.Sprintf("comments=%d", totalComments)
	}

	if err = ctx.Err(); err != nil { // the timeout can also cut short the deletes or title updates of the last post
		return fmt.Errorf("cleanup interrupted, %s: %w", msg, err)
	}
	log.Printf("[INFO] completed, %s", msg)
	if failedPosts > 0 {
		return fmt.Errorf("can't get comments for %d of %d posts", failedPosts, len(posts))
	}
	return nil
}

func (cc *CleanupCommand) procSpam(ctx context.Context, comments []store.Comment) int {
	spamComments := 0
	for _, comment := range comments {
		spam, score := cc.isSpam(comment)
		if spam {
			spamComments++
			if !cc.Dry {
				if err := cc.deleteComment(ctx, comment); err != nil {
					log.Printf("[WARN] can't remove comment, %v", err)
				}
			}
			comment.Text = strings.ReplaceAll(comment.Text, "\n", " ")
			log.Printf("[SPAM] %+v [%.0f%%]", comment, score)
		}
	}
	return spamComments
}

func (cc *CleanupCommand) procTitles(ctx context.Context, comments []store.Comment) {
	for _, comment := range comments {
		if !cc.Dry {
			if err := cc.setTitle(ctx, comment); err != nil {
				log.Printf("[WARN] can't set title for comment, %v", err)
			}
		}
	}
}

// get list of posts in from/to represented as yyyymmdd. this is [from-to] inclusive
func (cc *CleanupCommand) postsInRange(ctx context.Context, fromS, toS string) ([]store.PostInfo, error) {
	posts, err := cc.listPosts(ctx)
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
func (cc *CleanupCommand) listPosts(ctx context.Context) ([]store.PostInfo, error) {
	listURL := fmt.Sprintf("%s/api/v1/list?site=%s&limit=10000", cc.RemarkURL, url.QueryEscape(cc.Site))
	list := []store.PostInfo{}
	if err := getJSON(ctx, listURL, &list); err != nil {
		return nil, fmt.Errorf("can't get list of posts for site %s: %w", cc.Site, err)
	}
	return list, nil
}

// get all comments for post url via /find?site=siteID&url=post-url&format=[tree|plain]
func (cc *CleanupCommand) listComments(ctx context.Context, postURL string) ([]store.Comment, error) {
	commentsURL := fmt.Sprintf("%s/api/v1/find?site=%s&url=%s&format=plain", cc.RemarkURL, url.QueryEscape(cc.Site), url.QueryEscape(postURL))
	commentsWithInfo := struct {
		Comments []store.Comment `json:"comments"`
		Info     store.PostInfo  `json:"info"`
	}{}
	if err := getJSON(ctx, commentsURL, &commentsWithInfo); err != nil {
		return nil, fmt.Errorf("can't get comments for %s: %w", postURL, err)
	}
	return commentsWithInfo.Comments, nil
}

// getJSON decodes the response to a GET of reqURL into v. A 429 from the rate limiter is retried
// until ctx is done, any other status but 200 is an error
func getJSON(ctx context.Context, reqURL string, v any) error {
	client := http.Client{Timeout: 30 * time.Second}
	defer client.CloseIdleConnections()
	var resp *http.Response
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
		if err != nil {
			return fmt.Errorf("can't make request for %s: %w", reqURL, err)
		}
		if resp, err = client.Do(req); err != nil { //nolint:gosec // reqURL is built from operator-supplied CLI flags, not user input
			return fmt.Errorf("request failed for %s: %w", reqURL, err)
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			break
		}
		_ = resp.Body.Close()
		select {
		case <-ctx.Done():
			return fmt.Errorf("request %s rate limited: %w", reqURL, ctx.Err())
		case <-time.After(500 * time.Millisecond):
		}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request %s failed with status %d", reqURL, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("can't decode response of %s: %w", reqURL, err)
	}
	return nil
}

// deleteComment with DELETE /admin/comment/{id}?site=siteID&url=post-url
func (cc *CleanupCommand) deleteComment(ctx context.Context, c store.Comment) error {
	return cc.commentRequest(ctx, http.MethodDelete, "comment", c)
}

// setTitle with PUT /admin/title/{id}?site=siteID&url=post-url
func (cc *CleanupCommand) setTitle(ctx context.Context, c store.Comment) error {
	return cc.commentRequest(ctx, http.MethodPut, "title", c)
}

// commentRequest sends method to /admin/{route}/{id}?site=siteID&url=post-url
func (cc *CleanupCommand) commentRequest(ctx context.Context, method, route string, c store.Comment) error {
	endpoint := fmt.Sprintf("%s/api/v1/admin/%s/%s", cc.RemarkURL, route, url.PathEscape(c.ID))
	query := url.Values{"site": {cc.Site}, "url": {c.Locator.URL}, "format": {"plain"}}
	return cc.adminRequest(ctx, method, endpoint, query, http.NoBody, nil)
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
