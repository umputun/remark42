package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/token"
	log "github.com/go-pkgz/lgr"

	"github.com/umputun/remark42/backend/app/safehttp"
)

// customAvatarTimeout is the time a picture of a custom OAuth2 provider's user has to arrive in. The
// auth routes answer within five seconds, and the sign-in has the token and the user info to get too
const customAvatarTimeout = 3 * time.Second

// customAvatarSaver stores the picture of a user signing in through the custom OAuth2 provider. The
// address of the picture comes from the provider's user info, where it can be whatever a user of
// that provider has set, so it is requested through safehttp, which refuses private and reserved
// addresses, at every redirect too, and without the provider's access token, which the client handed
// to Put adds to its requests. A picture that can't be fetched or is not an image is replaced by an
// identicon and the user is signed in all the same.
//
// It is also where a user without an id is refused. The function mapping the user info to a user
// has no way to fail the sign-in, and storing the picture is the next step of it that has.
type customAvatarSaver struct {
	proxy     *avatar.Proxy
	transport http.RoundTripper // safehttp.Transport() when nil
}

// Put stores the picture of u and returns the address remark42 serves it from
func (a customAvatarSaver) Put(u token.User, _ *http.Client) (avatarURL string, err error) {
	if u.ID == "" {
		return "", errors.New("user info has no user id")
	}
	if u.Picture != "" {
		if avatarURL, err = a.fetch(u); err == nil {
			return avatarURL, nil
		}
		log.Printf("[WARN] can't store the picture of %s from %s, identicon is used: %v", u.ID, pictureHost(u.Picture), err)
		u.Picture = ""
	}
	return a.proxy.Put(u, nil) // no picture, so nothing is requested and no client is needed
}

// fetch requests the picture of u and hands it to the avatar proxy, which limits its size, checks
// that it is an image and stores it
func (a customAvatarSaver) fetch(u token.User) (avatarURL string, err error) {
	transport := a.transport
	if transport == nil {
		transport = safehttp.Transport()
	}
	client := http.Client{Timeout: customAvatarTimeout, Transport: transport}
	defer client.CloseIdleConnections()

	resp, err := client.Get(u.Picture) //nolint:gosec // the transport refuses private and reserved addresses
	if err != nil {
		// the error of a request quotes its url, which may carry a token of a signed link
		if urlErr, ok := errors.AsType[*url.Error](err); ok {
			err = urlErr.Err
		}
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to do about a failed close of a response read to its limit
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %s", resp.Status)
	}
	return a.proxy.PutContent(u.ID, resp.Body)
}

// pictureHost returns the host of a picture address for the log, where the rest of it has no place
func pictureHost(picture string) string {
	if u, err := url.Parse(picture); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return "an address without a host"
}
