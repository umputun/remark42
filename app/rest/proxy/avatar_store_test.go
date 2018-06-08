package proxy

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAvatarStore_Put(t *testing.T) {
	p := NewFSAvatarStore("/tmp/avatars.test")
	os.MkdirAll("/tmp/avatars.test", 0700)
	defer os.RemoveAll("/tmp/avatars.test")

	avatar, err := p.Put("user1", strings.NewReader("some picture bin data"))
	require.Nil(t, err)
	assert.Equal(t, "b3daa77b4c04a9551b8781d03191fe098f325e67.image", avatar)
	fi, err := os.Stat("/tmp/avatars.test/30/b3daa77b4c04a9551b8781d03191fe098f325e67.image")
	assert.NoError(t, err)
	assert.Equal(t, int64(21), fi.Size())

	avatar, err = p.Put("user2", strings.NewReader("some picture bin data 123"))
	require.Nil(t, err)
	assert.Equal(t, "a1881c06eec96db9901c7bbfe41c42a3f08e9cb4.image", avatar)
	fi, err = os.Stat("/tmp/avatars.test/84/a1881c06eec96db9901c7bbfe41c42a3f08e9cb4.image")
	assert.NoError(t, err)
	assert.Equal(t, int64(25), fi.Size())

	p = NewFSAvatarStore("/dev/null")
	_, err = p.Put("user1", strings.NewReader("some picture bin data"))
	assert.EqualError(t, err, "can't create file /dev/null/30/b3daa77b4c04a9551b8781d03191fe098f325e67.image: open /dev/null/30/b3daa77b4c04a9551b8781d03191fe098f325e67.image: not a directory")
}

func TestAvatarStore_Get(t *testing.T) {
	p := NewFSAvatarStore("/tmp/avatars.test")
	os.MkdirAll("/tmp/avatars.test/30", 0700)
	defer os.RemoveAll("/tmp/avatars.test")
	ioutil.WriteFile("/tmp/avatars.test/30/b3daa77b4c04a9551b8781d03191fe098f325e67.image", []byte("something"), 0666)
	r, size, err := p.Get("b3daa77b4c04a9551b8781d03191fe098f325e67.image")
	assert.Nil(t, err)
	assert.Equal(t, 9, size)
	data, err := ioutil.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(t, "something", string(data))
}

func TestAvatarStore_Location(t *testing.T) {
	p := NewFSAvatarStore("/tmp/avatars.test")

	tbl := []struct {
		id  string
		res string
	}{
		{"abc", "/tmp/avatars.test/35"},
		{"xyz", "/tmp/avatars.test/69"},
		{"blah blah", "/tmp/avatars.test/29"},
	}

	for i, tt := range tbl {
		assert.Equal(t, tt.res, p.location(tt.id), "test #%d", i)
	}
}
