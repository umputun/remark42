package image

import (
	"bytes"
	"context"
	"image"
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_ExtractPictures(t *testing.T) {
	svc := Service{ImageAPI: "/blah/"}
	html := `blah <img src="/blah/user1/pic1.png"/> foo 
<img src="/blah/user2/pic3.png"/> xyz <p>123</p> <img src="/pic3.png"/>`
	ids, err := svc.ExtractPictures(html)
	require.NoError(t, err)
	assert.Equal(t, 2, len(ids), "two images")
	assert.Equal(t, "user1/pic1.png", ids[0])
	assert.Equal(t, "user2/pic3.png", ids[1])
}

func TestService_Cleanup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := NewMockStore(ctrl)
	store.EXPECT().Cleanup(gomock.Any(), gomock.Any()).Times(10)

	svc := Service{Store: store, TTL: 100 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*549)
	defer cancel()
	svc.Cleanup(ctx)
}

func TestService_Submit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := NewMockStore(ctrl)

	store.EXPECT().Commit(gomock.Any()).Times(5) // all 5 should be committed
	svc := Service{Store: store, ImageAPI: "/blah/", TTL: time.Millisecond * 100}
	svc.Submit(func() []string { return []string{"id1", "id2", "id3"} })
	svc.Submit(func() []string { return []string{"id4", "id5"} })
	svc.Submit(nil)
	time.Sleep(time.Millisecond * 500)
}

func TestService_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := NewMockStore(ctrl)

	store.EXPECT().Commit(gomock.Any()).Times(5) // all 5 should be committed
	svc := Service{Store: store, ImageAPI: "/blah/", TTL: time.Millisecond * 500}
	svc.Submit(func() []string { return []string{"id1", "id2", "id3"} })
	svc.Submit(func() []string { return []string{"id4", "id5"} })
	svc.Submit(nil)
	svc.Close()
}

func TestService_SubmitDelay(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		ctrl.Finish()
	}()

	store := NewMockStore(ctrl)

	store.EXPECT().Commit(gomock.Any()).Times(3) // first batch should be committed
	svc := Service{Store: store, ImageAPI: "/blah/", TTL: time.Millisecond * 100}
	svc.Submit(func() []string { return []string{"id1", "id2", "id3"} })
	time.Sleep(150 * time.Millisecond) // let first batch to pass TTL
	svc.Submit(func() []string { return []string{"id4", "id5"} })
	svc.Submit(nil)
}

func TestService_resize(t *testing.T) {

	checkC := func(t *testing.T, r io.Reader, cExp []byte) {
		content, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		assert.Equal(t, cExp, content)
	}

	// Reader is nil.
	resizedR, ok := resize(nil, 100)
	assert.Nil(t, resizedR)
	assert.False(t, ok)

	// Negative limit error.
	resizedR, ok = resize(strings.NewReader("some picture bin data"), -1)
	require.NotNil(t, resizedR)
	checkC(t, resizedR, []byte("some picture bin data"))
	assert.False(t, ok)

	// Decode error.
	resizedR, ok = resize(strings.NewReader("invalid image content"), 100)
	assert.NotNil(t, resizedR)
	checkC(t, resizedR, []byte("invalid image content"))
	assert.False(t, ok)

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

		// No need for resize, image dimensions are smaller than resize limit.
		resizedR, ok = resize(bytes.NewReader(img), 800)
		assert.NotNil(t, resizedR, "file %s", c.file)
		checkC(t, resizedR, img)
		assert.False(t, ok)

		// Resizing to half of width. Check resizedR image format PNG.
		resizedR, ok = resize(bytes.NewReader(img), 400)
		assert.NotNil(t, resizedR, "file %s", c.file)
		assert.True(t, ok)

		imgRz, format, err := image.Decode(resizedR)
		assert.Nil(t, err, "file %s", c.file)
		assert.Equal(t, "png", format, "file %s", c.file)
		bounds := imgRz.Bounds()
		assert.Equal(t, c.wr, bounds.Dx(), "file %s", c.file)
		assert.Equal(t, c.hr, bounds.Dy(), "file %s", c.file)
	}

}
