package store

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockConverter struct{}

func (m mockConverter) Convert(text string) string { return text + "!converted" }

func TestFormatter_FormatText(t *testing.T) {
	tbl := []struct {
		in, out string
		name    string
	}{
		{"", "!converted", "empty"},
		{"12345 abc", "<p>12345 abc</p>\n!converted", "simple"},
		{"**xyz** _aaa_ - \"sfs\"", "<p><strong>xyz</strong> <em>aaa</em> – «sfs»</p>\n!converted", "format"},
		{
			"http://127.0.0.1/some-long-link/12345/678901234567890",
			"<p><a href=\"http://127.0.0.1/some-long-link/12345/678901234567890\">http://127.0.0." +
				"1/some-long-link/12345/6789012...</a></p>\n!converted", "links",
		},
		{
			"something <img src=\"some.png\"/>  _aaa_",
			"<p>something <img src=\"some.png\" loading=\"lazy\"/>  <em>aaa</em></p>\n!converted",
			"lazy image",
		},
		{"&mdash; not translated #354", "<p>— not translated #354</p>\n!converted", "mdash"},
		{"smth\n```go\nfunc main(aa string) int {return 0}\n```", `<p>smth</p>
<pre class="chroma"><span class="kd">func</span> <span class="nf">main</span><span class="p">(</span><span class="nx">aa</span> <span class="kt">string</span><span class="p">)</span> <span class="kt">int</span> <span class="p">{</span><span class="k">return</span> <span class="mi">0</span><span class="p">}</span>
</pre>!converted`, "code with language"},
		{"```\ntest_code\n```", `<pre class="chroma">test_code
</pre>!converted`, "code without language"},
	}
	f := NewCommentFormatter(mockConverter{})
	for _, tt := range tbl {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, f.FormatText(tt.in))
		})
	}
}

func TestFormatter_FormatTextNoConverter(t *testing.T) {
	f := NewCommentFormatter()
	assert.Equal(t, "<p>12345</p>\n", f.FormatText("12345"))
}

func TestFormatter_FormatTextConverterFunc(t *testing.T) {
	fn := CommentConverterFunc(func(text string) string { return "zz!" + text })
	f := NewCommentFormatter(fn)
	assert.Equal(t, "zz!<p>12345</p>\n", f.FormatText("12345"))
}

func TestFormatter_FormatComment(t *testing.T) {
	comment := Comment{
		Text:      "blah\n\nxyz",
		User:      User{ID: "username"},
		ParentID:  "p123",
		ID:        "123",
		Locator:   Locator{SiteID: "site", URL: "http://example.com?foo=bar&x=123"},
		Score:     10,
		Pin:       true,
		Deleted:   true,
		Timestamp: time.Date(2018, 1, 1, 9, 30, 0, 0, time.Local),
		Votes:     map[string]bool{"uu": true},
	}

	f := NewCommentFormatter(mockConverter{})
	exp := comment
	exp.Text = "<p>blah</p>\n\n<p>xyz</p>\n!converted"
	assert.Equal(t, exp, f.Format(comment))
}

func TestFormatter_ShortenAutoLinks(t *testing.T) {
	f := NewCommentFormatter(nil)
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

func TestCommentFormatter_lazyImage(t *testing.T) {

	tbl := []struct {
		inp, out string
	}{
		{"", ""},
		{`blah <img src="some.png" />`, `blah <img src="some.png" loading="lazy"/>`},
		{`blah <img src="some.png" loading="lazy"/>`, `blah <img src="some.png" loading="lazy"/>`},
		{`blah <img src="some.png"/> ххх <img src=http://example.com/pp2.jpg>`, `blah <img src="some.png" loading="lazy"/> ххх <img src="http://example.com/pp2.jpg" loading="lazy"/>`},
	}

	f := NewCommentFormatter(nil)
	for i, tt := range tbl {
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.out, f.lazyImage(tt.inp))
		})
	}

}
