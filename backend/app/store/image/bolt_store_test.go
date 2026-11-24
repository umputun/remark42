package image

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

func TestBoltStore_SaveCommit(t *testing.T) {
	svc, teardown := prepareBoltImageStorageTest(t)
	defer teardown()

	id := "test_img"

	err := svc.Save(id, gopherPNGBytes())
	assert.NoError(t, err)

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

	id := "test_img"
	err := svc.Save(id, gopherPNGBytes())
	assert.NoError(t, err)

	data, err := svc.Load(id)
	assert.NoError(t, err)
	assert.Equal(t, 1462, len(data))
	assert.Equal(t, gopherPNGBytes(), data)

	_, err = svc.Load("abcd")
	assert.Error(t, err)
}

// pictures saved under the 20-character ids of earlier versions load, commit and delete like new ones
func TestBoltStore_OldAndNewIDs(t *testing.T) {
	for _, id := range []string{"user1/dc0ctt0j3vck07oc07s0", "user1/" + guid()} {
		t.Run(id, func(t *testing.T) {
			svc, teardown := prepareBoltImageStorageTest(t)
			defer teardown()

			require.NoError(t, svc.Save(id, gopherPNGBytes()))
			data, err := svc.Load(id)
			require.NoError(t, err, "staged")
			assert.Equal(t, gopherPNGBytes(), data)

			require.NoError(t, svc.Commit(id))
			data, err = svc.Load(id)
			require.NoError(t, err, "committed")
			assert.Equal(t, gopherPNGBytes(), data)

			require.NoError(t, svc.Delete(id))
			_, err = svc.Load(id)
			assert.Error(t, err, "deleted")
		})
	}
}

func TestBoltStore_LoadedDataSurvivesLaterWrites(t *testing.T) {
	svc, teardown := prepareBoltImageStorageTest(t)
	defer teardown()

	// the image is large enough to take its own overflow pages, which bolt reuses once it's deleted,
	// and the later saves write enough of the same size to reuse them
	const imageSize, laterSaves = 8192, 50
	orig := bytes.Repeat([]byte("A"), imageSize)
	require.NoError(t, svc.Save("u/first", orig))
	got, err := svc.Load("u/first")
	require.NoError(t, err)

	require.NoError(t, svc.Delete("u/first"))
	for i := range laterSaves {
		require.NoError(t, svc.Save(fmt.Sprintf("u/other%d", i), bytes.Repeat([]byte("C"), imageSize)))
	}
	assert.True(t, bytes.Equal(orig, got), "loaded image changed after later writes to the store, %d bytes starting with %q",
		len(got), got[:min(16, len(got))])
}

func TestBoltStore_LoadAfterDelete(t *testing.T) {
	svc, teardown := prepareBoltImageStorageTest(t)
	defer teardown()

	// delete image from permanent storage
	id := "test_img"
	err := svc.Save(id, gopherPNGBytes())
	assert.NoError(t, err)

	err = svc.Commit(id)
	require.NoError(t, err)

	err = svc.Delete(id)
	assert.NoError(t, err)

	_, err = svc.Load(id)
	assert.Error(t, err)

	// delete staging image
	id = "staging_img"
	err = svc.Save(id, gopherPNGBytes())
	assert.NoError(t, err)

	err = svc.Delete(id)
	assert.NoError(t, err)

	_, err = svc.Load(id)
	assert.Error(t, err)
}

func TestBoltStore_CommitCommitted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svc, teardown := prepareBoltImageStorageTest(t)
		defer teardown()

		id := "user1/test_img"
		require.NoError(t, svc.Save(id, gopherPNGBytes()))
		require.NoError(t, svc.Commit(id))
		time.Sleep(time.Second)
		require.NoError(t, svc.Cleanup(context.Background(), time.Millisecond)) // removes the staged copy
		assertBoltImgNil(t, svc.db, imagesStagedBktName, id)

		assert.NoError(t, svc.Commit(id), "committed image committed again")
		assertBoltImgNotNil(t, svc.db, imagesBktName, id)
		assert.Error(t, svc.Commit("user1/missing_img"), "image in neither bucket")
	})
}

func TestBoltStore_Cleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		svc, teardown := prepareBoltImageStorageTest(t)
		defer teardown()

		save := func(file string) (id string) {
			err := svc.Save(file, gopherPNGBytes())
			require.NoError(t, err)

			checkBoltImgData(t, svc.db, imagesStagedBktName, file, func(data []byte) error {
				require.NotNil(t, data)
				assert.Equal(t, 1462, len(data))
				return nil
			})
			return file
		}

		// save 3 images to staging
		img1 := save("blah_ff1.png")
		img1ts := time.Now()
		time.Sleep(100 * time.Millisecond)
		img2 := save("blah_ff2.png")
		time.Sleep(100 * time.Millisecond)
		img3 := save("blah_ff3.png")

		// Cleanup check is `age > ttl` (strict), so pick a ttl strictly less than img1's age
		err := svc.Cleanup(context.Background(), time.Since(img1ts)-time.Millisecond)
		assert.NoError(t, err)

		assertBoltImgNil(t, svc.db, imagesStagedBktName, img1)
		assertBoltImgNil(t, svc.db, imagesBktName, img1)
		assertBoltImgNotNil(t, svc.db, imagesStagedBktName, img2)
		assertBoltImgNotNil(t, svc.db, imagesStagedBktName, img3)

		err = svc.Commit(img3)
		require.NoError(t, err)

		// reset the time to cleanup
		err = svc.ResetCleanupTimer(img2)
		require.NoError(t, err)
		err = svc.Cleanup(context.Background(), time.Millisecond*100)
		assert.NoError(t, err)

		assertBoltImgNotNil(t, svc.db, imagesStagedBktName, img2)
		assertBoltImgNil(t, svc.db, imagesBktName, img2)
		assertBoltImgNotNil(t, svc.db, imagesBktName, img3)
		assert.NoError(t, err)
	})
}

func TestBolt_Info(t *testing.T) {
	svc, teardown := prepareBoltImageStorageTest(t)
	defer teardown()

	// get info on empty storage, should be zero
	info, err := svc.Info()
	assert.NoError(t, err)
	assert.True(t, info.FirstStagingImageTS.IsZero())

	// save image
	err = svc.Save("test_img", gopherPNGBytes())
	assert.NoError(t, err)

	// get info after saving, should be non-zero
	info, err = svc.Info()
	assert.NoError(t, err)
	assert.False(t, info.FirstStagingImageTS.IsZero())
}

// TestBoltStore_InsertTimeBytes pins the stored insertion time to the bytes earlier versions wrote, so a
// database made by one of them keeps the age of its staged images
func TestBoltStore_InsertTimeBytes(t *testing.T) {
	svc, teardown := prepareBoltImageStorageTest(t)
	defer teardown()

	ts := time.Date(2026, 11, 21, 10, 30, 15, 123456789, time.UTC)
	// what binary.Write(buf, binary.LittleEndian, ts.UnixNano()) wrote for ts before putTS existed
	stored := []byte{0x15, 0x33, 0xf6, 0x5a, 0xbe, 0x08, 0xea, 0x18}

	require.NoError(t, svc.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(insertTimeBktName)).Put([]byte("old"), stored)
	}))
	info, err := svc.Info()
	require.NoError(t, err)
	assert.Equal(t, ts, info.FirstStagingImageTS.UTC(), "Info reads the stored time")

	require.NoError(t, svc.Cleanup(context.Background(), time.Since(ts)+time.Hour))
	assertBoltImgNotNil(t, svc.db, insertTimeBktName, "old")
	require.NoError(t, svc.Cleanup(context.Background(), time.Since(ts)-time.Hour))
	assertBoltImgNil(t, svc.db, insertTimeBktName, "old")

	require.NoError(t, svc.db.Update(func(tx *bolt.Tx) error { return putTS(tx, "new", ts) }))
	checkBoltImgData(t, svc.db, insertTimeBktName, "new", func(data []byte) error {
		assert.Equal(t, stored, data, "putTS writes the bytes earlier versions wrote")
		return nil
	})
}

func assertBoltImgNil(t *testing.T, db *bolt.DB, bucket, id string) {
	checkBoltImgData(t, db, bucket, id, func(data []byte) error {
		assert.Nil(t, data, id)
		return nil
	})
}

func assertBoltImgNotNil(t *testing.T, db *bolt.DB, bucket, id string) {
	checkBoltImgData(t, db, bucket, id, func(data []byte) error {
		assert.NotNil(t, data, id)
		return nil
	})
}

func checkBoltImgData(t *testing.T, db *bolt.DB, bucket, id string, callback func([]byte) error) {
	err := db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(bucket))
		assert.NotNil(t, bkt, "bucket %s not found", bucket)
		data := bkt.Get([]byte(id))
		return callback(data)
	})
	assert.NoError(t, err)
}

func prepareBoltImageStorageTest(t *testing.T) (svc *Bolt, teardown func()) {
	loc, err := os.MkdirTemp("", "test_image_r42")
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
