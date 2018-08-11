package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockConvertor struct{}

func (m mockConvertor) Convert(text string) string { return text + "!converted" }

func TestFormater_FormatText(t *testing.T) {
	tbl := []struct {
		in, out string
	}{
		{"", "!converted"},
		{"12345 abc", "<p>12345 abc</p>\n!converted"},
		{"**xyz** _aaa_", "<p><strong>xyz</strong> <em>aaa</em></p>\n!converted"},
		{
			"http://127.0.0.1/some-long-link/12345/678901234567890", "<p><a href=\"http://127.0.0.1/some-long-link/12345/678901234567890\">http://127.0.0.1/some-long-link/12345/6789012...</a></p>\n!converted",
		},
	}
	f := NewCommentFormater(mockConvertor{})
	for n, tt := range tbl {
		assert.Equal(t, tt.out, f.FormatText(tt.in), "check #%d", n)
	}
}

func TestFormater_FormatTextNoConvertor(t *testing.T) {
	f := NewCommentFormater()
	assert.Equal(t, "<p>12345</p>\n", f.FormatText("12345"))
}

func TestFormater_FormatComment(t *testing.T) {
	comment := Comment{
		Text:      `blah`,
		User:      User{ID: "username"},
		ParentID:  "p123",
		ID:        "123",
		Locator:   Locator{SiteID: "site", URL: "url"},
		Score:     10,
		Pin:       true,
		Deleted:   true,
		Timestamp: time.Date(2018, 1, 1, 9, 30, 0, 0, time.Local),
		Votes:     map[string]bool{"uu": true},
	}

	f := NewCommentFormater(mockConvertor{})
	exp := comment
	exp.Text = "<p>blah</p>\n!converted"
	assert.Equal(t, exp, f.Format(comment))
}

func TestFormater_ShortenAutoLinks(t *testing.T) {
	f := NewCommentFormater(nil)
	tbl := []struct {
		max     int
		in, out string
	}{
		{32, "", ""},
		{32, "text", "text"},
		{32, "<p>asd</p>", "<p>asd</p>"},
		{5, `<a href="incorrect-url">incorrect-url</a>`, `<a href="incorrect-url">incorrect-url</a>`},
		{32, `<a href="https://blah.com">some text, not href</a>`, `<a href="https://blah.com">some text, not href</a>`},
		{
			32,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=123#anc</a>`,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=123#anc</a>`,
		},
		{
			31,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=123#anc</a>`,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=1...</a>`,
		},
		{
			15,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=123#anc</a>`,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com...</a>`,
		},
		{
			3,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=123#anc</a>`,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com...</a>`,
		},
		{
			-1,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=123#anc</a>`,
			`<a href="https://blah.com/a/b/c/d?g=123#anc">https://blah.com/a/b/c/d?g=123#anc</a>`,
		},
	}

	for n, tt := range tbl {
		got := f.shortenAutoLinks(tt.in, tt.max)
		assert.Equalf(t, tt.out, got, "check #%d", n)
	}
}
