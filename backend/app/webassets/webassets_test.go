package webassets

import (
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
		"custom.svg", "deleteme.html", "dev.svg", "discord.svg", "facebook.svg", "ghost.svg", "github-dark.svg",
		"github-light.svg", "google.svg", "index.html", "last-comments.html", "markdown-help.html", "microsoft.svg",
		"patreon.svg", "privacy.html", "telegram.svg", "twitter-dark.svg", "twitter-light.svg", "yandex.svg"}, names)
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

// TestFS_RelativeReferencesResolve keeps the two standalone pages self-contained: every relative
// src and href they use has to name a sibling that ships alongside them, since nothing else supplies
// one. The demo, counter, last-comments, deleteme and comments pages are left out on purpose: they
// load the widget's scripts, which the frontend build supplies.
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
