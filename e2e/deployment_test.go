//go:build e2e

package e2e

import (
	"testing"
	"time"

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
	page := newPage(t)
	url := threadURLOn(t, adminEditURL)
	frame := openURL(t, page, url)
	signInEmail(t, page, frame, "admineditor", adminEditAddress)

	text := "admin edit " + runID
	postedAt := time.Now()
	postComment(t, frame, text)

	// the countdown is what #2001 left running. an admin has no deadline, so it should never
	// have been rendered at all
	// asserted on a loaded thread, not on the comment the widget has just added to the
	// page. The optimistic render puts a countdown on an admin's own new comment and drops it on
	// the next load, so asserting here would fail on that instead of on the deadline logic this
	// case is about
	frame = reload(t, page)
	count, err := actions(frame, text).Locator(`[role="timer"]`).Count()
	require.NoError(t, err)
	assert.Zero(t, count, "an admin's comment is counting down an edit window that does not apply to it")

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
	page := newPage(t)
	frame := openURL(t, page, threadURLOn(t, jwtHeaderURL))
	signInAnon(t, page, frame, anonName("headerjwt"))

	frame = reload(t, page)
	assertSignedIn(t, page, frame)

	pauseForAuthLimit()
	require.NoError(t, frame.Locator(`[title="Sign Out"]`).Click())
	waitVisible(t, frame.Locator(".auth-button"))

	frame = reload(t, page)
	waitVisible(t, frame.Locator(".auth-button"))
	waitHidden(t, frame.Locator(`[title="Sign Out"]`),
		"the header-borne session came back after signing out")
}

// TestAuth_NoProvidersSaysSo covers #1456, where an instance with no auth provider rendered a
// sign-in panel offering nothing and no explanation, so the operator's own misconfiguration read
// as the widget being broken
func TestAuth_NoProvidersSaysSo(t *testing.T) {
	page := newPage(t)
	frame := openURL(t, page, threadURLOn(t, noAuthURL))

	require.NoError(t, frame.Locator(".auth-button").Click())
	waitVisible(t, frame.Locator("text=No providers available"))

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
