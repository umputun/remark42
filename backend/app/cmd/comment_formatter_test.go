package cmd

import (
	"bytes"
	stdimage "image"
	"image/png"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerCommand_makeCommentFormatterNonASCIIHost formats a comment the way the server does on an instance
// whose host name is not ASCII. The comment text is sanitized before the converters see it, which percent-encodes
// that host in image sources, so remark42's own pictures must still be told from external images, sized and found.
func TestServerCommand_makeCommentFormatterNonASCIIHost(t *testing.T) {
	cmd := ServerCommand{RemarkURL: "https://комменты.example.com"}
	cmd.Image.Type = "fs"
	cmd.Image.FS.Path, cmd.Image.FS.Staging, cmd.Image.FS.Partitions = t.TempDir(), t.TempDir(), 100
	cmd.Image.MaxSize, cmd.Image.ResizeWidth, cmd.Image.ResizeHeight = 5000000, 2400, 900
	cmd.ImageProxy.CacheExternal = true
	imageService, err := cmd.makePicturesStore()
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, stdimage.NewRGBA(stdimage.Rect(0, 0, 40, 20))))
	id, err := imageService.Save("dev", &buf)
	require.NoError(t, err)

	_, formatter := cmd.makeCommentFormatter(imageService)
	text := formatter.FormatText("![](https://комменты.example.com/api/v1/picture/"+id+") ![](https://images.example/a.png)", false)

	assert.Equal(t, 1, strings.Count(text, "/api/v1/img?src="), "only the external image is proxied: %s", text)
	assert.Contains(t, text, `width="40"`, "the own picture is sized: %s", text)
	assert.Equal(t, []string{id}, imageService.ExtractNonProxiedPictures(text), "the own picture is found")
	assert.Len(t, imageService.ExtractPictures(text), 2, "the proxied image is found as well")
}

// TestServerCommand_makeCommentFormatterFetchExternal checks that the image store fetches the external images of
// posted comments only when they are cached: with the proxy used only to serve http images over https, the stored
// text proxies them too, and nothing may be stored for that
func TestServerCommand_makeCommentFormatterFetchExternal(t *testing.T) {
	for _, tt := range []struct {
		name               string
		cache, http2https bool
	}{{"cache external", true, false}, {"http to https only", false, true}, {"both", true, true}} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := ServerCommand{RemarkURL: "https://remark42.example.com"}
			cmd.Image.Type = "fs"
			cmd.Image.FS.Path, cmd.Image.FS.Staging, cmd.Image.FS.Partitions = t.TempDir(), t.TempDir(), 100
			cmd.ImageProxy.CacheExternal, cmd.ImageProxy.HTTP2HTTPS = tt.cache, tt.http2https
			imageService, err := cmd.makePicturesStore()
			require.NoError(t, err)

			imgProxy, formatter := cmd.makeCommentFormatter(imageService)
			text := formatter.FormatText("![](http://images.example/a.png)", false)
			assert.Len(t, imageService.ExternalPictures(text), 1, "the stored text proxies the image: %s", text)
			assert.Equal(t, tt.cache, imageService.FetchExternal != nil, "fetches the images of posted comments")
			assert.Equal(t, tt.cache, imgProxy.CacheExternal)
		})
	}
}
