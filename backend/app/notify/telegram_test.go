package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

func TestTelegram_New(t *testing.T) {
	ts := mockTelegramServer(nil)
	defer ts.Close()

	tb, err := NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "good-token",
		apiPrefix:      ts.URL + "/",
	})
	assert.NoError(t, err)
	assert.NotNil(t, tb)
	assert.Equal(t, tb.Timeout, time.Second*5)
	assert.Equal(t, "remark_test", tb.AdminChannelID, "@ added")

	_, err = NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "empty-json",
		apiPrefix:      ts.URL + "/",
	})
	assert.EqualError(t, err, "can't retrieve bot info from Telegram API: received empty result")

	st := time.Now()
	_, err = NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "non-json-resp",
		Timeout:        2 * time.Second,
		apiPrefix:      ts.URL + "/",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode json response:")
	assert.True(t, time.Since(st) >= 250*3*time.Millisecond)

	_, err = NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "404",
		Timeout:        2 * time.Second,
		apiPrefix:      ts.URL + "/",
	})
	assert.EqualError(t, err, "can't retrieve bot info from Telegram API: unexpected telegram API status code 404")

	_, err = NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "no-such-thing",
		apiPrefix:      "http://127.0.0.1:4321/",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can't retrieve bot info from Telegram API")
	assert.Contains(t, err.Error(), "dial tcp 127.0.0.1:4321: connect: connection refused")

	_, err = NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "no-such-thing",
		apiPrefix:      "",
	})
	assert.Error(t, err, "empty api url not allowed")

	_, err = NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "good-token",
		Timeout:        2 * time.Second,
		apiPrefix:      ts.URL + "/",
	})
	assert.NoError(t, err, "0 timeout allowed as default")

	tb, err = NewTelegram(TelegramParams{
		AdminChannelID: "1234567890",
		Token:          "good-token",
		apiPrefix:      ts.URL + "/",
	})
	assert.NoError(t, err)
	assert.NotNil(t, tb)
	assert.Equal(t, "1234567890", tb.AdminChannelID, "no @ prefix")
}

func TestTelegram_GetBotUsername(t *testing.T) {
	ts := mockTelegramServer(nil)
	defer ts.Close()

	tb, err := NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "good-token",
		apiPrefix:      ts.URL + "/",
	})
	assert.NoError(t, err)
	assert.NotNil(t, tb)
	assert.Equal(t, "remark42_test_bot", tb.GetBotUsername())
}

func TestTelegram_Send(t *testing.T) {
	ts := mockTelegramServer(nil)
	defer ts.Close()

	tb, err := NewTelegram(TelegramParams{
		AdminChannelID:    "remark_test",
		Token:             "good-token",
		UserNotifications: true,
		apiPrefix:         ts.URL + "/",
	})
	assert.NoError(t, err)
	assert.NotNil(t, tb)
	c := store.Comment{Text: "some text", ParentID: "1", ID: "999", Locator: store.Locator{URL: "http://example.org/"}}
	c.User.Name = "from"
	cp := store.Comment{Text: `<p>some parent text with a <a href="http://example.org">link</a> and special text:<br>& < > &</p>`}
	cp.User.Name = "to"

	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp, Telegrams: []string{"test_user_channel"}})
	assert.NoError(t, err)
	c.PostTitle = "test title"
	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp})
	assert.NoError(t, err)

	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp})
	assert.NoError(t, err)
	c.PostTitle = "[test title]"
	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp})
	assert.NoError(t, err)

	tb, err = NewTelegram(TelegramParams{
		AdminChannelID:    "remark_test",
		Token:             "non-json-resp",
		UserNotifications: true,
		apiPrefix:         ts.URL + "/",
	})
	assert.Nil(t, tb)
	assert.Error(t, err, "should fail")
	tb = &Telegram{
		TelegramParams: TelegramParams{
			AdminChannelID:    "remark_test",
			Token:             "non-json-resp",
			UserNotifications: true,
			apiPrefix:         ts.URL + "/",
		}}
	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp, Telegrams: []string{"test_user_channel"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected telegram API status code 404", "send on broken tg")

	assert.Equal(t, "telegram with admin notifications to remark_test with user notifications enabled", tb.String())

	// bad API URL
	tb.apiPrefix = "http://non-existent"
	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp, Telegrams: []string{"test_user_channel"}})
	assert.Error(t, err)

	// test buildMessage separately for message text
	res, err := buildMessage(Request{Comment: c, parent: cp})
	assert.NoError(t, err)
	assert.Equal(t, `{"text":"\u003ca href=\"http://example.org/#remark42__comment-999\"\u003efrom\u003c/a\u003e -\u003e \u003ca href=\"http://example.org/#remark42__comment-\"\u003eto\u003c/a\u003e\n\n`+
		`some text\n\n`+
		` \"_some parent text with a \u003ca href=\"http://example.org\"\u003elink\u003c/a\u003e and special text:\u0026amp; \u0026lt; \u0026gt; \u0026amp;_\"\n\n`+
		`↦  \u003ca href=\"http://example.org/\"\u003e[test title]\u003c/a\u003e","parse_mode":"HTML"}`,
		string(res))
}

func TestTelegram_SendVerification(t *testing.T) {
	ts := mockTelegramServer(nil)
	defer ts.Close()

	tb, err := NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "good-token",
		apiPrefix:      ts.URL + "/",
	})
	assert.NoError(t, err)
	assert.NotNil(t, tb)

	// empty VerificationRequest should return no error and no nothing, as well as any other
	assert.NoError(t, tb.SendVerification(context.TODO(), VerificationRequest{}))
}

const getUpdatesResp = `{
  "ok": true,
  "result": [
     {
        "update_id": 998,
        "message": {
           "chat": {
              "type": "group"
           }
        }
     },
     {
        "update_id": 999,
        "message": {
					 "text": "not starting with /start",
           "chat": {
              "type": "private"
           }
        }
     },
     {
        "update_id": 1000,
        "message": {
           "message_id": 4,
           "from": {
              "id": 313131313,
              "is_bot": false,
              "first_name": "Joe",
              "username": "joe123",
              "language_code": "en"
           },
           "chat": {
              "id": 313131313,
              "first_name": "Joe",
              "username": "joe123",
              "type": "private"
           },
           "date": 1601665548,
           "text": "/start token",
           "entities": [
              {
                 "offset": 0,
                 "length": 6,
                 "type": "bot_command"
              }
           ]
        }
     }
  ]
}`

func TestTelegram_GetUpdatesFlow(t *testing.T) {
	first := true
	ts := mockTelegramServer(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), "sendMessage") {
			// respond normally to processUpdates attempt to send message back to user
			_, _ = w.Write([]byte("{}"))
			return
		}
		// responses to get updates calls to API
		if first {
			assert.Equal(t, "", r.URL.Query().Get("offset"))
			first = false
		} else {
			assert.Equal(t, "1001", r.URL.Query().Get("offset"))
		}
		_, _ = w.Write([]byte(getUpdatesResp))
	})
	defer ts.Close()
	tb, err := NewTelegram(TelegramParams{
		AdminChannelID:    "remark_test",
		Token:             "xxxsupersecretxxx",
		UserNotifications: true,
		apiPrefix:         ts.URL + "/",
	})
	assert.NoError(t, err)

	// send request with no offset
	upd, err := tb.getUpdates(context.Background())
	assert.NoError(t, err)

	assert.Len(t, upd.Result, 3)
	assert.Equal(t, 1001, tb.updateOffset)
	assert.Equal(t, "/start token", upd.Result[len(upd.Result)-1].Message.Text)

	tb.AddToken("token", "user", "site", time.Now().Add(time.Minute))
	_, _, err = tb.CheckToken("token", "user")
	assert.Error(t, err)
	tb.processUpdates(context.Background(), upd)
	tgID, site, err := tb.CheckToken("token", "user")
	assert.NoError(t, err)
	assert.Equal(t, "313131313", tgID)
	assert.Equal(t, "site", site)

	// send request with offset
	_, err = tb.getUpdates(context.Background())
	assert.NoError(t, err)
}

func TestTelegram_ProcessUpdateFlow(t *testing.T) {
	ts := mockTelegramServer(func(w http.ResponseWriter, r *http.Request) {
		// respond normally to processUpdates attempt to send message back to user
		_, _ = w.Write([]byte("{}"))
	})
	defer ts.Close()
	tb, err := NewTelegram(TelegramParams{
		AdminChannelID:    "remark_test",
		Token:             "xxxsupersecretxxx",
		UserNotifications: true,
		apiPrefix:         ts.URL + "/",
	})
	assert.NoError(t, err)

	tb.AddToken("token", "user", "site", time.Now().Add(time.Minute))
	tb.AddToken("expired token", "user", "site", time.Now().Add(-time.Minute))
	assert.Len(t, tb.requests.data, 2)
	_, _, err = tb.CheckToken("token", "user")
	assert.Error(t, err)
	assert.NoError(t, tb.ProcessUpdate(context.Background(), getUpdatesResp))
	assert.Len(t, tb.requests.data, 1, "expired token was cleaned up")
	tgID, site, err := tb.CheckToken("token", "user")
	assert.NoError(t, err)
	assert.Len(t, tb.requests.data, 0, "token is deleted after successful check")
	assert.Equal(t, "313131313", tgID)
	assert.Equal(t, "site", site)

	tb.AddToken("expired token", "user", "site", time.Now().Add(-time.Minute))
	assert.Len(t, tb.requests.data, 1)
	assert.EqualError(t, tb.ProcessUpdate(context.Background(), ""), "failed to decode provided telegram update: unexpected end of JSON input")
	assert.Len(t, tb.requests.data, 0, "expired token should be cleaned up despite the error")
}

const sendMessageResp = `{
  "ok": true,
  "result": {
     "message_id": 100,
     "from": {
        "id": 666666666,
        "is_bot": true,
        "first_name": "Test auth bot",
        "username": "TestAuthBot"
     },
     "chat": {
        "id": 313131313,
        "first_name": "Joe",
        "username": "joe123",
        "type": "private"
     },
     "date": 1602430546,
     "text": "123"
  }
}`

func TestTelegram_SendText(t *testing.T) {
	ts := mockTelegramServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "123", r.URL.Query().Get("chat_id"))
		assert.Equal(t, "hello there", r.URL.Query().Get("text"))
		_, _ = w.Write([]byte(sendMessageResp))
	})
	defer ts.Close()
	tb, err := NewTelegram(TelegramParams{
		AdminChannelID:    "remark_test",
		Token:             "xxxsupersecretxxx",
		UserNotifications: true,
		apiPrefix:         ts.URL + "/",
	})
	assert.NoError(t, err)

	err = tb.sendText(context.Background(), 123, "hello there")
	assert.NoError(t, err)
}

const errorResp = `{"ok":false,"error_code":400,"description":"Very bad request"}`

func TestTelegram_Error(t *testing.T) {
	ts := mockTelegramServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(errorResp))
	})
	defer ts.Close()
	tb, err := NewTelegram(TelegramParams{
		AdminChannelID:    "remark_test",
		Token:             "xxxsupersecretxxx",
		UserNotifications: true,
		apiPrefix:         ts.URL + "/",
	})
	assert.NoError(t, err)

	_, err = tb.getUpdates(context.Background())
	assert.EqualError(t, err, "failed to fetch updates: unexpected telegram API status code 400, error: \"Very bad request\"")
}

func TestTelegram_TokenVerification(t *testing.T) {
	ts := mockTelegramServer(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.String(), "sendMessage") {
			// respond normally to processUpdates attempt to send message back to user
			_, _ = w.Write([]byte("{}"))
			return
		}
		// responses to get updates calls to API
		_, _ = w.Write([]byte(getUpdatesResp))
	})
	defer ts.Close()

	tb, err := NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "good-token",
		apiPrefix:      ts.URL + "/",
	})
	assert.NoError(t, err)
	assert.NotNil(t, tb)
	tb.AddToken("token", "user", "site", time.Now().Add(time.Minute))
	assert.Len(t, tb.requests.data, 1)

	// wrong token
	tgID, site, err := tb.CheckToken("unknown token", "user")
	assert.Empty(t, tgID)
	assert.Empty(t, site)
	assert.EqualError(t, err, "request is not found")

	// right token and user, not verified yet
	tgID, site, err = tb.CheckToken("token", "user")
	assert.Empty(t, tgID)
	assert.Empty(t, site)
	assert.EqualError(t, err, "request is not verified yet")

	// confirm request
	authRequest, ok := tb.requests.data["token"]
	assert.True(t, ok)
	authRequest.confirmed = true
	authRequest.telegramID = "telegramID"
	tb.requests.data["token"] = authRequest

	// wrong user
	tgID, site, err = tb.CheckToken("token", "wrong user")
	assert.Empty(t, tgID)
	assert.Empty(t, site)
	assert.EqualError(t, err, "user does not match original requester")

	// successful check
	tgID, site, err = tb.CheckToken("token", "user")
	assert.NoError(t, err)
	assert.Equal(t, "telegramID", tgID)
	assert.Equal(t, "site", site)

	// expired token
	tb.AddToken("expired token", "user", "site", time.Now().Add(-time.Minute))
	tgID, site, err = tb.CheckToken("expired token", "user")
	assert.Empty(t, tgID)
	assert.Empty(t, site)
	assert.EqualError(t, err, "request expired")
	assert.Len(t, tb.requests.data, 0)

	// expired token, cleaned up by the cleanup
	tb.apiPollInterval = time.Millisecond * 15
	tb.expiredCleanupInterval = time.Millisecond * 10
	ctx, cancel := context.WithCancel(context.Background())
	go tb.Run(ctx)
	assert.Eventually(t, func() bool {
		return tb.ProcessUpdate(ctx, "").Error() == "Run goroutine should not be used with ProcessUpdate"
	}, time.Millisecond*100, time.Millisecond*10, "ProcessUpdate should not work same time as Run")
	tb.AddToken("expired token", "user", "site", time.Now().Add(-time.Minute))
	tb.requests.RLock()
	assert.Len(t, tb.requests.data, 1)
	tb.requests.RUnlock()
	time.Sleep(tb.expiredCleanupInterval * 2)
	tb.requests.RLock()
	assert.Len(t, tb.requests.data, 0)
	tb.requests.RUnlock()
	cancel()
	// give enough time for Run() to finish
	time.Sleep(tb.expiredCleanupInterval)
}

const getMeResp = `{"ok": true,
				"result": {
					"first_name": "comments_test",
					"id": 707381019,
					"is_bot": true,
					"username": "remark42_test_bot"
				}}`

func mockTelegramServer(h http.HandlerFunc) *httptest.Server {
	if h != nil {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.String(), "getMe") {
				_, _ = w.Write([]byte(getMeResp))
				return
			}
			h(w, r)
		}))
	}
	router := chi.NewRouter()
	router.Get("/good-token/getMe", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(getMeResp))
	})
	router.Get("/empty-json/getMe", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	router.Get("/non-json-resp/getMe", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not-a-json`))
	})
	router.Get("/404/getMe", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	router.Post("/good-token/sendMessage", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok": true}`))
	})

	return httptest.NewServer(router)
}
