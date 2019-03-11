package image

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImage_Save(t *testing.T) {
	loc, err := ioutil.TempDir("", "test_image_r42")
	require.NoError(t, err, "failed to make temp dir")
	defer os.RemoveAll(loc)

	svc := FileSystem{
		Location:  loc,
		Partitons: 100,
		MaxSize:   50,
	}
	id, err := svc.Save("blah_ff1.png", strings.NewReader("blah blah"))
	assert.NoError(t, err)
	assert.Equal(t, "6851dcde6024e03258a66705f29e14b506048c74.png", id)

	dst := path.Join(loc, "02", id)
	data, err := ioutil.ReadFile(dst)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(data))
}

func TestImage_SaveTooLarge(t *testing.T) {
	loc, err := ioutil.TempDir("", "test_image_r42")
	require.NoError(t, err, "failed to make temp dir")
	defer os.RemoveAll(loc)

	svc := FileSystem{
		Location:  loc,
		Partitons: 100,
		MaxSize:   5,
	}
	_, err = svc.Save("blah_ff1.png", strings.NewReader("blah blah"))
	assert.Error(t, err)
	assert.EqualError(t, err, "file blah_ff1.png is too large")
}

func TestImage_Load(t *testing.T) {
	loc, err := ioutil.TempDir("", "test_image_r42")
	require.NoError(t, err, "failed to make temp dir")
	defer os.RemoveAll(loc)

	// save image
	svc := FileSystem{
		Location:  loc,
		Partitons: 100,
		MaxSize:   50,
	}
	id, err := svc.Save("blah_ff1.png", strings.NewReader("blah blah"))
	assert.NoError(t, err)

	r, err := svc.Load(id)
	assert.NoError(t, err)
	defer r.Close()
	data, err := ioutil.ReadAll(r)
	assert.NoError(t, err)
	assert.Equal(t, "blah blah", string(data))

	_, err = svc.Load("abcd")
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
		{0, "12345", "/tmp"},
	}
	for n, tt := range tbl {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			svc := FileSystem{Location: "/tmp", Partitons: tt.partitions}
			assert.Equal(t, tt.res, svc.location(tt.id))
		})
	}
}
