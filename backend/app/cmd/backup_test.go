package cmd

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackup_Execute(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/export")
		assert.Equal(t, "GET", r.Method)
		t.Logf("Authorization header: %+v", r.Header.Get("Authorization"))
		auth, err := base64.StdEncoding.DecodeString(strings.Split(r.Header.Get("Authorization"), " ")[1])
		require.NoError(t, err)
		assert.Equal(t, "admin:secret", string(auth))
		fmt.Fprint(w, "blah\nblah2\n12345678\n")
	}))
	defer ts.Close()

	cmd := BackupCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL, SharedSecret: "123456"})
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--site=remark", "--path=/tmp", "--file={{.SITE}}-test.export", "--admin-passwd=secret"})
	require.NoError(t, err)
	err = cmd.Execute(nil)
	assert.NoError(t, err)
	defer os.Remove("/tmp/remark-test.export")

	data, err := os.ReadFile("/tmp/remark-test.export")
	require.NoError(t, err)
	assert.Equal(t, "blah\nblah2\n12345678\n", string(data))
}

func TestBackup_ExecuteNoPassword(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/export")
		assert.Equal(t, "GET", r.Method)
		t.Logf("Authorization: %+v", r.Header.Get("Authorization"))
		auth, err := base64.StdEncoding.DecodeString(strings.Split(r.Header.Get("Authorization"), " ")[1])
		require.NoError(t, err)
		require.Equal(t, "admin:", string(auth))
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Unauthorized")
	}))
	defer ts.Close()

	cmd := BackupCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL})
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--site=remark", "--path=/tmp", "--file={{.SITE}}-test.export"})
	require.NoError(t, err)
	err = cmd.Execute(nil)
	assert.EqualError(t, err, "error response \"401 Unauthorized\", ensure you have set ADMIN_PASSWD and provided it to the command you're running: Unauthorized")
}

func TestBackup_ExecuteFailedStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/export")
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(400)
		fmt.Fprint(w, "some error")
	}))
	defer ts.Close()

	cmd := BackupCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL, SharedSecret: "123456"})

	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--site=remark", "--path=/tmp", "--file={{.SITE}}-test.export", "--admin-passwd=secret"})
	require.NoError(t, err)
	err = cmd.Execute(nil)
	assert.EqualError(t, err, `error response "400 Bad Request", some error`)
}

func TestBackup_ExecuteFailedWrite(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/export")
		assert.Equal(t, "GET", r.Method)
		fmt.Fprint(w, "blah\nblah2\n12345678\n")
	}))
	defer ts.Close()

	cmd := BackupCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL, SharedSecret: "123456"})

	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--site=remark", "--path=/tmp",
		"--file=/tmp/no-such-dir/{{.SITE}}-test.export", "--admin-passwd=secret"})
	require.NoError(t, err)
	err = cmd.Execute(nil)
	assert.EqualError(t, err, `can't create backup file /tmp/no-such-dir/remark-test.export: open /tmp/no-such-dir/remark-test.export: no such file or directory`)
}
