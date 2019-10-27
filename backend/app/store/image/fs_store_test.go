package image

import (
	"context"
	"encoding/base64"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gopher png for test, from https://golang.org/src/image/png/example_test.go
const gopher = "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"

func gopherPNG() io.Reader { return base64.NewDecoder(base64.StdEncoding, strings.NewReader(gopher)) }

func TestFsStore_Save(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()

	id, err := svc.Save("file1.png", "user1", gopherPNG())
	assert.NoError(t, err)
	assert.Contains(t, id, "user1/")
	assert.Contains(t, id, ".png")
	t.Log(id)

	img := svc.location(svc.Staging, id)
	t.Log(img)
	data, err := ioutil.ReadFile(img)
	assert.NoError(t, err)
	assert.Equal(t, 1462, len(data))
}

func TestFsStore_SaveWithResize(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()
	svc.MaxWidth, svc.MaxHeight = 32, 32

	id, err := svc.Save("file1.png", "user1", gopherPNG())
	assert.NoError(t, err)
	assert.Contains(t, id, "user1/")
	assert.Contains(t, id, ".png")
	t.Log(id)

	img := svc.location(svc.Staging, id)
	t.Log(img)
	data, err := ioutil.ReadFile(img)
	assert.NoError(t, err)
	assert.Equal(t, 1142, len(data))
}

func TestFsStore_SaveWithResizeJpeg(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()
	svc.MaxWidth, svc.MaxHeight = 400, 300
	svc.MaxSize = 32000

	fh, err := os.Open("testdata/circles.jpg")
	defer func() { assert.NoError(t, fh.Close()) }()
	assert.NoError(t, err)
	id, err := svc.Save("circles.jpg", "user1", fh)
	assert.NoError(t, err)
	assert.Contains(t, id, "user1/")
	assert.Contains(t, id, ".png")
	t.Log(id)

	img := svc.location(svc.Staging, id)
	t.Log(img)
	data, err := ioutil.ReadFile(img)
	assert.NoError(t, err)
	assert.Equal(t, 10786, len(data))
}

func TestFsStore_SaveNoResizeJpeg(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()
	svc.MaxWidth, svc.MaxHeight = 1400, 1300
	svc.MaxSize = 32000

	fh, err := os.Open("testdata/circles.jpg")
	defer func() { assert.NoError(t, fh.Close()) }()
	assert.NoError(t, err)
	id, err := svc.Save("circles.jpg", "user1", fh)
	assert.NoError(t, err)
	assert.Contains(t, id, "user1/")
	assert.Contains(t, id, ".jpg")
	t.Log(id)

	img := svc.location(svc.Staging, id)
	t.Log(img)
	data, err := ioutil.ReadFile(img)
	assert.NoError(t, err)
	assert.Equal(t, 23983, len(data))
}

func TestFsStore_WrongFormat(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()

	_, err := svc.Save("file1.png", "user1", strings.NewReader("blah blah bad image"))
	assert.EqualError(t, err, "can't load image file1.png: file format is not allowed")
}

func TestFsStore_SaveAndCommit(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()

	id, err := svc.Save("file1.png", "user1", gopherPNG())
	require.NoError(t, err)
	err = svc.Commit(id)
	require.NoError(t, err)

	imgStaging := svc.location(svc.Staging, id)
	_, err = os.Stat(imgStaging)
	assert.NotNil(t, err, "no file on staging anymore")

	img := svc.location(svc.Location, id)
	t.Log(img)
	data, err := ioutil.ReadFile(img)
	assert.NoError(t, err)
	assert.Equal(t, 1462, len(data))
}

func TestFsStore_SaveTooLarge(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()
	svc.MaxSize = 2000
	_, err := svc.Save("blah_ff1.png", "user2", io.MultiReader(gopherPNG(), gopherPNG()))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is too large")
}

func TestFsStore_LoadAfterSave(t *testing.T) {

	svc, teardown := prepareImageTest(t)
	defer teardown()

	id, err := svc.Save("blah_ff1.png", "user1", gopherPNG())
	assert.NoError(t, err)
	t.Log(id)

	r, sz, err := svc.Load(id)
	assert.NoError(t, err)
	defer func() { assert.NoError(t, r.Close()) }()
	data, err := ioutil.ReadAll(r)
	assert.NoError(t, err)
	assert.Equal(t, 1462, len(data))
	assert.Equal(t, int64(1462), sz)
	_, _, err = svc.Load("abcd")
	assert.NotNil(t, err)
}

func TestFsStore_LoadAfterCommit(t *testing.T) {

	svc, teardown := prepareImageTest(t)
	defer teardown()

	id, err := svc.Save("blah_ff1.png", "user1", gopherPNG())
	assert.NoError(t, err)
	t.Log(id)
	err = svc.Commit(id)
	require.NoError(t, err)

	r, sz, err := svc.Load(id)
	assert.NoError(t, err)
	defer func() { assert.NoError(t, r.Close()) }()
	data, err := ioutil.ReadAll(r)
	assert.NoError(t, err)
	assert.Equal(t, 1462, len(data))
	assert.Equal(t, int64(1462), sz)
	_, _, err = svc.Load("abcd")
	assert.NotNil(t, err)
}

func TestFsStore_location(t *testing.T) {
	tbl := []struct {
		partitions int
		id, res    string
	}{
		{10, "u1/abcdefg.png", "/tmp/u1/4/abcdefg.png"},
		{10, "u2/abcdefe", "/tmp/u2/0/abcdefe"},
		{10, "u3/12345", "/tmp/u3/4/12345"},
		{100, "12345", "/tmp/unknown/69/12345"},
		{100, "xyzz", "/tmp/unknown/58/xyzz"},
		{100, "u4/6851dcde6024e03258a66705f29e14b506048c74.png", "/tmp/u4/07/6851dcde6024e03258a66705f29e14b506048c74.png"},
		{5, "user/6851dcde6024e03258a66705f29e14b506048c74.png", "/tmp/user/1/6851dcde6024e03258a66705f29e14b506048c74.png"},
		{5, "aa-xxxyz.png", "/tmp/unknown/3/aa-xxxyz.png"},
		{0, "12345", "/tmp/unknown/12345"},
		{0, "user/12345", "/tmp/user/12345"},
	}
	for n, tt := range tbl {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			svc := FileSystem{Location: "/tmp", Partitions: tt.partitions}
			assert.Equal(t, tt.res, svc.location("/tmp", tt.id))
		})
	}

	// generate random names and make sure partition never runs out of allowed
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	randomID := func(n int) string {
		b := make([]rune, n)
		for i := range b {
			b[i] = letterRunes[rand.Intn(len(letterRunes))]
		}
		return "user1" + "/" + string(b)
	}

	svc := FileSystem{Location: "/tmp", Partitions: 10}
	for i := 0; i < 1000; i++ {
		v := randomID(rand.Intn(64))
		location := svc.location("/tmp", v)
		elems := strings.Split(location, "/")
		p, err := strconv.Atoi(elems[3])
		require.NoError(t, err, location)
		assert.True(t, p >= 0 && p < 10)
	}
}

func TestFsStore_Cleanup(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()

	save := func(file string, user string) (path string) {
		id, err := svc.Save(file, user, gopherPNG())
		require.NoError(t, err)
		img := svc.location(svc.Staging, id)
		data, err := ioutil.ReadFile(img)
		require.NoError(t, err)
		assert.Equal(t, 1462, len(data))
		return img
	}

	// save 3 images to staging
	img1 := save("blah_ff1.png", "user1")
	time.Sleep(100 * time.Millisecond)
	img2 := save("blah_ff2.png", "user1")
	time.Sleep(100 * time.Millisecond)
	img3 := save("blah_ff3.png", "user2")

	time.Sleep(100 * time.Millisecond) // make first image expired
	err := svc.Cleanup(context.Background(), time.Millisecond*300)
	assert.NoError(t, err)

	_, err = os.Stat(img1)
	assert.NotNil(t, err, "no file on staging anymore")
	_, err = os.Stat(path.Dir(img1))
	assert.NotNil(t, err, "no dir %s on staging anymore", path.Dir(img1))

	_, err = os.Stat(img2)
	assert.NoError(t, err, "file on staging")
	_, err = os.Stat(img3)
	assert.NoError(t, err, "file on staging")

	time.Sleep(200 * time.Millisecond) // make all images expired
	err = svc.Cleanup(context.Background(), time.Millisecond*300)
	assert.NoError(t, err)

	_, err = os.Stat(img2)
	assert.NotNil(t, err, "no file on staging anymore")
	_, err = os.Stat(img3)
	assert.NotNil(t, err, "no file on staging anymore")
}

func prepareImageTest(t *testing.T) (svc *FileSystem, teardown func()) {
	loc, err := ioutil.TempDir("", "test_image_r42")
	require.NoError(t, err, "failed to make temp dir")

	staging, err := ioutil.TempDir("", "test_image_r42.staging")
	require.NoError(t, err, "failed to make temp staging dir")

	svc = &FileSystem{
		Location:   loc,
		Staging:    staging,
		Partitions: 100,
		MaxSize:    1500,
	}

	teardown = func() {
		defer func() {
			assert.NoError(t, os.RemoveAll(loc))
			assert.NoError(t, os.RemoveAll(staging))
		}()
	}

	return svc, teardown
}
