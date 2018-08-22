package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	flags "github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExport_Execute(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/api/v1/admin/export")
		assert.Equal(t, "GET", r.Method)
		fmt.Fprint(w, "blah\nblah2\n12345678\n")
	}))
	defer ts.Close()

	cmd := ExportCommand{}
	p := flags.NewParser(&cmd, flags.Default)
	_, err := p.ParseArgs([]string{"--secret=123456", "--site=remark", "--path=/tmp",
		"--file={{.SITE}}-test.export", "--url=" + ts.URL})
	require.Nil(t, err)
	err = cmd.Execute(nil)
	assert.NoError(t, err)
	defer os.Remove("/tmp/remark-test.export")

	data, err := ioutil.ReadFile("/tmp/remark-test.export")
	require.Nil(t, err)
	assert.Equal(t, "blah\nblah2\n12345678\n", string(data))
}

func TestExport_ParseFileName(t *testing.T) {
	tbl := []struct {
		ex  ExportCommand
		res string
		err bool
	}{
		{ExportCommand{}, "", false},
		{ExportCommand{ExportPath: "/tmp/blah", ExportFile: "fname.gz"}, "/tmp/blah/fname.gz", false},
		{ExportCommand{Site: "remark", ExportPath: "/tmp/blah", ExportFile: "fname-{{.SITE}}-{{.YYYYMMDD}}.gz"},
			"/tmp/blah/fname-remark-20180821.gz", false},
		{ExportCommand{Site: "remark", ExportPath: "/tmp/blah", ExportFile: "fname-{{.SITE}}-{{.YYYY}}-{{.MM}}.gz"},
			"/tmp/blah/fname-remark-2018-08.gz", false},
		{ExportCommand{Site: "remark", ExportPath: "/tmp/blah", ExportFile: "/tmp/fname-{{.SITE}}-{{.YYYY}}-{{.MM}}.gz"},
			"/tmp/fname-remark-2018-08.gz", false},
		{ExportCommand{Site: "remark", ExportPath: "/tmp/blah", ExportFile: "fname-{{.XXX}}-{{.YYYY}}-{{.MM}}.gz"},
			"", true},
	}

	now := time.Date(2018, 8, 21, 21, 26, 15, 0, time.UTC)
	for i, tt := range tbl {
		r, err := tt.ex.parseFileName(now)
		if tt.err {
			assert.NotNil(t, err)
			continue
		}
		assert.Equal(t, tt.res, r, "check #%d", i)
	}
}
