//go:build e2e

package e2e

import (
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
	text := "vote target " + runID
	voter, voterFrame, target := voteScenario(t, "voteauthor", text)

	require.NoError(t, target.Locator(`button[title="Vote up"]`).Click())

	eventually(t, waitTimeout, "score did not reach 1", func() bool {
		v, err := pollText(score(voterFrame, text))
		return err == nil && v == "1"
	})

	// the vote is stored rather than only reflected in local state
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
// The score is optimistic, so a failed request has to put it back rather than leave the
// reader believing a vote landed.
func TestVote_FailureShowsAnErrorAndRestoresTheScore(t *testing.T) {
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
		// 409 rather than 500: the widget maps a handful of statuses to copy of their own and
		// everything else to a generic "something went wrong", which is also what it shows when
		// error handling fails altogether. asserting a distinct string is what makes this
		// assertion mean anything
		_ = route.Fulfill(playwright.RouteFulfillOptions{
			Status:      playwright.Int(409),
			ContentType: playwright.String("application/json"),
			Body:        `{"code":19,"details":"vote rejected","error":"failed"}`,
		})
	}))

	require.NoError(t, target.Locator(`button[title="Vote up"]`).Click())

	// while the request is in flight the widget shows the vote as though it had landed
	eventually(t, waitTimeout, "the score was never incremented optimistically", func() bool {
		v, err := pollText(score(voterFrame, text))
		return err == nil && v == "1"
	})

	unblock()

	// the widget shows its own copy for the status rather than the raw body
	waitVisible(t, target.Locator("text=Conflict."))

	eventually(t, waitTimeout, "the optimistic score was not rolled back", func() bool {
		v, err := pollText(score(voterFrame, text))
		return err == nil && v == "0"
	})
}
