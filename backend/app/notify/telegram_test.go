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

	tb, err = NewTelegram("bad-resp", "remark_test", 2*time.Second, ts.URL+"/")
	assert.NotNil(t, err)

	tb, err = NewTelegram("404", "remark_test", 2*time.Second, ts.URL+"/")
	assert.NotNil(t, err)
}

func TestTelegram_Send(t *testing.T) {
	ts := mockTelegramServer()
	defer ts.Close()

	tb, err := NewTelegram("good-token", "remark_test", 2*time.Second, ts.URL+"/")
	assert.NoError(t, err)
	assert.NotNil(t, tb)
	err = tb.Send(context.TODO(), request{comment: store.Comment{Text: "some text"}})
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
		w.Write([]byte(s))
	})
	router.Get("/bad-resp/getMe", func(w http.ResponseWriter, r *http.Request) {
		s := `{"ok": false,
				"result": {
					"first_name": "comments_test",
					"id": 707381019,
					"is_bot": false,
					"username": "remark42_test_bot"
				}}`
		w.Write([]byte(s))
	})
	router.Get("/404/getMe", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	router.Get("/good-token/sendMessage", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok": true}`))
	})

	return httptest.NewServer(router)
}
