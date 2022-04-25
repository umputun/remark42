// Package notify provides notification functionality.
package notify

import (
	"context"
	"fmt"
	"strings"
)

// Notifier defines common interface among all notifiers
type Notifier interface {
	fmt.Stringer
	Schema() string                                           // returns schema prefix supported by this client
	Send(ctx context.Context, destination, text string) error // sends message to provided destination
}

// Send sends message to provided destination, picking the right one based on destination schema
func Send(ctx context.Context, notifiers []Notifier, destination, text string) error {
	for _, n := range notifiers {
		if strings.HasPrefix(destination, n.Schema()) {
			return n.Send(ctx, destination, text)
		}
	}
	if strings.Contains(destination, ":") {
		return fmt.Errorf("unsupported destination schema: %s", strings.Split(destination, ":")[0])
	}
	return fmt.Errorf("unsupported destination schema: %s", destination)
}
