//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// remark_config is the widget's public surface: whatever an integrator sets there has to keep
// working, and discussion #1714 asks for it to be written down. None of it was covered, so a
// setting could stop reaching the widget without anything here going red.

// TestConfig_ColorsReachTheWidget covers __colors__, the only setting that does not travel in the
// iframe's query string: the parent puts it in window.name and an inline script in the widget
// document, in templates/iframe.ejs, reads it back before the bundle runs. Nothing else opens
// window.name, so the whole path could be removed unnoticed
func TestConfig_ColorsReachTheWidget(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	stubSignedOut(t, page)
	embedConfig(t, page, map[string]any{"__colors__": map[string]any{"--color15": "rgb(1, 2, 3)"}})
	widget(t, page)

	value, err := page.FrameLocator("#remark42 iframe").Locator(":root").Evaluate(
		`(el) => getComputedStyle(el).getPropertyValue('--color15').trim()`, nil,
		playwright.LocatorEvaluateOptions{Timeout: playwright.Float(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err)
	assert.Equal(t, "rgb(1, 2, 3)", value, "the colors an integrator sets never reached the widget document")
}

// TestConfig_URLOverrideDecidesTheThread covers remark_config.url, which is how a canonical
// address keeps one conversation across pages that differ: a print view, a path with tracking
// parameters, a page that moved. Two host pages at different addresses name the same thread here,
// and the second has to see what the first posted.
//
// The override carries a query string on purpose, since createInstance strips the fragment and
// nothing else, so the query is part of the thread's identity and dropping it would split the
// conversation in two. One parameter and not two: a url containing "&" cannot be commented on
// at all, see #2204
func TestConfig_URLOverrideDecidesTheThread(t *testing.T) {
	t.Parallel()

	shared := fmt.Sprintf("%s/web/?e2e=config-url-%s", baseURL, runID)
	text := "shared thread " + runID

	page := newPage(t)
	embedConfigOn(t, page, "/web/privacy.html", map[string]any{"url": shared})
	frame := widget(t, page)
	signInAnon(t, page, frame, anonName("urloverride"))
	postComment(t, frame, text)

	// a different host page, same override
	other := newPage(t)
	stubSignedOut(t, other)
	embedConfigOn(t, other, "/web/markdown-help.html", map[string]any{"url": shared})
	otherFrame := widget(t, other)
	// markdown-help.html is long enough to leave the widget below the fold, and comments render
	// through an IntersectionObserver, so one that has not been reached is an empty article no
	// text filter matches
	require.NoError(t, other.Locator("#remark42 iframe").ScrollIntoViewIfNeeded())
	waitVisible(t, comment(otherFrame, text))

	// and the query string reached the widget, which is what makes this a test of the override
	// and not of two pages agreeing on a default
	src, err := other.Locator("#remark42 iframe").GetAttribute("src")
	require.NoError(t, err)
	assert.Contains(t, src, "e2e%3Dconfig-url", "the query string was dropped from the thread url")
}

// TestConfig_SubscriptionControlsCanBeHidden covers show_rss_subscription and
// show_email_subscription, which an integrator turns off to keep the form to itself, read in
// common/settings.ts. The both-shown case is the control: without it these would hold on a
// widget that offers neither
func TestConfig_SubscriptionControlsCanBeHidden(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		config      map[string]any
		rss, byMail bool
	}{
		{"both shown", map[string]any{}, true, true},
		{"rss hidden", map[string]any{"show_rss_subscription": false}, false, true},
		{"email hidden", map[string]any{"show_email_subscription": false}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := newPage(t)
			embedConfig(t, page, tc.config)
			frame := widget(t, page)
			// the email control is offered to a registered reader only, which anonymous is not
			signInDev(t, page, frame)

			rss := frame.Locator(`[title="Subscribe by RSS"]`)
			byMail := frame.Locator(`[title="Subscribe by Email"]`)

			// this instance runs NOTIFY_USERS=email, so the telegram control has to be absent
			// whatever the display settings say. The email control beside it is the positive
			// control: a widget offering no subscriptions at all would satisfy the absence alone
			waitHidden(t, frame.Locator(`[title="Subscribe by Telegram"]`),
				"the telegram control was offered on an instance with telegram notifications off")

			if tc.rss {
				waitVisible(t, rss)
			} else {
				waitHidden(t, rss, "show_rss_subscription=false left the RSS control in place")
			}
			if tc.byMail {
				waitVisible(t, byMail)
			} else {
				waitHidden(t, byMail, "show_email_subscription=false left the email control in place")
			}
		})
	}
}

// TestConfig_UnknownLocaleFallsBackToEnglish covers loadLocale's only observable guarantee. It
// returns the English defaults both for a name it does not recognize and for a chunk it fails to
// fetch, and the two are indistinguishable from outside, so a widget rendering English for
// locale=xx is all a caller can be promised. Without this, a build that stopped resolving
// catalogs altogether would still look correct to anyone reading English
func TestConfig_UnknownLocaleFallsBackToEnglish(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	stubSignedOut(t, page)
	embedConfig(t, page, map[string]any{"locale": "xx"})

	frame := widget(t, page)
	placeholder, err := frame.Locator(commentFormSel).First().Locator("textarea").GetAttribute("placeholder")
	require.NoError(t, err)
	assert.Equal(t, "Your comment here", placeholder,
		"an unrecognized locale did not fall back to the english defaults")
}

// TestConfig_TimesRenderInTheReadersTimezone covers the one part of a comment's rendering that
// cannot move to the server. The locale is configured and the server knows it, but the timezone
// belongs to the reader's machine, so a design that renders timestamps anywhere else has to keep
// this working. Kiritimati is UTC+14, far enough that a wrong timezone usually lands on the wrong
// day as well as the wrong hour
func TestConfig_TimesRenderInTheReadersTimezone(t *testing.T) {
	t.Parallel()

	const zone = "Pacific/Kiritimati"

	page := newPageInContext(t, browser, playwright.BrowserNewContextOptions{
		TimezoneId: playwright.String(zone),
	})
	frame := openThread(t, page)
	signInAnon(t, page, frame, anonName("timezone"))

	text := "timezone " + runID
	posted := postCommentMatching(t, frame, text, text)

	shown, err := posted.Locator(`a[href*="#remark42__comment-"]`).First().InnerText()
	require.NoError(t, err)

	// the server's own timestamp for that comment, so the comparison is against what was stored
	// and not against the clock this process reads
	status, body := pageFetch(t, page, "GET",
		fmt.Sprintf("%s/api/v1/find?site=remark&url=%s&format=tree", baseURL, neturl.QueryEscape(threadURL(t))), nil)
	require.Equal(t, http.StatusOK, status, "could not read the thread back: %s", body)

	var thread struct {
		Comments []struct {
			Comment struct {
				Time string `json:"time"`
			} `json:"comment"`
		} `json:"comments"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &thread))
	require.NotEmpty(t, thread.Comments, "the thread came back empty")

	// formatted in the page, by the same Intl the widget uses, so this asserts the timezone
	// reached the rendering and not that two libraries agree on a format
	want, err := page.Evaluate(`([iso, zone]) => {
		const d = new Date(iso);
		const day = new Intl.DateTimeFormat('en', {timeZone: zone}).format(d);
		const time = new Intl.DateTimeFormat('en', {timeZone: zone, hour: 'numeric', minute: 'numeric'}).format(d);
		return {day, time};
	}`, []any{thread.Comments[0].Comment.Time, zone})
	require.NoError(t, err)

	parts, ok := want.(map[string]any)
	require.True(t, ok, "unexpected shape from the page: %#v", want)
	day, _ := parts["day"].(string)
	require.NotEmpty(t, day)

	assert.Contains(t, shown, day,
		"the comment is dated %q, which is not the reader's own day in %s", shown, zone)
}

// TestConfig_PageTitleReachesTheStoredComment covers the title path, which runs the other way
// from everything else here: the host page's title is posted into the widget, the widget sends it
// with the comment, and the backend stores it against the thread. It is what a feed and the admin
// listing show, and nothing else here would notice it going
func TestConfig_PageTitleReachesTheStoredComment(t *testing.T) {
	t.Parallel()

	title := "A title only this test uses " + runID

	page := newPage(t)
	embedConfig(t, page, map[string]any{})
	frame := widget(t, page)

	// set after the widget is up, so this covers the observer embed.ts installs on the title
	// element and not merely the value read once at boot
	_, err := page.Evaluate(`(t) => { document.title = t; }`, title)
	require.NoError(t, err)

	signInAnon(t, page, frame, anonName("pagetitle"))
	text := "titled " + runID
	postComment(t, frame, text)

	status, body := pageFetch(t, page, "GET",
		fmt.Sprintf("%s/api/v1/find?site=remark&url=%s&format=tree", baseURL, neturl.QueryEscape(threadURL(t))), nil)
	require.Equal(t, http.StatusOK, status, "could not read the thread back: %s", body)

	var thread struct {
		Comments []struct {
			Comment struct {
				Title string `json:"title"`
			} `json:"comment"`
		} `json:"comments"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &thread))
	require.NotEmpty(t, thread.Comments)
	assert.Equal(t, title, thread.Comments[0].Comment.Title,
		"the host page's title never reached the stored comment")
}
