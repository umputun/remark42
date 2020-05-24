package api

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStream_Timeout(t *testing.T) {
	s := Streamer{
		Refresh:   10 * time.Millisecond,
		TimeOut:   100 * time.Millisecond,
		MaxActive: 10,
	}

	eventFn := func() steamEventFn {
		n := 0
		return func() (event string, data []byte, upd bool, err error) {
			n++
			if n%2 == 0 || n > 10 {
				return "test", nil, false, nil
			}
			return "test", []byte(fmt.Sprintf("some data %d\n", n)), true, nil
		}
	}

	buf := bytes.Buffer{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := s.Activate(ctx, eventFn, &buf)
	assert.NoError(t, err)
	assert.Equal(t, "event: test\ndata: some data 1\n\nevent: test\ndata: some data 3\n\nevent: test\ndata: some data 5\n\nevent: test\ndata: some data 7\n\nevent: test\ndata: some data 9\n\n", buf.String())
}

func TestStream_Cancel(t *testing.T) {
	s := Streamer{
		Refresh:   10 * time.Millisecond,
		TimeOut:   100 * time.Millisecond,
		MaxActive: 10,
	}

	eventFn := func() steamEventFn {
		n := 0
		return func() (event string, data []byte, upd bool, err error) {
			n++
			if n%2 == 0 {
				return "test", nil, false, nil
			}
			return "test", []byte(fmt.Sprintf("some data %d\n", n)), true, nil
		}
	}

	buf := bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := s.Activate(ctx, eventFn, &buf)
	assert.NoError(t, err)
	assert.Equal(t, "event: test\ndata: some data 1\n\nevent: test\ndata: some data 3\n\nevent: test\ndata: some data 5\n\nevent: test\ndata: some data 7\n\nevent: test\ndata: some data 9\n\n", buf.String())
}
