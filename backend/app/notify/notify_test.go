package notify

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"

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

	assert.Equal(t, 3, len(d1.data), "got all comments to d1")
	assert.Equal(t, 3, len(d2.data), "got all comments to d2")

	assert.Equal(t, "100", d1.data[0].ID)
	assert.Equal(t, "101", d1.data[1].ID)
	assert.Equal(t, "102", d1.data[2].ID)
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

	assert.Equal(t, 2, len(d1.data), "one comment dropped from d1")
	assert.Equal(t, 2, len(d2.data), "one comment dropped from d2")
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

	assert.NotEqual(t, 10, len(d1.data), "some comments dropped from d1")
	assert.NotEqual(t, 10, len(d2.data), "some comments dropped from d2")

	assert.True(t, d1.closed)
	assert.True(t, d2.closed)
}

type mockDest struct {
	data   []store.Comment
	id     int
	closed bool
}

func (m *mockDest) Send(ctx context.Context, r Request) {
	select {
	case <-time.After(100 * time.Millisecond):
		m.data = append(m.data, r.comment)
		log.Printf("sent %s -> %d", r.comment.ID, m.id)
	case <-ctx.Done():
		log.Printf("ctx closed %d", m.id)
		m.closed = true
		return
	}
}
