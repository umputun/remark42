//go:build e2e

package e2e

import (
	"net/http"
	"strings"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Telegram subscription also leaves the browser: the notify service polls for updates through its
// configurable API host. Component tests cover the panel's state machine with mocked fetches;
// these cases also exercise persistence, the backend and the bot exchange.
//
// Everything here runs on remark42-telegram, whose notify service and auth provider both answer
// through e2e/telegramstub.
// These cases stay serial because the stub exposes one shared update queue and sent-message log.

// TestTelegramSub_NotOfferedToAReaderWithoutAnAccount pins the control's precondition. A
// subscription is keyed to a user the backend can recognize later, so the widget offers none at all
// until the reader has an identity. The signed-in leg proves the same deployment offers the control.
func TestTelegramSub_NotOfferedToAReaderWithoutAnAccount(t *testing.T) {
	page := newPage(t)
	frame := openURL(t, page, threadURLOn(t, telegramURL))

	// the form is there, so the widget rendered
	waitVisible(t, frame.Locator(commentFormSel).First())

	control := frame.Locator(`[title="Subscribe by Telegram"]`)
	n, err := control.Count()
	require.NoError(t, err)
	assert.Zero(t, n, "Telegram subscription was offered to a reader with no account")

	token := tgStartSignIn(t, frame)
	tgSendToBot(t, "/start "+token, tgReaderID(t), "Control Reader")
	tgConfirmSignIn(t, page, frame)
	waitVisible(t, control)
}

// TestTelegramSub_RoundTrip drives the complete subscription lifecycle: the backend mints a
// one-time token, the reader sends it to the bot, the notify poll confirms it, and the widget
// persists and removes the subscription. It then starts again with a new token, since reusing the
// consumed token leaves the resubscribe control present but permanently unable to finish.
func TestTelegramSub_RoundTrip(t *testing.T) {
	page := newPage(t)
	frame := openURL(t, page, threadURLOn(t, telegramURL))
	readerID := tgReaderID(t)

	loginToken := tgStartSignIn(t, frame)
	tgSendToBot(t, "/start "+loginToken, readerID, "Subscription Reader")
	tgConfirmSignIn(t, page, frame)

	// a unique reader keeps runs isolated, and clearing here also makes a rerun against a kept
	// stack deterministic if the same E2E_RUN_ID and process id are reused
	status, body := pageFetch(t, page, http.MethodDelete, telegramURL+"/api/v1/telegram?site=remark", nil)
	require.Equal(t, http.StatusOK, status, "could not clear a telegram subscription left by an earlier run: %s", body)
	clearTelegramSubscriptionSession(t, page)

	frame = reload(t, page)
	firstToken := openTelegramSubscription(t, frame)
	confirmTelegramSubscription(t, page, frame, firstToken, readerID)

	// remove the panel state and reload, so the subscribed view comes from the backend's 409 and
	// not from the step the component kept in session storage
	clearTelegramSubscriptionSession(t, page)
	frame = reload(t, page)

	require.NoError(t, frame.Locator(`[title="Subscribe by Telegram"]`).Click())
	unsubscribe := frame.Locator(`button:text-is("Unsubscribe")`)
	waitVisible(t, unsubscribe)

	resp, err := page.ExpectResponse("**/api/v1/telegram**", func() error {
		return unsubscribe.Click()
	}, playwright.PageExpectResponseOptions{Timeout: playwright.Float(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err, "clicking Unsubscribe asked the server nothing")
	assert.Equal(t, http.MethodDelete, resp.Request().Method())
	assert.Equal(t, http.StatusOK, resp.Status(), "the server refused the telegram unsubscribe")
	waitVisible(t, frame.Locator(`text=You have been unsubscribed by telegram to updates`))

	require.NoError(t, frame.Locator(`button:text-is("Resubscribe")`).Click())
	secondToken := telegramSubscriptionToken(t, frame)
	require.NotEqual(t, firstToken, secondToken, "resubscribe reused the one-time token already consumed")
	confirmTelegramSubscription(t, page, frame, secondToken, readerID)
}

// TestTelegramSub_StartFailureIsShown covers the initial request, where no TelegramLink exists to
// carry its error. The dropdown must leave a diagnosis instead of becoming an empty panel.
func TestTelegramSub_StartFailureIsShown(t *testing.T) {
	page := newPage(t)
	frame := openURL(t, page, threadURLOn(t, telegramURL))

	loginToken := tgStartSignIn(t, frame)
	tgSendToBot(t, "/start "+loginToken, tgReaderID(t), "Error Reader")
	tgConfirmSignIn(t, page, frame)

	require.NoError(t, page.Route("**/api/v1/telegram/subscribe**", func(route playwright.Route) {
		if err := route.Fulfill(playwright.RouteFulfillOptions{
			Status:      playwright.Int(http.StatusInternalServerError),
			ContentType: playwright.String("application/json"),
			Body:        playwright.String(`{"error":"failed"}`),
		}); err != nil {
			t.Errorf("fulfill Telegram subscription failure: %v", err)
		}
	}))

	require.NoError(t, frame.Locator(`[title="Subscribe by Telegram"]`).Click())
	errorMessage := frame.Locator(`.auth-error`)
	waitVisible(t, errorMessage)
	text, err := errorMessage.TextContent()
	require.NoError(t, err)
	assert.Contains(t, strings.ToLower(text), "something went wrong")
}

func clearTelegramSubscriptionSession(t *testing.T, page playwright.Page) {
	t.Helper()
	_, err := page.Evaluate(`() => {
		sessionStorage.removeItem('telegram-subscription-step');
		sessionStorage.removeItem('telegram-subscription-telegram');
	}`)
	require.NoError(t, err)
}

func openTelegramSubscription(t *testing.T, frame playwright.FrameLocator) string {
	t.Helper()
	require.NoError(t, frame.Locator(`[title="Subscribe by Telegram"]`).Click())
	return telegramSubscriptionToken(t, frame)
}

func telegramSubscriptionToken(t *testing.T, frame playwright.FrameLocator) string {
	t.Helper()
	link := frame.Locator(`.telegram a`).First()
	waitVisible(t, link)
	href, err := link.GetAttribute("href")
	require.NoError(t, err)
	return telegramStartToken(t, href)
}

func confirmTelegramSubscription(
	t *testing.T, page playwright.Page, frame playwright.FrameLocator, token, readerID string,
) {
	t.Helper()
	before := len(tgSentMessages(t))
	tgSendToBot(t, "/start "+token, readerID, "Subscription Reader")
	tgWaitForMessage(t, before, "successfully subscribed")

	resp, err := page.ExpectResponse("**/api/v1/telegram/subscribe**", func() error {
		return frame.Locator(`button:text-is("Check")`).Click()
	}, playwright.PageExpectResponseOptions{Timeout: playwright.Float(float64(waitTimeout.Milliseconds()))})
	require.NoError(t, err, "clicking Check asked the server nothing")
	require.Equal(t, http.StatusOK, resp.Status(), "the server refused a subscription token the bot confirmed")
	waitVisible(t, frame.Locator(`text=You have been subscribed on updates by telegram`))
}
