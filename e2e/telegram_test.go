//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Telegram auth is the one provider whose flow leaves the browser: the reader is told to message a
// bot, and the widget waits for the backend to see that message. Component tests cover the widget's
// state machine with mocked fetches; these cases also exercise the backend and bot exchange.
//
// The stub in e2e/telegramstub answers as the bot API and takes an injected update through its
// control endpoint, which is the step no page can perform. The instance pointing at it is
// remark42-telegram, on its own service so the main instance's provider list stays what every other
// case reads out of the auth panel.
//
// These cases stay serial because the stub exposes one shared update queue and sent-message log.

// tgSendToBot injects the message a reader would send the bot. This process drives it because the
// step happens inside Telegram, where the browser has no reach
func tgSendToBot(t *testing.T, text, id, firstName string) {
	t.Helper()

	q := url.Values{"text": {text}, "id": {id}, "first_name": {firstName}}
	resp, err := probeClient.Get(telegramStubURL + "/control/send?" + q.Encode())
	require.NoError(t, err, "the telegram stub did not accept the update")
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "the telegram stub refused the update")
}

// tgConfirmSignIn presses the widget's check button until the reader is signed in.
//
// Calls are spaced because the provider polls the bot API every five seconds (apiPollInterval in
// go-pkgz/auth), so the first press is legitimately early, and everything under /auth/ is capped at
// two requests a second per reader. A tight retry loop manufactures the 429s it would then have to
// interpret, so this waits out one poll interval and then presses on the limiter's own spacing, a
// handful of times at most
func tgConfirmSignIn(t *testing.T, page playwright.Page, frame playwright.FrameLocator) {
	t.Helper()

	signOut := frame.Locator(`[title="Sign Out"]`)
	submit := frame.Locator(`.auth-submit`)

	// two ways in, and which one happens is a race the test has no business deciding: the panel
	// polls while it is open, so the sign-in can complete on its own, and the button is there for
	// the reader who comes back to the tab after it stopped.
	//
	// The presence check costs the limiter nothing, so it runs often; the presses are what have to
	// be spaced, since everything under /auth/ is capped at two requests a second. Detecting on the
	// press interval instead would report a sign-in that landed immediately five seconds late, and
	// a genuine failure half a minute late
	deadline := time.Now().Add(waitTimeout)
	var lastPress time.Time
	for time.Now().Before(deadline) {
		if n, err := signOut.Count(); err == nil && n > 0 {
			assertSignedIn(t, page, frame)
			return
		}

		if time.Since(lastPress) >= 5*time.Second {
			// the button goes as soon as the panel confirms on its own, so losing the race to it is
			// not a failure: the next pass reads the signed-in state instead
			if n, err := submit.Count(); err == nil && n > 0 {
				pauseForAuthLimit()
				_ = submit.Click()
			}
			lastPress = time.Now()
		}

		time.Sleep(250 * time.Millisecond)
	}

	// whatever the panel is showing is the diagnosis: a refused token and a token the backend never
	// saw fail identically from out here
	shown := ""
	if n, err := frame.Locator(`.auth-error`).Count(); err == nil && n > 0 {
		shown, _ = frame.Locator(`.auth-error`).First().TextContent()
	}
	t.Fatalf("the widget never confirmed the telegram sign-in, panel error: %q", shown)
}

// tgSentMessages returns the messages the provider has asked the bot to deliver. Matching the text
// gives each wait its own barrier even when another process shares the stub.
func tgSentMessages(t *testing.T) []string {
	t.Helper()

	resp, err := probeClient.Get(telegramStubURL + "/control/sent")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var sent []string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&sent))
	return sent
}

func tgWaitForMessage(t *testing.T, before int, contains string) {
	t.Helper()
	eventually(t, waitTimeout, "the bot never sent a message containing "+contains, func() bool {
		messages := tgSentMessages(t)
		for _, message := range messages[min(before, len(messages)):] {
			if strings.Contains(strings.ToLower(message), strings.ToLower(contains)) {
				return true
			}
		}
		return false
	})
}

// tgReaderID is stable within a test and distinct across tests, runs and processes.
func tgReaderID(t *testing.T) string {
	t.Helper()
	key := fmt.Sprintf("%s|%s|%d", runID, t.Name(), os.Getpid())
	return fmt.Sprintf("%d", 100_000_000+uint64(crc32.ChecksumIEEE([]byte(key)))%900_000_000)
}

func telegramStartToken(t *testing.T, href string) string {
	t.Helper()
	parsed, err := url.Parse(href)
	require.NoError(t, err, "parse telegram link %q", href)
	token := parsed.Query().Get("start")
	require.NotEmpty(t, token, "no token in the telegram link: %s", href)
	return token
}

// tgStartSignIn opens the auth panel, picks telegram and returns the token the backend minted.
// Reading it from the rendered link ties the panel to the request behind it
func tgStartSignIn(t *testing.T, frame playwright.FrameLocator) string {
	t.Helper()

	// the backend saying the handler registered, before anything is driven through the panel. /ping
	// answers while telegram auth is dead -- a stub the instance cannot reach leaves the provider
	// off the list -- and without this the case fails on a locator timeout that names nothing
	resp, err := probeClient.Get(telegramProbeURL + "/api/v1/config?site=remark")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var cfg struct {
		AuthProviders []string `json:"auth_providers"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&cfg))
	require.Contains(t, cfg.AuthProviders, "telegram",
		"the instance does not offer telegram auth, so it never reached the stub: %v", cfg.AuthProviders)

	pauseForAuthLimit()
	require.NoError(t, frame.Locator(".auth-button").Click())
	waitVisible(t, frame.Locator(".auth-dropdown"))

	// the attribute carries the provider's display name, capitalised, and this instance offers
	// exactly one provider, so the assertion is what ties the click to telegram, not the selector
	provider := frame.Locator(`.oauth-button`).First()
	waitVisible(t, provider)
	name, nameErr := provider.GetAttribute("data-provider-name")
	require.NoError(t, nameErr)
	require.Equal(t, "telegram", strings.ToLower(name), "the only provider offered is not telegram")
	require.NoError(t, provider.Click())

	link := frame.Locator(`.telegram a`).First()
	waitVisible(t, link)

	href, hrefErr := link.GetAttribute("href")
	require.NoError(t, hrefErr)
	require.Contains(t, href, "remark42_e2e_bot", "the panel names a bot the stub never reported: %s", href)

	return telegramStartToken(t, href)
}

// TestTelegram_SignInThroughTheBot drives the whole two-step flow the widget performs: ask the
// backend for a token, show the reader a link carrying it, and then verify once the reader has
// messaged the bot. Both halves are asserted, since a widget that renders the link and never
// verifies looks identical to one that works until the reader tries it.
func TestTelegram_SignInThroughTheBot(t *testing.T) {
	page := newPage(t)
	frame := openURL(t, page, threadURLOn(t, telegramURL))

	token := tgStartSignIn(t, frame)

	// the reader messages the bot, and the shared update dispatcher picks it up
	tgSendToBot(t, "/start "+token, tgReaderID(t), "E2E Reader")

	// the widget verifies on demand, so this is the reader pressing the button after switching back
	// from Telegram. The poll interval is five seconds, so the first
	// press can legitimately come too early: press until the panel changes or the wait gives up
	tgConfirmSignIn(t, page, frame)

	// the identity came from the update, not from a placeholder: the name is what the stub sent
	waitVisible(t, frame.Locator(`text=E2E Reader`).First())
}

// TestTelegram_CommentSurvivesAReload is the other half of an auth provider being usable: a reader
// who signed in through Telegram can post, and is still that reader after a reload. The token lives
// in a cookie the backend set during verification, so a flow that signs in without persisting looks
// identical until the page reloads
func TestTelegram_CommentSurvivesAReload(t *testing.T) {
	page := newPage(t)
	frame := openURL(t, page, threadURLOn(t, telegramURL))

	token := tgStartSignIn(t, frame)

	// a distinct id per run, so the comments of one run cannot be read as another's. Digits only,
	// the way anonName builds a name: E2E_RUN_ID is "<run>-<attempt>" on CI, so anything slicing it
	// raw hands the stub something strconv cannot read
	tgSendToBot(t, "/start "+token, tgReaderID(t), "Reload Reader")

	tgConfirmSignIn(t, page, frame)

	text := "posted after telegram sign-in " + runID
	postComment(t, frame, text)

	frame = reload(t, page)

	assertSignedIn(t, page, frame)
	waitVisible(t, comment(frame, text))
}

// TestTelegram_UnknownTokenIsRefused pins that a login token is only ever honored for the request
// that minted it. The interesting negative is not an invented token, which fails whether or not the
// bot was ever involved: it is a real token from a real request, against a bot message carrying a
// different one. That fails the moment processUpdates stops matching what the reader sent against
// the requests it is holding, and nothing else here would notice.
func TestTelegram_UnknownTokenIsRefused(t *testing.T) {
	page := newPage(t)
	frame := openURL(t, page, threadURLOn(t, telegramURL))

	minted := tgStartSignIn(t, frame)
	before := len(tgSentMessages(t))

	// the reader messages the bot with something else entirely
	tgSendToBot(t, "/start deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "9999", "Nobody")

	// the provider answers an unmatched token by messaging the reader, and the stub records every
	// send. That condition marks the observable moment it
	// has drained the update and decided
	tgWaitForMessage(t, before, "authentication request was not found or expired")

	pauseForAuthLimit()
	resp, err := probeClient.Get(telegramProbeURL + "/auth/telegram/login?site=remark&token=" + minted)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// an exact status, not "anything but 200": a 429 from the limiter, a 500, or the route being
	// gone altogether would all satisfy a looser assertion while proving nothing
	assert.Equal(t, http.StatusNotFound, resp.StatusCode,
		"a token nobody confirmed was answered with %d: %s", resp.StatusCode, body)
	assert.Contains(t, string(body), "not verified",
		"the refusal does not say the request was unverified: %s", body)

	// and nothing was handed out on the way past
	for _, c := range resp.Cookies() {
		assert.NotEqual(t, "JWT", c.Name, "an unverified request was given a token cookie")
	}
}
