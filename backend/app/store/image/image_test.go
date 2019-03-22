package image

import (
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImage_Save(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()

	id, err := svc.Save("file1.png", "user1", strings.NewReader("blah blah"))
	assert.NoError(t, err)
	assert.Contains(t, id, "user1/")
	assert.Contains(t, id, ".png")
	t.Log(id)

	data, err := ioutil.ReadFile(svc.location(id))
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(data))
}

func TestImage_SaveTooLarge(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()
	svc.MaxSize = 5
	_, err := svc.Save("blah_ff1.png", "user2", strings.NewReader("blah blah"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is too large")
}

func TestImage_Load(t *testing.T) {

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

func TestImage_location(t *testing.T) {
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
			assert.Equal(t, tt.res, svc.location(tt.id))
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
		location := svc.location(v)
		elems := strings.Split(location, "/")
		p, err := strconv.Atoi(elems[3])
		require.NoError(t, err, location)
		assert.True(t, p >= 0 && p < 10)
	}
}

func prepareImageTest(t *testing.T) (svc FileSystem, teardown func()) {
	loc, err := ioutil.TempDir("", "test_image_r42")
	require.NoError(t, err, "failed to make temp dir")

	svc = FileSystem{
		Location:   loc,
		Partitions: 100,
		MaxSize:    50,
	}

	teardown = func() {
		defer func() {
			assert.NoError(t, os.RemoveAll(loc))
		}()
	}

	return svc, teardown
}
