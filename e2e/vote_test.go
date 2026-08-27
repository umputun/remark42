//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"
	"sync"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// score reads the vote counter of the comment carrying the text
func score(frame playwright.FrameLocator, text string) playwright.Locator {
	return comment(frame, text).Locator(`[title="Votes score"]`).First()
}

// voteScenario posts a comment as one user and returns a second, signed-in user's page and
// its view of that comment. remark42 hides the vote buttons on your own comment, so the
// author and the voter cannot be the same person
func voteScenario(t *testing.T, author, text string) (playwright.Page, playwright.FrameLocator, playwright.Locator) {
	t.Helper()

	authorPage := newPage(t)
	authorFrame := openThread(t, authorPage)
	signInAnon(t, authorPage, authorFrame, author)
	postComment(t, authorFrame, text)

	voter := newPage(t)
	voterFrame := openThread(t, voter)
	signInDev(t, voter, voterFrame)

	target := comment(voterFrame, text)
	waitVisible(t, target)
	return voter, voterFrame, target
}

func TestVote_UpvoteCountsOnce(t *testing.T) {
	t.Parallel()

	text := "vote target " + runID
	voter, voterFrame, target := voteScenario(t, "voteauthor", text)

	require.NoError(t, target.Locator(`button[title="Vote up"]`).Click())

	eventually(t, waitTimeout, "score did not reach 1", func() bool {
		v, err := pollText(score(voterFrame, text))
		return err == nil && v == "1"
	})

	// the vote is stored and not only reflected in local state
	voterFrame = reload(t, voter)
	eventually(t, waitTimeout, "score did not survive reload", func() bool {
		v, err := pollText(score(voterFrame, text))
		return err == nil && v == "1"
	})

	// and the same voter cannot stack a second one. this has to come after the reload: for
	// 200ms after a click the button is disabled by the in-flight loading state, so checking
	// it earlier would pass without the caller's own vote ever coming back from /find
	disabled, err := comment(voterFrame, text).Locator(`button[title="Vote up"]`).IsDisabled()
	require.NoError(t, err)
	assert.True(t, disabled, "an already-cast upvote should not be repeatable")
}

// TestVote_FailureShowsAnErrorAndRestoresTheScore drives the catch branch in comment-votes.tsx.
// The score is optimistic, so a failed request has to put it back instead of leaving the
// reader believing a vote landed.
func TestVote_FailureShowsAnErrorAndRestoresTheScore(t *testing.T) {
	t.Parallel()

	text := "vote failure " + runID
	voter, voterFrame, target := voteScenario(t, "votefailauthor", text)

	// hold the response open. answering instantly would let the test pass with the optimistic
	// increment removed altogether, since the score would simply never leave 0
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	// on any failure below the handler would otherwise sit on <-release for the rest of the
	// process, with the intercepted request never answered
	defer unblock()
	require.NoError(t, voter.Route("**/api/v1/vote/**", func(route playwright.Route) {
		<-release
		// 409 and not 500: the widget maps a handful of statuses to copy of their own and
		// everything else to a generic "something went wrong", which is also what it shows when
		// error handling fails altogether. asserting a distinct string is what makes this
		// assertion mean anything
		if err := route.Fulfill(playwright.RouteFulfillOptions{
			Status:      playwright.Int(409),
			ContentType: playwright.String("application/json"),
			Body:        `{"code":19,"details":"vote rejected","error":"failed"}`,
		}); err != nil {
			t.Errorf("fulfill refused vote response: %v", err)
		}
	}))

	require.NoError(t, target.Locator(`button[title="Vote up"]`).Click())

	// while the request is in flight the widget shows the vote as though it had landed
	eventually(t, waitTimeout, "the score was never incremented optimistically", func() bool {
		v, err := pollText(score(voterFrame, text))
		return err == nil && v == "1"
	})

	unblock()

	// the widget shows its own copy for the status, not the raw body
	waitVisible(t, target.Locator("text=Conflict."))

	eventually(t, waitTimeout, "the optimistic score was not rolled back", func() bool {
		v, err := pollText(score(voterFrame, text))
		return err == nil && v == "0"
	})
}

// TestVote_AnonymousVoterCounts covers ANON_VOTE, which the default configuration refuses:
// rest_private.go turns a vote down when the user id carries the anonymous prefix unless the
// setting is on, and it is only meaningful alongside VOTES_IP, which is what scopes it. Its own
// instance, since both are server flags. The vote and the author are separate anonymous users
// because remark42 hides the buttons on your own comment either way
func TestVote_AnonymousVoterCounts(t *testing.T) {
	t.Parallel()

	thread := threadURLOn(t, anonVoteURL)
	text := "anon vote target " + runID

	author := newPage(t)
	authorFrame := openURL(t, author, thread)
	signInAnon(t, author, authorFrame, "anonvoteauthor")
	postComment(t, authorFrame, text)

	voter := newPage(t)
	voterFrame := openURL(t, voter, thread)
	signInAnon(t, voter, voterFrame, "anonvoter")

	target := comment(voterFrame, text)
	waitVisible(t, target)
	require.NoError(t, target.Locator(`button[title="Vote up"]`).Click())

	eventually(t, waitTimeout, "the anonymous vote did not register", func() bool {
		v, err := pollText(score(voterFrame, text))
		return err == nil && v == "1"
	})

	// stored, not merely reflected in the optimistic state the click set, which is the half a
	// refused vote would still satisfy
	voterFrame = reload(t, voter)
	eventually(t, waitTimeout, "the anonymous vote did not survive a reload", func() bool {
		v, err := pollText(score(voterFrame, text))
		return err == nil && v == "1"
	})
}

// TestVote_OwnCommentCannotBeVotedOn covers the rule every other vote case has to work around:
// remark42 does not offer the buttons on your own comment, and the backend refuses the vote even
// when the buttons are put back. Both halves, since the widget hiding them is only politeness
func TestVote_OwnCommentCannotBeVotedOn(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	signInAnon(t, page, frame, anonName("selfvoter"))

	text := "own comment " + runID
	posted := postCommentMatching(t, frame, text, text)

	buttons, err := posted.Locator(`button[title="Vote up"]`).Count()
	require.NoError(t, err)
	assert.Zero(t, buttons, "the widget offered a vote on the reader's own comment")

	id, err := posted.GetAttribute("id")
	require.NoError(t, err)
	commentID := strings.TrimPrefix(id, "remark42__comment-")

	// and the backend refuses it, which is what makes the absence above a courtesy and not the
	// rule itself
	vote := fmt.Sprintf("%s/api/v1/vote/%s?site=remark&url=%s&vote=1",
		baseURL, commentID, neturl.QueryEscape(threadURL(t)))
	status, body := pageFetch(t, page, "PUT", vote, nil)
	assert.GreaterOrEqual(t, status, http.StatusBadRequest,
		"the backend allowed a vote on the voter's own comment: %d %s", status, body)
}

// TestVote_DownvoteAndCorrection covers the other direction and the correction #728 asked for: a
// reader who changes their mind has to be able to, and the score has to end where the second vote
// leaves it. Nothing covered downvoting at all
func TestVote_DownvoteAndCorrection(t *testing.T) {
	t.Parallel()

	text := "downvote target " + runID
	voter, voterFrame, target := voteScenario(t, "downvoteauthor", text)

	require.NoError(t, target.Locator(`button[title="Vote down"]`).Click())
	eventually(t, waitTimeout, "the downvote did not register", func() bool {
		v, err := pollText(score(voterFrame, text))
		return err == nil && v == "-1"
	})

	// and it is the server's, not the optimistic state the click set
	voterFrame = reload(t, voter)
	eventually(t, waitTimeout, "the downvote did not survive a reload", func() bool {
		v, err := pollText(score(voterFrame, text))
		return err == nil && v == "-1"
	})

	// then the reader changes their mind, which is what #728 asked for and #729 fixed: the
	// opposite vote has to be accepted and not refused as a repeat. It takes the vote back
	// instead of flipping it, so the score returns to zero and does not become +1. Asserted
	// after a reload as well, since a correction the server never recorded leaves the reader
	// looking at a number nobody else sees
	require.NoError(t, comment(voterFrame, text).Locator(`button[title="Vote up"]`).Click())
	eventually(t, waitTimeout, "the opposite vote did not take the downvote back", func() bool {
		v, err := pollText(score(voterFrame, text))
		return err == nil && v == "0"
	})

	voterFrame = reload(t, voter)
	eventually(t, waitTimeout, "the correction did not survive a reload", func() bool {
		v, err := pollText(score(voterFrame, text))
		return err == nil && v == "0"
	})
}

// TestVote_WithoutTheXSRFHeaderIsRefused pins the mechanism that decides how the widget can be
// rendered at all. go-pkgz/auth rejects a cookie-borne token whose X-XSRF-TOKEN header does not
// match the jti in it, with no exemption by method, which is why a document navigation, an iframe
// src among them, is always anonymous and why the widget hydrates its user over XHR. Anyone
// designing around that needs it pinned, since nothing else here would notice the check going
func TestVote_WithoutTheXSRFHeaderIsRefused(t *testing.T) {
	t.Parallel()

	text := "xsrf vote " + runID
	voter, voterFrame, target := voteScenario(t, "xsrfauthor", text)

	// strip the header the widget attaches to every call, and let the rest of the request go as
	// it was: same cookie, same body, same origin
	require.NoError(t, voter.Route("**/api/v1/vote/**", func(route playwright.Route) {
		headers := route.Request().Headers()
		delete(headers, "x-xsrf-token")
		if err := route.Fallback(playwright.RouteFallbackOptions{Headers: headers}); err != nil {
			t.Errorf("forward vote without XSRF header: %v", err)
		}
	}))

	resp, err := voter.ExpectResponse("**/api/v1/vote/**", func() error {
		return target.Locator(`button[title="Vote up"]`).Click()
	}, playwright.PageExpectResponseOptions{Timeout: playwright.Float(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err, "the vote was never sent")
	assert.GreaterOrEqual(t, resp.Status(), http.StatusBadRequest,
		"a vote without the XSRF header was accepted, so the check that forces anonymous-first "+
			"rendering is no longer there")

	// and the reader is not left believing it landed
	eventually(t, waitTimeout, "the optimistic score was not rolled back", func() bool {
		v, err := pollText(score(voterFrame, text))
		return err == nil && v == "0"
	})
}
