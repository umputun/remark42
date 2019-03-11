package image

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImage_Save(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()

	id, err := svc.Save("blah_ff1.png", strings.NewReader("blah blah"))
	assert.NoError(t, err)
	assert.Equal(t, "fc77a87ad3c898b9603119711f99305145e272e103c904d85ee2deda.png", id)

	dst := path.Join(svc.Location, "56", id)
	data, err := ioutil.ReadFile(dst)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(data))
}

func TestImage_SaveTooLarge(t *testing.T) {
	svc, teardown := prepareImageTest(t)
	defer teardown()
	svc.MaxSize = 5
	_, err := svc.Save("blah_ff1.png", strings.NewReader("blah blah"))
	assert.Error(t, err)
	assert.EqualError(t, err, "file blah_ff1.png is too large")
}

func TestImage_Load(t *testing.T) {

	svc, teardown := prepareImageTest(t)
	defer teardown()

	id, err := svc.Save("blah_ff1.png", strings.NewReader("blah blah"))
	assert.NoError(t, err)

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
		{10, "abcdefg", "/tmp/2"},
		{10, "abcdefe", "/tmp/1"},
		{10, "12345", "/tmp/9"},
		{100, "12345", "/tmp/69"},
		{100, "xyzz", "/tmp/58"},
		{100, "6851dcde6024e03258a66705f29e14b506048c74.png", "/tmp/02"},
		{5, "6851dcde6024e03258a66705f29e14b506048c74.png", "/tmp/2"},
		{5, "xxxyz.png", "/tmp/0"},
		{0, "12345", "/tmp"},
	}
	for n, tt := range tbl {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			svc := FileSystem{Location: "/tmp", Partitions: tt.partitions}
			assert.Equal(t, tt.res, svc.location(tt.id))
		})
	}

	// generate random names and make sure partition never runs out of allowed
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	randomString := func(n int) string {
		b := make([]rune, n)
		for i := range b {
			b[i] = letterRunes[rand.Intn(len(letterRunes))]
		}
		return string(b)
	}

	svc := FileSystem{Location: "/tmp", Partitions: 10}
	for i := 0; i < 1000; i++ {
		v := randomString(rand.Intn(64))
		parts := strings.Split(svc.location(v), "/")
		p, err := strconv.Atoi(parts[len(parts)-1])
		require.NoError(t, err)
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
