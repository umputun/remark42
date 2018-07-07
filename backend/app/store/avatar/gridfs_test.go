// +build mongo

package avatar

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/globalsign/mgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store/engine/mongo"
)

func TestGridFS_PutAndGet(t *testing.T) {
	p := prepGFStore(t)

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
}

func prepGFStore(t *testing.T) Store {
	mg := mongo.NewTesting("fs")
	conn, err := mg.Get()
	require.Nil(t, err)

	_ = conn.WithCustomCollection("fs.chunks", func(coll *mgo.Collection) error {
		return coll.DropCollection()
	})
	_ = conn.WithCustomCollection("fs.files", func(coll *mgo.Collection) error {
		return coll.DropCollection()
	})
	return NewGridFS(conn, 0)
}
