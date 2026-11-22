//go:build e2e

package e2e

import (
	neturl "net/url"
	"regexp"
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
	t.Parallel()

	page := newPage(t)

	_, err := page.Goto(baseURL + "/web/")
	require.NoError(t, err)

	// every file type the distributions rewrite, not only the one the widget happens to load
	for _, name := range []string{"embed.mjs", "counter.mjs", "last-comments.mjs", "index.html", "comments.html"} {
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

// TestAssets_WidgetAddressesTheInstanceThatServedIt pins where the widget document sends its
// requests. iframe.html builds remark_config out of its own query string, and any page can frame
// it with a query of its choosing. Every request the widget makes carries the reader's session
// and XSRF token, so the host must come from the address the document was served from: a host in
// the query ending in `#` or an open query would otherwise swallow the paths the widget appends
// and send all of them to the one endpoint it names.
//
// The no-host case is the plain document opened on its own. The prefixed case serves the instance
// under /remark42 on this origin, as a reverse proxy mounting it on a path would, so the address the
// document derives differs from the instance's REMARK_URL and the prefix has to survive.
func TestAssets_WidgetAddressesTheInstanceThatServedIt(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ name, prefix, host string }{
		{name: "no host"},
		{name: "host swallowing the path in a fragment", host: baseURL + "/api/v1/user?site=remark#"},
		{name: "host swallowing the path in a query", host: baseURL + "/api/v1/user?site=remark&x="},
		{name: "instance mounted under a path", prefix: "/remark42"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := newPage(t)
			instance := baseURL + tc.prefix

			if tc.prefix != "" {
				// strip the prefix on the way out, as the proxy would. Fallback keeps the context's
				// own route, which sets the reader address, in the chain
				mounted := regexp.MustCompile("^" + regexp.QuoteMeta(instance) + "/")
				require.NoError(t, page.Route(mounted, func(route playwright.Route) {
					target := baseURL + "/" + mounted.ReplaceAllString(route.Request().URL(), "")
					if err := route.Fallback(playwright.RouteFallbackOptions{URL: &target}); err != nil {
						t.Errorf("route %s to the instance: %v", route.Request().URL(), err)
					}
				}))
			}

			// the config request by its path, not by a substring: a swallowing host turns it into
			// /api/v1/user?site=remark&x=/api/v1/config?site=remark, which still contains the name
			var configURL string
			page.OnRequest(func(r playwright.Request) {
				if u, err := neturl.Parse(r.URL()); err == nil && configURL == "" && u.Path == tc.prefix+"/api/v1/config" {
					configURL = r.URL()
				}
			})

			query := neturl.Values{"site_id": {"remark"}, "url": {threadURL(t)}}
			if tc.host != "" {
				query.Set("host", tc.host)
			}
			_, err := page.Goto(instance + "/web/iframe.html?" + query.Encode())
			require.NoError(t, err)

			host, err := page.Evaluate(`() => window.remark_config && window.remark_config.host`)
			require.NoError(t, err)
			assert.Equal(t, instance, host, "the widget should address the instance that served it")

			// it rendered, so the url it was addressing resolved
			waitVisible(t, page.Locator(commentFormSel).First())

			eventually(t, waitTimeout, "the widget never asked this instance for its config", func() bool {
				return configURL != ""
			})
			assert.True(t, strings.HasPrefix(configURL, instance+"/api/v1/config?"),
				"the widget asked %q rather than this instance for its config", configURL)
		})
	}
}

// TestAssets_WidgetImagesActuallyLoad covers a class of defect the unit suite structurally cannot
// see. The ghost avatar and the oauth provider icons are files the server embeds from
// app/webassets, and the widget builds their URLs from the instance address. A unit test sees only
// the string it built, never whether the server answers it: a doubled origin such as
// `https://hosthttps://host/web/google.svg`, which once made these images 404 on every master
// build, or a file missing from the embedded set, both look right there.
//
// naturalWidth is the discriminator. A broken <img> still exists in the DOM with its src attribute
// intact, so asserting the element or the attribute proves nothing; only a decoded image has a
// non-zero natural size. That catches a 404, a malformed URL and a CSP refusal alike.
func TestAssets_WidgetImagesActuallyLoad(t *testing.T) {
	t.Parallel()

	page := newPage(t)

	_, err := page.Goto(threadURL(t))
	require.NoError(t, err)

	frame := page.FrameLocator("#remark42 iframe")

	// the auth panel is where the provider icons live, and they are the case that regressed
	require.NoError(t, frame.Locator(".auth-button").Click())
	waitVisible(t, frame.Locator(".auth-dropdown"))
	waitVisible(t, frame.Locator(".oauth-icon").First())

	report, err := frame.Locator(":root").Evaluate(`() => {
		const imgs = [...document.querySelectorAll('img')];
		return imgs.map(i => ({
			src: i.getAttribute('src') || '',
			loaded: i.complete && i.naturalWidth > 0,
		}));
	}`, nil)
	require.NoError(t, err)

	entries, ok := report.([]any)
	require.True(t, ok, "unexpected probe shape %T", report)
	require.NotEmpty(t, entries, "no images in the widget, so this asserts nothing")

	checked := 0
	for _, e := range entries {
		m, ok := e.(map[string]any)
		require.True(t, ok)
		src, _ := m["src"].(string)
		loaded, _ := m["loaded"].(bool)
		if src == "" {
			continue
		}
		checked++

		// the doubling is worth naming separately, because "did not load" alone sends the reader
		// looking at the file rather than at the url that asked for it
		assert.LessOrEqual(t, strings.Count(src, "://"), 1,
			"image url carries more than one origin, so something prefixed an absolute asset url: %s", src)
		assert.True(t, loaded, "image did not load: %s", src)
	}
	require.NotZero(t, checked, "every image had an empty src, so nothing was checked")
}
