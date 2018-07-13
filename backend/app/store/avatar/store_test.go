package avatar

import (
	"bytes"
	"image"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAvatarStore_resize(t *testing.T) {
	checkC := func(t *testing.T, r io.Reader, cExp []byte) {
		content, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, cExp, content)
	}

	// Reader is nil.
	resizedR := resize(nil, 100)
	// assert.EqualError(t, err, "limit should be greater than 0")
	assert.Nil(t, resizedR)

	// Negative limit error.
	resizedR = resize(strings.NewReader("some picture bin data"), -1)
	require.NotNil(t, resizedR)
	checkC(t, resizedR, []byte("some picture bin data"))

	// Decode error.
	resizedR = resize(strings.NewReader("invalid image content"), 100)
	assert.NotNil(t, resizedR)
	checkC(t, resizedR, []byte("invalid image content"))

	cases := []struct {
		file   string
		wr, hr int
	}{
		{"testdata/circles.png", 400, 300}, // full size: 800x600 px
		{"testdata/circles.jpg", 300, 400}, // full size: 600x800 px
	}

	for _, c := range cases {
		img, err := ioutil.ReadFile(c.file)
		require.Nil(t, err, "can't open test file %s", c.file)

		// No need for resize, avatar dimensions are smaller than resize limit.
		resizedR = resize(bytes.NewReader(img), 800)
		assert.NotNilf(t, resizedR, "file %s", c.file)
		checkC(t, resizedR, img)

		// Resizing to half of width. Check resizedR avatar format PNG.
		resizedR = resize(bytes.NewReader(img), 400)
		assert.NotNilf(t, resizedR, "file %s", c.file)

		imgRz, format, err := image.Decode(resizedR)
		assert.Nilf(t, err, "file %s", c.file)
		assert.Equalf(t, "png", format, "file %s", c.file)
		bounds := imgRz.Bounds()
		assert.Equalf(t, c.wr, bounds.Dx(), "file %s", c.file)
		assert.Equalf(t, c.hr, bounds.Dy(), "file %s", c.file)
	}
}
