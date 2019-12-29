package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/go-pkgz/lgr"
	flags "github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImport_Execute(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/import")
		assert.Equal(t, "POST", r.Method)
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "blah\nblah2\n12345678\n", string(body))

		fmt.Fprintln(w, "some response")
		fmt.Fprintln(w, string(body))
	}))
	defer ts.Close()

	cmd := ImportCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL, SharedSecret: "123456"})

	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--site=remark", "--file=testdata/import.txt", "--admin-passwd=secret"})
	require.NoError(t, err)
	err = cmd.Execute(nil)
	assert.NoError(t, err)

	cmd = ImportCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL, SharedSecret: "123456"})

	p = flags.NewParser(&cmd, flags.Default)
	_, err = p.ParseArgs([]string{"--site=remark", "--file=testdata/import.txt.gz", "--admin-passwd=secret"})
	require.NoError(t, err)
	err = cmd.Execute(nil)
	assert.NoError(t, err)
}

func TestImport_ExecuteFailed(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/import")
		assert.Equal(t, "POST", r.Method)
		fmt.Fprintln(w, "some response")
	}))
	defer ts.Close()

	cmd := ImportCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL, SharedSecret: "123456"})
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--site=remark", "--file=testdata/import-no.txt", "--admin-passwd=secret"})
	require.NoError(t, err)
	err = cmd.Execute(nil)
	t.Log(err)
	assert.Error(t, err, "fail on no such file")
	assert.Contains(t, err.Error(), "no such file or directory")

	cmd = ImportCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: "http://127.0.0.1:12345", SharedSecret: "123456"})
	p = flags.NewParser(&cmd, flags.Default)
	_, err = p.ParseArgs([]string{"--site=remark", "--file=testdata/import.txt", "--admin-passwd=secret"})
	require.NoError(t, err)
	err = cmd.Execute(nil)
	t.Log(err)
	assert.Error(t, err, "fail on connection refused")
	assert.Contains(t, err.Error(), "connection refused")

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%+v", r)
		w.WriteHeader(400)
		fmt.Fprintln(w, "some response with 400")
	}))
	defer ts2.Close()
	cmd = ImportCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts2.URL, SharedSecret: "123456"})
	p = flags.NewParser(&cmd, flags.Default)
	_, err = p.ParseArgs([]string{"--site=remark", "--file=testdata/import.txt", "--admin-passwd=secret"})
	require.NoError(t, err)
	err = cmd.Execute(nil)
	t.Log(err)
	assert.Error(t, err)
}

func TestImport_ExecuteTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/import")
		assert.Equal(t, "POST", r.Method)
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "blah\nblah2\n12345678\n", string(body))
		time.Sleep(500 * time.Millisecond)
		fmt.Fprintln(w, "some response")
		fmt.Fprintln(w, string(body))

	}))
	defer ts.Close()

	cmd := ImportCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL, SharedSecret: "123456"})

	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--site=remark", "--file=testdata/import.txt", "--timeout=300ms", "--admin-passwd=secret"})
	require.NoError(t, err)
	err = cmd.Execute(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline exceeded")
}
