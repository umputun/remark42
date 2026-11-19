package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	ntf "github.com/go-pkgz/notify"
	"github.com/stretchr/testify/assert"
)

func TestDispatchTelegramUpdates(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		poolPeriod := time.Millisecond * 100
		requester := &mockTGRequester{t: t}
		receivers := []*mockTGUpdatesReceiver{{t: t}, {t: t}}
		done := make(chan struct{})
		go func() {
			DispatchTelegramUpdates(ctx, requester, []TGUpdatesReceiver{receivers[0], receivers[1]}, poolPeriod)
			close(done)
		}()
		time.Sleep(poolPeriod*3 + poolPeriod/2) // three polls: two answered with an update, one failed
		cancel()
		<-done

		// the offset is the last update seen plus one, and the failed third poll keeps it
		assert.Equal(t, []string{
			`getUpdates?allowed_updates=["message"]`,
			`getUpdates?allowed_updates=["message"]&offset=999`,
			`getUpdates?allowed_updates=["message"]&offset=999`,
		}, requester.urls)
		// every receiver gets each answered poll, and nothing from the failed one
		for i, r := range receivers {
			assert.Equal(t, []int{998, 998}, r.updateIDs, "receiver %d", i)
		}
	})
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

// mockTGRequester answers the first two polls with one update and fails the rest
type mockTGRequester struct {
	t    *testing.T
	urls []string
}

func (m *mockTGRequester) Request(_ context.Context, url string, _ []byte, data any) error {
	m.urls = append(m.urls, url)
	if len(m.urls) <= 2 {
		assert.NoError(m.t, json.Unmarshal([]byte(getUpdatesResp), data))
		return nil
	}
	return fmt.Errorf("test error")
}

// mockTGUpdatesReceiver records the ids of the updates it is given
type mockTGUpdatesReceiver struct {
	t         *testing.T
	updateIDs []int
}

func (m *mockTGUpdatesReceiver) String() string {
	return "mock updater"
}

func (m *mockTGUpdatesReceiver) ProcessUpdate(_ context.Context, textUpdate string) error {
	var result ntf.TelegramUpdate
	assert.NoError(m.t, json.Unmarshal([]byte(textUpdate), &result))
	for _, u := range result.Result {
		m.updateIDs = append(m.updateIDs, u.UpdateID)
	}
	return nil
}
