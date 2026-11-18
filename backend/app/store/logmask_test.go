package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskForLog(t *testing.T) {
	tbl := []struct {
		name, in, out string
	}{
		{"query", "GET - /auth/email/login?address=john.doe+x@example.com&site=remark42&user=john",
			"GET - /auth/email/login?address=********&site=remark42&user=john"},
		{"query left encoded", "GET - /auth/email/login?address=john.doe%2Bx%40example.com&user=50%",
			"GET - /auth/email/login?address=********&user=50%"},
		{"json body", `POST - /api/v1/email/subscribe - {"site":"remark42","address":"john@example.com"}`,
			`POST - /api/v1/email/subscribe - {"site":"remark42","address":"********"}`},
		{"body cut after the at sign", `{"address":"john@exa...`, `{"address":"********`},
		{"apostrophe", "address=o'brien@example.com", "address=********"},
		{"mistyped", "address=john@@example.com&x=john@example@com", "address=********&x=********"},
		{"unicode", "пишите на почту иван@пример.рф, спасибо", "пишите на почту ********, спасибо"},
		{"after no-break space", "write to\u00a0john@example.com", "write to\u00a0********"},
		{"in error", `problem sending user email notification to "john@example.com": 550 <john@example.com> rejected`,
			`problem sending user email notification to "********": 550 <********> rejected`},
		{"jwt", "/email/unsubscribe.html?site=remark42&tkn=eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOiJyZW1hcms0MiJ9.c2ln-_x",
			"/email/unsubscribe.html?site=remark42&tkn=********"},
		{"jwt cut short", `{"token":"eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOi...`, `{"token":"********..`},
		{"handle in a page url", "GET - /api/v1/find?url=https://example.com/@handle/post&site=remark42",
			"GET - /api/v1/find?url=https://example.com/@handle/post&site=remark42"},
		{"mention in text", `{"text":"thanks @umputun, see https://example.com/post?id=1"}`,
			`{"text":"thanks @umputun, see https://example.com/post?id=1"}`},
		{"nothing to hide", "GET - /api/v1/last/10?site=remark42 - 200 (123)", "GET - /api/v1/last/10?site=remark42 - 200 (123)"},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, MaskForLog(tt.in))
		})
	}
}
