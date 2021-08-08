package notify

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

type funcWebhookClient func(*http.Request) (*http.Response, error)

func (c funcWebhookClient) Do(r *http.Request) (*http.Response, error) {
	return c(r)
}

var okWebhookClient = funcWebhookClient(func(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString("ok")),
	}, nil
})

type errReader struct {
}

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func TestWebhook_NewWebhook(t *testing.T) {

	wh, err := NewWebhook(okWebhookClient, WebhookParams{
		WebhookURL: "https://example.org/webhook",
		Headers:    []string{"Authorization:Basic AXVubzpwQDU1dzByYM=="},
	})
	assert.NoError(t, err)
	assert.NotNil(t, wh)

	assert.Equal(t, "https://example.org/webhook", wh.WebhookURL)
	assert.Equal(t, []string{"Authorization:Basic AXVubzpwQDU1dzByYM=="}, wh.Headers)
	assert.Equal(t, `{"text": "{{.Text}}"}`, wh.Template)

	wh, err = NewWebhook(okWebhookClient, WebhookParams{
		WebhookURL: "https://example.org/webhook",
		Headers:    []string{"Authorization:Basic AXVubzpwQDU1dzByYM=="},
		Template:   "{{.Text}}",
	})
	assert.NoError(t, err)
	assert.NotNil(t, wh)
	assert.Equal(t, "{{.Text}}", wh.Template)

	wh, err = NewWebhook(okWebhookClient, WebhookParams{})
	assert.Nil(t, wh)
	assert.Error(t, err)
	assert.Equal(t, "webhook URL is required for webhook notifications", err.Error())

	wh, err = NewWebhook(okWebhookClient, WebhookParams{WebhookURL: "https://example.org/webhook", Template: "{{.Text"})
	assert.Nil(t, wh)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse webhook template")
}

func TestWebhook_Send(t *testing.T) {

	wh, err := NewWebhook(funcWebhookClient(func(r *http.Request) (*http.Response, error) {
		assert.Len(t, r.Header, 1)
		assert.Equal(t, r.Header.Get("Content-Type"), "application/json,text/plain")

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBufferString("")),
		}, nil
	}), WebhookParams{
		WebhookURL: "https://example.org/webhook",
		Headers:    []string{"Content-Type:application/json,text/plain", ""},
	})
	assert.NoError(t, err)
	assert.NotNil(t, wh)

	c := store.Comment{Text: "some text", ParentID: "1", ID: "999"}
	c.User.Name = "from"

	err = wh.Send(context.TODO(), Request{Comment: c})
	assert.NoError(t, err)

	wh, err = NewWebhook(okWebhookClient, WebhookParams{
		WebhookURL: "https://example.org/webhook",
		Template:   "{{.InvalidProperty}}",
	})
	assert.NoError(t, err)
	err = wh.Send(context.TODO(), Request{Comment: c})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook template")

	wh, err = NewWebhook(okWebhookClient, WebhookParams{WebhookURL: "https://example.org/webhook"})
	assert.NoError(t, err)
	err = wh.Send(nil, Request{Comment: c}) // nolint
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to create webhook request")

	wh, err = NewWebhook(funcWebhookClient(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("request failed")
	}), WebhookParams{WebhookURL: "https://not-existing-url.net"})
	assert.NoError(t, err)
	err = wh.Send(context.TODO(), Request{Comment: c})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook request failed")

	wh, err = NewWebhook(funcWebhookClient(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(bytes.NewBufferString("not found")),
		}, nil
	}), WebhookParams{
		WebhookURL: "http:/example.org/invalid-url",
	})
	assert.NoError(t, err)
	err = wh.Send(context.TODO(), Request{Comment: c})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-OK status code: 404, body: not found")

	wh, err = NewWebhook(funcWebhookClient(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(errReader{}),
		}, nil
	}), WebhookParams{
		WebhookURL: "http:/example.org/invalid-url",
	})
	assert.NoError(t, err)
	err = wh.Send(context.TODO(), Request{Comment: c})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-OK status code: 404")
	assert.NotContains(t, err.Error(), "body")
}

func TestWebhook_SendVerification(t *testing.T) {
	wh, err := NewWebhook(okWebhookClient, WebhookParams{WebhookURL: "https://example.org/webhook"})
	assert.NoError(t, err)
	assert.NotNil(t, wh)

	err = wh.SendVerification(context.TODO(), VerificationRequest{})
	assert.NoError(t, err)
}

func TestWebhook_String(t *testing.T) {
	wh, err := NewWebhook(okWebhookClient, WebhookParams{WebhookURL: "https://example.org/webhook"})
	assert.NoError(t, err)
	assert.NotNil(t, wh)

	str := wh.String()
	assert.Equal(t, "webhook notification to https://example.org/webhook", str)
}
