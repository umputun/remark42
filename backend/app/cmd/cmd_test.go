package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
		{fileParser{site: "remark", path: "/tmp/blah", file: "fname-{{.XXX}}-{{.YYYY}}-{{.MM}}.gz"},
			"", true},
	}

	now := time.Date(2018, 8, 21, 21, 26, 15, 0, time.UTC)
	for i, tt := range tbl {
		r, err := tt.p.parse(now)
		if tt.err {
			assert.NotNil(t, err)
			continue
		}
		assert.Equal(t, tt.res, r, "check #%d", i)
	}
}
