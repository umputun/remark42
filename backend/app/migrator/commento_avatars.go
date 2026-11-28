package migrator

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/remark42/backend/app/safehttp"
)

const (
	commentoPhotoPause        = 100 * time.Millisecond // between photo requests
	commentoPhotoTimeout      = 10 * time.Second       // for one photo, its redirects and its body included
	commentoPhotoMaxRedirects = 3
	commentoPhotoMaxSize      = 2 << 20 // ample for an avatar, which the avatar proxy resizes anyway
	commentoPhotoMaxFailures  = 5       // photos in a row that got no answer, then the rest go without
)

// commentoPhoto is the photo address of a commento commenter, with the remark42 id of that user
type commentoPhoto struct {
	userID string
	url    string
}

// commentoAvatars fetches the photos of commento commenters and stores them with an AvatarSaver.
// The addresses come from the uploaded export and point to any host, so they are requested through
// safehttp, which refuses private and reserved addresses when connecting and at every redirect.
type commentoAvatars struct {
	saver     AvatarSaver
	transport http.RoundTripper // safehttp.Transport() when nil
	timeout   time.Duration
	pause     time.Duration
	sleep     func(time.Duration)
}

// errPhotoUnanswered is a photo request that got no answer: a network failure, a timeout, a server
// error or throttling. Several of them in a row suggest the server can't reach the photo hosts at all.
var errPhotoUnanswered = errors.New("no answer")

func newCommentoAvatars(saver AvatarSaver) *commentoAvatars {
	return &commentoAvatars{saver: saver, timeout: commentoPhotoTimeout, pause: commentoPhotoPause, sleep: time.Sleep}
}

// fetch stores the given photos and returns their remark42 avatar URLs by user id. A user whose photo
// could not be fetched, or is not an image, is left out of the result.
func (a *commentoAvatars) fetch(photos []commentoPhoto) map[string]string {
	transport := a.transport
	if transport == nil {
		transport = safehttp.Transport()
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   a.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("%w: to a %q address", errRedirectRefused, req.URL.Scheme)
			}
			if len(via) > commentoPhotoMaxRedirects {
				return fmt.Errorf("%w: more than %d redirects", errRedirectRefused, commentoPhotoMaxRedirects)
			}
			req.Header.Del("Referer") // the address redirected from may carry a token of a signed link
			return nil
		},
	}
	defer client.CloseIdleConnections()

	res := map[string]string{}
	failures := 0
	for i, photo := range photos {
		if failures >= commentoPhotoMaxFailures {
			log.Printf("[WARN] commento photos unavailable, %d stored, %d users imported without avatars", len(res), len(photos)-i)
			break
		}
		if i > 0 {
			a.sleep(a.pause)
		}
		avatarURL, err := a.put(client, photo)
		if err != nil {
			if errors.Is(err, errPhotoUnanswered) {
				failures++
			} else {
				failures = 0
			}
			log.Printf("[DEBUG] no avatar for %s from %s, %v", photo.userID, photoHost(photo.url), err)
			continue
		}
		failures = 0
		res[photo.userID] = avatarURL
	}
	log.Printf("[INFO] stored commento photos of %d of %d users", len(res), len(photos))
	return res
}

// put requests a photo and hands it to the avatar saver, which refuses what is not an image
func (a *commentoAvatars) put(client *http.Client, photo commentoPhoto) (string, error) {
	resp, err := client.Get(photo.url) //nolint:noctx // the import runs without a request context
	if err != nil {
		if urlErr, ok := errors.AsType[*url.Error](err); ok {
			err = urlErr.Err // the error of a request quotes its url, which may carry a token of a signed link
		}
		if photoUnanswered(err) {
			return "", fmt.Errorf("%w: %w", errPhotoUnanswered, err)
		}
		return "", err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to do about a failed close of a response read to its limit
	switch {
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		return "", fmt.Errorf("%w: status %s", errPhotoUnanswered, resp.Status)
	case resp.StatusCode != http.StatusOK:
		return "", fmt.Errorf("status %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, commentoPhotoMaxSize+1))
	switch {
	case err != nil:
		return "", fmt.Errorf("%w: %w", errPhotoUnanswered, err)
	case len(body) > commentoPhotoMaxSize:
		return "", fmt.Errorf("photo larger than %d bytes", commentoPhotoMaxSize)
	}
	return a.saver.PutContent(photo.userID, bytes.NewReader(body))
}

// photoUnanswered tells a request that got no answer from one refused for its address or its redirects,
// or for a host name that does not exist, which are answers about that photo alone
func photoUnanswered(err error) bool {
	if errors.Is(err, safehttp.ErrPrivateAddress) || errors.Is(err, errRedirectRefused) {
		return false
	}
	if dnsErr, ok := errors.AsType[*net.DNSError](err); ok && dnsErr.IsNotFound {
		return false
	}
	return true
}

// photoHost returns the host of a photo address for the log, where the rest of it has no place
func photoHost(photo string) string {
	if u, err := url.Parse(photo); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return "an address without a host"
}
