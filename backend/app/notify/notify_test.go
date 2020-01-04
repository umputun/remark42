package notify

import (
	"errors"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	require.Equal(t, 3, len(d1.Get()), "got all comments to d1")
	require.Equal(t, 3, len(d2.Get()), "got all comments to d2")

	assert.Equal(t, "100", d1.Get()[0].Comment.ID)
	assert.Equal(t, "101", d1.Get()[1].Comment.ID)
	assert.Equal(t, "102", d1.Get()[2].Comment.ID)
}

func TestService_WithDrops(t *testing.T) {
	d1, d2 := &MockDest{id: 1}, &MockDest{id: 2}
	s := NewService(nil, 1, d1, d2)
	assert.NotNil(t, s)

	s.Submit(Request{Comment: store.Comment{ID: "100"}})
	s.Submit(Request{Comment: store.Comment{ID: "101"}})
	time.Sleep(time.Millisecond * 11)
	s.Submit(Request{Comment: store.Comment{ID: "102"}})
	time.Sleep(time.Millisecond * 11)
	s.Close()

	s.Submit(Request{Comment: store.Comment{ID: "111"}}) // safe to send after close

	assert.Equal(t, 2, len(d1.Get()), "one comment from three dropped from d1, got: %v", d1.Get())
	assert.Equal(t, 2, len(d2.Get()), "one comment from three dropped from d2, got: %v", d2.Get())
}

func TestService_Many(t *testing.T) {
	d1, d2 := &MockDest{id: 1}, &MockDest{id: 2}
	s := NewService(nil, 5, d1, d2)
	assert.NotNil(t, s)

	for i := 0; i < 10; i++ {
		s.Submit(Request{Comment: store.Comment{ID: fmt.Sprintf("%d", 100+i)}})
		time.Sleep(time.Millisecond * time.Duration(rand.Int31n(20)))
	}
	s.Close()
	time.Sleep(time.Millisecond * 10)

	assert.NotEqual(t, 10, len(d1.Get()), "some comments dropped from d1")
	assert.NotEqual(t, 10, len(d2.Get()), "some comments dropped from d2")

	assert.True(t, d1.closed)
	assert.True(t, d2.closed)
}

func TestService_WithParent(t *testing.T) {
	dest := &MockDest{id: 1}
	dataStore := &mockStore{data: map[string]store.Comment{}}

	dataStore.data["p1"] = store.Comment{ID: "p1"}
	dataStore.data["p2"] = store.Comment{ID: "p2"}

	s := NewService(dataStore, 1, dest)
	assert.NotNil(t, s)

	s.Submit(Request{Comment: store.Comment{ID: "c1", ParentID: "p1"}})
	time.Sleep(time.Millisecond * 110)
	s.Submit(Request{Comment: store.Comment{ID: "c11", ParentID: "p11"}})
	time.Sleep(time.Millisecond * 110)
	s.Close()

	destRes := dest.Get()
	require.Equal(t, 2, len(destRes), "two comment notified")
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

type mockStore struct{ data map[string]store.Comment }

func (m mockStore) Get(_ store.Locator, id string, _ store.User) (store.Comment, error) {
	res, ok := m.data[id]
	if !ok {
		return store.Comment{}, errors.New("no such id")
	}
	return res, nil
}

func (m mockStore) GetUserEmail(_ store.Locator, _ string) (string, error) {
	return "", errors.New("no such user")
}
