package store

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Depado/bfchroma/v2"
	"github.com/PuerkitoBio/goquery"
	"github.com/alecthomas/chroma/v2/formatters/html"
	bf "github.com/russross/blackfriday/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			"<p><a href=\"http://127.0.0.1/some-long-link/12345/678901234567890\" rel=\"nofollow\">http://127.0.0." +
				"1/some-long-link/12345/6789012...</a></p>\n!converted", "links",
		},
		{
			"something https://amp.dw.com/ru/илон-маск-купил-twitter/a-61590911",
			"<p>something <a href=\"https://amp.dw.com/ru/%D0%B8%D0%BB%D0%BE%D0%BD-%D0%BC%D0%B0%D1%81%D0%BA-%D0%BA%D1%83%D0%BF%D0%B8%D0%BB-twitter/a-61590911\" rel=\"nofollow\">https://amp.dw.com/ru/илон-маск-купил-twitter...</a></p>\n!converted", "links",
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

func TestFormatter_FormatTextPanic(t *testing.T) {
	// the lexers of these languages and the markdown parser itself panic on such text
	tbl := []struct {
		name, in, out string
	}{
		{"yang", "```yang\n/* /*\n```", "<pre><code>/* /*\n</code></pre>\n"},
		{"spade", "```spade\nstruct 'a\n```", "<pre><code>struct &#39;a\n</code></pre>\n"},
		{"psl", "```psl\n#WARN\n```", "<pre><code>#WARN\n</code></pre>\n"},
		{"raku pod", "```raku\n=begin pod\n```", "<pre><code>=begin pod\n</code></pre>\n"},
		{"raku comment", "```raku\n#`[\n```", "<pre><code>#`[\n</code></pre>\n"},
		{
			"text around code", "before **b**\n\n```yang\n/* /*\n```\n\n```go\nx := 1\n```\n\nafter",
			"<p>before <strong>b</strong></p>\n\n<pre><code>/* /*\n</code></pre>\n" +
				`<pre class="chroma"><code><span class="line"><span class="cl"><span class="nx">x</span><span class="w"> </span>` +
				`<span class="o">:=</span><span class="w"> </span><span class="mi">1</span><span class="w">` + "\n" +
				"</span></span></span></code></pre>\n<p>after</p>\n",
		},
		{"link reference", "[0]:<", "<p>[0]:&lt;</p>\n"},
		{"link reference with text", "<b>some</b> **text**\n[0]:<", "<p>&lt;b&gt;some&lt;/b&gt; **text**<br/>\n[0]:&lt;</p>\n"},
	}
	f := NewCommentFormatter()
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			var out string
			require.NotPanics(t, func() { out = f.FormatText(tt.in, false) })
			assert.Equal(t, tt.out, out)
		})
	}
}

// panicsAtEnd writes the start of a highlighted block and panics where the block ends
type panicsAtEnd struct{}

func (panicsAtEnd) Start(bool, string) string { return "<pre>written before the panic" }
func (panicsAtEnd) End(bool) string           { panic("highlighting failed halfway") }

func TestFormatter_plainCodeOnPanicWritesNothingHalfway(t *testing.T) {
	base := bf.NewHTMLRenderer(bf.HTMLRendererParameters{})
	rend := plainCodeOnPanic{bfchroma.NewRenderer(bfchroma.Extend(base),
		bfchroma.ChromaOptions(html.WithClasses(true), html.WithPreWrapper(panicsAtEnd{})))}
	var res string
	require.NotPanics(t, func() {
		res = string(bf.Run([]byte("```go\nx := 1\n```\n"), bf.WithExtensions(bf.FencedCode), bf.WithRenderer(rend)))
	})
	assert.Equal(t, "<pre><code class=\"language-go\">x := 1\n</code></pre>\n", res,
		"the block is plain, with nothing of the highlighting that failed halfway")
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

// TestFormatter_ConvertersSeeStoredImages checks that a converter finding images with an HTML parser sees every
// image the stored text holds. Each comment hides an image where the parser reads text and the sanitizer, which
// has the last word on what is stored, reads a tag; the converter marks every image it finds.
func TestFormatter_ConvertersSeeStoredImages(t *testing.T) {
	mark := CommentConverterFunc(func(text string) string {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))
		require.NoError(t, err)
		doc.Find("img").Each(func(_ int, s *goquery.Selection) { s.SetAttr("src", "https://remark42.example/seen.png") })
		res, err := doc.Find("body").Html()
		require.NoError(t, err)
		return res
	})
	f := NewCommentFormatter(mark)
	for name, hidden := range map[string]string{
		"svg title":     `<svg><title><!-- </title><img src="http://attacker.example/hidden.png" --></title></svg>`,
		"svg style":     `<svg><style><!-- </style><img src="http://attacker.example/hidden.png" --></style></svg>`,
		"svg textarea":  `<svg><textarea><!-- </textarea><img src="http://attacker.example/hidden.png" --></textarea></svg>`,
		"math title":    `<math><title><!-- </title><img src="http://attacker.example/hidden.png" --></title></math>`,
		"nested deeply": strings.Repeat("<b>", 520) + `<img src="http://attacker.example/hidden.png">`,
	} {
		t.Run(name, func(t *testing.T) {
			c := Comment{Text: f.FormatText(`plain <img src="http://attacker.example/plain.png"> `+hidden, false)}
			c.Sanitize()
			assert.NotContains(t, c.Text, "attacker.example", "an image the converter did not see is stored")
			assert.Equal(t, strings.Count(c.Text, "<img"), strings.Count(c.Text, `loading="lazy"`), "every stored image is lazy")
		})
	}
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
		{ // 52 bytes, 36 runes: longer than the limit in bytes, shorter in runes
			45,
			`<a href="https://example.com/яяяяяяяяяяяяяяяя">https://example.com/яяяяяяяяяяяяяяяя</a>`,
			`<a href="https://example.com/яяяяяяяяяяяяяяяя">https://example.com/яяяяяяяяяяяяяяяя</a>`,
		},
		{
			25,
			`<a href="https://example.com/яяяяяяяяяяяяяяяя">https://example.com/яяяяяяяяяяяяяяяя</a>`,
			`<a href="https://example.com/яяяяяяяяяяяяяяяя">https://example.com/яя...</a>`,
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
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.out, f.lazyImage(tt.inp))
		})
	}
}
