package image

import (
	"bytes"
	"context"
	"image"
	"io/ioutil"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	store := MockStore{}
	store.On("Cleanup", mock.Anything, mock.Anything).Times(10).Return(nil)

	svc := Service{Store: &store, TTL: 100 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*549)
	defer cancel()
	svc.Cleanup(ctx)
	store.AssertNumberOfCalls(t, "Cleanup", 10)
}

func TestService_Submit(t *testing.T) {
	store := MockStore{}
	store.On("Commit", mock.Anything, mock.Anything).Times(5).Return(nil)
	svc := Service{Store: &store, ImageAPI: "/blah/", TTL: time.Millisecond * 100}
	svc.Submit(func() []string { return []string{"id1", "id2", "id3"} })
	svc.Submit(func() []string { return []string{"id4", "id5"} })
	svc.Submit(nil)
	store.AssertNumberOfCalls(t, "Commit", 0)
	time.Sleep(time.Millisecond * 150)
	store.AssertNumberOfCalls(t, "Commit", 5)
}

func TestService_Close(t *testing.T) {
	store := MockStore{}
	store.On("Commit", mock.Anything, mock.Anything).Times(5).Return(nil)
	svc := Service{Store: &store, ImageAPI: "/blah/", TTL: time.Millisecond * 500}
	svc.Submit(func() []string { return []string{"id1", "id2", "id3"} })
	svc.Submit(func() []string { return []string{"id4", "id5"} })
	svc.Submit(nil)
	svc.Close()
	store.AssertNumberOfCalls(t, "Commit", 5)
}

func TestService_SubmitDelay(t *testing.T) {
	store := MockStore{}
	store.On("Commit", mock.Anything, mock.Anything).Times(5).Return(nil)
	svc := Service{Store: &store, ImageAPI: "/blah/", TTL: time.Millisecond * 100}
	svc.Submit(func() []string { return []string{"id1", "id2", "id3"} })
	time.Sleep(150 * time.Millisecond) // let first batch to pass TTL
	svc.Submit(func() []string { return []string{"id4", "id5"} })
	svc.Submit(nil)
	store.AssertNumberOfCalls(t, "Commit", 3)
	svc.Close()
	store.AssertNumberOfCalls(t, "Commit", 5)
}

func TestService_resize(t *testing.T) {

	// Reader is nil.
	resized, ok := resize(nil, 100, 100)
	assert.Nil(t, resized)
	assert.False(t, ok)

	// Negative limit error.
	resized, ok = resize([]byte("some picture bin data"), -1, -1)
	require.NotNil(t, resized)
	assert.Equal(t, resized, []byte("some picture bin data"))
	assert.False(t, ok)

	// Decode error.
	resized, ok = resize([]byte("invalid image content"), 100, 100)
	assert.NotNil(t, resized)
	assert.Equal(t, resized, []byte("invalid image content"))
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
		resized, ok = resize(img, 800, 800)
		assert.NotNil(t, resized, "file %s", c.file)
		assert.Equal(t, resized, img)
		assert.False(t, ok)

		// Resizing to half of width. Check resized image format PNG.
		resized, ok = resize(img, 400, 400)
		assert.NotNil(t, resized, "file %s", c.file)
		assert.True(t, ok)

		imgRz, format, err := image.Decode(bytes.NewBuffer(resized))
		assert.Nil(t, err, "file %s", c.file)
		assert.Equal(t, "png", format, "file %s", c.file)
		bounds := imgRz.Bounds()
		assert.Equal(t, c.wr, bounds.Dx(), "file %s", c.file)
		assert.Equal(t, c.hr, bounds.Dy(), "file %s", c.file)
	}

}

func TestGetProportionalSizes(t *testing.T) {
	tbl := []struct {
		inpW, inpH     int
		limitW, limitH int
		resW, resH     int
	}{
		{10, 20, 50, 25, 10, 20},
		{400, 200, 50, 25, 50, 25},
		{100, 100, 50, 25, 25, 25},
		{100, 200, 50, 25, 12, 25},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			resW, resH := getProportionalSizes(tt.inpW, tt.inpH, tt.limitW, tt.limitH)
			assert.Equal(t, tt.resW, resW, "width")
			assert.Equal(t, tt.resH, resH, "height")
		})
	}
}
