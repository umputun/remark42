package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
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

func TestService_SaveAndLoad(t *testing.T) {
	store := MockStore{}
	svc := NewService(&store, ServiceParams{MaxSize: 1500, MaxWidth: 32, MaxHeight: 32})

	store.On("Save", "test_id", mock.Anything).Return(nil)
	err := svc.SaveWithID("test_id", gopherPNG())
	assert.NoError(t, err)

	store.On("Load", "test_id", mock.Anything).Return(nil, nil)
	img, err := svc.Load("test_id")
	assert.NoError(t, err)
	assert.Nil(t, img)
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
	svc := Service{ServiceParams: ServiceParams{ImageAPI: "/blah/"}}
	svc.MaxSize = 2000
	_, err := svc.Save("user2", io.MultiReader(gopherPNG(), gopherPNG()))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is too large")
	err = svc.SaveWithID("test_id", io.MultiReader(gopherPNG(), gopherPNG()))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is too large")
}

func TestService_WrongFormat(t *testing.T) {
	svc := Service{ServiceParams: ServiceParams{ImageAPI: "/blah/"}}

	_, err := svc.Save("user1", strings.NewReader("blah blah bad image"))
	assert.Error(t, err)
}

func TestService_ExtractPictures(t *testing.T) {
	svc := Service{ServiceParams: ServiceParams{ImageAPI: "/blah/", ProxyAPI: "/non_existent"}}
	html := `blah <img src="/blah/user1/pic1.png"/> foo
<img src="/blah/user2/pic3.png"/> xyz <p>123</p> <img src="/pic3.png"/> <img src="https://i.ibb.co/0cqqqnD/ezgif-5-3b07b6b97610.png" alt="">`
	ids, err := svc.ExtractPictures(html)
	require.NoError(t, err)
	require.Equal(t, 2, len(ids), "two images")
	assert.Equal(t, "user1/pic1.png", ids[0])
	assert.Equal(t, "user2/pic3.png", ids[1])

	svc = Service{ServiceParams: ServiceParams{ImageAPI: "https://remark42.radio-t.com/api/v1/picture/", ProxyAPI: "https://remark42.radio-t.com/api/v1/img"}}
	html = `<p>TLDR: такое в go пока правильно посчитать трудно. То, что они считают это общее количество go packages в коде.
</p>\n\n<p>Пакеты в го это средство организации кода, они могут быть связанны друг с другом в рамках одной библиотеки (модуля).
Например одна из моих вот так выглядит на libraries.io:</p>\n\n
<p><img src="https://remark42.radio-t.com/api/v1/picture/github_ef0f706a79cc24b17bbbb374cd234a691d034128/bjttt8ahajfmrhsula10.png" alt="bjtr0-201906-08110846-i324c.png"/></p>\n\n<p>
По форме все верно, это все packages, но по сути это все одна библиотека организованная таким образом. При ее импорте, например посредством go mod, она выглядит как один модуль, т.е.
<code>github.com/go-pkgz/auth v0.5.2</code>.</p>\n`
	ids, err = svc.ExtractPictures(html)
	require.NoError(t, err)
	require.Equal(t, 1, len(ids), "one image in")
	assert.Equal(t, "github_ef0f706a79cc24b17bbbb374cd234a691d034128/bjttt8ahajfmrhsula10.png", ids[0])

	// proxied image
	html = `<img src="https://remark42.radio-t.com/api/v1/img?src=aHR0cHM6Ly9ob21lcGFnZXMuY2FlLndpc2MuZWR1L35lY2U1MzMvaW1hZ2VzL2JvYXQucG5n" alt="cat.png">`
	ids, err = svc.ExtractPictures(html)
	require.NoError(t, err)
	require.Equal(t, 1, len(ids), "one image in")
	assert.Equal(t, "cached_images/12318fbd4c55e9d177b8b5ae197bc89c5afd8e07-a41fcb00643f28d700504256ec81cbf2e1aac53e", ids[0])

	// bad url
	html = `<img src=" https://remark42.radio-t.com/api/v1/img">`
	ids, err = svc.ExtractPictures(html)
	require.NoError(t, err)
	require.Empty(t, ids)

	// bad src
	html = `<img src="https://remark42.radio-t.com/api/v1/img?src=bad">`
	ids, err = svc.ExtractPictures(html)
	require.NoError(t, err)
	require.Empty(t, ids)

	// good src with bad content
	badURL := base64.URLEncoding.EncodeToString([]byte(" http://foo.bar"))
	html = fmt.Sprintf(`<img src="https://remark42.radio-t.com/api/v1/img?src=%s">`, badURL)
	ids, err = svc.ExtractPictures(html)
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestService_Cleanup(t *testing.T) {
	store := MockStore{}
	store.On("Cleanup", mock.Anything, mock.Anything).Times(10).Return(nil)

	svc := NewService(&store, ServiceParams{EditDuration: 20 * time.Millisecond})
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*549)
	defer cancel()
	svc.Cleanup(ctx)
	store.AssertNumberOfCalls(t, "Cleanup", 10)
}

func TestService_Submit(t *testing.T) {
	store := MockStore{}
	store.On("Commit", mock.Anything, mock.Anything).Times(7).Return(nil)
	svc := NewService(&store, ServiceParams{ImageAPI: "/blah/", EditDuration: time.Millisecond * 100})
	svc.Submit(func() []string { return []string{"id1", "id2", "id3"} })
	err := svc.SubmitAndCommit(func() []string { return []string{"id4", "id5"} })
	assert.NoError(t, err)
	svc.Submit(func() []string { return []string{"id6", "id7"} })
	svc.Submit(nil)
	store.AssertNumberOfCalls(t, "Commit", 2)
	time.Sleep(time.Millisecond * 175)
	store.AssertNumberOfCalls(t, "Commit", 7)
	svc.Close(context.TODO())
}

func TestService_Close(t *testing.T) {
	store := MockStore{}
	store.On("Commit", mock.Anything, mock.Anything).Times(5).Return(nil)
	svc := Service{store: &store, ServiceParams: ServiceParams{ImageAPI: "/blah/", EditDuration: time.Hour * 24}}
	svc.Submit(func() []string { return []string{"id1", "id2", "id3"} })
	svc.Submit(func() []string { return []string{"id4", "id5"} })
	svc.Submit(nil)
	svc.Close(context.TODO())
	store.AssertNumberOfCalls(t, "Commit", 5)
}

func TestService_SubmitDelay(t *testing.T) {
	store := MockStore{}
	store.On("Commit", mock.Anything, mock.Anything).Times(5).Return(nil)
	svc := NewService(&store, ServiceParams{EditDuration: 20 * time.Millisecond})
	svc.Submit(func() []string { return []string{"id1", "id2", "id3"} })
	time.Sleep(150 * time.Millisecond) // let first batch to pass TTL
	svc.Submit(func() []string { return []string{"id4", "id5"} })
	svc.Submit(nil)
	store.AssertNumberOfCalls(t, "Commit", 3)
	svc.Close(context.TODO())
	store.AssertNumberOfCalls(t, "Commit", 5)
}

func TestService_Info(t *testing.T) {
	store := MockStore{}
	store.On("Info", mock.Anything, mock.Anything).Once().Return(StoreInfo{}, nil)

	svc := Service{store: &store, ServiceParams: ServiceParams{}}
	info, err := svc.Info()
	assert.NoError(t, err)
	assert.True(t, info.FirstStagingImageTS.IsZero())
	store.AssertNumberOfCalls(t, "Info", 1)
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
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			resW, resH := getProportionalSizes(tt.inpW, tt.inpH, tt.limitW, tt.limitH)
			assert.Equal(t, tt.resW, resW, "width")
			assert.Equal(t, tt.resH, resH, "height")
		})
	}
}

func TestCachedImgID(t *testing.T) {
	img, err := CachedImgID(" http://foo.com")
	assert.Error(t, err)
	assert.Empty(t, img)
	imgURL := "http://example.org/img/1.png"
	img, err = CachedImgID(imgURL)
	assert.NoError(t, err)
	assert.Equal(t, "cached_images/"+Sha1Str("example.org")+"-"+Sha1Str(imgURL), img)
}
