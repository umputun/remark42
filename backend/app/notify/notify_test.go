package notify

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark/backend/app/store"
)

func TestService_NoDestinations(t *testing.T) {
	s := NewService(nil, 1)
	assert.NotNil(t, s)
	s.Submit(store.Comment{ID: "123"})
	s.Submit(store.Comment{ID: "123"})
	s.Submit(store.Comment{ID: "123"})
	s.Close()
}

func TestService_WithDestinations(t *testing.T) {
	d1, d2 := &mockDest{id: 1}, &mockDest{id: 2}
	s := NewService(nil, 1, d1, d2)
	assert.NotNil(t, s)

	s.Submit(store.Comment{ID: "100"})
	time.Sleep(time.Millisecond * 110)
	s.Submit(store.Comment{ID: "101"})
	time.Sleep(time.Millisecond * 110)
	s.Submit(store.Comment{ID: "102"})
	time.Sleep(time.Millisecond * 110)
	s.Close()

	assert.Equal(t, 3, len(d1.get()), "got all comments to d1")
	assert.Equal(t, 3, len(d2.get()), "got all comments to d2")

	assert.Equal(t, "100", d1.get()[0].comment.ID)
	assert.Equal(t, "101", d1.get()[1].comment.ID)
	assert.Equal(t, "102", d1.get()[2].comment.ID)
}

func TestService_WithDrops(t *testing.T) {
	d1, d2 := &mockDest{id: 1}, &mockDest{id: 2}
	s := NewService(nil, 1, d1, d2)
	assert.NotNil(t, s)

	s.Submit(store.Comment{ID: "100"})
	s.Submit(store.Comment{ID: "101"})
	time.Sleep(time.Millisecond * 110)
	s.Submit(store.Comment{ID: "102"})
	time.Sleep(time.Millisecond * 110)
	s.Close()

	s.Submit(store.Comment{ID: "111"}) // safe to send after close

	assert.Equal(t, 2, len(d1.get()), "one comment dropped from d1")
	assert.Equal(t, 2, len(d2.get()), "one comment dropped from d2")
}

func TestService_Many(t *testing.T) {
	d1, d2 := &mockDest{id: 1}, &mockDest{id: 2}
	s := NewService(nil, 5, d1, d2)
	assert.NotNil(t, s)

	for i := 0; i < 10; i++ {
		s.Submit(store.Comment{ID: fmt.Sprintf("%d", 100+i)})
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
	dest := &mockDest{id: 1}
	dataStore := &mockStore{data: map[string]store.Comment{}}

	dataStore.data["p1"] = store.Comment{ID: "p1"}
	dataStore.data["p2"] = store.Comment{ID: "p2"}

	s := NewService(dataStore, 1, dest)
	assert.NotNil(t, s)

	s.Submit(store.Comment{ID: "c1", ParentID: "p1"})
	time.Sleep(time.Millisecond * 110)
	s.Submit(store.Comment{ID: "c11", ParentID: "p11"})
	time.Sleep(time.Millisecond * 110)
	s.Close()

	destRes := dest.get()
	assert.Equal(t, 2, len(destRes), "two comment notified")
	assert.Equal(t, "p1", destRes[0].comment.ParentID)
	assert.Equal(t, "p1", destRes[0].parent.ID)
	assert.Equal(t, "p11", destRes[1].comment.ParentID)
	assert.Equal(t, "", destRes[1].parent.ID)
}

func TestService_Nop(t *testing.T) {
	s := NopService
	s.Submit(store.Comment{})
	s.Close()
	assert.Equal(t, uint32(1), atomic.LoadUint32(&s.closed))
}

type mockDest struct {
	data   []request
	id     int
	closed bool
	lock   sync.Mutex
}

func (m *mockDest) Send(ctx context.Context, r request) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	select {
	case <-time.After(100 * time.Millisecond):
		m.data = append(m.data, r)
		log.Printf("sent %s -> %d", r.comment.ID, m.id)
	case <-ctx.Done():
		log.Printf("ctx closed %d", m.id)
		m.closed = true
	}
	return nil
}

func (m *mockDest) get() []request {
	m.lock.Lock()
	defer m.lock.Unlock()
	res := make([]request, len(m.data))
	copy(res, m.data)
	return res
}
func (m *mockDest) String() string { return fmt.Sprintf("mock id=%d, closed=%v", m.id, m.closed) }

type mockStore struct{ data map[string]store.Comment }

func (m *mockStore) Get(_ store.Locator, id string, user store.User) (store.Comment, error) {
	res, ok := m.data[id]
	if !ok {
		return store.Comment{}, errors.New("no such id")
	}
	return res, nil
}
