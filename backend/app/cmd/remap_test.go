package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemap_Execute(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/remap")
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "remark", r.URL.Query().Get("site"))
		user, passwd, ok := r.BasicAuth()
		assert.True(t, ok, "request carries basic auth")
		assert.Equal(t, "admin:secret", user+":"+passwd)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "http://oldsite.com* https://newsite.com*\nhttp://oldsite.com/from-old-page/1 https://newsite.com/to-new-page/1", string(body))

		w.WriteHeader(202)
	}))
	defer ts.Close()

	cmd := RemapCommand{}
	parseCmd(t, &cmd, ts.URL, "--site=remark", "--file=testdata/remap_urls.txt", "--admin-passwd=secret")
	err := cmd.Execute(nil)
	assert.NoError(t, err)
}

func TestRemap_ExecuteNoPassword(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/remap")
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "remark", r.URL.Query().Get("site"))
		user, passwd, ok := r.BasicAuth()
		assert.True(t, ok, "request carries basic auth")
		assert.Equal(t, "admin:", user+":"+passwd)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "http://oldsite.com* https://newsite.com*\nhttp://oldsite.com/from-old-page/1 https://newsite.com/to-new-page/1", string(body))

		w.WriteHeader(401)
		fmt.Fprint(w, "Unauthorized")
	}))
	defer ts.Close()

	cmd := RemapCommand{}
	parseCmd(t, &cmd, ts.URL, "--site=remark", "--file=testdata/remap_urls.txt")
	err := cmd.Execute(nil)
	assert.EqualError(t, err, "error response \"401 Unauthorized\", ensure you have set ADMIN_PASSWD and provided it to the command you're running: Unauthorized")
}
