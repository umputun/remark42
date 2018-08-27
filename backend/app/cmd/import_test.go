package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	flags "github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImport_Execute(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/import")
		assert.Equal(t, "POST", r.Method)
		body, err := ioutil.ReadAll(r.Body)
		assert.Nil(t, err)
		assert.Equal(t, "blah\nblah2\n12345678\n", string(body))

		fmt.Fprintln(w, "some response")
		fmt.Fprintln(w, string(body))
	}))
	defer ts.Close()

	cmd := ImportCommand{}
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--secret=123456", "--site=remark", "--file=testdata/import.txt", "--url=" + ts.URL})
	require.Nil(t, err)
	err = cmd.Execute(nil)
	assert.NoError(t, err)

	cmd = ImportCommand{}
	p = flags.NewParser(&cmd, flags.Default)
	_, err = p.ParseArgs([]string{"--secret=123456", "--site=remark", "--file=testdata/import.txt.gz", "--url=" + ts.URL})
	require.Nil(t, err)
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
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--secret=123456", "--site=remark", "--file=testdata/import-no.txt", "--url=" + ts.URL})
	require.Nil(t, err)
	err = cmd.Execute(nil)
	t.Log(err)
	assert.NotNil(t, err, "fail on no such file")
	assert.True(t, strings.Contains(err.Error(), "no such file or directory"))

	cmd = ImportCommand{}
	p = flags.NewParser(&cmd, flags.Default)
	_, err = p.ParseArgs([]string{"--secret=123456", "--site=remark", "--file=testdata/import.txt",
		"--url=http://127.0.0.1:12345"})
	require.Nil(t, err)
	err = cmd.Execute(nil)
	t.Log(err)
	assert.NotNil(t, err, "fail on connection refused")
	assert.True(t, strings.Contains(err.Error(), "connection refused"))

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%+v", r)
		w.WriteHeader(400)
		fmt.Fprintln(w, "some response with 400")
	}))
	defer ts2.Close()
	cmd = ImportCommand{}
	p = flags.NewParser(&cmd, flags.Default)
	_, err = p.ParseArgs([]string{"--secret=123456", "--site=remark", "--file=testdata/import.txt", "--url=" + ts2.URL})
	require.Nil(t, err)
	err = cmd.Execute(nil)
	t.Log(err)
	assert.NotNil(t, err)
}

func TestImport_ExecuteTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/import")
		assert.Equal(t, "POST", r.Method)
		body, err := ioutil.ReadAll(r.Body)
		assert.Nil(t, err)
		assert.Equal(t, "blah\nblah2\n12345678\n", string(body))
		time.Sleep(500 * time.Millisecond)
		fmt.Fprintln(w, "some response")
		fmt.Fprintln(w, string(body))

	}))
	defer ts.Close()

	cmd := ImportCommand{}
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--secret=123456", "--site=remark", "--file=testdata/import.txt",
		"--url=" + ts.URL, "--timeout=300ms"})
	require.Nil(t, err)
	err = cmd.Execute(nil)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "deadline exceeded"))
}
