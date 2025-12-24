package cmd

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemap_Execute(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/remap")
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "remark", r.URL.Query().Get("site"))
		t.Logf("Authorization header: %+v", r.Header.Get("Authorization"))
		auth, err := base64.StdEncoding.DecodeString(strings.Split(r.Header.Get("Authorization"), " ")[1])
		require.NoError(t, err)
		assert.Equal(t, "admin:secret", string(auth))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "http://oldsite.com* https://newsite.com*\nhttp://oldsite.com/from-old-page/1 https://newsite.com/to-new-page/1", string(body))

		w.WriteHeader(202)
	}))
	defer ts.Close()

	cmd := RemapCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL, SharedSecret: "123456"})

	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--site=remark", "--file=testdata/remap_urls.txt", "--admin-passwd=secret"})
	require.NoError(t, err)
	err = cmd.Execute(nil)
	assert.NoError(t, err)
}

func TestRemap_ExecuteNoPassword(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/remap")
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "remark", r.URL.Query().Get("site"))
		t.Logf("Authorization header: %+v", r.Header.Get("Authorization"))
		auth, err := base64.StdEncoding.DecodeString(strings.Split(r.Header.Get("Authorization"), " ")[1])
		require.NoError(t, err)
		assert.Equal(t, "admin:", string(auth))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "http://oldsite.com* https://newsite.com*\nhttp://oldsite.com/from-old-page/1 https://newsite.com/to-new-page/1", string(body))

		w.WriteHeader(401)
		fmt.Fprint(w, "Unauthorized")
	}))
	defer ts.Close()

	cmd := RemapCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL})

	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--site=remark", "--file=testdata/remap_urls.txt"})
	require.NoError(t, err)
	err = cmd.Execute(nil)
	assert.EqualError(t, err, "error response \"401 Unauthorized\", ensure you have set ADMIN_PASSWD and provided it to the command you're running: Unauthorized")
}
