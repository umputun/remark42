package providers

// Both Telegram auth and notifications need to receive messages received by Telegram bot in the loop,
// and below is the implementation of such loop which dispatched received events to both receivers,
// so that they could work at the same time.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/go-pkgz/lgr"

	"github.com/umputun/remark42/backend/app/notify"
)

type tgRequester interface {
	Request(ctx context.Context, method string, b []byte, data interface{}) error
}

// TGUpdatesReceiver used to dispatch telegram updates to multiple receivers
type TGUpdatesReceiver interface {
	fmt.Stringer
	ProcessUpdate(ctx context.Context, textUpdate string) error
}

// DispatchTelegramUpdates dispatches telegram updates to provided list of receivers
// Blocks caller
func DispatchTelegramUpdates(ctx context.Context, requester tgRequester, receivers []TGUpdatesReceiver, period time.Duration) {
	// Identifier of the first update to be requested.
	// Should be equal to LastSeenUpdateID + 1
	// See https://core.telegram.org/bots/api#getupdates
	var updateOffset int

	processUpdatedTicker := time.NewTicker(period)
	for {
		select {
		case <-ctx.Done():
			processUpdatedTicker.Stop()
			return
		case <-processUpdatedTicker.C:
			url := `getUpdates?allowed_updates=["message"]`
			if updateOffset != 0 {
				url += fmt.Sprintf("&offset=%d", updateOffset)
			}

			var update notify.TelegramUpdate

			err := requester.Request(ctx, url, nil, &update)
			if err != nil {
				log.Printf("[WARN] failed to fetch updates: %v", err)
				continue
			}

			for _, u := range update.Result {
				if u.UpdateID >= updateOffset {
					updateOffset = u.UpdateID + 1
				}
			}

			if raw, err := json.Marshal(update); err == nil {
				for _, r := range receivers {
					e := r.ProcessUpdate(ctx, string(raw))
					if e != nil {
						log.Printf("[ERROR] failure from destination %s on processing telegram update %v", r, e)
					}
				}
			}
		}
	}
}
