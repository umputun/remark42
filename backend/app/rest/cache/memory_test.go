package cache

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCache_Get(t *testing.T) {
	var postFnCall, coldCalls int32
	lc, err := NewMemoryCache(PostFlushFn(func() { atomic.AddInt32(&postFnCall, 1) }))
	require.Nil(t, err)
	res, err := lc.Get("key", "site", func() ([]byte, error) {
		atomic.AddInt32(&coldCalls, 1)
		return []byte("result"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result", string(res))
	assert.Equal(t, int32(1), atomic.LoadInt32(&coldCalls))
	assert.Equal(t, int32(0), atomic.LoadInt32(&postFnCall))

	res, err = lc.Get("key", "site", func() ([]byte, error) {
		atomic.AddInt32(&coldCalls, 1)
		return []byte("result"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result", string(res))
	assert.Equal(t, int32(1), atomic.LoadInt32(&coldCalls))
	assert.Equal(t, int32(0), atomic.LoadInt32(&postFnCall))

	lc.Flush()
	time.Sleep(100 * time.Millisecond) // let postFn to do its thing
	assert.Equal(t, int32(1), atomic.LoadInt32(&postFnCall))

	_, err = lc.Get("key", "site", func() ([]byte, error) {
		return nil, errors.New("err")
	})
	assert.NotNil(t, err)
}

func TestMemoryCache_MaxKeys(t *testing.T) {
	var postFnCall, coldCalls int32
	lc, err := NewMemoryCache(PostFlushFn(func() { atomic.AddInt32(&postFnCall, 1) }),
		MaxKeys(5), MaxValSize(10))
	require.Nil(t, err)

	// put 5 keys to cache
	for i := 0; i < 5; i++ {
		res, e := lc.Get(fmt.Sprintf("key-%d", i), "site", func() ([]byte, error) {
			atomic.AddInt32(&coldCalls, 1)
			return []byte(fmt.Sprintf("result-%d", i)), nil
		})
		assert.Nil(t, e)
		assert.Equal(t, fmt.Sprintf("result-%d", i), string(res))
		assert.Equal(t, int32(i+1), atomic.LoadInt32(&coldCalls))
		assert.Equal(t, int32(0), atomic.LoadInt32(&postFnCall))
	}

	// check if really cached
	res, err := lc.Get("key-3", "site", func() ([]byte, error) {
		return []byte("result-blah"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-3", string(res), "should be cached")

	// try to cache after maxKeys reached
	res, err = lc.Get("key-X", "site", func() ([]byte, error) {
		return []byte("result-X"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-X", string(res))

	assert.Equal(t, 5, lc.(*memoryCache).bytesCache.Len())

	// put to cache and make sure it cached
	res, err = lc.Get("key-Z", "site", func() ([]byte, error) {
		return []byte("result-Z"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-Z", string(res))

	res, err = lc.Get("key-Z", "site", func() ([]byte, error) {
		return []byte("result-Zzzz"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-Z", string(res), "got cached value")
	assert.Equal(t, 5, lc.(*memoryCache).bytesCache.Len())
}

func TestMemoryCache_MaxValueSize(t *testing.T) {
	lc, err := NewMemoryCache(MaxKeys(5), MaxValSize(10))
	require.Nil(t, err)
	// put good size value to cache and make sure it cached
	res, err := lc.Get("key-Z", "site", func() ([]byte, error) {
		return []byte("result-Z"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-Z", string(res))

	res, err = lc.Get("key-Z", "site", func() ([]byte, error) {
		return []byte("result-Zzzz"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-Z", string(res), "got cached value")

	// put too big value to cache and make sure it is not cached
	res, err = lc.Get("key-Big", "site", func() ([]byte, error) {
		return []byte("1234567890"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "1234567890", string(res))

	res, err = lc.Get("key-Big", "site", func() ([]byte, error) {
		return []byte("result-big"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-big", string(res), "got not cached value")
}

func TestMemoryCache_MaxCacheSize(t *testing.T) {
	lc, err := NewMemoryCache(MaxKeys(50), MaxCacheSize(20))
	require.Nil(t, err)

	// put good size value to cache and make sure it cached
	res, err := lc.Get("key-Z", "site", func() ([]byte, error) {
		return []byte("result-Z"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-Z", string(res))
	assert.Equal(t, int64(8), lc.(*memoryCache).currentSize)

	_, err = lc.Get("key-Z2", "site", func() ([]byte, error) {
		return []byte("result-Z"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, int64(16), lc.(*memoryCache).currentSize)

	// this will cause removal
	_, err = lc.Get("key-Z3", "site", func() ([]byte, error) {
		return []byte("result-Z"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, int64(16), lc.(*memoryCache).currentSize)

	assert.Equal(t, 2, lc.(*memoryCache).bytesCache.Len())
}

func TestMemoryCache_MaxCacheSizeParallel(t *testing.T) {
	lc, err := NewMemoryCache(MaxCacheSize(123), MaxKeys(10000))
	require.Nil(t, err)

	wg := sync.WaitGroup{}
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		i := i
		go func() {
			time.Sleep(time.Duration(rand.Intn(100)) * time.Nanosecond)
			defer wg.Done()
			res, err := lc.Get(fmt.Sprintf("key-%d", i), "site", func() ([]byte, error) {
				return []byte(fmt.Sprintf("result-%d", i)), nil
			})
			require.Nil(t, err)
			require.Equal(t, fmt.Sprintf("result-%d", i), string(res))
			size := atomic.LoadInt64(&lc.(*memoryCache).currentSize)
			require.True(t, size < 200 && size >= 0, "unexpected size=%d", size) // won't be exactly 123 due parallel
		}()
	}
	wg.Wait()
	assert.True(t, lc.(*memoryCache).currentSize < 123 && lc.(*memoryCache).currentSize >= 0)
	t.Log("size=", lc.(*memoryCache).currentSize)
}

func TestMemoryCache_Parallel(t *testing.T) {
	var coldCalls int32
	lc, err := NewMemoryCache()
	require.Nil(t, err)

	res, err := lc.Get("key", "site", func() ([]byte, error) {
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
			res, err := lc.Get("key", "site", func() ([]byte, error) {
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

func TestMemoryCache_Scopes(t *testing.T) {
	lc, err := NewMemoryCache()
	require.Nil(t, err)

	res, err := lc.Get(Key("key", "s1", "s2"), "site", func() ([]byte, error) {
		return []byte("value"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "value", string(res))

	res, err = lc.Get(Key("key2", "s2"), "site", func() ([]byte, error) {
		return []byte("value2"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "value2", string(res))

	assert.Equal(t, 2, lc.(*memoryCache).bytesCache.Len())
	lc.Flush("s1")
	assert.Equal(t, 1, lc.(*memoryCache).bytesCache.Len())

	_, err = lc.Get(Key("key2", "s2"), "site", func() ([]byte, error) {
		assert.Fail(t, "should stay")
		return nil, nil
	})
	assert.Nil(t, err)
	res, err = lc.Get(Key("key", "s1", "s2"), "site", func() ([]byte, error) {
		return []byte("value-upd"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "value-upd", string(res), "was deleted, update")
}

func TestMemoryCache_Flush(t *testing.T) {
	lc, err := NewMemoryCache()
	require.Nil(t, err)

	addToCache := func(key string, scopes ...string) {
		res, err := lc.Get(key, "site", func() ([]byte, error) {
			return []byte("value" + key), nil
		})
		require.Nil(t, err)
		require.Equal(t, "value"+key, string(res))
	}

	init := func() {
		lc.Flush()
		addToCache(Key("key1", "s1", "s2"))
		addToCache(Key("key2", "s1", "s2", "s3"))
		addToCache(Key("key3", "s1", "s2", "s3"))
		addToCache(Key("key4", "s2", "s3"))
		addToCache(Key("key5", "s2"))
		addToCache(Key("key6"))
		addToCache(Key("key7", "s4", "s3"))
		require.Equal(t, 7, lc.(*memoryCache).bytesCache.Len(), "cache init")
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
		lc.Flush(tt.scopes...)
		assert.Equal(t, tt.left, lc.(*memoryCache).bytesCache.Len(), "keys size, %s #%d", tt.msg, i)
	}
}

func TestMemoryCache_FlushFailed(t *testing.T) {
	lc, err := NewMemoryCache()
	require.Nil(t, err)
	val, err := lc.Get("invalid-composite", "site", func() ([]byte, error) {
		return []byte("value"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "value", string(val))
	assert.Equal(t, 1, lc.(*memoryCache).bytesCache.Len())

	lc.Flush("invalid-composite")
	assert.Equal(t, 1, lc.(*memoryCache).bytesCache.Len())
}

func TestMemoryCache_BadOptions(t *testing.T) {
	_, err := NewMemoryCache(MaxCacheSize(-1))
	assert.EqualError(t, err, "failed to set cache option: negative size or MaxCacheSize, -1")

	_, err = NewMemoryCache(MaxKeys(-1))
	assert.EqualError(t, err, "failed to set cache option: negative size for MaxKeys, -1")

	_, err = NewMemoryCache(MaxValSize(-1))
	assert.EqualError(t, err, "failed to set cache option: negative size for MaxValSize, -1")
}
