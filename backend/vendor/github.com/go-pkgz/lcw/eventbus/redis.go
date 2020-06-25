package eventbus

import (
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// NewRedisPubSub creates new RedisPubSub with given parameters.
// Returns an error in case of problems with creating PubSub client for specified channel.
func NewRedisPubSub(addr, channel string) (*RedisPubSub, error) {
	client := redis.NewClient(&redis.Options{Addr: addr})
	pubSub := client.Subscribe(channel)
	// wait for subscription to be created and ignore the message
	if _, err := pubSub.Receive(); err != nil {
		_ = client.Close()
		return nil, errors.Wrapf(err, "problem subscribing to channel %s on address %s", channel, addr)
	}
	return &RedisPubSub{client: client, pubSub: pubSub, channel: channel, done: make(chan struct{})}, nil
}

// RedisPubSub provides Redis implementation for PubSub interface
type RedisPubSub struct {
	client  *redis.Client
	pubSub  *redis.PubSub
	channel string

	done chan struct{}
}

// Subscribe calls provided function on subscription channel provided on new RedisPubSub instance creation.
// Should not be called more than once. Spawns a goroutine and does not return an error.
func (m *RedisPubSub) Subscribe(fn func(fromID, key string)) error {
	go func(done <-chan struct{}, pubsub *redis.PubSub) {
		for {
			select {
			case <-done:
				return
			default:
			}
			msg, err := pubsub.ReceiveTimeout(time.Second * 10)
			if err != nil {
				continue
			}

			// Process the message
			if msg, ok := msg.(*redis.Message); ok {
				payload := strings.Split(msg.Payload, "$")
				fn(payload[0], strings.Join(payload[1:], "$"))
			}
		}
	}(m.done, m.pubSub)

	return nil
}

// Publish publishes provided message to channel provided on new RedisPubSub instance creation
func (m *RedisPubSub) Publish(fromID, key string) error {
	return m.client.Publish(m.channel, fromID+"$"+key).Err()
}

// Close cleans up running goroutines and closes Redis clients
func (m *RedisPubSub) Close() error {
	close(m.done)
	errs := new(multierror.Error)
	errs = multierror.Append(errs, errors.Wrap(m.pubSub.Close(), "problem closing pubSub client"))
	errs = multierror.Append(errs, errors.Wrap(m.client.Close(), "problem closing redis client"))
	return errs.ErrorOrNil()
}
