package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExport_ParseFileName(t *testing.T) {
	tbl := []struct {
		p   fileParser
		res string
		err bool
	}{
		{fileParser{}, "", false},
		{fileParser{path: "/tmp/blah", file: "fname.gz"}, "/tmp/blah/fname.gz", false},
		{fileParser{site: "remark", path: "/tmp/blah", file: "fname-{{.SITE}}-{{.YYYYMMDD}}.gz"},
			"/tmp/blah/fname-remark-20180821.gz", false},
		{fileParser{site: "remark", path: "/tmp/blah", file: "fname-{{.SITE}}-{{.YYYY}}-{{.MM}}.gz"},
			"/tmp/blah/fname-remark-2018-08.gz", false},
		{fileParser{site: "remark", path: "/tmp/blah", file: "/tmp/fname-{{.SITE}}-{{.YYYY}}-{{.MM}}.gz"},
			"/tmp/fname-remark-2018-08.gz", false},
		{fileParser{site: "remark", path: "/tmp/blah", file: "/tmp/fname-{{.SITE}}-{{.TS}}.gz"},
			"/tmp/fname-remark-20180821T212615.gz", false},
		{fileParser{site: "remark", path: "/tmp/blah", file: "fname-{{.XXX}}-{{.YYYY}}-{{.MM}}.gz"},
			"", true},
	}

	now := time.Date(2018, 8, 21, 21, 26, 15, 0, time.UTC)
	for i, tt := range tbl {
		r, err := tt.p.parse(now)
		if tt.err {
			assert.Error(t, err)
			continue
		}
		assert.Equal(t, tt.res, r, "check #%d", i)
	}
}

// TestAdminRequest_EscapesQuery runs each admin command for a site id carrying query and fragment
// delimiters and checks the server receives the site, and the import provider, exactly as given
func TestAdminRequest_EscapesQuery(t *testing.T) {
	const site = "a&b#c"
	tbl := []struct {
		name  string
		cmd   CommonOptionsCommander
		args  []string
		query url.Values
	}{
		{name: "backup", cmd: &BackupCommand{},
			args:  []string{"--path=" + t.TempDir(), "--file=out.gz"},
			query: url.Values{"mode": {"file"}, "site": {site}}},
		{name: "import", cmd: &ImportCommand{},
			args:  []string{"--file=testdata/import.txt", "--provider=wordpress"},
			query: url.Values{"site": {site}, "provider": {"wordpress"}}},
		{name: "remap", cmd: &RemapCommand{},
			args:  []string{"--file=testdata/remap_urls.txt"},
			query: url.Values{"site": {site}}},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			var got url.Values
			ts := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = r.URL.Query()
			}))
			defer ts.Close()

			parseCmd(t, tt.cmd, ts.URL, append([]string{"--site=" + site}, tt.args...)...)
			require.NoError(t, tt.cmd.Execute(nil))
			assert.Equal(t, tt.query, got, "query the server received")
		})
	}
}

// testSharedSecret is the shared secret parseCmd sets, which signRawToken signs with
const testSharedSecret = "secret"

// parseCmd sets the common options the way main does and parses args into cmd
func parseCmd(t *testing.T, cmd interface{ SetCommon(CommonOpts) }, remarkURL string, args ...string) {
	t.Helper()
	cmd.SetCommon(CommonOpts{RemarkURL: remarkURL, SharedSecret: testSharedSecret})
	_, err := flags.NewParser(cmd, flags.Default).ParseArgs(args)
	require.NoError(t, err)
}
