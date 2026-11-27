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
