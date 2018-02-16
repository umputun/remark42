package notifier

import (
	"log"

	"github.com/umputun/remark/app/store"
)

// Interface defines notifier, sending messages triggered by topic/reply updates
type Interface interface {
	Subscribe(locator store.Locator, user store.User) error
	UnSubscribe(locator store.Locator, user store.User) error
	OnUpdate(locator store.Locator) error
}

// NoOperation implements Interface doing nothing but logging
type NoOperation struct{}

// Subscribe is a fake, just loging attempt
func (n NoOperation) Subscribe(locator store.Locator, user store.User) error {
	log.Printf("[DEBUG] user %+v subscribed to %+v", user, locator)
	return nil
}

// UnSubscribe is a fake, just loging attempt
func (n NoOperation) UnSubscribe(locator store.Locator, user store.User) error {
	log.Printf("[DEBUG] user %+v unsubscribed from %+v", user, locator)
	return nil
}

// OnUpdate is a fake, just loging event
func (n NoOperation) OnUpdate(locator store.Locator) error {
	log.Printf("[DEBUG] update for %+v", locator)
	return nil
}
