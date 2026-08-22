//go:build e2e

package e2e

import (
	"fmt"
	neturl "net/url"
	"strings"
	"testing"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// remarkURLPlaceholder is what the frontend build emits wherever the instance url belongs. The
// bundler cannot know that url, so every distribution fills the marker in: the docker image
// rewrites its web root at container start, the binary substitutes as it serves.
const remarkURLPlaceholder = "{% REMARK_URL %}"

// TestAssets_InstanceURLIsFilledIn covers the substitution, which nothing else here exercises.
// The demo pages set remark_config.host from location.origin themselves, so the compiled-in
// fallback is never read and the marker could survive into a release without a test noticing.
func TestAssets_InstanceURLIsFilledIn(t *testing.T) {
	page := newPage(t)

	pauseForAuthLimit()
	_, err := page.Goto(baseURL + "/web/")
	require.NoError(t, err)

	// every file type the distributions rewrite, not only the one the widget happens to load
	for _, name := range []string{"embed.mjs", "counter.mjs", "last-comments.mjs", "index.html"} {
		status, body := pageFetch(t, page, "GET", baseURL+"/web/"+name, nil)
		require.Equal(t, 200, status, name)
		assert.NotContains(t, body, remarkURLPlaceholder, "%s still carries the marker", name)
	}

	// and the value that replaced it is this instance, not some other host. only the bundles
	// carry it; index.html sets the host from the page's own origin
	status, body := pageFetch(t, page, "GET", baseURL+"/web/embed.mjs", nil)
	require.Equal(t, 200, status)
	assert.Contains(t, body, baseURL, "embed.mjs should fall back to this instance")
}

// TestAssets_WidgetRunsOnTheCompiledInURL is the behavior the substitution exists for.
//
// The widget document carries no host: iframe.html builds remark_config out of its own query
// string, and the parent never passes one, so everything the bundle requests is addressed with
// the url the build was substituted with. A marker left in place leaves the widget asking for
// `{% REMARK_URL %}/api/v1/config` and rendering nothing.
//
// The demo pages cannot show this. Their loader builds the bundle's own script url from
// remark_config.host, so a page without one never gets as far as loading the widget.
func TestAssets_WidgetRunsOnTheCompiledInURL(t *testing.T) {
	page := newPage(t)

	var configURL string
	page.OnRequest(func(r playwright.Request) {
		if configURL == "" && strings.Contains(r.URL(), "/api/v1/config") {
			configURL = r.URL()
		}
	})

	pauseForAuthLimit()
	_, err := page.Goto(fmt.Sprintf("%s/web/iframe.html?site_id=remark&url=%s",
		baseURL, neturl.QueryEscape(threadURL(t))))
	require.NoError(t, err)

	// the document really had no host of its own, or the fallback was never reached
	host, err := page.Evaluate(`() => window.remark_config && window.remark_config.host`)
	require.NoError(t, err)
	require.Nil(t, host, "the widget document should carry no host")

	// it rendered, so the url it was addressing resolved
	waitVisible(t, page.Locator(commentFormSel).First())

	eventually(t, waitTimeout, "the widget never asked for its config", func() bool {
		return configURL != ""
	})
	assert.True(t, strings.HasPrefix(configURL, baseURL),
		"the widget addressed %q rather than this instance", configURL)
	assert.NotContains(t, configURL, remarkURLPlaceholder)
}
