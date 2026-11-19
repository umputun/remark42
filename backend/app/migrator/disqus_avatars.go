package migrator

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/remark42/backend/app/store"
)

// AvatarSaver stores an avatar image for a user and returns the URL remark42 serves it from.
// The avatar proxy of go-pkgz/auth implements it, and refuses content that is not an image.
type AvatarSaver interface {
	PutContent(userID string, content io.Reader) (avatarURL string, err error)
}

const (
	disqusAvatarsURL         = "https://disqus.com/api/users/avatars/" // public endpoint, documented at https://disqus.com/api/docs/images/
	disqusAvatarPause        = 100 * time.Millisecond                  // between requests to disqus
	disqusAvatarRetries      = 3                                       // for a request disqus throttled or failed to answer
	disqusAvatarMaxRetryWait = time.Minute                             // longest Retry-After honored
	disqusAvatarMaxFailures  = 5                                       // users in a row disqus could not answer for, then the rest go without
	disqusAvatarMaxSize      = 10 << 20
)

// disqusAvatars fetches avatars of disqus users and stores them with an AvatarSaver
type disqusAvatars struct {
	saver   AvatarSaver
	client  *http.Client
	baseURL string
	pause   time.Duration
	sleep   func(time.Duration)
}

// errDisqusUnavailable is a failure to get an answer from disqus, as opposed to an answer that the
// user has no avatar
var errDisqusUnavailable = errors.New("disqus avatar service unavailable")

// errRedirectRefused is a redirect away from disqus, or a chain of them, which is not followed
var errRedirectRefused = errors.New("redirect refused")

func newDisqusAvatars(saver AvatarSaver) *disqusAvatars {
	return &disqusAvatars{
		saver:   saver,
		baseURL: disqusAvatarsURL,
		pause:   disqusAvatarPause,
		sleep:   time.Sleep,
		client: &http.Client{
			Timeout: 10 * time.Second,
			// the avatar endpoint redirects to an image on the disqus CDN, or to its generic "noavatar" image
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("%w: too many redirects", errRedirectRefused)
				}
				if req.URL.Host != via[0].URL.Host && !strings.HasSuffix(req.URL.Hostname(), ".disquscdn.com") {
					return fmt.Errorf("%w: unexpected host %s", errRedirectRefused, req.URL.Hostname())
				}
				return nil
			},
		},
	}
}

// fetch stores the avatars of the given disqus usernames and returns their remark42 avatar URLs.
// A user without an avatar, or whose avatar could not be stored, is left out of the result.
func (a *disqusAvatars) fetch(usernames []string) map[string]string {
	res := map[string]string{}
	failures, stored := 0, 0
	for i, username := range usernames {
		if failures >= disqusAvatarMaxFailures {
			log.Printf("[WARN] disqus avatars unavailable, %d stored, %d users imported without avatars",
				stored, len(usernames)-i)
			break
		}
		if i > 0 {
			a.sleep(a.pause)
		}
		avatarURL, err := a.put(username)
		switch {
		case errors.Is(err, errDisqusUnavailable):
			failures++
			log.Printf("[WARN] can't get disqus avatar of %s, %v", username, err)
			continue
		case err != nil:
			failures = 0
			log.Printf("[DEBUG] no disqus avatar for %s, %v", username, err)
			continue
		}
		failures = 0
		if avatarURL != "" {
			res[username] = avatarURL
			stored++
		}
		if (i+1)%100 == 0 {
			log.Printf("[INFO] processed disqus avatars of %d of %d users", i+1, len(usernames))
		}
	}
	log.Printf("[INFO] stored disqus avatars of %d of %d users", stored, len(usernames))
	return res
}

// put fetches the avatar of a disqus user and stores it, returning an empty URL when the user has none
func (a *disqusAvatars) put(username string) (string, error) {
	body, err := a.load(a.baseURL + url.PathEscape(username) + ".jpg")
	if err != nil || body == nil {
		return "", err
	}
	return a.saver.PutContent(disqusUserID(username), strings.NewReader(string(body)))
}

// load gets the avatar image, retrying what disqus throttled or failed to answer.
// It returns nil without an error when disqus answers with its generic avatar or a client error.
func (a *disqusAvatars) load(avatarURL string) ([]byte, error) {
	var lastErr error
	for attempt := range disqusAvatarRetries + 1 {
		if attempt > 0 {
			a.sleep(lastWait(lastErr, attempt))
		}
		resp, err := a.client.Get(avatarURL) //nolint:noctx // the import runs without a request context
		if errors.Is(err, errRedirectRefused) {
			return nil, err // the same on every attempt, and no sign disqus is down
		}
		if err != nil {
			lastErr = retryAfter{err: err}
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, disqusAvatarMaxSize+1))
		_ = resp.Body.Close()
		switch {
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			lastErr = retryAfter{err: fmt.Errorf("status %s", resp.Status), header: resp.Header.Get("Retry-After")}
			continue
		case resp.StatusCode != http.StatusOK:
			return nil, nil // disqus has no such user
		case err != nil:
			lastErr = retryAfter{err: err}
			continue
		case strings.Contains(resp.Request.URL.Path, "/noavatar"):
			return nil, nil // the generic avatar disqus gives users without their own
		case len(body) > disqusAvatarMaxSize:
			return nil, fmt.Errorf("avatar larger than %d bytes", disqusAvatarMaxSize)
		}
		return body, nil
	}
	return nil, fmt.Errorf("%w: %w", errDisqusUnavailable, lastErr)
}

// retryAfter is a failed attempt, with the Retry-After header disqus sent along, if any
type retryAfter struct {
	err    error
	header string
}

func (r retryAfter) Error() string { return r.err.Error() }
func (r retryAfter) Unwrap() error { return r.err }

// lastWait is how long to wait before the given attempt: what Retry-After asked for, up to a minute,
// or a second doubling with each attempt
func lastWait(err error, attempt int) time.Duration {
	var ra retryAfter
	if errors.As(err, &ra) && ra.header != "" {
		if secs, e := strconv.Atoi(ra.header); e == nil && secs >= 0 {
			return min(time.Duration(secs)*time.Second, disqusAvatarMaxRetryWait)
		}
		if t, e := http.ParseTime(ra.header); e == nil {
			return min(max(time.Until(t), 0), disqusAvatarMaxRetryWait)
		}
	}
	return time.Second << (attempt - 1)
}

// disqusUsernames returns the usernames of the authors of the comments an import of the disqus
// export in r keeps, in the order they first appear
func disqusUsernames(r io.Reader) ([]string, error) {
	decoder := xml.NewDecoder(r)
	threads := map[string]bool{} // ids of threads that are not deleted
	seen := map[string]bool{}
	var res []string
	for {
		t, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return res, nil
		}
		if err != nil {
			return nil, err
		}
		se, ok := t.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "thread":
			thread := disqusThread{}
			if decoder.DecodeElement(&thread, &se) == nil && !thread.Deleted {
				threads[thread.UID] = true
			}
		case "post":
			comment := disqusComment{}
			if decoder.DecodeElement(&comment, &se) != nil || comment.Deleted || comment.IsSpam || !threads[comment.Tid.Val] {
				continue
			}
			if comment.AuthorUserName != "" && !seen[comment.AuthorUserName] {
				seen[comment.AuthorUserName] = true
				res = append(res, comment.AuthorUserName)
			}
		}
	}
}

// disqusUserID is the remark42 user id of a disqus user
func disqusUserID(username string) string {
	return "disqus_" + store.EncodeID(username)
}
