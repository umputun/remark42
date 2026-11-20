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
		{"address after an escaped backslash", `{"text":"DOMAIN\\john@example.com"}`, `{"text":"DOMAIN\\********"}`},
		{"one-letter address after an escaped backslash", `{"text":"\\j@example.com"}`, `{"text":"\\********"}`},
		{"address starting with a unicode escape", `{"text":"\u00e9me@example.com"}`, `{"text":"\********"}`},
		{"encoded address after a json newline", `{"text":"line\n%2Bjohn%40example.com"}`, `{"text":"line\********"}`},
		{"address glued to a mention after a json newline", `{"text":"\n@umputun.com@example.com"}`, `{"text":"\********"}`},
		{"jwt-like run inside the local part", `{"text":"write to RileyJane.Doe@gmail.com"}`, `{"text":"write to ********"}`},
		{"jwt-like run inside the local part, mixed case", "to KelseyJones.Smith@example.com", "to ********"},
		{"jwt-like run inside the domain", "address=john@HarveyJones.com&site=remark42", "address=********&site=remark42"},
		{"star before the at sign", "address=secret*@example.com", "address=********"},
		{"star inside the local part", "address=secret*tag@example.com", "address=********"},
		{"quoted local part", `[DEBUG] send 12-byte message to "john"@example.com`, "[DEBUG] send 12-byte message to ********"},
		{"quoted local part with a space is not matched", `send to "john doe"@example.com`, `send to "john doe"@example.com`},
		{"quoted local part in a query", `GET - /auth/email/login?address="john"@example.com&site=remark42`,
			"GET - /auth/email/login?address=********&site=remark42"},
		{"quoted local part in a json body", `{"site":"remark42","address":"\"john\"@example.com"}`,
			`{"site":"remark42","address":"\********"}`},
		{"json value starting with a mention", `{"user":"john","text":"@umputun thanks"}`,
			`{"user":"john","text":"@umputun thanks"}`},
		{"json array item starting with a mention", `["john", "@umputun"]`, `["john", "@umputun"]`},
		{"quoted values, the second starting with a mention", `name="john" text="@umputun"`, `name="john" text="@umputun"`},
		{"two quoted phrases in a comment, the second a mention", `{"text":"I agree with \"this\" and \"@umputun\" too"}`,
			`{"text":"I agree with \"this\" and \"@umputun\" too"}`},
		{"two %q values, the second a mention", `name "john" by "@umputun"`, `name "john" by "@umputun"`},
		{"quoted phrase, full stop, then a quoted mention", `{"text":"I said \"this\". Then \"@umputun\" replied"}`,
			`{"text":"I said \"this\". Then \"@umputun\" replied"}`},
		{"quoted phrase, exclamation mark, then a quoted mention", `{"text":"\"ok\"! Please ask \"@umputun\" for help"}`,
			`{"text":"\"ok\"! Please ask \"@umputun\" for help"}`},
		{"quoted phrase, no-break space, then a quoted mention", "he said \"hi\"\u00a0and \"@umputun\"",
			"he said \"hi\"\u00a0and \"@umputun\""},
		{"json key starting with an at sign", `{"x":"y","@type":"z"}`, `{"x":"y","@type":"z"}`},
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
