package image

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	bolt "github.com/coreos/bbolt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoltStore_Save(t *testing.T) {
	svc, teardown := prepareBoltImageStorageTest(t)
	defer teardown()

	id, err := svc.Save("file1.png", "user1", gopherPNG())
	assert.NoError(t, err)
	assert.Contains(t, id, "user1")
	t.Log(id)

	err = svc.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket([]byte(imagesBktName)).Get([]byte(id))
		assert.NotNil(t, data)
		assert.Equal(t, 1462, len(data))
		return nil
	})
	assert.NoError(t, err)
}

func TestBoltStore_LoadAfterSave(t *testing.T) {
	svc, teardown := prepareBoltImageStorageTest(t)
	defer teardown()

	id, err := svc.Save("file1.png", "user1", gopherPNG())
	assert.NoError(t, err)
	assert.Contains(t, id, "user1")
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

func TestBoltStore_Cleanup(t *testing.T) {
	svc, teardown := prepareBoltImageStorageTest(t)
	defer teardown()

	save := func(file string, user string) (id string) {
		id, err := svc.Save(file, user, gopherPNG())
		require.NoError(t, err)

		checkBoltImgData(t, svc.db, id, func(data []byte) error {
			assert.NotNil(t, data)
			assert.Equal(t, 1462, len(data))
			return nil
		})
		return id
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

	assertBoltImgNil(t, svc.db, img1)
	assertBoltImgNotNil(t, svc.db, img2)
	assertBoltImgNotNil(t, svc.db, img3)
	err = svc.Commit(img3)
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond) // make all images except commited expired
	err = svc.Cleanup(context.Background(), time.Millisecond*300)
	assert.NoError(t, err)

	assertBoltImgNil(t, svc.db, img2)
	assertBoltImgNotNil(t, svc.db, img3)
	assert.NoError(t, err)
}

func assertBoltImgNil(t *testing.T, db *bolt.DB, id string) {
	checkBoltImgData(t, db, id, func(data []byte) error {
		assert.Nil(t, data)
		return nil
	})
}

func assertBoltImgNotNil(t *testing.T, db *bolt.DB, id string) {
	checkBoltImgData(t, db, id, func(data []byte) error {
		assert.NotNil(t, data)
		return nil
	})
}

func checkBoltImgData(t *testing.T, db *bolt.DB, id string, callback func([]byte) error) {
	err := db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket([]byte(imagesBktName)).Get([]byte(id))
		return callback(data)
	})
	assert.NoError(t, err)
}

func prepareBoltImageStorageTest(t *testing.T) (svc *Bolt, teardown func()) {
	loc, err := ioutil.TempDir("", "test_image_r42")
	require.NoError(t, err, "failed to make temp dir")

	svc, err = NewBoltStorage(path.Join(loc, "picture.db"), 1500, 0, 0, bolt.Options{})
	assert.NoError(t, err, "new bolt storage")

	teardown = func() {
		defer func() {
			assert.NoError(t, os.RemoveAll(loc))
		}()
	}

	return svc, teardown
}
