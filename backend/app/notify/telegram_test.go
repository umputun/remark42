package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

func TestTelegram_New(t *testing.T) {

	ts := mockTelegramServer()
	defer ts.Close()

	tb, err := NewTelegram("good-token", "remark_test", 2*time.Second, ts.URL+"/")
	assert.NoError(t, err)
	assert.NotNil(t, tb)
	assert.Equal(t, "@remark_test", tb.channelID, "@ added")

	st := time.Now()
	_, err = NewTelegram("bad-resp", "remark_test", 2*time.Second, ts.URL+"/")
	assert.EqualError(t, err, "unexpected telegram response {OK:false Result:{FirstName:comments_test ID:707381019 IsBot:false UserName:remark42_test_bot}}")
	assert.True(t, time.Since(st) >= 250*5*time.Millisecond)

	_, err = NewTelegram("non-json-resp", "remark_test", 2*time.Second, ts.URL+"/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can't decode response:")

	_, err = NewTelegram("404", "remark_test", 2*time.Second, ts.URL+"/")
	assert.EqualError(t, err, "unexpected telegram status code 404")

	_, err = NewTelegram("no-such-thing", "remark_test", 2*time.Second, "http://127.0.0.1:4321/")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can't initialize telegram notifications")
	assert.Contains(t, err.Error(), "dial tcp 127.0.0.1:4321: connect: connection refused")

	_, err = NewTelegram("good-token", "remark_test", 2*time.Second, "")
	assert.Error(t, err, "empty api url not allowed")

	_, err = NewTelegram("good-token", "remark_test", 0, ts.URL+"/")
	assert.NoError(t, err, "0 timeout allowed as default")

	tb, err = NewTelegram("good-token", "1234567890", 2*time.Second, ts.URL+"/")
	assert.NoError(t, err)
	assert.NotNil(t, tb)
	assert.Equal(t, "1234567890", tb.channelID, "no @ prefix")
}

func TestTelegram_Send(t *testing.T) {
	ts := mockTelegramServer()
	defer ts.Close()

	tb, err := NewTelegram("good-token", "remark_test", 2*time.Second, ts.URL+"/")
	assert.NoError(t, err)
	assert.NotNil(t, tb)
	c := store.Comment{Text: "some text", ParentID: "1", ID: "999"}
	c.User.Name = "from"
	cp := store.Comment{Text: "some parent text"}
	cp.User.Name = "to"

	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp})
	assert.NoError(t, err)
	c.PostTitle = "test title"
	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp})
	assert.NoError(t, err)

	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp})
	assert.NoError(t, err)
	c.PostTitle = "[test title]"
	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp})
	assert.NoError(t, err)

	tb, err = NewTelegram("non-json-resp", "remark_test", 2*time.Second, ts.URL+"/")
	assert.Error(t, err, "should failed")
	err = tb.Send(context.TODO(), Request{Comment: c, parent: cp})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected telegram status code 404", "send on broken tg")

	assert.Equal(t, "telegram: @remark_test", tb.String())
}

func TestTelegram_SendVerification(t *testing.T) {
	ts := mockTelegramServer()
	defer ts.Close()

	tb, err := NewTelegram("good-token", "remark_test", 2*time.Second, ts.URL+"/")
	assert.NoError(t, err)
	assert.NotNil(t, tb)

	err = tb.SendVerification(context.TODO(), VerificationRequest{})
	assert.NoError(t, err)
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

func TestTelegram_escapeTitle(t *testing.T) {

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

	tb := Telegram{}
	for i, tt := range tbl {
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.out, tb.escapeTitle(tt.inp))
		})
	}

}
