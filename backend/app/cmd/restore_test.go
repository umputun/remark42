package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	flags "github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestore_Execute(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/import")
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "native", r.URL.Query().Get("provider"))
		body, err := ioutil.ReadAll(r.Body)
		assert.Nil(t, err)
		assert.Equal(t, "blah\nblah2\n12345678\n", string(body))

		fmt.Fprintln(w, "some response")
		fmt.Fprintln(w, string(body))
	}))
	defer ts.Close()

	cmd := RestoreCommand{}
	cmd.SetCommon(CommonOpts{RemarkURL: ts.URL, SharedSecret: "123456"})

	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--site=remark", "--path=testdata", "--file=import.txt"})
	require.Nil(t, err)
	err = cmd.Execute(nil)
	assert.NoError(t, err)
}
