package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

func TestTelegram_New(t *testing.T) {

	ts := mockTelegramServer()
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

	st := time.Now()
	_, err = NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "bad-resp",
		apiPrefix:      ts.URL + "/",
	})
	assert.EqualError(t, err, "unexpected telegram response {OK:false Result:{FirstName:comments_test ID:707381019 IsBot:false UserName:remark42_test_bot}}")
	assert.True(t, time.Since(st) >= 250*5*time.Millisecond)

	_, err = NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "non-json-resp",
		Timeout:        2 * time.Second,
		apiPrefix:      ts.URL + "/",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can't decode response:")

	_, err = NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "404",
		Timeout:        2 * time.Second,
		apiPrefix:      ts.URL + "/",
	})
	assert.EqualError(t, err, "unexpected telegram API status code 404")

	_, err = NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "no-such-thing",
		apiPrefix:      "http://127.0.0.1:4321/",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can't initialize telegram notifications")
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

func TestTelegram_Send(t *testing.T) {
	ts := mockTelegramServer()
	defer ts.Close()

	tb, err := NewTelegram(TelegramParams{
		AdminChannelID:    "remark_test",
		Token:             "good-token",
		UserNotifications: true,
		apiPrefix:         ts.URL + "/",
	})
	assert.NoError(t, err)
	assert.NotNil(t, tb)
	c := store.Comment{Text: "some text", ParentID: "1", ID: "999"}
	c.User.Name = "from"
	cp := store.Comment{Text: "some parent text"}
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
	assert.Error(t, err, "should fail")
	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp, Telegrams: []string{"test_user_channel"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected telegram API status code 404", "send on broken tg")

	assert.Equal(t, "telegram with admin notifications to remark_test with user notifications enabled", tb.String())

	// bad API URL
	tb.apiPrefix = "http://non-existent"
	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp, Telegrams: []string{"test_user_channel"}})
	assert.Error(t, err)
}

func TestTelegram_SendVerification(t *testing.T) {
	ts := mockTelegramServer()
	defer ts.Close()

	tb, err := NewTelegram(TelegramParams{
		AdminChannelID: "remark_test",
		Token:          "good-token",
		apiPrefix:      ts.URL + "/",
	})
	assert.NoError(t, err)
	assert.NotNil(t, tb)

	// proper VerificationRequest without telegram
	req := VerificationRequest{
		SiteID: "remark",
		User:   "test_username",
		Token:  "secret_",
	}
	assert.NoError(t, tb.SendVerification(context.TODO(), req))

	// proper VerificationRequest with telegram
	req.Telegram = "test"
	assert.NoError(t, tb.SendVerification(context.TODO(), req))

	// VerificationRequest with canceled context
	ctx, cancel := context.WithCancel(context.TODO())
	cancel()
	assert.EqualError(t, tb.SendVerification(ctx, req), "sending message to \"test_username\" aborted due to canceled context")

	// test buildVerificationMessage separately for message text
	res, err := tb.buildVerificationMessage(req.User, req.Token, req.SiteID)
	assert.NoError(t, err)
	assert.Contains(t, string(res), "This is confirmation for test\\\\_username on site remark")
	assert.Contains(t, string(res), `secret_`)
}

func mockTelegramServer() *httptest.Server {
	router := chi.NewRouter()
	router.Get("/good-token/getMe", func(w http.ResponseWriter, r *http.Request) {
		s := `{"ok": true,
				"result": {
					"first_name": "comments_test",
					"id": 707381019,
					"is_bot": true,
					"username": "remark42_test_bot"
				}}`
		_, _ = w.Write([]byte(s))
	})
	router.Get("/bad-resp/getMe", func(w http.ResponseWriter, r *http.Request) {
		s := `{"ok": false,
				"result": {
					"first_name": "comments_test",
					"id": 707381019,
					"is_bot": false,
					"username": "remark42_test_bot"
				}}`
		_, _ = w.Write([]byte(s))
	})
	router.Get("/non-json-resp/getMe", func(w http.ResponseWriter, r *http.Request) {
		s := `"ok": false,
				"result": {
					"first_name": "comments_test",
					"id": 707381019,
					"is_bot": false,
					"username": "remark42_test_bot"
				`
		_, _ = w.Write([]byte(s))
	})
	router.Get("/404/getMe", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	router.Post("/good-token/sendMessage", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok": true}`))
	})

	return httptest.NewServer(router)
}

func Test_escapeTitle(t *testing.T) {
	tbl := []struct {
		inp string
		out string
	}{
		{"", ""},
		{"something 123", "something 123"},
		{"something [123]", "something \\[123\\]"},
		{"something (123)", "something \\(123\\)"},
		{"something (123) [aaa]", "something \\(123\\) \\[aaa\\]"},
	}

	for i, tt := range tbl {
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.out, escapeText(tt.inp))
		})
	}

}
