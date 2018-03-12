// Package notifier handles update notification as well as subscriptions to notification
package notifier

import (
	"fmt"
	"log"
	"sync"

	"github.com/umputun/remark/app/store"
)

// Interface defines notifier, sending messages triggered by topic/reply updates
type Interface interface {
	Subscribe(user store.User) error
	UnSubscribe(user store.User) error
	OnUpdate(comment store.Comment) error
	Status(user store.User) (bool, error)
}

// NoOperation implements Interface doing nothing but logging
type NoOperation struct {
	sync.RWMutex
	status map[string]struct{}
}

// NewNoOperation makes NoOperation fake notifier
func NewNoOperation() *NoOperation {
	res := NoOperation{status: map[string]struct{}{}}
	return &res
}

// Subscribe is a fake, just logging attempt
func (n *NoOperation) Subscribe(user store.User) error {
	n.Lock()
	n.status[user.ID] = struct{}{}
	n.Unlock()
	log.Printf("[DEBUG] user %+v subscribed to updates", user)
	return nil
}

// UnSubscribe is a fake, just logging attempt
func (n *NoOperation) UnSubscribe(user store.User) error {
	n.Lock()
	delete(n.status, user.ID)
	n.Unlock()
	log.Printf("[DEBUG] user %+v unsubscribed from updates", user)
	return nil
}

// OnUpdate is a fake, just logging event
func (n *NoOperation) OnUpdate(comment store.Comment) error {
	log.Printf("[DEBUG] update for %+v", comment)
	return nil
}

// Status returns from in-memory map
func (n *NoOperation) Status(user store.User) (bool, error) {
	n.RLock()
	defer n.RUnlock()
	_, found := n.status[user.ID]
	return found, nil
}

func (n *NoOperation) key(locator store.Locator, user store.User) string {
	return fmt.Sprintf("%+v-%s", locator, user.ID)
}
