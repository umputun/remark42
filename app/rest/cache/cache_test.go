package cache

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/umputun/remark/app/rest"
	"github.com/umputun/remark/app/store"
)

func TestLoadingCache_Get(t *testing.T) {
	var postFnCall, coldCalls int32
	lc := NewLoadingCache(CleanupInterval(200*time.Millisecond), PostFlushFn(func() { atomic.AddInt32(&postFnCall, 1) }))

	res, err := lc.Get("key", time.Minute, func() ([]byte, error) {
		atomic.AddInt32(&coldCalls, 1)
		return []byte("result"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result", string(res))
	assert.Equal(t, int32(1), atomic.LoadInt32(&coldCalls))
	assert.Equal(t, int32(0), atomic.LoadInt32(&postFnCall))

	res, err = lc.Get("key", time.Minute, func() ([]byte, error) {
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
}

func TestLoadingCache_MaxKeys(t *testing.T) {
	var postFnCall, coldCalls int32
	lc := NewLoadingCache(CleanupInterval(200*time.Millisecond), PostFlushFn(func() { atomic.AddInt32(&postFnCall, 1) }),
		MaxKeys(5), MaxValSize(10))

	// put 5 keys to cache
	for i := 0; i < 5; i++ {
		res, err := lc.Get(fmt.Sprintf("key-%d", i), 500*time.Millisecond, func() ([]byte, error) {
			atomic.AddInt32(&coldCalls, 1)
			return []byte(fmt.Sprintf("result-%d", i)), nil
		})
		assert.Nil(t, err)
		assert.Equal(t, fmt.Sprintf("result-%d", i), string(res))
		assert.Equal(t, int32(i+1), atomic.LoadInt32(&coldCalls))
		assert.Equal(t, int32(0), atomic.LoadInt32(&postFnCall))
	}

	// check if really cached
	res, err := lc.Get("key-3", time.Minute, func() ([]byte, error) {
		return []byte("result-blah"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-3", string(res), "should get cached")

	// try to cache after maxKeys reached
	res, err = lc.Get("key-X", time.Minute, func() ([]byte, error) {
		return []byte("result-X"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-X", string(res))

	cc := atomic.LoadInt32(&coldCalls)
	res, err = lc.Get("key-X", time.Minute, func() ([]byte, error) {
		atomic.AddInt32(&coldCalls, 1)
		return []byte("result-not-cached"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, cc+1, atomic.LoadInt32(&coldCalls))
	assert.Equal(t, "result-not-cached", string(res), "not cached")

	time.Sleep(time.Second) // let cleanup to remove

	// put to cache and make sure it cached
	res, err = lc.Get("key-Z", time.Minute, func() ([]byte, error) {
		return []byte("result-Z"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-Z", string(res))

	res, err = lc.Get("key-Z", time.Minute, func() ([]byte, error) {
		return []byte("result-Zzzz"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-Z", string(res), "got cached value")
}

func TestLoadingCache_MaxSize(t *testing.T) {
	lc := NewLoadingCache(CleanupInterval(200*time.Millisecond), MaxKeys(5), MaxValSize(10))

	// put good size value to cache and make sure it cached
	res, err := lc.Get("key-Z", time.Minute, func() ([]byte, error) {
		return []byte("result-Z"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-Z", string(res))

	res, err = lc.Get("key-Z", time.Minute, func() ([]byte, error) {
		return []byte("result-Zzzz"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-Z", string(res), "got cached value")

	// put too big value to cache and make sure it is not cached
	res, err = lc.Get("key-Big", time.Minute, func() ([]byte, error) {
		return []byte("1234567890"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "1234567890", string(res))

	res, err = lc.Get("key-Big", time.Minute, func() ([]byte, error) {
		return []byte("result-big"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result-big", string(res), "got not cached value")

}

func TestLoadingCache_URLKey(t *testing.T) {
	r, err := http.NewRequest("GET", "http://blah/123", nil)
	assert.Nil(t, err)
	key := URLKey(r)
	assert.Equal(t, "http://blah/123", key)

	r, err = http.NewRequest("GET", "http://blah/123?key=v&k2=v2", nil)
	assert.Nil(t, err)
	key = URLKey(r)
	assert.Equal(t, "http://blah/123?key=v&k2=v2", key)

	user := store.User{Admin: true}
	r = rest.SetUserInfo(r, user)
	key = URLKey(r)
	assert.Equal(t, "admin!!http://blah/123?key=v&k2=v2", key)
}

func TestLoadingCache_Parallel(t *testing.T) {
	var coldCalls int32
	lc := NewLoadingCache(CleanupInterval(time.Second))

	res, err := lc.Get("key", time.Minute, func() ([]byte, error) {
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
			res, err := lc.Get("key", time.Minute, func() ([]byte, error) {
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

func TestLoadingCache_Scopes(t *testing.T) {
	lc := NewLoadingCache(CleanupInterval(time.Second))

	res, err := lc.Get(Key("key", "s1", "s2"), time.Minute, func() ([]byte, error) {
		return []byte("value"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "value", string(res))

	res, err = lc.Get(Key("key2", "s2"), time.Minute, func() ([]byte, error) {
		return []byte("value2"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "value2", string(res))

	assert.Equal(t, 2, len(lc.(*loadingCache).activeKeys))
	lc.Flush("s1")
	assert.Equal(t, 1, len(lc.(*loadingCache).activeKeys))

	lc.Get(Key("key2", "s2"), time.Minute, func() ([]byte, error) {
		assert.Fail(t, "should stay")
		return nil, nil
	})

	res, err = lc.Get(Key("key", "s1", "s2"), time.Minute, func() ([]byte, error) {
		return []byte("value-upd"), nil
	})
	assert.Equal(t, "value-upd", string(res), "was deleted, update")
}

func TestLoadingCache_Flush(t *testing.T) {
	lc := NewLoadingCache(CleanupInterval(time.Second))

	addToCache := func(key string, scopes ...string) {
		res, err := lc.Get(key, time.Minute, func() ([]byte, error) {
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
		require.Equal(t, 7, len(lc.(*loadingCache).activeKeys), "cache init")
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
		assert.Equal(t, tt.left, len(lc.(*loadingCache).activeKeys), "keys size, %s #%d", tt.msg, i)
		assert.Equal(t, tt.left, len(lc.(*loadingCache).bytesCache.Items()), "items size, %s #%d", tt.msg, i)

	}
}

func TestLoadingCache_Keys(t *testing.T) {
	tbl := []struct {
		key    string
		scopes []string
		full   string
	}{
		{"key1", []string{"s1"}, "s1@@key1"},
		{"key2", []string{"s11", "s2"}, "s11$$s2@@key2"},
		{"key3", []string{}, "@@key3"},
	}

	for n, tt := range tbl {
		full := Key(tt.key, tt.scopes...)
		assert.Equal(t, tt.full, full, "making key, #%d", n)

		k, s, e := parseKey(full)
		assert.Nil(t, e)
		assert.Equal(t, tt.scopes, s)
		assert.Equal(t, tt.key, k)
	}

	_, _, err := parseKey("abc")
	assert.Error(t, err)
	_, _, err = parseKey("")
	assert.Error(t, err)
}
