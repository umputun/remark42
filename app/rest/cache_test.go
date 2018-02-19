package rest

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/remark/app/store"
)

func TestLoadingCache_Get(t *testing.T) {
	var postFnCall, coldCalls int
	lc := NewLoadingCache(1*time.Minute, 200*time.Millisecond, func() {
		postFnCall++
	})

	res, err := lc.Get("key", time.Minute, func() ([]byte, error) {
		coldCalls++
		return []byte("result"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result", string(res))
	assert.Equal(t, 1, coldCalls)
	assert.Equal(t, 0, postFnCall)

	res, err = lc.Get("key", time.Minute, func() ([]byte, error) {
		coldCalls++
		return []byte("result"), nil
	})
	assert.Nil(t, err)
	assert.Equal(t, "result", string(res))
	assert.Equal(t, 1, coldCalls)
	assert.Equal(t, 0, postFnCall)

	lc.Flush()
	time.Sleep(100 * time.Millisecond) // let postFn to do its thing
	assert.Equal(t, 1, postFnCall)
}

func TestLoadingCache_URLKey(t *testing.T) {
	r, err := http.NewRequest("GET", "http://blah/123", nil)
	assert.Nil(t, err)
	key := URLKey(r)
	assert.Equal(t, "http://blah/123", key)

	ctx := context.Background()
	user := store.User{Admin: true}
	ctx = context.WithValue(ctx, ContextKey("user"), user)
	r = r.WithContext(ctx)
	key = URLKey(r)
	assert.Equal(t, "admin!!http://blah/123", key)
}
