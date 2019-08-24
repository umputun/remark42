package rest

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/umputun/remark/backend/app/store"
)

func TestSendErrorJSON(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/error" {
			t.Log("http err request", r.URL)
			SendErrorJSON(w, r, 500, errors.New("error 500"), "error details 123456", 123)
			return
		}
		w.WriteHeader(404)
	}))

	defer ts.Close()

	resp, err := http.Get(ts.URL + "/error")
	require.Nil(t, err)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	assert.Equal(t, `{"code":123,"details":"error details 123456","error":"error 500"}`+"\n", string(body))
}

func TestErrorDetailsMsg(t *testing.T) {
	callerFn := func() {
		req, err := http.NewRequest("GET", "https://example.com/test?k1=v1&k2=v2", nil)
		require.Nil(t, err)
		req.RemoteAddr = "1.2.3.4"
		msg := errDetailsMsg(req, 500, errors.New("error 500"), "error details 123456", 123)
		assert.Equal(t, "error details 123456 - error 500 - 500 (123) - 1.2.3.4 - https://example.com/test?k1=v1&k2=v2 [caused by app/rest/httperrors_test.go:47 rest.TestErrorDetailsMsg]", msg)
	}
	callerFn()
}

func TestErrorDetailsMsgWithUser(t *testing.T) {
	callerFn := func() {
		req, err := http.NewRequest("GET", "https://example.com/test?k1=v1&k2=v2", nil)
		require.NoError(t, err)
		req.RemoteAddr = "127.0.0.1:1234"
		req = SetUserInfo(req, store.User{Name: "test", ID: "id"})
		require.Nil(t, err)
		msg := errDetailsMsg(req, 500, errors.New("error 500"), "error details 123456", 34567)
		assert.Equal(t, "error details 123456 - error 500 - 500 (34567) - test/id - 127.0.0.1 - https://example." +
			"com/test?k1=v1&k2=v2 [caused by app/rest/httperrors_test.go:61 rest.TestErrorDetailsMsgWithUser]", msg)
	}
	callerFn()
}
