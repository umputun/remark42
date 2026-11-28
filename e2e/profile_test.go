//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/mxschmitt/playwright-go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The profile overlay is a second document the widget opens against its own origin, and everything
// about it beyond "it opens" is covered by jest alone: the comment list, the pagination and the
// controls a reader is or is not offered. Those are the states a reader actually meets, and jsdom
// sees none of the layout, none of the real API and none of the iframe boundary they live behind.

// profileFrame opens the overlay from a signed-in widget and hands back its frame. The overlay is
// appended to the host page's body, outside the widget's container, so it is addressed through the
// page and not through the widget frame
func profileFrame(t *testing.T, page playwright.Page, frame playwright.FrameLocator) playwright.FrameLocator {
	t.Helper()

	require.NoError(t, frame.Locator(`[title="Open your profile"]`).Click())
	waitVisible(t, page.Locator(`iframe[src*="page=profile"]`).First())
	return page.FrameLocator(`iframe[src*="page=profile"]`)
}

// TestProfile_ListsTheReadersOwnCommentsAndPaginates covers the load-more control, which appears
// only past the page size and has to disappear once the last page arrives. Both halves matter: a
// control that never appears hides a reader's history, and one that never goes away asks for a page
// that does not exist.
//
// COMMENTS_LIMIT in profile.tsx is 10, so eleven comments is the smallest thread that pages
func TestProfile_ListsTheReadersOwnCommentsAndPaginates(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	signInAnon(t, page, frame, anonName("profilepager"))

	// COMMENTS_LIMIT in profile.tsx. One past it is the smallest thread that pages
	const pageSize = 10
	const total = pageSize + 1
	// the oldest is the one that can only arrive on the second page, so it is the marker the
	// assertions below turn on
	oldest := fmt.Sprintf("profile paging 0 %s", runID)
	for i := range total {
		postComment(t, frame, fmt.Sprintf("profile paging %d %s", i, runID))
	}

	overlay := profileFrame(t, page, frame)

	// exactly the page size, not at least it: an API that ignored limit and returned all eleven
	// would satisfy every ">=" here and still show a load-more control, so the whole case would
	// pass against pagination that is not happening
	eventually(t, waitTimeout, "the profile never showed a full first page", func() bool {
		n, err := overlay.Locator("article").Count()
		return err == nil && n == pageSize
	})

	// and it is a page of different comments, not the same one counted twice: the oldest cannot be
	// on it. A second fetch repeating the first, or returning somebody else's comments, satisfies
	// any assertion made on counts alone.
	//
	// A text filter is sound for an absence here only because profile.tsx renders every comment
	// eagerly. The thread does not, which is why comment() says the same filter cannot be trusted
	// there: an article below the fold renders empty under in-view and matches no text. Should the
	// profile adopt that too, this assertion passes without checking anything
	absent, err := overlay.Locator("article", playwright.FrameLocatorLocatorOptions{HasText: oldest}).Count()
	require.NoError(t, err)
	require.Zero(t, absent, "the oldest comment is on the first page, so the page size is not being applied")

	// the count beside the heading, which nothing else reads. The class is kept outside the css
	// modules for this; the unit suite selects the same element by its data-testid
	counter := overlay.Locator(".comments-counter").First()
	waitVisible(t, counter)
	eventually(t, waitTimeout, "the profile never showed the number of comments the reader has", func() bool {
		shown, err := counter.TextContent()
		return err == nil && strings.TrimSpace(shown) == strconv.Itoa(total)
	})

	loadMore := overlay.Locator(`button:has-text("Load more")`)
	waitVisible(t, loadMore)

	require.NoError(t, loadMore.Click())

	// the eleventh arrives and the control goes, since nothing is left to fetch
	eventually(t, waitTimeout, "the profile did not load the last page", func() bool {
		n, err := overlay.Locator("article").Count()
		return err == nil && n == total
	})

	// the second page brought the one the first could not hold
	waitVisible(t, overlay.Locator("article", playwright.FrameLocatorLocatorOptions{HasText: oldest}).First())

	// the control must be gone. While a fetch is in flight it is replaced by a spinner, so "not
	// visible" is briefly true for a reason that has nothing to do with the last page
	eventually(t, waitTimeout, "the load-more control survived the last page, so it asks for comments that do not exist",
		func() bool {
			n, err := loadMore.Count()
			return err == nil && n == 0
		})
}

// TestProfile_AnonymousReaderIsNotOfferedRemoval covers the one control whose absence is the
// feature. Removal deletes every comment a reader has written, and an anonymous identity is derived
// from the name alone, so the next reader to pick that name would be handed the button.
//
// The signed-in half is the control: without it a selector that stopped matching would report the
// button as absent for everyone and pass
func TestProfile_AnonymousReaderIsNotOfferedRemoval(t *testing.T) {
	t.Parallel()

	removal := `button:has-text("Request deletion of my data")`

	t.Run("anonymous", func(t *testing.T) {
		page := newPage(t)
		frame := openThread(t, page)
		signInAnon(t, page, frame, anonName("profileanon"))
		postComment(t, frame, "profile anon "+runID)

		overlay := profileFrame(t, page, frame)
		waitVisible(t, overlay.Locator("article").First())

		n, err := overlay.Locator(removal).Count()
		require.NoError(t, err)
		assert.Zero(t, n, "an anonymous reader was offered data removal, which deletes every comment "+
			"written under a name anyone can claim")
	})

	t.Run("registered", func(t *testing.T) {
		page := newPage(t)
		frame := openThread(t, page)
		signInDev(t, page, frame)
		postComment(t, frame, "profile dev "+runID)

		overlay := profileFrame(t, page, frame)
		waitVisible(t, overlay.Locator("article").First())

		waitVisible(t, overlay.Locator(removal))
	})
}

// TestProfile_LoadingAndFailureStates covers the two states the profile shows instead of a list,
// which nothing else reaches: every other case answers at once and succeeds, so a profile that
// never says it is working, or that swallows a failure and shows an empty list, passes them all.
func TestProfile_LoadingAndFailureStates(t *testing.T) {
	t.Parallel()

	page := newPage(t)
	frame := openThread(t, page)
	signInAnon(t, page, frame, anonName("profilestates"))
	postComment(t, frame, "profile states "+runID)

	// held open so the loading state can be read, then failed so the error state follows
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()

	// a regexp and not a glob: `?` is a single-character wildcard in playwright's url matching, so
	// a pattern written with the query string never matches the request it names
	profileList := regexp.MustCompile(`/api/v1/comments\?.*user=`)
	require.NoError(t, page.Route(profileList, func(route playwright.Route) {
		<-release
		fulfillJSON(t, route, http.StatusInternalServerError, `{"error":"failed"}`)
	}))

	overlay := profileFrame(t, page, frame)

	// while the list is out: the preloader, and none of what a settled profile shows
	waitVisible(t, overlay.Locator(".preloader"))
	waitHidden(t, overlay.Locator(`button:has-text("Retry")`),
		"the profile offered a retry before anything had failed")
	waitHidden(t, overlay.Locator(".comments-counter"),
		"the profile showed a comment count before the list arrived")
	waitHidden(t, overlay.Locator(`button:has-text("Load more")`),
		"the profile offered another page before the first had arrived")
	waitHidden(t, overlay.Locator("h3.profile-title"),
		"the profile showed its comments heading before the list arrived")

	unblock()

	// after it fails: the message and a way to try again, and still no count
	waitVisible(t, overlay.Locator(".profile-error"))
	waitVisible(t, overlay.Locator(`button:has-text("Retry")`))
	waitHidden(t, overlay.Locator(".preloader"),
		"the profile was still loading after the request had failed")
	waitHidden(t, overlay.Locator(".comments-counter"),
		"the profile showed a comment count after the list failed to load")
	waitHidden(t, overlay.Locator(`button:has-text("Load more")`),
		"the profile offered another page after the list failed to load")
	waitHidden(t, overlay.Locator("h3.profile-title"),
		"the profile showed its comments heading after the list failed to load")
}

// TestProfile_PageLearnsNothingOfTheReader covers the messages the widget posts to open a profile,
// which any page framing it reads. Opening one's own profile has to name nobody, and opening the
// profile of the author of a comment has to leave out whether that author is the reader. The
// profile frame still recognizes the reader in both, by asking the widget frame, so each half also
// requires the sign-out the reader's own profile offers: without it, a widget that sent nothing at
// all would pass
func TestProfile_PageLearnsNothingOfTheReader(t *testing.T) {
	t.Parallel()

	t.Run("own profile from the auth panel", func(t *testing.T) {
		t.Parallel()

		page, frame, _ := quietReader(t, threadURL(t), "profilequietpanel")
		require.NoError(t, frame.Locator(`[title="Open your profile"]`).Click())
		waitVisible(t, profileSignout(page))

		messages := profileMessages(t, page)
		require.NotEmpty(t, messages, "the page saw no profile message, so nothing here was checked")
		for _, m := range messages {
			for _, key := range []string{"id", "name", "picture", "current"} {
				assert.NotContains(t, m, key, "the page was told the %s of the reader: %v", key, m)
			}
		}

		// the sidebar slides in once the widget frame has answered, and its close button is on it.
		// A sidebar that never slid in leaves the reader under a dimmed page with no way out
		require.NoError(t, page.FrameLocator(`iframe[src*="page=profile"]`).
			GetByTitle("Close profile").Click())
		waitHidden(t, page.Locator(`iframe[src*="page=profile"]`), "the reader's own profile did not close")
	})

	t.Run("own name above a comment", func(t *testing.T) {
		t.Parallel()

		page, frame, reader := quietReader(t, threadURL(t), "profilequietname")
		comment := frame.Locator("article").Filter(playwright.LocatorFilterOptions{HasText: "profile quiet " + reader})
		require.NoError(t, comment.Locator("button", playwright.LocatorLocatorOptions{HasText: reader}).First().Click())
		waitVisible(t, profileSignout(page))

		messages := profileMessages(t, page)
		require.NotEmpty(t, messages, "the page saw no profile message, so nothing here was checked")
		for _, m := range messages {
			assert.NotContains(t, m, "current", "the page was told the author is the reader: %v", m)
		}
	})

	// with AUTH_SEND_JWT_HEADER the widget frame keeps the session as a token in its memory. A
	// browser that blocks third-party cookies stores none for the frame, which is the case here once
	// the cookies are gone: the profile frame then has no session, and only the widget frame can say
	// who the reader is. The page is on another site, so the two frames find each other as frames
	// of a foreign page
	t.Run("own profile in header mode on another site", func(t *testing.T) {
		t.Parallel()

		page, frame, _ := quietReader(t, httpsThread(t, "profile-quiet-header"), "profilequietheader")
		require.NoError(t, page.Context().ClearCookies())

		require.NoError(t, frame.Locator(`[title="Open your profile"]`).Click())
		waitVisible(t, profileSignout(page))

		// read after the profile opened, since a response on the way could have set a cookie again
		cookies, err := page.Context().Cookies(httpsURL)
		require.NoError(t, err)
		assert.False(t, slices.ContainsFunc(cookies, func(c playwright.Cookie) bool { return c.Name == "JWT" }),
			"the profile frame had a cookie session of its own, so this case does not show the widget frame answering")
	})
}

// quietReader signs a new reader in, posts a comment as them and starts recording every profile
// message the widget posts to the page from then on
func quietReader(t *testing.T, thread, prefix string) (playwright.Page, playwright.FrameLocator, string) {
	t.Helper()

	page := newPage(t)
	frame := openURL(t, page, thread)
	reader := anonName(prefix)
	signInAnon(t, page, frame, reader)
	postComment(t, frame, "profile quiet "+reader)

	_, err := page.Evaluate(`() => {
		window.e2eMessages = [];
		window.addEventListener('message', (e) => {
			if (e.data && typeof e.data === 'object') window.e2eMessages.push(e.data);
		});
	}`)
	require.NoError(t, err)
	return page, frame, reader
}

// profileMessages hands back the profile messages the page has received since quietReader. The
// answer the widget frame gives the profile frame must not be among what the page received at all
func profileMessages(t *testing.T, page playwright.Page) []map[string]any {
	t.Helper()

	got, err := page.Evaluate(`() => window.e2eMessages`)
	require.NoError(t, err)
	var out []map[string]any
	for _, m := range got.([]any) {
		msg, ok := m.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range []string{"reader", "readerAnswer", "readerAsk"} {
			assert.NotContains(t, msg, key, "the page received the exchange between the frames: %v", msg)
		}
		if p, ok := msg["profile"].(map[string]any); ok {
			out = append(out, p)
		}
	}
	return out
}

// profileSignout is the sign-out the profile offers on the reader's own profile only
func profileSignout(page playwright.Page) playwright.Locator {
	return page.FrameLocator(`iframe[src*="page=profile"]`).Locator(".profile-signout")
}

// TestProfile_SlidesInAndClosesOnEscape opens the reader's own profile, waits for the sidebar to
// slide into the window, and closes the profile with Escape pressed inside it
func TestProfile_SlidesInAndClosesOnEscape(t *testing.T) {
	t.Parallel()

	page, frame, _ := quietReader(t, threadURL(t), "profileescape")
	profile := profileFrame(t, page, frame)
	sidebar := profile.Locator(".profile-sidebar")
	waitVisible(t, profileSignout(page))

	eventually(t, waitTimeout, "the sidebar did not slide into the window", func() bool {
		box, err := sidebar.BoundingBox()
		if err != nil || box == nil {
			return false
		}
		size := page.ViewportSize()
		return box.X >= 0 && box.X+box.Width <= float64(size.Width)+0.5
	})

	require.NoError(t, profile.Locator(".profile-id").Click())
	require.NoError(t, page.Keyboard().Press("Escape"))
	waitHidden(t, page.Locator(`iframe[src*="page=profile"]`), "Escape did not close the profile")
}
