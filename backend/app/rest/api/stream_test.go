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
	s := streamer{
		refresh:   10 * time.Millisecond,
		timeout:   100 * time.Millisecond,
		maxActive: 10,
	}

	eventFn := func() steamEventFn {
		n := 0
		return func() (data []byte, upd bool, err error) {
			n++
			if n%2 == 0 || n > 10 {
				return nil, false, nil
			}
			return []byte(fmt.Sprintf("some data %d\n", n)), true, nil
		}
	}

	buf := bytes.Buffer{}
	err := s.activate(context.Background(), eventFn, &buf)
	assert.NoError(t, err)
	assert.Equal(t, "some data 1\nsome data 3\nsome data 5\nsome data 7\nsome data 9\n", buf.String())
}

func TestStream_Cancel(t *testing.T) {
	s := streamer{
		refresh:   10 * time.Millisecond,
		timeout:   100 * time.Millisecond,
		maxActive: 10,
	}

	eventFn := func() steamEventFn {
		n := 0
		return func() (data []byte, upd bool, err error) {
			n++
			if n%2 == 0 {
				return nil, false, nil
			}
			return []byte(fmt.Sprintf("some data %d\n", n)), true, nil
		}
	}

	buf := bytes.Buffer{}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := s.activate(ctx, eventFn, &buf)
	assert.NoError(t, err)
	assert.Equal(t, "some data 1\nsome data 3\nsome data 5\nsome data 7\nsome data 9\n", buf.String())
}
