package notifier

import "context"

// Notifier delivers alert notifications to a channel.
type Notifier interface {
	Type() string
	Send(ctx context.Context, webhookURL, secret, message string) error
}
