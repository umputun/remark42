//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mxschmitt/playwright-go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWidgets_LastCommentsRendersIntoTheHostPage covers the one component that writes into the
// embedding page and not into the widget's iframe, so a change to the embed script can
// break it without any iframe test noticing
func TestWidgets_LastCommentsRendersIntoTheHostPage(t *testing.T) {
	text := "last comments " + runID

	poster := newPage(t)
	frame := openThread(t, poster)
	signInAnon(t, poster, frame, "lastcommenter")
	postComment(t, frame, text)

	page := newPage(t)

	// the stylesheet is appended at runtime and nothing waits for it, so the comments render
	// whether or not it arrives. wait for the response itself instead of sampling afterwards,
	// which reads whatever has landed by then and passes when the miss is still in flight
	pauseForAuthLimit()
	css, err := page.ExpectResponse("**/last-comments.css", func() error {
		_, gerr := page.Goto(baseURL + "/web/last-comments.html")
		return gerr
	}, playwright.PageExpectResponseOptions{Timeout: playwright.Float(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err, "the page never asked for its stylesheet")
	assert.Equal(t, 200, css.Status(), "the last-comments stylesheet did not load")

	list := page.Locator(".remark42__last-comments")
	waitVisible(t, list)
	waitVisible(t, list.Locator("text="+text))
}

// TestWidgets_DeleteMePageServesAndRuns covers the GDPR delete page, which nothing else opens. It
// needs an admin token to do its work, so this drives the branch it takes without one: reaching
// that message proves the html and its bundle were both served and executed.
func TestWidgets_DeleteMePageServesAndRuns(t *testing.T) {
	page := newPage(t)

	pauseForAuthLimit()
	resp, err := page.Goto(baseURL + "/web/deleteme.html")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 200, resp.Status())

	waitVisible(t, page.Locator("text=You are not logged in"))
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
	signInAnon(t, poster, frame, "countertester")

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

	// and the data-url branch resolves to its own thread and not the page's
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
	signInAnon(t, poster, frame, "aliastester")
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

	// match on the page parameter and not the word: the widget's own src carries the
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

// TestWidgets_EveryLocaleLoadsAndRenders drives the dynamic import behind every translation, once
// per catalog. The catalogs are separate chunks the bundle fetches at runtime, so a change to
// how chunks are named or emitted breaks them without touching a line of widget code, and the
// failure is silent: loadLocale falls back to english instead of throwing, which is also what an
// unrecognized name does. Each case therefore compares against the catalog on disk, so english
// coming back is a failure and not a pass.
//
// The catalog set comes from the locales directory and not a list here, so a language added to
// the app is covered without an edit, and injected into a plain page and not the demo one,
// which is the only way to hand the widget a remark_config of this test's choosing
func TestWidgets_EveryLocaleLoadsAndRenders(t *testing.T) {
	const localesDir = "../frontend/apps/remark42/app/locales"

	entries, err := os.ReadDir(localesDir)
	require.NoError(t, err, "reading %s", localesDir)
	require.NotEmpty(t, entries, "no catalogs in %s, so this test would assert nothing", localesDir)

	for _, entry := range entries {
		locale, found := strings.CutSuffix(entry.Name(), ".json")
		if !found {
			continue
		}

		t.Run(locale, func(t *testing.T) {
			raw, rerr := os.ReadFile(filepath.Join(localesDir, entry.Name())) //nolint:gosec // name from the walk
			require.NoError(t, rerr)

			var catalog map[string]string
			require.NoError(t, json.Unmarshal(raw, &catalog))
			want := catalog["commentForm.input-placeholder"]
			require.NotEmpty(t, want, "%s carries no placeholder message to compare against", entry.Name())

			page := newPage(t)
			// this case never signs in, and the widget probes /auth/status on every load. that
			// probe is capped at 2/s for the whole suite, so twenty four of them would spend a
			// budget the sign-in cases need and manufacture 429s for whichever test runs next
			require.NoError(t, page.Route("**/auth/status**", func(route playwright.Route) {
				require.NoError(t, route.Fulfill(playwright.RouteFulfillOptions{
					Status:      playwright.Int(http.StatusOK),
					ContentType: playwright.String("application/json"),
					Body:        playwright.String(`{"status":"not logged in"}`),
				}))
			}))

			_, gerr := page.Goto(baseURL + "/web/privacy.html")
			require.NoError(t, gerr)

			_, eerr := page.Evaluate(`([host, url, locale]) => {
				window.remark_config = { host, site_id: 'remark', url, locale };
				const node = document.createElement('div');
				node.id = 'remark42';
				document.body.appendChild(node);
			}`, []any{baseURL, threadURL(t), locale})
			require.NoError(t, eerr)

			_, aerr := page.AddScriptTag(playwright.PageAddScriptTagOptions{URL: playwright.String(baseURL + "/web/embed.mjs")})
			require.NoError(t, aerr)

			// not widget(): commentFormSel matches the form's aria-label, which is itself
			// translated, so the shared helper only ever finds an english widget
			frame := page.FrameLocator("#remark42 iframe")
			textarea := frame.Locator("form textarea").First()
			waitVisible(t, textarea)

			got, perr := textarea.GetAttribute("placeholder")
			require.NoError(t, perr)
			assert.Equal(t, want, got, "the %s catalog did not render, so the widget fell back to english", locale)
		})
	}
}

// TestWidgets_SimpleViewHidesTheEditingFurniture covers simple_view, the one mode of this kind
// that is a query parameter and not a server flag, so both branches run against the same
// instance. It takes the markdown toolbar and the preview away and leaves everything else,
// including the markdown help line, which is why that is not asserted either way. The full-view
// branch is the control: without it these would hold on a widget that never had a toolbar
func TestWidgets_SimpleViewHidesTheEditingFurniture(t *testing.T) {
	for _, tc := range []struct {
		name   string
		simple bool
	}{
		{"full view", false},
		{"simple view", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := newPage(t)

			config := map[string]any{}
			if tc.simple {
				config["simple_view"] = true
			}
			embedConfig(t, page, config)
			frame := widget(t, page)
			// signed in, since the preview button is only offered to somebody who could post
			signInAnon(t, page, frame, anonName("simpleview"))

			// by element name and not by test id, which the production bundle strips, or by
			// class, which it hashes
			toolbar := frame.Locator(`md-bold`).First()
			preview := frame.Locator(`button:has-text("Preview")`).First()

			if tc.simple {
				waitHidden(t, toolbar, "simple_view left the markdown toolbar in place")
				waitHidden(t, preview, "simple_view left the preview button in place")
				return
			}
			waitVisible(t, toolbar)
			waitVisible(t, preview)
		})
	}
}
