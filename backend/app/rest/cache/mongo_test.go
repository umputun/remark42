package cache

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/go-pkgz/mongo"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMongoCache_Get(t *testing.T) {
	conn, err := mongo.MakeTestConnection(t)
	assert.NoError(t, err)
	defer mongo.RemoveTestCollections(t, conn, "cache")

	var postFnCall, coldCalls int32
	lc, err := NewMongoCache(conn, PostFlushFn(func() { atomic.AddInt32(&postFnCall, 1) }))
	require.Nil(t, err)
	res, err := lc.Get(NewKey("site").ID("key"), func() ([]byte, error) {
		atomic.AddInt32(&coldCalls, 1)
		return []byte("result"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result", string(res))
	assert.Equal(t, int32(1), atomic.LoadInt32(&coldCalls))
	assert.Equal(t, int32(0), atomic.LoadInt32(&postFnCall))

	res, err = lc.Get(NewKey("site").ID("key"), func() ([]byte, error) {
		atomic.AddInt32(&coldCalls, 1)
		return []byte("result"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result", string(res))
	assert.Equal(t, int32(1), atomic.LoadInt32(&coldCalls))
	assert.Equal(t, int32(0), atomic.LoadInt32(&postFnCall))

	lc.Flush(Flusher("site"))
	time.Sleep(100 * time.Millisecond) // let postFn to do its thing
	assert.Equal(t, int32(1), atomic.LoadInt32(&postFnCall))

	_, err = lc.Get(NewKey("site").ID("key"), func() ([]byte, error) {
		return nil, errors.New("err")
	})
	assert.NotNil(t, err)
}

func TestMongoCache_MaxKeys(t *testing.T) {
	var postFnCall, coldCalls int32
	conn, err := mongo.MakeTestConnection(t)
	assert.NoError(t, err)
	defer mongo.RemoveTestCollections(t, conn, "cache")

	lc, err := NewMongoCache(conn, PostFlushFn(func() { atomic.AddInt32(&postFnCall, 1) }),
		MaxKeys(5), MaxValSize(10))
	require.Nil(t, err)

	// put 5 keys to cache
	for i := 0; i < 5; i++ {
		res, e := lc.Get(NewKey("site").ID(fmt.Sprintf("key-%d", i)), func() ([]byte, error) {
			atomic.AddInt32(&coldCalls, 1)
			return []byte(fmt.Sprintf("result-%d", i)), nil
		})
		assert.Nil(t, e)
		assert.Equal(t, fmt.Sprintf("result-%d", i), string(res))
		assert.Equal(t, int32(i+1), atomic.LoadInt32(&coldCalls))
		assert.Equal(t, int32(0), atomic.LoadInt32(&postFnCall))
	}

	// check if really cached
	res, err := lc.Get(NewKey("site").ID("key-3"), func() ([]byte, error) {
		return []byte("result-blah"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-3", string(res), "should be cached")

	// try to cache after maxKeys reached
	res, err = lc.Get(NewKey("site").ID("key-X"), func() ([]byte, error) {
		return []byte("result-X"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-X", string(res))

	conn.WithCustomCollection("cache", func(coll *mgo.Collection) error {
		n, e := coll.Find(bson.M{"site": "site"}).Count()
		require.NoError(t, e)
		require.Equal(t, 5, n)
		r := mongoDoc{}
		require.NoError(t, coll.Find(bson.M{"site": "site"}).Sort("+_id").One(&r))
		assert.Equal(t, "key-1", r.Key)
		return nil
	})

	// put to cache and make sure it cached
	res, err = lc.Get(NewKey("site").ID("key-Z"), func() ([]byte, error) {
		return []byte("result-Z"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-Z", string(res))

	res, err = lc.Get(NewKey("site").ID("key-Z"), func() ([]byte, error) {
		return []byte("result-Zzzz"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-Z", string(res), "got cached value")

	conn.WithCustomCollection("cache", func(coll *mgo.Collection) error {
		n, e := coll.Find(bson.M{"site": "site"}).Count()
		require.NoError(t, e)
		require.Equal(t, 5, n)
		r := mongoDoc{}
		require.NoError(t, coll.Find(bson.M{"site": "site"}).Sort("+_id").One(&r))
		assert.Equal(t, "key-2", r.Key)
		return nil
	})
}

func TestMongoCache_Parallel(t *testing.T) {
	var coldCalls int32
	conn, err := mongo.MakeTestConnection(t)
	assert.NoError(t, err)
	defer mongo.RemoveTestCollections(t, conn, "cache")
	lc, err := NewMongoCache(conn)
	require.Nil(t, err)

	res, err := lc.Get(NewKey("site").ID("key"), func() ([]byte, error) {
		return []byte("value"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "value", string(res))

	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			res, err := lc.Get(NewKey("site").ID("key"), func() ([]byte, error) {
				atomic.AddInt32(&coldCalls, 1)
				return []byte(fmt.Sprintf("result-%d", i)), nil
			})
			require.Nil(t, err)
			require.Equal(t, "value", string(res))
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(0), atomic.LoadInt32(&coldCalls))
}

func TestMongoCache_Flush(t *testing.T) {
	conn, err := mongo.MakeTestConnection(t)
	assert.NoError(t, err)
	defer mongo.RemoveTestCollections(t, conn, "cache")
	lc, err := NewMongoCache(conn)
	require.Nil(t, err)

	addToCache := func(id string, scopes ...string) {
		res, err := lc.Get(NewKey("site").ID(id).Scopes(scopes...), func() ([]byte, error) {
			return []byte("value" + id), nil
		})
		require.Nil(t, err)
		require.Equal(t, "value"+id, string(res))
	}

	cacheSize := func() (count int) {
		conn.WithCustomCollection("cache", func(coll *mgo.Collection) error {
			n, e := coll.Find(bson.M{"site": "site"}).Count()
			require.NoError(t, e)
			count = n
			return nil
		})
		return count
	}

	init := func() {
		lc.Flush(Flusher("site"))
		addToCache("key1", "s1", "s2")
		addToCache("key2", "s1", "s2", "s3")
		addToCache("key3", "s1", "s2", "s3")
		addToCache("key4", "s2", "s3")
		addToCache("key5", "s2")
		addToCache("key6")
		addToCache("key7", "s4", "s3")
		require.Equal(t, 7, cacheSize(), "cache init")
	}

	tbl := []struct {
		scopes []string
		left   int
		msg    string
	}{
		{[]string{}, 0, "full flush, no scopes"},
		{[]string{"s0"}, 7, "flush wrong scope"},
		{[]string{"s1"}, 4, "flush s1 scope"},
		{[]string{"s2", "s1"}, 2, "flush s2+s1 scope"},
		{[]string{"s1", "s2"}, 2, "flush s1+s2 scope"},
		{[]string{"s1", "s2", "s4"}, 1, "flush s1+s2+s4 scope"},
		{[]string{"s1", "s2", "s3"}, 1, "flush s1+s2+s3 scope"},
		{[]string{"s1", "s2", "ss"}, 2, "flush s1+s2+wrong scope"},
	}

	for i, tt := range tbl {
		init()
		lc.Flush(Flusher("site").Scopes(tt.scopes...))
		assert.Equal(t, tt.left, cacheSize(), "keys size, %s #%d", tt.msg, i)
	}
}
