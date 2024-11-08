package notify

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const webhookTimeOut = 5000 * time.Millisecond

// WebhookParams contain settings for webhook notifications
type WebhookParams struct {
	Timeout time.Duration
	Headers []string // headers in format "header:value"
}

// Webhook notifications client
type Webhook struct {
	WebhookParams
	webhookClient webhookClient
}

// webhookClient defines an interface of client for webhook
type webhookClient interface {
	Do(*http.Request) (*http.Response, error)
}

// NewWebhook makes Webhook
func NewWebhook(params WebhookParams) *Webhook {
	res := &Webhook{WebhookParams: params}

	if res.Timeout == 0 {
		res.Timeout = webhookTimeOut
	}

	res.webhookClient = &http.Client{Timeout: res.Timeout}

	return res
}

// Send sends Webhook notification. Destination field is expected to have http:// or https:// schema.
//
// Example:
//
// - https://example.com/webhook
func (wh *Webhook) Send(ctx context.Context, destination, text string) error {
	payload := bytes.NewBufferString(text)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", destination, payload)
	if err != nil {
		return fmt.Errorf("unable to create webhook request: %w", err)
	}

	for _, h := range wh.Headers {
		elems := strings.Split(h, ":")
		if len(elems) != 2 {
			continue
		}
		httpReq.Header.Set(strings.TrimSpace(elems[0]), strings.TrimSpace(elems[1]))
	}

	resp, err := wh.webhookClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		errMsg := fmt.Sprintf("webhook request failed with non-OK status code: %d", resp.StatusCode)
		respBody, e := io.ReadAll(resp.Body)
		if e != nil {
			return errors.New(errMsg)
		}
		return fmt.Errorf("%s, body: %s", errMsg, respBody)
	}

	return nil
}

// Schema returns schema prefix supported by this client
func (wh *Webhook) Schema() string {
	return "http"
}

// String describes the webhook instance
func (wh *Webhook) String() string {
	str := fmt.Sprintf("webhook notification with timeout %s", wh.Timeout)
	if wh.Headers != nil {
		str += fmt.Sprintf(" and headers %v", wh.Headers)
	}
	return str
}
