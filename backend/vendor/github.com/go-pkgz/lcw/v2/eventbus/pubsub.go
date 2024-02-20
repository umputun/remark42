// Package eventbus provides PubSub interface used for distributed cache invalidation,
// as well as NopPubSub and RedisPubSub implementations.
package eventbus

// PubSub interface is used for distributed cache invalidation.
// Publish is called on each entry invalidation,
// Subscribe is used for subscription for these events.
type PubSub interface {
	Publish(fromID, key string) error
	Subscribe(fn func(fromID, key string)) error
}

// NopPubSub implements default do-nothing pub-sub (event bus)
type NopPubSub struct{}

// Subscribe does nothing for NopPubSub
func (n *NopPubSub) Subscribe(func(fromID string, key string)) error {
	return nil
}

// Publish does nothing for NopPubSub
func (n *NopPubSub) Publish(string, string) error {
	return nil
}
