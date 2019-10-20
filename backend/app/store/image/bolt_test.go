package image

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	bolt "github.com/coreos/bbolt"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var testDB = "/tmp/test-pictures-remark.db"

func TestBoltDB_NewBoltDB(t *testing.T) {
	store, cleanup := prep(t)
	defer cleanup()

	assert.NotNil(t, store.db)

	verifyBucket := func(bktName string) error {
		return store.db.View(func(tx *bolt.Tx) error {
			if b := tx.Bucket([]byte(bktName)); b == nil {
				return errors.Errorf("didn't find bucket %s", bktName)
			}
			return nil
		})
	}

	assert.Nil(t, verifyBucket(stagingBucketName))
	assert.Nil(t, verifyBucket(committedBucketName))
	assert.Nil(t, verifyBucket(metasBucketName))
}

func TestBoltDB_Save(t *testing.T) {
	store, cleanup := prep(t)
	defer cleanup()

	data, _ := ioutil.ReadFile("./testdata/circles.png")

	id, err := store.Save("circles.png", "smaant", data)
	assert.NoError(t, err)
	assert.NotEqual(t, "", id)
	assert.Equal(t, data, getKey(store, t, stagingBucketName, id))
}

func TestBoltDB_LoadFromStaging(t *testing.T) {
	store, cleanup := prep(t)
	defer cleanup()

	data, _ := ioutil.ReadFile("./testdata/circles.png")

	id, _ := store.Save("circles.png", "smaant", data)
	r, size, err := store.Load(id)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(data)), size)

	res, _ := ioutil.ReadAll(r)
	assert.Equal(t, data, res)
}

func TestBoltDB_LoadCommitted(t *testing.T) {
	store, cleanup := prep(t)
	defer cleanup()

	data, _ := ioutil.ReadFile("./testdata/circles.png")

	id, _ := store.Save("circles.png", "smaant", data)
	store.Commit(id)
	r, size, err := store.Load(id)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(data)), size)

	res, _ := ioutil.ReadAll(r)
	assert.Equal(t, data, res)
}

func TestBoltDB_LoadNotExisting(t *testing.T) {
	store, cleanup := prep(t)
	defer cleanup()

	_, size, err := store.Load("dev/123.jpg")
	assert.Error(t, err)
	assert.Equal(t, int64(0), size)
}

func TestBoltDB_Commit(t *testing.T) {
	store, cleanup := prep(t)
	defer cleanup()

	data, _ := ioutil.ReadFile("./testdata/circles.png")

	id, _ := store.Save("circles.png", "smaant", data)
	err := store.Commit(id)
	assert.NoError(t, err)

	assert.Equal(t, data, getKey(store, t, committedBucketName, id))
	assert.Equal(t, []byte(nil), getKey(store, t, stagingBucketName, id))
}

func TestBoltDB_CleanupRemovesOutdatedFiles(t *testing.T) {
	store, cleanup := prep(t)
	defer cleanup()

	data, _ := ioutil.ReadFile("./testdata/circles.png")

	id, _ := store.Save("circles.png", "smaant", data)
	err := store.Cleanup(context.Background(), 0)
	assert.NoError(t, err)

	assert.Equal(t, []byte(nil), getKey(store, t, committedBucketName, id))
	assert.Equal(t, []byte(nil), getKey(store, t, stagingBucketName, id))
}

func TestBoltDB_CleanupRemovesMultipleFiles(t *testing.T) {
	store, cleanup := prep(t)
	defer cleanup()

	data, _ := ioutil.ReadFile("./testdata/circles.png")

	id1, _ := store.Save("circles1.png", "smaant", data)
	id2, _ := store.Save("circles2.png", "smaant", data)
	err := store.Cleanup(context.Background(), 0)
	assert.NoError(t, err)

	assert.Equal(t, []byte(nil), getKey(store, t, committedBucketName, id1))
	assert.Equal(t, []byte(nil), getKey(store, t, stagingBucketName, id1))

	assert.Equal(t, []byte(nil), getKey(store, t, committedBucketName, id2))
	assert.Equal(t, []byte(nil), getKey(store, t, stagingBucketName, id2))
}

func TestBoltDB_CleanupRespectsTTL(t *testing.T) {
	store, cleanup := prep(t)
	defer cleanup()

	data, _ := ioutil.ReadFile("./testdata/circles.png")

	id, _ := store.Save("circles.png", "smaant", data)
	err := store.Cleanup(context.Background(), time.Second*5)
	assert.NoError(t, err)

	assert.Equal(t, data, getKey(store, t, stagingBucketName, id))
}

func prep(t *testing.T) (*BoltDB, func()) {
	cleanUp := func() { os.Remove(testDB) }
	store, err := NewBoltDB(BoltStore{FileName: testDB}, bolt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return store, cleanUp
}

func getKey(store *BoltDB, t *testing.T, bktName string, key string) []byte {
	var buf bytes.Buffer
	err := store.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket([]byte(bktName)).Get([]byte(key))
		buf.Write(v)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
