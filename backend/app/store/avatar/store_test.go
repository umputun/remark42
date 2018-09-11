package avatar

import (
	"bytes"
	"image"
	"io"
	"io/ioutil"
	"os"
	"sort"
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

func TestAvatarStore_Migrate(t *testing.T) {
	// prep localfs
	plocal := NewLocalFS("/tmp/avatars.test", 300)
	err := os.MkdirAll("/tmp/avatars.test", 0700)
	require.NoError(t, err)
	defer os.RemoveAll("/tmp/avatars.test")

	// prep gridfs
	pgfs, skip := prepGFStore(t)
	if skip {
		return
	}

	// write to localfs
	_, err = plocal.Put("user1", strings.NewReader("some picture bin data 1"))
	require.Nil(t, err)
	_, err = plocal.Put("user2", strings.NewReader("some picture bin data 2"))
	require.Nil(t, err)
	_, err = plocal.Put("user3", strings.NewReader("some picture bin data 3"))
	require.Nil(t, err)

	// migrate and check reported count
	count, err := Migrate(pgfs, plocal)
	require.NoError(t, err)
	assert.Equal(t, 3, count, "all 3 recs migrated")

	// list avatars
	l, err := pgfs.List()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(l), "3 avatars listed in destination store")
	sort.Strings(l)
	assert.Equal(t, []string{"0b7f849446d3383546d15a480966084442cd2193.image", "a1881c06eec96db9901c7bbfe41c42a3f08e9cb4.image", "b3daa77b4c04a9551b8781d03191fe098f325e67.image"}, l)

	// try to read one of migrated avatars
	r, size, err := pgfs.Get("0b7f849446d3383546d15a480966084442cd2193.image")
	assert.Nil(t, err)
	assert.Equal(t, 23, size)
	data, err := ioutil.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(t, "some picture bin data 3", string(data))
}
