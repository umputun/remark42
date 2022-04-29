package notify

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

func TestWebhook_NewWebhook(t *testing.T) {
	wh, err := NewWebhook(WebhookParams{
		URL:     "https://example.org/webhook",
		Headers: []string{"Authorization:Basic AXVubzpwQDU1dzByYM=="},
	})
	assert.NoError(t, err)
	assert.NotNil(t, wh)

	assert.Equal(t, "https://example.org/webhook", wh.url)
	assert.Equal(t, []string{"Authorization:Basic AXVubzpwQDU1dzByYM=="}, wh.Headers)
	assert.NotNil(t, wh.template)

	wh, err = NewWebhook(WebhookParams{})
	assert.Nil(t, wh)
	assert.Error(t, err)
	assert.Equal(t, "webhook URL is required for webhook notifications", err.Error())

	wh, err = NewWebhook(WebhookParams{URL: "https://example.org/webhook", Template: "{{.Text"})
	assert.Nil(t, wh)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse webhook template")
}

func TestWebhook_Send(t *testing.T) {
	wh, err := NewWebhook(WebhookParams{
		URL:     "bad-url",
		Headers: []string{"Content-Type:application/json,text/plain", ""},
	})
	assert.NoError(t, err)
	assert.NotNil(t, wh)

	c := store.Comment{Text: "some text", ParentID: "1", ID: "999"}
	c.User.Name = "from"

	err = wh.Send(context.Background(), Request{Comment: c})
	assert.Error(t, err)

	wh, err = NewWebhook(WebhookParams{
		URL:      "https://example.org/webhook",
		Template: "{{.InvalidProperty}}",
	})
	assert.NoError(t, err)
	err = wh.Send(context.Background(), Request{Comment: c})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook template")

	wh, err = NewWebhook(WebhookParams{URL: "https://example.org/webhook"})
	assert.NoError(t, err)
	err = wh.Send(nil, Request{Comment: c}) // nolint
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to create webhook request")

	wh, err = NewWebhook(WebhookParams{URL: "https://not-existing-url.net"})
	assert.NoError(t, err)
	err = wh.Send(context.Background(), Request{Comment: c})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook request failed")
}

func TestWebhook_SendVerification(t *testing.T) {
	wh, err := NewWebhook(WebhookParams{URL: "https://example.org/webhook"})
	assert.NoError(t, err)
	assert.NotNil(t, wh)

	err = wh.SendVerification(context.Background(), VerificationRequest{})
	assert.NoError(t, err)
}

func TestWebhook_String(t *testing.T) {
	wh, err := NewWebhook(WebhookParams{URL: "https://example.org/webhook", Timeout: time.Minute * 5})
	assert.NoError(t, err)
	assert.NotNil(t, wh)

	str := wh.String()
	assert.Equal(t, "webhook notification with timeout 5m0s to https://example.org/webhook", str)
}
