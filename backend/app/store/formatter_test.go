package store

import (
	"strconv"
	"strings"
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
			"something https://amp.dw.com/ru/илон-маск-купил-twitter/a-61590911",
			"<p>something <a href=\"https://amp.dw.com/ru/илон-маск-купил-twitter/a-61590911\">https://amp.dw.com/ru/илон-маск-купил-twitter...</a></p>\n!converted", "links",
		},
		{
			"something <img src=\"some.png\"/>  _aaa_",
			"<p>something <img src=\"some.png\" loading=\"lazy\"/>  <em>aaa</em></p>\n!converted",
			"lazy image",
		},
		{"&mdash; not translated #354", "<p>— not translated #354</p>\n!converted", "mdash"},
		{`no_smartpants "quoted" text`, "<p>no_smartpants &#34;quoted&#34; text</p>\n!converted", "normal quotes without smartpants"},
		{`"quoted" text`, "<p>«quoted» text</p>\n!converted", "normal quotes with smartpants"},
		{`no_smartpants “quoted” text`, "<p>no_smartpants “quoted” text</p>\n!converted", "curly quotes without smartpants"},
		{`“quoted” text`, "<p>“quoted” text</p>\n!converted", "curly quotes with smartpants"},
		{`no_smartpants «quoted» text`, "<p>no_smartpants «quoted» text</p>\n!converted", "French guillemets without smartpants"},
		{`«quoted» text`, "<p>«quoted» text</p>\n!converted", "French guillemets with smartpants"},
		{"smth\n```go\nfunc main(aa string) int {return 0}\n```", `<p>smth</p>
<pre class="chroma"><code><span class="line"><span class="cl"><span class="kd">func</span><span class="w"> </span><span class="nf">main</span><span class="p">(</span><span class="nx">aa</span><span class="w"> </span><span class="kt">string</span><span class="p">)</span><span class="w"> </span><span class="kt">int</span><span class="w"> </span><span class="p">{</span><span class="k">return</span><span class="w"> </span><span class="mi">0</span><span class="p">}</span><span class="w">
</span></span></span></code></pre>!converted`, "code with language"},
		{"```\ntest_code\n```", `<pre class="chroma"><code><span class="line"><span class="cl">test_code
</span></span></code></pre>!converted`, "code without language"},
	}
	f := NewCommentFormatter(mockConverter{})
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			raw := strings.HasPrefix(tt.in, `no_smartpants`)
			t.Logf("raw: %v", raw)
			assert.Equal(t, tt.out, f.FormatText(tt.in, raw))
		})
	}
}

func TestFormatter_FormatTextNoConverter(t *testing.T) {
	f := NewCommentFormatter()
	assert.Equal(t, "<p>12345</p>\n", f.FormatText("12345", false))
}

func TestFormatter_FormatTextConverterFunc(t *testing.T) {
	fn := CommentConverterFunc(func(text string) string { return "zz!" + text })
	f := NewCommentFormatter(fn)
	assert.Equal(t, "zz!<p>12345</p>\n", f.FormatText("12345", false))
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
		Timestamp: time.Date(2018, 1, 1, 9, 30, 0, 0, time.UTC),
		Votes:     map[string]bool{"uu": true},
	}

	f := NewCommentFormatter(mockConverter{})
	exp := comment
	exp.Text = "<p>blah</p>\n\n<p>xyz</p>\n!converted"
	assert.Equal(t, exp, f.Format(comment, false))
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

func TestStripImages(t *testing.T) {
	tbl := []struct {
		name, in, out string
	}{
		{"empty", "", ""},
		{"no image", "<p>text</p>\n", "<p>text</p>\n"},
		{"external image", `<p>before <img src="https://example.com/a.png" alt="a"> after</p>` + "\n", "<p>before  after</p>\n"},
		{"lazy image", `<p>x <img src="https://example.com/a.png" loading="lazy"></p>` + "\n", "<p>x </p>\n"},
		{"uploaded image only", `<p><img src="/api/v1/picture/user/id.png" alt=""></p>` + "\n", "\n"},
		{"proxied image only", `<p><img src="/api/v1/img?src=aHR0cDovL2E" alt=""></p>` + "\n", "\n"},
		{"linked image only", `<p><a href="https://example.com"><img src="a.png"></a></p>` + "\n", "\n"},
		{"linked image with text", `<p><a href="https://example.com">text <img src="a.png"></a></p>` + "\n", `<p><a href="https://example.com">text </a></p>` + "\n"},
		{"several images", `<p><img src="a.png"></p><p>keep</p><p><img src="b.png"></p>`, "<p>keep</p>"},
		{"image markup shown as code", "<pre><code>&lt;img src=x&gt;</code></pre>\n", "<pre><code>&lt;img src=x&gt;</code></pre>\n"},
		{"empty link kept without image", `<p><a href="https://example.com"></a>text</p>` + "\n", `<p><a href="https://example.com"></a>text</p>` + "\n"},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.out, StripImages(tt.in))
		})
	}
}

func TestFormatter_StripImagesConverter(t *testing.T) {
	f := NewCommentFormatter(CommentConverterFunc(StripImages))

	t.Run("markdown image", func(t *testing.T) {
		res := f.FormatText("hello ![pic](https://example.com/a.png) world", false)
		assert.NotContains(t, res, "<img")
		assert.Contains(t, res, "hello")
		assert.Contains(t, res, "world")
	})

	t.Run("raw html image", func(t *testing.T) {
		res := f.FormatText(`hello <img src="https://example.com/a.png" onerror="x()"> world`, false)
		assert.NotContains(t, res, "<img")
	})

	t.Run("image only leaves no content", func(t *testing.T) {
		res := f.FormatText("![pic](https://example.com/a.png)", false)
		assert.Empty(t, strings.TrimSpace(res))
	})
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
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.out, f.lazyImage(tt.inp))
		})
	}
}
