//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/mxschmitt/playwright-go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWidgets_LastCommentsRendersIntoTheHostPage covers the one component that writes into the
// embedding page rather than into the widget's iframe, so a change to the embed script can
// break it without any iframe test noticing
func TestWidgets_LastCommentsRendersIntoTheHostPage(t *testing.T) {
	text := "last comments " + runID

	poster := newPage(t)
	frame := openThread(t, poster)
	signInAnon(t, frame, "lastcommenter")
	postComment(t, frame, text)

	page := newPage(t)
	pauseForAuthLimit()
	_, err := page.Goto(baseURL + "/web/last-comments.html")
	require.NoError(t, err)

	list := page.Locator(".remark42__last-comments")
	waitVisible(t, list)
	waitVisible(t, list.Locator("text="+text))
}

// TestWidgets_CounterFillsInTheCommentCount covers the other host-page script.
//
// The demo page hard-codes both counters to one fixed url, so "some digits appeared" would
// hold just as well if the script always wrote a constant. Post to that same url and assert
// the rendered number moves by exactly as many comments as were added.
func TestWidgets_CounterFillsInTheCommentCount(t *testing.T) {
	const counted = "https://remark42.com/demo/"

	poster := newPage(t)
	frame := openThread(t, poster)
	signInAnon(t, frame, "countertester")

	before := commentCount(t, poster, counted)
	for i := range 2 {
		status, body := pageFetch(t, poster, "POST", baseURL+"/api/v1/comment?site=remark", map[string]any{
			"text":    fmt.Sprintf("counted %d %s", i, runID),
			"locator": map[string]string{"site": "remark", "url": counted},
		})
		require.Equal(t, 201, status, "could not seed a comment: %s", body)
	}

	page := newPage(t)

	// the demo page points both of its counters at the same url, one through data-url and one
	// through remark_config, so as shipped the two branches are indistinguishable. add a third
	// node carrying this test's own thread, which only the data-url branch can resolve
	thread := threadURL(t)
	err := page.AddInitScript(playwright.Script{
		Content: playwright.String(`document.addEventListener('DOMContentLoaded', () => {
			const node = document.createElement('span');
			node.className = 'remark42__counter';
			node.id = 'own-thread-counter';
			node.dataset.url = ` + fmt.Sprintf("%q", thread) + `;
			document.body.appendChild(node);
		})`),
	})
	require.NoError(t, err)

	pauseForAuthLimit()
	_, err = page.Goto(baseURL + "/web/counter.html")
	require.NoError(t, err)

	counters := page.Locator(".remark42__counter")
	count, err := counters.Count()
	require.NoError(t, err)
	require.NotZero(t, count, "the counter demo page should carry at least one counter node")

	want := strconv.Itoa(before + 2)
	for i := range count {
		node := counters.Nth(i)
		id, aerr := node.GetAttribute("id")
		require.NoError(t, aerr)
		if id == "own-thread-counter" {
			continue // asserted separately below, it counts a different url
		}
		eventually(t, waitTimeout, "counter never reported the seeded comments", func() bool {
			txt, ierr := pollText(node)
			return ierr == nil && txt == want
		})
	}

	// and the data-url branch resolves to its own thread rather than the page's
	own := strconv.Itoa(commentCount(t, poster, thread))
	eventually(t, waitTimeout, "the data-url counter did not report its own thread", func() bool {
		txt, ierr := pollText(page.Locator("#own-thread-counter"))
		return ierr == nil && txt == own
	})
}

func TestWidgets_LegacyJSURLLoadsAsAClassicScript(t *testing.T) {
	thread := threadURL(t)

	poster := newPage(t)
	frame := openThread(t, poster)
	signInAnon(t, frame, "aliastester")
	postComment(t, frame, "alias "+runID)

	want := strconv.Itoa(commentCount(t, poster, thread))

	page := newPage(t)
	pauseForAuthLimit()
	_, err := page.Goto(baseURL + "/web/privacy.html")
	require.NoError(t, err)

	_, err = page.Evaluate(`([host, url]) => {
		window.remark_config = { host, site_id: 'remark' };
		const node = document.createElement('span');
		node.className = 'remark42__counter';
		node.dataset.url = url;
		document.body.appendChild(node);
	}`, []any{baseURL, thread})
	require.NoError(t, err)

	counter := page.Locator(".remark42__counter")
	blank, err := pollText(counter)
	require.NoError(t, err)
	require.Empty(t, blank, "the counter must start blank or the assertion below is vacuous")

	_, err = page.AddScriptTag(playwright.PageAddScriptTagOptions{URL: playwright.String(baseURL + "/web/counter.js")})
	require.NoError(t, err)

	eventually(t, waitTimeout, "the legacy counter.js url never filled the counter", func() bool {
		txt, ierr := pollText(counter)
		return ierr == nil && txt == want
	})
}

// commentCount asks the API what the counter should be showing
func commentCount(t *testing.T, page playwright.Page, url string) int {
	t.Helper()
	status, body := pageFetch(t, page, "POST", baseURL+"/api/v1/counts?site=remark", []string{url})
	require.Equal(t, 200, status, "counts: %s", body)

	var counts []struct {
		Count int `json:"count"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &counts))
	require.Len(t, counts, 1)
	return counts[0].Count
}

// TestWidgets_ProfileOpensInItsOwnIframe covers the postMessage handoff: the widget asks the
// parent to open the profile, and the parent creates a second iframe outside #remark42
func TestWidgets_ProfileOpensInItsOwnIframe(t *testing.T) {
	page := newPage(t)
	frame := openThread(t, page)
	signInDev(t, page, frame)

	// match on the page parameter rather than the word: the widget's own src carries the
	// thread url, which contains this test's name
	profile := page.Locator(`iframe[src*="page=profile"]`)
	before, err := profile.Count()
	require.NoError(t, err)
	require.Zero(t, before, "no profile iframe should exist before it is asked for")

	require.NoError(t, frame.Locator(`[title="Open My Profile"]`).Click())

	waitVisible(t, profile.First())
	src, err := profile.First().GetAttribute("src")
	require.NoError(t, err)
	assert.Contains(t, src, "current=1", "the profile should open on the signed-in user")

	// the frame reveals itself after five seconds whether or not its bundle ran, so a 404 or a
	// dead script would satisfy a visibility check on its own. look inside it
	profileFrame := page.FrameLocator(`iframe[src*="page=profile"]`)
	waitVisible(t, profileFrame.Locator("text=dev_user").First())

	// it lives outside the widget's own container, appended straight to the body
	inWidget, err := page.Locator(`#remark42 iframe[src*="page=profile"]`).Count()
	require.NoError(t, err)
	assert.Zero(t, inWidget)
}
