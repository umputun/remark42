package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/umputun/remark/backend/app/store"
)

func TestTelegram_New(t *testing.T) {

	ts := mockTelegramServer()
	defer ts.Close()

	tb, err := NewTelegram("good-token", "remark_test", 2*time.Second, ts.URL+"/")
	assert.NoError(t, err)
	assert.NotNil(t, tb)

	_, err = NewTelegram("bad-resp", "remark_test", 2*time.Second, ts.URL+"/")
	assert.EqualError(t, err, "unexpected telegram response {OK:false Result:{FirstName:comments_test ID:707381019 IsBot:false UserName:remark42_test_bot}}")

	_, err = NewTelegram("non-json-resp", "remark_test", 2*time.Second, ts.URL+"/")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "can't decode response:")

	_, err = NewTelegram("404", "remark_test", 2*time.Second, ts.URL+"/")
	assert.EqualError(t, err, "unexpected telegram status code 404")

	_, err = NewTelegram("no-such-thing", "remark_test", 2*time.Second, "http://127.0.0.1:4321/")
	assert.EqualError(t, err, "can't initialize telegram notifications: Get http://127.0.0.1:4321/no-such-thing/getMe: dial tcp 127.0.0.1:4321: connect: connection refused")
}

func TestTelegram_Send(t *testing.T) {
	ts := mockTelegramServer()
	defer ts.Close()

	tb, err := NewTelegram("good-token", "remark_test", 2*time.Second, ts.URL+"/")
	assert.NoError(t, err)
	assert.NotNil(t, tb)
	c := store.Comment{Text: "some text", ParentID: "1"}
	c.User.Name = "from"
	cp := store.Comment{Text: "some parent text"}
	cp.User.Name = "to"

	err = tb.Send(context.TODO(), request{comment: c, parent: cp})
	assert.NoError(t, err)

	assert.Equal(t, "telegram: remark_test", tb.String())
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

	router.Get("/good-token/sendMessage", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok": true}`))
	})

	return httptest.NewServer(router)
}
