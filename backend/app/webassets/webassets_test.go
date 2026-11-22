package webassets

import (
	"encoding/json"
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFS_Contents(t *testing.T) {
	entries, err := fs.ReadDir(FS, ".")
	require.NoError(t, err)

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		assert.False(t, e.IsDir(), "the assets are served flat under /web, %s is a directory", e.Name())
		info, err := e.Info()
		require.NoError(t, err)
		assert.NotZero(t, info.Size(), "%s is empty", e.Name())
		names = append(names, e.Name())
	}
	assert.Equal(t, []string{"400x400.jpeg", "apple-dark.svg", "apple-light.svg", "comments.html", "counter.html",
		"counter.mjs", "custom.svg", "deleteme.html", "dev.svg", "discord.svg", "emoji.json", "facebook.svg", "ghost.svg",
		"github-dark.svg", "github-light.svg", "google.svg", "iframe.html", "index.html", "last-comments.html", "markdown-help.html",
		"microsoft.svg", "patreon.svg", "privacy.html", "telegram.svg", "twitter-dark.svg", "twitter-light.svg",
		"yandex.svg"}, names)
}

// TestFS_ContentShape catches an asset that has been truncated or replaced by something of the
// wrong kind, which a size check alone lets through.
func TestFS_ContentShape(t *testing.T) {
	tbl := []struct {
		name   string
		prefix []byte
		want   string
	}{
		{name: "400x400.jpeg", prefix: []byte{0xff, 0xd8, 0xff}},
		{name: "comments.html", want: "<!DOCTYPE html>"},
		{name: "counter.html", want: "<!DOCTYPE html>"},
		{name: "deleteme.html", want: "<!DOCTYPE html>"},
		{name: "iframe.html", want: "<!DOCTYPE html>"},
		{name: "index.html", want: "<!DOCTYPE html>"},
		{name: "last-comments.html", want: "<!DOCTYPE html>"},
		{name: "markdown-help.html", want: "<!DOCTYPE html>"},
		{name: "privacy.html", want: "<!DOCTYPE html>"},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			b, err := fs.ReadFile(FS, tt.name)
			require.NoError(t, err)

			if tt.prefix != nil {
				require.GreaterOrEqual(t, len(b), len(tt.prefix))
				assert.Equal(t, tt.prefix, b[:len(tt.prefix)], "not a JPEG")
				return
			}
			assert.True(t, strings.HasPrefix(strings.TrimSpace(string(b)), tt.want), "not an HTML document")
			assert.Contains(t, string(b), "</html>", "the document is truncated")
		})
	}
}

// TestFS_SVGShape does the same for the images the widget loads by name: the ghost avatar and the
// sign-in provider icons. A file that is not a whole SVG document shows as a broken image.
func TestFS_SVGShape(t *testing.T) {
	names, err := fs.Glob(FS, "*.svg")
	require.NoError(t, err)
	require.Len(t, names, 16, "the ghost avatar and fifteen provider icons")

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			b, err := fs.ReadFile(FS, name)
			require.NoError(t, err)
			s := strings.TrimSpace(string(b))
			assert.True(t, strings.HasPrefix(s, "<svg"), "not an SVG document")
			assert.True(t, strings.HasSuffix(s, "</svg>"), "the document is truncated")
		})
	}
}

// TestFS_CounterIsAClassicScript keeps counter.mjs loadable under its legacy name: the server answers
// counter.js with the same bytes, and an integration that loads that name with a plain script tag
// gets a syntax error from any import, export or import.meta. It also checks the file carries the
// placeholder the server fills in, which is its address when a page sets no remark_config.host.
func TestFS_CounterIsAClassicScript(t *testing.T) {
	b, err := fs.ReadFile(FS, "counter.mjs")
	require.NoError(t, err)
	src := string(b)

	moduleSyntax := regexp.MustCompile(`(?m)^\s*(import|export)\b|\bimport\.meta\b|\bimport\s*\(`)
	assert.Empty(t, moduleSyntax.FindAllString(src, -1), "module syntax in counter.mjs")
	assert.Contains(t, src, "{% REMARK_URL %}")
	assert.Contains(t, src, "remark42__counter")
}

// TestFS_EmojiTable checks the table the comment form fetches for its emoji suggestions: a JSON
// object of name to emoji. A file that does not parse, or parses to something else, leaves the
// form with no suggestions and nothing else to show for it.
func TestFS_EmojiTable(t *testing.T) {
	b, err := fs.ReadFile(FS, "emoji.json")
	require.NoError(t, err)

	table := map[string]string{}
	require.NoError(t, json.Unmarshal(b, &table), "not a JSON object of strings")
	assert.Len(t, table, 1878)
	assert.Equal(t, "😄", table["smile"])
	for name, emoji := range table {
		assert.NotEmpty(t, name, "a name is empty")
		assert.NotEmpty(t, emoji, "%s has no emoji", name)
	}
}

// TestFS_RelativeReferencesResolve keeps the two standalone pages self-contained: every relative
// src and href they use has to name a sibling that ships alongside them, since nothing else supplies
// one. The widget document and the demo, counter, last-comments, deleteme and comments pages are left
// out on purpose: they load the widget's scripts and styles, which the frontend build supplies.
func TestFS_RelativeReferencesResolve(t *testing.T) {
	ref := regexp.MustCompile(`(?:src|href)="([^"]+)"`)
	external := regexp.MustCompile(`^(?:[a-z]+:|//|#|mailto:)`)

	for _, page := range []string{"markdown-help.html", "privacy.html"} {
		t.Run(page, func(t *testing.T) {
			b, err := fs.ReadFile(FS, page)
			require.NoError(t, err)

			for _, m := range ref.FindAllStringSubmatch(string(b), -1) {
				target := m[1]
				if external.MatchString(target) {
					continue
				}
				_, err := fs.Stat(FS, target)
				assert.NoError(t, err, "%s references %q, which ships nowhere", page, target)
			}
		})
	}
}

// TestFS_Robots keeps every page under /web out of search results. Each one is the same stock page on
// every installation, and comments.html takes any site and address in its query, so an indexed copy
// only multiplies one page under many hosts. The widget document is the exception: it carries
// indexifembedded as well, which lets Google keep the comments as part of the page that embeds them,
// and a plain noindex there would drop them.
func TestFS_Robots(t *testing.T) {
	robots := regexp.MustCompile(`<meta\s+name="robots"\s+content="([^"]*)"`)
	want := map[string]string{
		"comments.html":      "noindex",
		"counter.html":       "noindex",
		"deleteme.html":      "noindex",
		"iframe.html":        "noindex, indexifembedded",
		"index.html":         "noindex",
		"last-comments.html": "noindex",
		"markdown-help.html": "noindex",
		"privacy.html":       "noindex",
	}

	pages, err := fs.Glob(FS, "*.html")
	require.NoError(t, err)
	require.Len(t, pages, len(want), "a page was added or removed; decide its robots rule here")

	for _, page := range pages {
		t.Run(page, func(t *testing.T) {
			b, err := fs.ReadFile(FS, page)
			require.NoError(t, err)

			rules := robots.FindAllStringSubmatch(string(b), -1)
			require.Len(t, rules, 1, "%s needs exactly one robots meta tag", page)
			assert.Equal(t, want[page], rules[0][1])
		})
	}
}
