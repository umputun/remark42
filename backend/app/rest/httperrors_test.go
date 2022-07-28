package rest

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

func TestSendErrorJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/error" {
			t.Log("http err request", r.URL)
			SendErrorJSON(w, r, 500, fmt.Errorf("error 500"), "error details 123456", 123)
			return
		}
		w.WriteHeader(404)
	}))

	defer ts.Close()

	resp, err := http.Get(ts.URL + "/error")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	assert.Equal(t, `{"code":123,"details":"error details 123456","error":"error 500"}`+"\n", string(body))
}

func TestSendErrorHTML(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/error" {
			t.Log("http err request", r.URL)
			SendErrorHTML(w, r, 500, fmt.Errorf("error 500"), "error details 123456", 987)
			return
		}
		w.WriteHeader(404)
	}))

	defer ts.Close()

	resp, err := http.Get(ts.URL + "/error")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	assert.NotContains(t, string(body), `987`, "user html should not contain internal error code")
	assert.Contains(t, string(body), `error details 123456`)
	assert.Contains(t, string(body), `error 500`)
}

func TestErrorDetailsMsg(t *testing.T) {
	callerFn := func() {
		req, err := http.NewRequest("GET", "https://example.com/test?k1=v1&k2=v2", http.NoBody)
		require.NoError(t, err)
		req.RemoteAddr = "1.2.3.4"
		msg := errDetailsMsg(req, 500, fmt.Errorf("error 500"), "error details 123456", 123)
		assert.Contains(t, msg, "error details 123456 - error 500 - 500 (123) - https://example.com/test?k1=v1&k2=v2 - [app/rest/httperrors_test.go:")
		// error line in the middle of the message is not checked
		assert.Contains(t, msg, " rest.TestErrorDetailsMsg]")
	}
	callerFn()
}

func TestErrorDetailsMsgWithUser(t *testing.T) {
	callerFn := func() {
		req, err := http.NewRequest("GET", "https://example.com/test?k1=v1&k2=v2", http.NoBody)
		require.NoError(t, err)
		req.RemoteAddr = "127.0.0.1:1234"
		req = SetUserInfo(req, store.User{Name: "test", ID: "id"})
		require.NoError(t, err)
		msg := errDetailsMsg(req, 500, fmt.Errorf("error 500"), "error details 123456", 34567)
		assert.Contains(t, msg, "error details 123456 - error 500 - 500 (34567) - test/id - https://example.com/test?k1=v1&k2=v2 - [app/rest/httperrors_test.go:")
		// error line in the middle of the message is not checked
		assert.Contains(t, msg, " rest.TestErrorDetailsMsgWithUser]")
	}
	callerFn()
}
