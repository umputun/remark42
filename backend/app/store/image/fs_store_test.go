package image

import (
	"context"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFsStore_Save(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()

	id, err := svc.Save("file1.png", "user1", strings.NewReader("blah blah"))
	assert.NoError(t, err)
	assert.Contains(t, id, "user1/")
	assert.Contains(t, id, ".png")
	t.Log(id)

	img := svc.location(svc.Staging, id)
	t.Log(img)
	data, err := ioutil.ReadFile(img)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(data))
}

func TestFsStore_SaveAndCommit(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()

	id, err := svc.Save("file1.png", "user1", strings.NewReader("blah blah"))
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
	assert.Equal(t, "blah blah", string(data))
}

func TestFsStore_SaveTooLarge(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()
	svc.MaxSize = 5
	_, err := svc.Save("blah_ff1.png", "user2", strings.NewReader("blah blah"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is too large")
}

func TestFsStore_LoadAfterSave(t *testing.T) {

	svc, teardown := prepareImageTest(t)
	defer teardown()

	id, err := svc.Save("blah_ff1.png", "user1", strings.NewReader("blah blah"))
	assert.NoError(t, err)
	t.Log(id)

	r, sz, err := svc.Load(id)
	assert.NoError(t, err)
	defer func() { assert.NoError(t, r.Close()) }()
	data, err := ioutil.ReadAll(r)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(data))
	assert.Equal(t, int64(9), sz)
	_, _, err = svc.Load("abcd")
	assert.NotNil(t, err)
}

func TestFsStore_LoadAfterCommit(t *testing.T) {

	svc, teardown := prepareImageTest(t)
	defer teardown()

	id, err := svc.Save("blah_ff1.png", "user1", strings.NewReader("blah blah"))
	assert.NoError(t, err)
	t.Log(id)
	err = svc.Commit(id)
	require.NoError(t, err)

	r, sz, err := svc.Load(id)
	assert.NoError(t, err)
	defer func() { assert.NoError(t, r.Close()) }()
	data, err := ioutil.ReadAll(r)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(data))
	assert.Equal(t, int64(9), sz)
	_, _, err = svc.Load("abcd")
	assert.NotNil(t, err)
}

func TestFsStore_location(t *testing.T) {
	tbl := []struct {
		partitions int
		id, res    string
	}{
		{10, "u1/abcdefg.png", "/tmp/u1/4/abcdefg.png"},
		{10, "abcdefe", "/tmp/unknown/1/abcdefe"},
		{10, "12345", "/tmp/unknown/9/12345"},
		{100, "12345", "/tmp/unknown/69/12345"},
		{100, "xyzz", "/tmp/unknown/58/xyzz"},
		{100, "6851dcde6024e03258a66705f29e14b506048c74.png", "/tmp/unknown/02/6851dcde6024e03258a66705f29e14b506048c74.png"},
		{5, "6851dcde6024e03258a66705f29e14b506048c74.png", "/tmp/unknown/2/6851dcde6024e03258a66705f29e14b506048c74.png"},
		{5, "xxxyz.png", "/tmp/unknown/0/xxxyz.png"},
		{0, "12345", "/tmp/unknown/12345"},
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

	save := func(file string, user string, content string) (path string) {
		id, err := svc.Save(file, user, strings.NewReader(content))
		require.NoError(t, err)
		img := svc.location(svc.Staging, id)
		data, err := ioutil.ReadFile(img)
		require.NoError(t, err)
		require.Equal(t, content, string(data))
		return img
	}

	// save 3 images to staging
	img1 := save("blah_ff1.png", "user1", "blah blah1")
	time.Sleep(100 * time.Millisecond)
	img2 := save("blah_ff2.png", "user1", "blah blah2")
	time.Sleep(100 * time.Millisecond)
	img3 := save("blah_ff3.png", "user2", "blah blah3")

	time.Sleep(100 * time.Millisecond) // make first image expired
	err := svc.Cleanup(context.Background(), time.Millisecond*300)
	assert.NoError(t, err)

	_, err = os.Stat(img1)
	assert.NotNil(t, err, "no file on staging anymore")
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

func prepareImageTest(t *testing.T) (svc FileSystem, teardown func()) {
	loc, err := ioutil.TempDir("", "test_image_r42")
	require.NoError(t, err, "failed to make temp dir")

	staging, err := ioutil.TempDir("", "test_image_r42.staging")
	require.NoError(t, err, "failed to make temp staging dir")

	svc = FileSystem{
		Location:   loc,
		Staging:    staging,
		Partitions: 100,
		MaxSize:    50,
	}

	teardown = func() {
		defer func() {
			assert.NoError(t, os.RemoveAll(loc))
			assert.NoError(t, os.RemoveAll(staging))
		}()
	}

	return svc, teardown
}
