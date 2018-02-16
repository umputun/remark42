package notifier

import (
	"fmt"
	"log"
	"sync"

	"github.com/umputun/remark/app/store"
)

// Interface defines notifier, sending messages triggered by topic/reply updates
type Interface interface {
	Subscribe(locator store.Locator, user store.User) error
	UnSubscribe(locator store.Locator, user store.User) error
	OnUpdate(locator store.Locator) error
	Status(locator store.Locator, user store.User) (bool, error)
}

// NoOperation implements Interface doing nothing but logging
type NoOperation struct {
	sync.RWMutex
	status map[string]struct{}
}

// NewNoperation makes NoOperation fake notifier
func NewNoperation() *NoOperation {
	res := NoOperation{status: map[string]struct{}{}}
	return &res
}

// Subscribe is a fake, just loging attempt
func (n *NoOperation) Subscribe(locator store.Locator, user store.User) error {
	n.Lock()
	n.status[n.key(locator, user)] = struct{}{}
	n.Unlock()
	log.Printf("[DEBUG] user %+v subscribed to %+v", user, locator)
	return nil
}

// UnSubscribe is a fake, just loging attempt
func (n *NoOperation) UnSubscribe(locator store.Locator, user store.User) error {
	n.Lock()
	delete(n.status, n.key(locator, user))
	n.Unlock()
	log.Printf("[DEBUG] user %+v unsubscribed from %+v", user, locator)
	return nil
}

// OnUpdate is a fake, just loging event
func (n *NoOperation) OnUpdate(locator store.Locator) error {
	log.Printf("[DEBUG] update for %+v", locator)
	return nil
}

// Status returns from in-memory
func (n *NoOperation) Status(locator store.Locator, user store.User) (bool, error) {
	n.RLock()
	defer n.RUnlock()
	_, found := n.status[n.key(locator, user)]
	return found, nil
}

func (n *NoOperation) key(locator store.Locator, user store.User) string {
	return fmt.Sprintf("%+v-%s", locator, user.ID)
}
