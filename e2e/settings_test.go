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

// TestSettings_BlockedUserIsUnblockedAndBlockedAgain covers the blocked users list in the
// settings, which no other case opens: the admin finds a user blocked from a comment there, with
// the period of the block, unblocks them and blocks them again from the same row. The row stays
// in the list after the unblock and its action turns into the way back. The row flips before the
// server answers, so the case reads the requests themselves: each one for the user the comment
// blocked, and each one answered with success
func TestSettings_BlockedUserIsUnblockedAndBlockedAgain(t *testing.T) {
	t.Parallel()

	name := anonName("settingsblock")
	text := "blocked from the settings " + runID

	author := newPage(t)
	authorFrame := openThread(t, author)
	signInAnon(t, author, authorFrame, name)
	postComment(t, authorFrame, text)

	admin := newPage(t)
	frame := openURL(t, admin, threadURL(t))
	signInDev(t, admin, frame)
	admin.OnDialog(func(d playwright.Dialog) { _ = d.Accept() })

	// each request with the user it names and the status the server answered it with
	var mu sync.Mutex
	var sent []string
	admin.OnResponse(func(r playwright.Response) {
		u, err := neturl.Parse(r.URL())
		if err != nil || r.Request().Method() != http.MethodPut || !strings.HasPrefix(u.Path, "/api/v1/admin/user/") {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		sent = append(sent, fmt.Sprintf("%s block=%s ttl=%s %d", strings.TrimPrefix(u.Path, "/api/v1/admin/user/"),
			u.Query().Get("block"), u.Query().Get("ttl"), r.Status()))
	})
	requests := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), sent...)
	}

	moderation := actions(frame, text)
	_, err := moderation.Locator("select").SelectOption(playwright.SelectOptionValues{Values: &[]string{"1440m"}})
	require.NoError(t, err)
	waitVisible(t, moderation.Locator(`button:has-text("Unblock")`))

	require.NoError(t, frame.Locator(`button:has-text("Show settings")`).Click())
	blocked := frame.Locator(`[role="region"][aria-label="Blocked users"]`)
	waitVisible(t, blocked)
	// other cases block users on the same instance, so the row is found by this run's name
	row := blocked.Locator("li", playwright.LocatorLocatorOptions{HasText: name})
	waitVisible(t, row)
	shown, err := row.InnerText()
	require.NoError(t, err)
	assert.Contains(t, shown, "until", "a block for a day shows when it ends")

	press := func(action string, want int) {
		t.Helper()
		require.NoError(t, row.GetByText(action, playwright.LocatorGetByTextOptions{Exact: new(true)}).Click())
		eventually(t, waitTimeout, "the "+action+" press did not reach the server", func() bool {
			return len(requests()) == want
		})
	}
	press("unblock", 2)
	waitVisible(t, row.GetByText("block", playwright.LocatorGetByTextOptions{Exact: new(true)}))
	press("block", 3)
	waitVisible(t, row.GetByText("unblock", playwright.LocatorGetByTextOptions{Exact: new(true)}))

	got := requests()
	require.Len(t, got, 3)
	user := strings.Fields(got[0])[0]
	assert.Equal(t, []string{
		user + " block=1 ttl=1440m 200",
		user + " block=0 ttl= 200",
		user + " block=1 ttl= 200",
	}, got, "a block from the comment, then the unblock and the permanent block from the settings, all for that user")
}
