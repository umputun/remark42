package avatar

import (
	"io/ioutil"
	"sort"
	"strings"
	"testing"

	"github.com/globalsign/mgo"
	"github.com/go-pkgz/mongo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGridFS_PutAndGet(t *testing.T) {
	p, skip := prepGFStore(t)
	if skip {
		return
	}
	avatar, err := p.Put("user1", strings.NewReader("some picture bin data"))
	require.Nil(t, err)
	assert.Equal(t, "b3daa77b4c04a9551b8781d03191fe098f325e67.image", avatar)

	rd, size, err := p.Get(avatar)
	require.Nil(t, err)
	assert.Equal(t, 21, size)
	data, err := ioutil.ReadAll(rd)
	require.Nil(t, err)
	assert.Equal(t, "some picture bin data", string(data))

	_, _, err = p.Get("bad avatar")
	assert.NotNil(t, err)

	assert.Equal(t, "8ce5568f7f9a1c9da5b897bc8642e397", p.ID(avatar))
	assert.Equal(t, "70c881d4a26984ddce795f6f71817c9cf4480e79", p.ID("aaaa"), "no data, encode avatar id")

	l, err := p.List()
	require.Nil(t, err)
	assert.Equal(t, 1, len(l))
	assert.Equal(t, "b3daa77b4c04a9551b8781d03191fe098f325e67.image", l[0])
}

func TestGridFS_Remove(t *testing.T) {
	p, skip := prepGFStore(t)
	if skip {
		return
	}

	assert.NotNil(t, p.Remove("no-such-thing.image"))

	avatar, err := p.Put("user1", strings.NewReader("some picture bin data"))
	require.Nil(t, err)
	assert.Equal(t, "b3daa77b4c04a9551b8781d03191fe098f325e67.image", avatar)
	assert.NoError(t, p.Remove("b3daa77b4c04a9551b8781d03191fe098f325e67.image"), "remove real one")
	assert.NotNil(t, p.Remove("b3daa77b4c04a9551b8781d03191fe098f325e67.image"), "already removed")
}

func TestGridFS_List(t *testing.T) {
	p, skip := prepGFStore(t)
	if skip {
		return
	}
	// write some avatars
	_, err := p.Put("user1", strings.NewReader("some picture bin data 1"))
	require.Nil(t, err)
	_, err = p.Put("user2", strings.NewReader("some picture bin data 2"))
	require.Nil(t, err)
	_, err = p.Put("user3", strings.NewReader("some picture bin data 3"))
	require.Nil(t, err)

	l, err := p.List()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(l), "3 avatars listed")
	sort.Strings(l)
	assert.Equal(t, []string{"0b7f849446d3383546d15a480966084442cd2193.image", "a1881c06eec96db9901c7bbfe41c42a3f08e9cb4.image", "b3daa77b4c04a9551b8781d03191fe098f325e67.image"}, l)

	r, size, err := p.Get("0b7f849446d3383546d15a480966084442cd2193.image")
	assert.Nil(t, err)
	assert.Equal(t, 23, size)
	data, err := ioutil.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(t, "some picture bin data 3", string(data))
}

func prepGFStore(t *testing.T) (Store, bool) {
	conn, err := mongo.MakeTestConnection(t)
	if err != nil {
		return nil, true
	}
	_ = conn.WithCustomCollection("fs.chunks", func(coll *mgo.Collection) error {
		return coll.DropCollection()
	})
	_ = conn.WithCustomCollection("fs.files", func(coll *mgo.Collection) error {
		return coll.DropCollection()
	})
	return NewGridFS(conn, 0), false
}
