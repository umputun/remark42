package notify

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
)

const (
	webhookDefaultTemplate = `{"text": "{{.Text}}"}`
)

// WebhookClient defines an interface of client for webhook
type WebhookClient interface {
	Do(*http.Request) (*http.Response, error)
}

// WebhookParams contain settings for webhook notifications
type WebhookParams struct {
	WebhookURL string
	Template   string
	Headers    []string
}

// Webhook implements notify.Destination for Webhook notifications
type Webhook struct {
	WebhookParams
	webhookClient   WebhookClient
	webhookTemplate *template.Template
}

// NewWebhook makes Webhook
func NewWebhook(client WebhookClient, params WebhookParams) (*Webhook, error) {
	res := &Webhook{WebhookParams: params}
	if res.WebhookURL == "" {
		return nil, errors.New("webhook URL is required for webhook notifications")
	}

	if res.Template == "" {
		res.Template = webhookDefaultTemplate
	}

	payloadTmpl, err := template.New("webhook").Parse(res.Template)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse webhook template")
	}

	res.webhookClient = client
	res.webhookTemplate = payloadTmpl

	log.Printf("[DEBUG] create new webhook notifier for %s", res.WebhookURL)

	return res, nil
}

// Send sends Webhook notification
func (t *Webhook) Send(ctx context.Context, req Request) error {
	var payload bytes.Buffer
	err := t.webhookTemplate.Execute(&payload, req.Comment)
	if err != nil {
		return errors.Wrap(err, "unable to compile webhook template")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", t.WebhookURL, &payload)
	if err != nil {
		return errors.Wrap(err, "unable to create webhook request")
	}

	for _, h := range t.Headers {
		elems := strings.Split(h, ":")
		if len(elems) != 2 {
			continue
		}
		httpReq.Header.Set(strings.TrimSpace(elems[0]), strings.TrimSpace(elems[1]))
	}

	resp, err := t.webhookClient.Do(httpReq)
	if err != nil {
		return errors.Wrap(err, "webhook request failed")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("webhook request failed with non-OK status code: %d", resp.StatusCode)
		respBody, e := io.ReadAll(resp.Body)
		if e != nil {
			return fmt.Errorf(errMsg)
		}
		return fmt.Errorf("%s, body: %s", errMsg, respBody)
	}

	log.Printf("[DEBUG] send webhook notification, comment id %s", req.Comment.ID)

	return nil
}

// SendVerification is not implemented for Webhook
func (t *Webhook) SendVerification(_ context.Context, _ VerificationRequest) error {
	return nil
}

// String describes the webhook instance
func (t *Webhook) String() string {
	return fmt.Sprintf("webhook notification to %s", t.WebhookURL)
}
