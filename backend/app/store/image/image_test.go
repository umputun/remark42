package image

import (
	"bytes"
	"context"
	"image"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_Save(t *testing.T) {
	store := MockStore{}
	svc := Service{Store: &store}
	svc.MaxSize = 1500
	svc.MaxWidth, svc.MaxHeight = 32, 32

	store.On("Save", "user1", mock.Anything).Return("user1/test_id", nil)
	id, err := svc.Save("user1", gopherPNG())
	assert.NoError(t, err)
	assert.Equal(t, "user1/test_id", id)

	store.On("SaveWithID", "test_id", mock.Anything).Return("test_id", nil)
	id, err = svc.SaveWithID("test_id", gopherPNG())
	assert.NoError(t, err)
	assert.Equal(t, "test_id", id)
}

func TestService_Resize(t *testing.T) {
	img, err := readAndValidateImage(gopherPNG(), 1500)
	assert.NoError(t, err)
	assert.Equal(t, 1462, len(img))

	img = resize(img, 32, 32)
	assert.Equal(t, 1135, len(img))
}

func TestService_ResizeJpeg(t *testing.T) {
	fh, err := os.Open("testdata/circles.jpg")
	defer func() { assert.NoError(t, fh.Close()) }()
	assert.NoError(t, err)

	img, err := readAndValidateImage(fh, 32000)
	assert.NoError(t, err)
	assert.Equal(t, 23983, len(img))

	img = resize(img, 400, 300)
	assert.Equal(t, 10918, len(img))
}

func TestService_SaveTooLarge(t *testing.T) {
	svc := Service{ImageAPI: "/blah/"}
	svc.MaxSize = 2000
	_, err := svc.Save("user2", io.MultiReader(gopherPNG(), gopherPNG()))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is too large")
	_, err = svc.SaveWithID("test_id", io.MultiReader(gopherPNG(), gopherPNG()))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is too large")
}

func TestService_WrongFormat(t *testing.T) {
	svc := Service{ImageAPI: "/blah/"}

	_, err := svc.Save("user1", strings.NewReader("blah blah bad image"))
	assert.Error(t, err)
}

func TestService_SizeLimit(t *testing.T) {
	svc := Service{MaxSize: 666}

	size := svc.SizeLimit()
	assert.Equal(t, 666, size)
}

func TestService_ExtractPictures(t *testing.T) {
	svc := Service{ImageAPI: "/blah/"}
	html := `blah <img src="/blah/user1/pic1.png"/> foo 
<img src="/blah/user2/pic3.png"/> xyz <p>123</p> <img src="/pic3.png"/> <img src="https://i.ibb.co/0cqqqnD/ezgif-5-3b07b6b97610.png" alt="">`
	ids, err := svc.ExtractPictures(html)
	require.NoError(t, err)
	require.Equal(t, 2, len(ids), "two images")
	assert.Equal(t, "user1/pic1.png", ids[0])
	assert.Equal(t, "user2/pic3.png", ids[1])
}

func TestService_ExtractPictures2(t *testing.T) {
	svc := Service{ImageAPI: "https://remark42.radio-t.com/api/v1/picture/"}
	html := "<p>TLDR: такое в go пока правильно посчитать трудно. То, что они считают это общее количество go packages в коде." +
		"</p>\n\n<p>Пакеты в го это средство организации кода, они могут быть связанны друг с другом в рамках одной библиотеки (модуля). Например одна из моих вот так выглядит на libraries.io:</p>\n\n<p><img src=\"https://remark42.radio-t.com/api/v1/picture/github_ef0f706a79cc24b17bbbb374cd234a691d034128/bjttt8ahajfmrhsula10.png\" alt=\"bjtr0-201906-08110846-i324c.png\"/></p>\n\n<p>По форме все верно, это все packages, но по сути это все одна библиотека организованная таким образом. При ее импорте, например посредством go mod, она выглядит как один модуль, т.е. <code>github.com/go-pkgz/auth v0.5.2</code>.</p>\n"
	ids, err := svc.ExtractPictures(html)
	require.NoError(t, err)
	require.Equal(t, 1, len(ids), "one image in")
	assert.Equal(t, "github_ef0f706a79cc24b17bbbb374cd234a691d034128/bjttt8ahajfmrhsula10.png", ids[0])
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

	// reader is nil
	resized := resize(nil, 100, 100)
	assert.Nil(t, resized)

	// negative limit error
	resized = resize([]byte("some picture bin data"), -1, -1)
	require.NotNil(t, resized)
	assert.Equal(t, resized, []byte("some picture bin data"))

	// decode error
	resized = resize([]byte("invalid image content"), 100, 100)
	assert.NotNil(t, resized)
	assert.Equal(t, resized, []byte("invalid image content"))

	cases := []struct {
		file   string
		wr, hr int
	}{
		{"testdata/circles.png", 400, 300}, // full size: 800x600 px
		{"testdata/circles.jpg", 300, 400}, // full size: 600x800 px
	}

	for _, c := range cases {
		img, err := ioutil.ReadFile(c.file)
		require.NoError(t, err, "can't open test file %s", c.file)

		// no need for resize, image dimensions are smaller than resize limit
		resized = resize(img, 800, 800)
		assert.NotNil(t, resized, "file %s", c.file)
		assert.Equal(t, resized, img)

		// resizing to half of width
		resized = resize(img, 400, 400)
		assert.NotNil(t, resized, "file %s", c.file)
		imgRz, format, err := image.Decode(bytes.NewBuffer(resized))
		assert.NoError(t, err, "file %s", c.file)
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
