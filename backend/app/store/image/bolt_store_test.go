package image

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoltStore_SaveCommit(t *testing.T) {
	svc, teardown := prepareBoltImageStorageTest(t)
	defer teardown()

	id, err := svc.Save("user1", gopherPNGBytes())
	assert.NoError(t, err)
	assert.Contains(t, id, "user1")
	t.Log(id)

	err = svc.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket([]byte(imagesStagedBktName)).Get([]byte(id))
		assert.NotNil(t, data)
		assert.Equal(t, 1462, len(data))
		return nil
	})
	assert.NoError(t, err)

	err = svc.Commit(id)
	require.NoError(t, err)

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

	id, err := svc.Save("user1", gopherPNGBytes())
	assert.NoError(t, err)
	assert.Contains(t, id, "user1")
	t.Log(id)

	data, err := svc.Load(id)
	assert.NoError(t, err)
	assert.Equal(t, 1462, len(data))
	assert.Equal(t, gopherPNGBytes(), data)

	_, err = svc.Load("abcd")
	assert.Error(t, err)
}

func TestBoltStore_Cleanup(t *testing.T) {
	svc, teardown := prepareBoltImageStorageTest(t)
	defer teardown()

	save := func(file string, user string) (id string) {
		id, err := svc.Save(user, gopherPNGBytes())
		require.NoError(t, err)

		checkBoltImgData(t, svc.db, imagesStagedBktName, id, func(data []byte) error {
			require.NotNil(t, data)
			assert.Equal(t, 1462, len(data))
			return nil
		})
		return id
	}

	// save 3 images to staging
	img1 := save("blah_ff1.png", "user1")
	img1ts := time.Now()
	time.Sleep(100 * time.Millisecond)
	img2 := save("blah_ff2.png", "user1")
	time.Sleep(100 * time.Millisecond)
	img3 := save("blah_ff3.png", "user2")

	err := svc.Cleanup(context.Background(), time.Since(img1ts)) // clean first images
	assert.NoError(t, err)

	assertBoltImgNil(t, svc.db, imagesStagedBktName, img1)
	assertBoltImgNil(t, svc.db, imagesBktName, img1)
	assertBoltImgNotNil(t, svc.db, imagesStagedBktName, img2)
	assertBoltImgNotNil(t, svc.db, imagesStagedBktName, img3)

	err = svc.Commit(img3)
	require.NoError(t, err)

	err = svc.Cleanup(context.Background(), time.Millisecond*10)
	assert.NoError(t, err)

	assertBoltImgNil(t, svc.db, imagesStagedBktName, img2)
	assertBoltImgNil(t, svc.db, imagesBktName, img2)
	assertBoltImgNotNil(t, svc.db, imagesBktName, img3)
	assert.NoError(t, err)
}

func assertBoltImgNil(t *testing.T, db *bolt.DB, bucket string, id string) {
	checkBoltImgData(t, db, bucket, id, func(data []byte) error {
		assert.Nil(t, data, id)
		return nil
	})
}

func assertBoltImgNotNil(t *testing.T, db *bolt.DB, bucket string, id string) {
	checkBoltImgData(t, db, bucket, id, func(data []byte) error {
		assert.NotNil(t, data, id)
		return nil
	})
}

func checkBoltImgData(t *testing.T, db *bolt.DB, bucket string, id string, callback func([]byte) error) {
	err := db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		assert.NotNil(t, bkt, "bucket %s not found", bucket)
		data := bkt.Get([]byte(id))
		return callback(data)
	})
	assert.NoError(t, err)
}

func prepareBoltImageStorageTest(t *testing.T) (svc *Bolt, teardown func()) {
	loc, err := ioutil.TempDir("", "test_image_r42")
	require.NoError(t, err, "failed to make temp dir")

	svc, err = NewBoltStorage(path.Join(loc, "picture.db"), bolt.Options{})
	assert.NoError(t, err, "new bolt storage")

	teardown = func() {
		defer func() {
			assert.NoError(t, os.RemoveAll(loc))
		}()
	}

	return svc, teardown
}
