package providers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/umputun/remark42/backend/app/notify"
)

func TestDispatchTelegramUpdates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	poolPeriod := time.Millisecond * 100
	go DispatchTelegramUpdates(ctx, &mockTGRequester{t: t}, []TGUpdatesReceiver{&mockTGUpdatesReceiver{t: t}}, poolPeriod)
	time.Sleep(poolPeriod * 3)
	cancel()
	time.Sleep(poolPeriod)
}

const getUpdatesResp = `{
  "ok": true,
  "result": [
     {
        "update_id": 998,
        "message": {
           "chat": {
              "type": "group"
           }
        }
     }
]
}`

type mockTGRequester struct {
	hit int
	t   *testing.T
}

func (m *mockTGRequester) Request(_ context.Context, _ string, _ []byte, data interface{}) error {
	if m.hit < 2 {
		m.hit++
		assert.NoError(m.t, json.Unmarshal([]byte(getUpdatesResp), data))
		return nil
	}
	return errors.New("test error")
}

type mockTGUpdatesReceiver struct {
	t   *testing.T
	hit int
}

func (m *mockTGUpdatesReceiver) String() string {
	return "mock updater"
}

func (m *mockTGUpdatesReceiver) ProcessUpdate(_ context.Context, textUpdate string) error {
	var result notify.TelegramUpdate
	err := json.Unmarshal([]byte(textUpdate), &result)
	assert.NoError(m.t, err)
	if m.hit < 2 {
		assert.NotNil(m.t, result.Result)
		m.hit++
		return nil
	}
	assert.Nil(m.t, result.Result)
	return errors.New("test error")
}
