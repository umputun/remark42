package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/notify"
	"github.com/umputun/remark42/backend/app/store"
)

func TestRest_CreateAndFindCommentWithQueryParams(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	locatorURL := "https://example.com/post?a=1&b=2"
	body, err := json.Marshal(map[string]any{
		"text": "query parameter locator",
		"locator": map[string]string{
			"site": "remark42",
			"url":  locatorURL,
		},
	})
	require.NoError(t, err)

	resp, err := post(t, ts.URL+"/api/v1/comment?site=remark42", string(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(responseBody))

	created := struct {
		ID string `json:"id"`
	}{}
	require.NoError(t, json.Unmarshal(responseBody, &created))
	require.NotEmpty(t, created.ID)

	result, status := get(t, ts.URL+"/api/v1/find?site=remark42&url="+url.QueryEscape(locatorURL))
	require.Equal(t, http.StatusOK, status, result)
	found := commentsWithInfo{}
	require.NoError(t, json.Unmarshal([]byte(result), &found))
	require.Len(t, found.Comments, 1)
	assert.Equal(t, created.ID, found.Comments[0].ID)
	assert.Equal(t, locatorURL, found.Info.URL)
}

func TestTelegramSendEscapesLocatorURLInHTMLHref(t *testing.T) {
	messageCh := make(chan string, 1)
	telegramAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bottest/getMe":
			_, _ = fmt.Fprint(w, `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"test","username":"testbot"}}`)
		case "/bottest/sendMessage":
			var payload struct {
				Text string `json:"text"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			messageCh <- payload.Text
			_, _ = fmt.Fprint(w, `{"ok":true,"result":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer telegramAPI.Close()

	tg, err := notify.NewTelegram(notify.TelegramParams{
		AdminChannelID: "test-channel",
		Token:          "test",
		APIURL:         telegramAPI.URL,
	})
	require.NoError(t, err)

	locatorURL := `https://example.com/?q="><b>spoof</b>`
	err = tg.Send(context.Background(), notify.Request{Comment: store.Comment{
		ID:        "comment-id",
		ParentID:  "parent-id",
		PostTitle: "Post title",
		Locator:   store.Locator{URL: locatorURL},
		User:      store.User{Name: "Alice"},
	}})
	require.NoError(t, err)
	message := <-messageCh
	escapedURL := `href="` + html.EscapeString(locatorURL)
	assert.Equal(t, 3, strings.Count(message, escapedURL))
	assert.Contains(t, message, escapedURL+`#remark42__comment-comment-id">Alice</a>`)
	assert.NotContains(t, message, `<b>spoof</b>`)
}
