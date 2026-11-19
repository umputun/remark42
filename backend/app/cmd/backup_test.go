package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackup_Execute(t *testing.T) {
	tmp := t.TempDir()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/export")
		assert.Equal(t, "GET", r.Method)
		user, passwd, ok := r.BasicAuth()
		assert.True(t, ok, "request carries basic auth")
		assert.Equal(t, "admin:secret", user+":"+passwd)
		fmt.Fprint(w, "blah\nblah2\n12345678\n")
	}))
	defer ts.Close()

	cmd := BackupCommand{}
	parseCmd(t, &cmd, ts.URL, "--site=remark", "--path="+tmp, "--file={{.SITE}}-test.export", "--admin-passwd=secret")
	err := cmd.Execute(nil)
	assert.NoError(t, err)
	data, err := os.ReadFile(tmp + "/remark-test.export")
	require.NoError(t, err)
	assert.Equal(t, "blah\nblah2\n12345678\n", string(data))
}

func TestBackup_ExecuteNoPassword(t *testing.T) {
	tmp := t.TempDir()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/export")
		assert.Equal(t, "GET", r.Method)
		user, passwd, ok := r.BasicAuth()
		assert.True(t, ok, "request carries basic auth")
		assert.Equal(t, "admin:", user+":"+passwd)
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Unauthorized")
	}))
	defer ts.Close()

	cmd := BackupCommand{}
	parseCmd(t, &cmd, ts.URL, "--site=remark", "--path="+tmp, "--file={{.SITE}}-test.export")
	err := cmd.Execute(nil)
	assert.EqualError(t, err, "error response \"401 Unauthorized\", ensure you have set ADMIN_PASSWD and provided it to the command you're running: Unauthorized")
}

func TestBackup_ExecuteFailedStatus(t *testing.T) {
	tmp := t.TempDir()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/export")
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(400)
		fmt.Fprint(w, "some error")
	}))
	defer ts.Close()

	cmd := BackupCommand{}
	parseCmd(t, &cmd, ts.URL, "--site=remark", "--path="+tmp, "--file={{.SITE}}-test.export", "--admin-passwd=secret")
	err := cmd.Execute(nil)
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
	tmp := t.TempDir()
	parseCmd(t, &cmd, ts.URL, "--site=remark", "--path="+tmp,
		"--file="+tmp+"/no-such-dir/{{.SITE}}-test.export", "--admin-passwd=secret")
	err := cmd.Execute(nil)
	fname := tmp + "/no-such-dir/remark-test.export"
	assert.EqualError(t, err, "can't create backup file "+fname+": open "+fname+": no such file or directory")
}
