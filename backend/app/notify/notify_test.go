package notify

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark/backend/app/store"
)

func TestService_NoDestinations(t *testing.T) {
	s := NewService(nil, 0)
	assert.Equal(t, defaultQueueSize, cap(s.queue))
	assert.NotNil(t, s)
	s.Submit(Request{Comment: store.Comment{ID: "123"}})
	s.Submit(Request{Comment: store.Comment{ID: "123"}})
	s.Submit(Request{Comment: store.Comment{ID: "123"}})
	s.Close()
}

func TestService_WithDestinations(t *testing.T) {
	d1, d2 := &MockDest{id: 1}, &MockDest{id: 2}
	s := NewService(nil, 1, d1, d2)
	assert.NotNil(t, s)

	s.Submit(Request{Comment: store.Comment{ID: "100"}})
	time.Sleep(time.Millisecond * 110)
	s.Submit(Request{Comment: store.Comment{ID: "101"}})
	time.Sleep(time.Millisecond * 110)
	s.Submit(Request{Comment: store.Comment{ID: "102"}})
	time.Sleep(time.Millisecond * 110)
	s.Close()

	assert.Equal(t, 3, len(d1.get()), "got all comments to d1")
	assert.Equal(t, 3, len(d2.get()), "got all comments to d2")

	assert.Equal(t, "100", d1.get()[0].Comment.ID)
	assert.Equal(t, "101", d1.get()[1].Comment.ID)
	assert.Equal(t, "102", d1.get()[2].Comment.ID)
}

func TestService_WithDrops(t *testing.T) {
	d1, d2 := &MockDest{id: 1}, &MockDest{id: 2}
	s := NewService(nil, 1, d1, d2)
	assert.NotNil(t, s)

	s.Submit(Request{Comment: store.Comment{ID: "100"}})
	s.Submit(Request{Comment: store.Comment{ID: "101"}})
	time.Sleep(time.Millisecond * 110)
	s.Submit(Request{Comment: store.Comment{ID: "102"}})
	time.Sleep(time.Millisecond * 110)
	s.Close()

	s.Submit(Request{Comment: store.Comment{ID: "111"}}) // safe to send after close

	assert.Equal(t, 2, len(d1.get()), "one comment dropped from d1")
	assert.Equal(t, 2, len(d2.get()), "one comment dropped from d2")
}

func TestService_Many(t *testing.T) {
	d1, d2 := &MockDest{id: 1}, &MockDest{id: 2}
	s := NewService(nil, 5, d1, d2)
	assert.NotNil(t, s)

	for i := 0; i < 10; i++ {
		s.Submit(Request{Comment: store.Comment{ID: fmt.Sprintf("%d", 100+i)}})
		time.Sleep(time.Millisecond * time.Duration(rand.Int31n(200)))
	}
	s.Close()
	time.Sleep(time.Millisecond * 10)

	assert.NotEqual(t, 10, len(d1.get()), "some comments dropped from d1")
	assert.NotEqual(t, 10, len(d2.get()), "some comments dropped from d2")

	assert.True(t, d1.closed)
	assert.True(t, d2.closed)
}

func TestService_WithParent(t *testing.T) {
	dest := &MockDest{id: 1}
	dataStore := &MockStore{data: map[string]store.Comment{}}

	dataStore.data["p1"] = store.Comment{ID: "p1"}
	dataStore.data["p2"] = store.Comment{ID: "p2"}

	s := NewService(dataStore, 1, dest)
	assert.NotNil(t, s)

	s.Submit(Request{Comment: store.Comment{ID: "c1", ParentID: "p1"}})
	time.Sleep(time.Millisecond * 110)
	s.Submit(Request{Comment: store.Comment{ID: "c11", ParentID: "p11"}})
	time.Sleep(time.Millisecond * 110)
	s.Close()

	destRes := dest.get()
	assert.Equal(t, 2, len(destRes), "two comment notified")
	assert.Equal(t, "p1", destRes[0].Comment.ParentID)
	assert.Equal(t, "p1", destRes[0].parent.ID)
	assert.Equal(t, "p11", destRes[1].Comment.ParentID)
	assert.Equal(t, "", destRes[1].parent.ID)
}

func TestService_Nop(t *testing.T) {
	s := NopService
	s.Submit(Request{Comment: store.Comment{}})
	s.Close()
	assert.Equal(t, uint32(1), atomic.LoadUint32(&s.closed))
}
