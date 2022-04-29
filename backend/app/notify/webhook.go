package notify

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	log "github.com/go-pkgz/lgr"
	ntf "github.com/go-pkgz/notify"
)

const (
	webhookDefaultTemplate = `{"text": "{{.Text}}"}`
)

// WebhookParams contain settings for webhook notifications
type WebhookParams struct {
	URL      string
	Template string
	Headers  []string
	Timeout  time.Duration
}

// Webhook implements notify.Destination for Webhook notifications
type Webhook struct {
	*ntf.Webhook

	url      string
	template *template.Template
}

// NewWebhook makes Webhook
func NewWebhook(params WebhookParams) (*Webhook, error) {
	res := &Webhook{
		Webhook: ntf.NewWebhook(ntf.WebhookParams{
			Timeout: params.Timeout,
			Headers: params.Headers,
		}),
		url: params.URL,
	}

	if res.url == "" {
		return nil, fmt.Errorf("webhook URL is required for webhook notifications")
	}

	if params.Template == "" {
		params.Template = webhookDefaultTemplate
	}

	payloadTmpl, err := template.New("webhook").Parse(params.Template)
	if err != nil {
		return nil, fmt.Errorf("unable to parse webhook template: %w", err)
	}

	res.template = payloadTmpl

	log.Printf("[DEBUG] create new webhook notifier for %s", res.url)

	return res, nil
}

// Send sends Webhook notification
func (w *Webhook) Send(ctx context.Context, req Request) error {
	log.Printf("[DEBUG] send webhook notification, comment id %s", req.Comment.ID)
	var payload bytes.Buffer
	err := w.template.Execute(&payload, req.Comment)
	if err != nil {
		return fmt.Errorf("unable to compile webhook template: %w", err)
	}

	return w.Webhook.Send(ctx, w.url, payload.String())
}

// SendVerification is not implemented for Webhook
func (w *Webhook) SendVerification(_ context.Context, _ VerificationRequest) error {
	return nil
}

// String describes the webhook instance
func (w *Webhook) String() string {
	return fmt.Sprintf("%s to %s", w.Webhook.String(), w.url)
}
