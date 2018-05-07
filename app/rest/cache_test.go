package rest

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/remark/app/store"
)

func TestLoadingCache_Get(t *testing.T) {
	var postFnCall, coldCalls int32
	lc := NewLoadingCache(1*time.Minute, 200*time.Millisecond, func() {
		atomic.AddInt32(&postFnCall, 1)
	})

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
