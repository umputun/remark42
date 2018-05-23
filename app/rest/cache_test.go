package rest

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		MaxKeys(5), MaxValueSize(10))

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
	lc := NewLoadingCache(CleanupInterval(200*time.Millisecond), MaxKeys(5), MaxValueSize(10))

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
	r = SetUserInfo(r, user)
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
