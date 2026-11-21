//go:build e2e

package e2e

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Three configurations that cannot be settings on an instance shared with anything else: each
// changes the widget for every reader of that instance, and each broke in a way nothing else in
// this suite can reach. Their services are in compose-e2e-test.yml.

// adminEditAddress is the address whose email-derived id compose names in ADMIN_SHARED_ID, so
// signing in with it on that instance produces an admin.
//
// Email and not anonymous: remark42 hashes an anonymous id from the name and the client address
// together (server.go:1220, to tell apart two people picking the same name), so an id written
// down here would belong to nobody on any machine but the one it was read from. The email id is
// sha1 of the address alone
const adminEditAddress = "adminedit@example.com"

// TestComment_AdminEditHasNoDeadline covers #1986: the backend honored ADMIN_EDIT while the
// frontend went on counting an admin's edit window down and taking the button away at the end,
// so the setting did nothing where it is visible. The instance's ordinary window is short, so the
// wait is the same one TestComment_EditExpiresAfterTheDeadline pays, and both halves are asserted
// against it: no countdown at any point, and an edit that still lands afterwards
func TestComment_AdminEditHasNoDeadline(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	url := threadURLOn(t, adminEditURL)
	frame := openURL(t, page, url)
	signInEmail(t, page, frame, "admineditor", adminEditAddress)

	text := "admin edit " + runID
	postedAt := time.Now()
	postComment(t, frame, text)

	// the countdown is what #2001 left running. an admin has no deadline, so it should never
	// have been rendered at all. Asserted on the comment just posted and again on a loaded
	// thread, because #2205 showed only on the first: right after sign-in the widget did not
	// know the user was an admin
	assertNoCountdown := func(frame playwright.FrameLocator, msg string) {
		count, err := actions(frame, text).Locator(`[role="timer"]`).Count()
		require.NoError(t, err)
		assert.Zero(t, count, msg)
	}
	assertNoCountdown(frame, "an admin's new comment is counting down an edit window that does not apply to it")

	frame = reload(t, page)
	assertNoCountdown(frame, "an admin's comment is counting down an edit window that does not apply to it")

	// past the window every other user on this instance is held to. editWindow is EDIT_TIME on
	// both short-window instances in compose-e2e-test.yml
	waitPastEditWindow(postedAt)

	require.NoError(t, actions(frame, text).Locator(`button:has-text("Edit")`).Click())
	edited := "admin edited after the deadline " + runID
	submitForm(t, replyForm(t, frame), edited)
	waitVisible(t, comment(frame, edited))

	// stored and not only rendered: the backend has its own view of the deadline, and the
	// widget showing the new text says nothing about which of the two answered
	frame = reload(t, page)
	waitVisible(t, comment(frame, edited))
}

// TestAuth_HeaderJWTSurvivesReload covers #1877. With AUTH_SEND_JWT_HEADER the token arrives in a
// response header instead of a cookie, so the frontend holds it itself and a reload starts with
// nothing in hand. Signing out matters as much as signing in: a token kept somewhere the sign-out
// does not clear leaves a session that outlives the button
func TestAuth_HeaderJWTSurvivesReload(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openURL(t, page, threadURLOn(t, jwtHeaderURL))
	signInAnon(t, page, frame, anonName("headerjwt"))

	frame = reload(t, page)
	assertSignedIn(t, page, frame)

	pauseForAuthLimit()
	require.NoError(t, frame.Locator(`[title="Sign out"]`).Click())
	waitVisible(t, frame.Locator(".auth-button"))

	frame = reload(t, page)
	waitVisible(t, frame.Locator(".auth-button"))
	waitHidden(t, frame.Locator(`[title="Sign out"]`),
		"the header-borne session came back after signing out")
}

// TestAuth_HeaderSessionOutlivesItsToken is the half of header mode a reload cannot show. The
// widget keeps the token from every X-JWT response in memory and used to send it back on each
// request for as long as the page stayed open. The backend reads that header before the cookie,
// refuses a token that has expired there, and renews only a token that arrived in the cookie. So the
// first authorized request made after AUTH_TTL_JWT got a 401, the widget dropped the session, and a
// reader who had left the page open for five minutes was signed out by posting a comment.
//
// The instance's token lives for 15 seconds, see compose-e2e-test.yml, and the wait is counted from
// the token the widget was actually handed.
func TestAuth_HeaderSessionOutlivesItsToken(t *testing.T) {
	t.Parallel()

	page := newPage(t)

	// the newest token the instance handed over, and when it arrived
	var mu sync.Mutex
	var issued string
	var issuedAt time.Time
	page.OnResponse(func(r playwright.Response) {
		if tkn, err := r.HeaderValue("x-jwt"); err == nil && tkn != "" {
			mu.Lock()
			issued, issuedAt = tkn, time.Now()
			mu.Unlock()
		}
	})

	frame := openURL(t, page, threadURLOn(t, jwtHeaderURL))
	signInAnon(t, page, frame, anonName("headerexpiry"))

	mu.Lock()
	tkn, receivedAt := issued, issuedAt
	mu.Unlock()
	require.NotEmpty(t, tkn, "sign-in answered without an X-JWT header, so remark42-jwtheader is no longer in "+
		"header mode and nothing below is about it")

	// both claims come from one clock, the server's, so their distance holds whatever this
	// machine's clock reads. counting it from the arrival of the response can only overshoot
	waitUntil(receivedAt.Add(tokenLifetime(t, tkn) + time.Second))

	text := "posted after the token expired " + runID
	req, err := page.ExpectRequest(regexp.MustCompile(`/api/v1/comment\?`), func() error {
		submitForm(t, frame.Locator(commentFormSel).First(), text)
		return nil
	}, playwright.PageExpectRequestOptions{Timeout: new(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err)
	require.Equal(t, http.MethodPost, req.Method())

	sent, err := req.HeaderValue("x-jwt")
	require.NoError(t, err)
	assert.Empty(t, sent, "the request made after the token expired still carried it in X-JWT. The backend "+
		"refuses an expired token in that header without looking at the cookie, so it has to be left out")

	resp, err := req.Response()
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.Status(),
		"posting after the token expired was refused, so the session ended with the token")
	renewed, err := resp.HeaderValue("x-jwt")
	require.NoError(t, err)
	assert.NotEmpty(t, renewed, "the comment was accepted without the token being renewed, so it had not "+
		"expired and the wait above was too short to test anything")

	waitVisible(t, comment(frame, text))
	assertSignedIn(t, page, frame)

	// and the renewed token reached the cookie, or the session would end at the reload instead
	frame = reload(t, page)
	assertSignedIn(t, page, frame)
	waitVisible(t, comment(frame, text))
}

// tokenLifetime reads how long a JWT was issued for out of its own claims
func tokenLifetime(t *testing.T, tkn string) time.Duration {
	t.Helper()

	parts := strings.Split(tkn, ".")
	require.Len(t, parts, 3, "not a JWT: %q", tkn)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims struct {
		Exp int64 `json:"exp"`
		Iat int64 `json:"iat"`
	}
	require.NoError(t, json.Unmarshal(payload, &claims))
	require.Greater(t, claims.Exp, claims.Iat, "the token carries no usable exp and iat: %s", payload)
	return time.Duration(claims.Exp-claims.Iat) * time.Second
}

// waitUntil sleeps until a deadline computed from something known, which is not the same as
// sleeping for a guess: there is nothing in the page to poll for while a token runs out
func waitUntil(deadline time.Time) {
	if wait := time.Until(deadline); wait > 0 {
		time.Sleep(wait)
	}
}

// TestAuth_NoProvidersSaysSo covers #1456, where an instance with no auth provider rendered a
// sign-in panel offering nothing and no explanation, so the operator's own misconfiguration read
// as the widget being broken
func TestAuth_NoProvidersSaysSo(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openURL(t, page, threadURLOn(t, noAuthURL))

	require.NoError(t, frame.Locator(".auth-button").Click())
	waitVisible(t, frame.Locator("text=No sign-in method available"))

	// and no empty form is offered alongside it
	inputs, err := frame.Locator(".auth-input-username").Count()
	require.NoError(t, err)
	assert.Zero(t, inputs, "a sign-in form is offered on an instance with nothing to sign in with")
}

// waitPastEditWindow waits out the instance's ordinary edit window, measured from the moment the
// comment was posted. There is nothing to poll for here: the admin's comment shows no countdown,
// which is the assertion above, so the deadline passing is not observable in the page. Sleeping
// until a deadline computed from a known setting is not the same as sleeping for a guess
func waitPastEditWindow(postedAt time.Time) {
	// a margin over the window itself, since the backend compares against its own clock and the
	// post round trip sits between the two
	deadline := postedAt.Add(editWindow + time.Second)
	if wait := time.Until(deadline); wait > 0 {
		time.Sleep(wait)
	}
}
