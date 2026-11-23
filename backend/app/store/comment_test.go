package store

import (
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestComment_Sanitize(t *testing.T) {
	tbl := []struct {
		inp Comment
		out Comment
	}{

		{inp: Comment{}, out: Comment{}},
		{
			inp: Comment{
				Text: `blah <a href="javascript:alert('XSS1')" onmouseover="alert('XSS2')">XSS</a>` + "\n\t",
				User: User{ID: `<a href="http://blah.com">username</a>`, Name: "name <b/>"},
			},
			out: Comment{
				Text: "blah XSS\n\t",
				User: User{ID: `&lt;a href=&#34;http://blah.com&#34;&gt;username&lt;/a&gt;`, Name: "name"},
			},
		},
		{
			inp: Comment{
				Text: "blah 123" + "\n\t",
				User: User{ID: "id", Name: "xyz-123"},
			},
			out: Comment{
				Text: `blah 123` + "\n\t",
				User: User{ID: "id", Name: "xyz-123"},
			},
		},
		{
			inp: Comment{Text: "blah `123`"},
			out: Comment{Text: "blah `123`"},
		},
		{
			inp: Comment{Text: "blah & & 123 &mdash; &mdash;"},
			out: Comment{Text: `blah &amp; &amp; 123 — —`},
		},
		{
			inp: Comment{Text: "blah & & 123 — —"},
			out: Comment{Text: `blah &amp; &amp; 123 — —`},
		},
		{
			inp: Comment{Text: "blah `123`", User: User{Name: "name \"xxx\" `yyy`"}},
			out: Comment{Text: "blah `123`", User: User{Name: "name \"xxx\" `yyy`"}},
		},
		{
			inp: Comment{Text: "blah & & 123", User: User{Name: "name <> & ' ` \""}},
			out: Comment{Text: `blah &amp; &amp; 123`, User: User{Name: "name &lt;&gt; &amp; ' ` \""}},
		},
		{
			inp: Comment{Text: "blah & & 123", User: User{Name: `name <a href="javascript:alert('XSS1')" onmouseover="alert('XSS2')">XSS</a>`}},
			out: Comment{Text: `blah &amp; &amp; 123`, User: User{Name: "name XSS"}},
		},

		{
			inp: Comment{Text: "blah blah", Locator: Locator{URL: "javascript:alert('XSS1')"}},
			out: Comment{Text: "blah blah", Locator: Locator{URL: ""}},
		},
		{
			inp: Comment{Text: "blah blah", Locator: Locator{URL: "javascript:alert(document.domain)//"}},
			out: Comment{Text: "blah blah", Locator: Locator{URL: ""}},
		},
		{
			inp: Comment{Text: "blah blah", Locator: Locator{URL: "<script>alert()</script>"}},
			out: Comment{Text: "blah blah", Locator: Locator{URL: "%3Cscript%3Ealert%28%29%3C/script%3E"}},
		},
		{
			inp: Comment{Text: "blah blah",
				Locator: Locator{URL: "/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085"}},
			out: Comment{Text: "blah blah",
				Locator: Locator{URL: "/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085"}},
		},
		{
			inp: Comment{Text: "<scrİpt>&lt;img src=x onerror=alert(1)&gt;",
				Locator: Locator{URL: "/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085"}},
			out: Comment{Text: "&lt;img src=x onerror=alert(1)&gt;",
				Locator: Locator{URL: "/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085"}},
		},
		{
			inp: Comment{Text: "blah blah", PostTitle: "<script>alert()</script>something"},
			out: Comment{Text: "blah blah", PostTitle: "something"},
		},
		{
			inp: Comment{Text: "blah blah", PostTitle: "<a href=\"https://example.com\">test</a>"},
			out: Comment{Text: "blah blah", PostTitle: "test"},
		},
		{
			inp: Comment{Text: "blah blah", PostTitle: "https://example.com/blah"}, // link is left as-is, but not rendered as <a>
			out: Comment{Text: "blah blah", PostTitle: "https://example.com/blah"},
		},
		{
			inp: Comment{Text: `<blockquote class="twitter-tweet"><p lang="es" dir="ltr">Silicon iMac Concept<a href="https://t.co/7ga95QxVXn">https://t.co/7ga95QxVXn</a> by <a href="https://twitter.com/marcsheep?ref_src=twsrc%5Etfw">@marcsheep</a> <a href="https://t.co/ULnVpG8w55">pic.twitter.com/ULnVpG8w55</a></p>&mdash; Andreas Storm (@avstorm) <a href="https://twitter.com/avstorm/status/1325693387798933504?ref_src=twsrc%5Etfw">November 9, 2020</a></blockquote> <script async src="https://platform.twitter.com/widgets.js" charset="utf-8"></script>`, PostTitle: "Twitter quote"},
			out: Comment{Text: `<blockquote class="twitter-tweet"><p lang="es" dir="ltr">Silicon iMac Concept<a href="https://t.co/7ga95QxVXn" rel="nofollow ugc">https://t.co/7ga95QxVXn</a> by <a href="https://twitter.com/marcsheep?ref_src=twsrc%5Etfw" rel="nofollow ugc">@marcsheep</a> <a href="https://t.co/ULnVpG8w55" rel="nofollow ugc">pic.twitter.com/ULnVpG8w55</a></p>— Andreas Storm (@avstorm) <a href="https://twitter.com/avstorm/status/1325693387798933504?ref_src=twsrc%5Etfw" rel="nofollow ugc">November 9, 2020</a></blockquote> `, PostTitle: "Twitter quote"},
		},
		{
			inp: Comment{Text: `<img src="https://example.com/a.png" width="600" height="300" class="sized" loading="lazy"/> <img src="https://example.com/b.png" class="big"/>`},
			out: Comment{Text: `<img src="https://example.com/a.png" width="600" height="300" class="sized" loading="lazy"/> <img src="https://example.com/b.png"/>`},
		},
	}

	for n, tt := range tbl {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			tt.inp.Sanitize()
			assert.Equal(t, tt.out, tt.inp, "check #%d", n)
		})
	}
}

// TestComment_SanitizeBalancesTags checks that the tags of a stored comment pair up. The sanitizer keeps
// or drops each tag on its own, and an end tag without its start tag would close an element of whatever
// the text is put into.
func TestComment_SanitizeBalancesTags(t *testing.T) {
	tbl := []struct{ name, inp, out string }{
		{name: "end tags left by dropped elements",
			inp: `<p>ok <svg><title><xmp></title></div></div><h1>Notice</h1><div><div></xmp></title></svg></p>`,
			out: `<p>ok </p><h1>Notice</h1><div><div><p></p></div></div>`},
		{name: "stray end tags", inp: `<p>ok </div></div><h1>Notice</h1><div><div></p>`,
			out: `<p>ok </p><h1>Notice</h1><div><div><p></p></div></div>`},
		{name: "unclosed tags", inp: `<div><h1><s>x`, out: `<div><h1><s>x</s></h1></div>`},
		{name: "misnested tags", inp: `<h1><div><h2><b><h3>x</b></div></div>`,
			out: `<h1><div><h2><b></b></h2><h3><b>x</b></h3></div></h1>`},
		{name: "nested too deep keeps the words only", inp: `</div><h1>h</h1>` + strings.Repeat("<b>", 600) + `x &amp; y`,
			out: `hx &amp; y`},
		{name: "table without tbody gets the one a browser adds", inp: `<table><tr><td>a</td></tr></table>`,
			out: `<table><tbody><tr><td>a</td></tr></tbody></table>`},
		{name: "balanced text is kept byte for byte",
			inp: `<p>plain <b>bold</b> <a id="z" href="https://example.com">l</a><br><img src="https://example.com/i.png"></p>`,
			out: `<p>plain <b>bold</b> <a id="z" href="https://example.com" rel="nofollow ugc">l</a><br><img src="https://example.com/i.png"></p>`},
	}
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			c := Comment{Text: tt.inp}
			c.Sanitize()
			assert.Equal(t, tt.out, c.Text)
			c.Sanitize()
			assert.Equal(t, tt.out, c.Text, "a second pass changes nothing")
		})
	}
}

// TestComment_SanitizeMarksLinksUGC checks that every link in a comment is marked as written by the
// commenter, and that the attribute text typed into a comment or an attribute value is left alone.
func TestComment_SanitizeMarksLinksUGC(t *testing.T) {
	tbl := []struct {
		name, inp, out string
	}{
		{name: "link", inp: `see <a href="https://example.com">this</a>`,
			out: `see <a href="https://example.com" rel="nofollow ugc">this</a>`},
		{name: "commenter's own rel is replaced", inp: `<a href="https://example.com" rel="author">this</a>`,
			out: `<a href="https://example.com" rel="nofollow ugc">this</a>`},
		{name: "area with the commenter's rel", inp: `<map name="m"><area href="https://example.com" rel="author" alt="a"></map>`,
			out: `<map name="m"><area href="https://example.com" rel="author nofollow ugc" alt="a"></map>`},
		{name: "area already nofollow", inp: `<map name="m"><area href="https://example.com" rel="nofollow external" alt="a"></map>`,
			out: `<map name="m"><area href="https://example.com" rel="nofollow external ugc" alt="a"></map>`},
		{name: "area with nofollow only as part of a word", inp: `<map name="m"><area href="https://example.com" rel="nofollowed" alt="a"></map>`,
			out: `<map name="m"><area href="https://example.com" rel="nofollowed nofollow ugc" alt="a"></map>`},
		{name: "id that looks like the start of rel", inp: `<a id="item rel=" href="https://example.com/post">post</a>`,
			out: `<a id="item rel=" href="https://example.com/post" rel="nofollow ugc">post</a>`},
		{name: "attribute typed as text", inp: `<p>a rel="nofollow" b</p>`, out: `<p>a rel=&#34;nofollow&#34; b</p>`},
		{name: "attribute typed into a link address", inp: `<a href='https://example.com/?q="x"rel="nofollow"'>x</a>`,
			out: `<a href="https://example.com/?q=&#34;x&#34;rel=&#34;nofollow&#34;" rel="nofollow ugc">x</a>`},
		{name: "attribute typed into code", inp: `<code> rel="nofollow"</code>`, out: `<code> rel=&#34;nofollow&#34;</code>`},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			c := Comment{Text: tt.inp}
			c.Sanitize()
			assert.Equal(t, tt.out, c.Text)

			c.Sanitize()
			assert.Equal(t, tt.out, c.Text, "a second pass changes nothing")
		})
	}
}

func TestComment_SanitizeEditSummary(t *testing.T) {
	ts := time.Date(2026, 11, 21, 9, 30, 0, 0, time.UTC)
	tbl := []struct {
		inp, out string
	}{
		{inp: "", out: ""},
		{inp: "fixed a typo", out: "fixed a typo"},
		{inp: `fixed typo & added "quote", it's done`, out: `fixed typo &amp; added "quote", it's done`},
		{inp: "<script>alert(1)</script><b>why</b>", out: "why"},
		{inp: `x <img src=x onerror="alert(1)"> y`, out: "x  y"},
		{inp: "5 > 3 && 2 < 4", out: "5 &gt; 3 &amp;&amp; 2 &lt; 4"},
		{inp: "already &amp; escaped &lt;b&gt;", out: "already &amp; escaped &lt;b&gt;"},
		{inp: "  padded  ", out: "padded"},
	}

	for _, tt := range tbl {
		t.Run(tt.inp, func(t *testing.T) {
			stored := &Edit{Timestamp: ts, Summary: tt.inp}
			c := Comment{Text: "blah", Edit: stored}
			c.Sanitize()
			assert.Equal(t, &Edit{Timestamp: ts, Summary: tt.out}, c.Edit)
			assert.Equal(t, tt.inp, stored.Summary, "the record the comment shared is left as it was")

			c.SanitizeEditSummary()
			assert.Equal(t, tt.out, c.Edit.Summary, "cleaning a clean summary again keeps it")
		})
	}

	c := Comment{Text: "blah"}
	c.Sanitize()
	assert.Nil(t, c.Edit, "a comment without an edit keeps none")
}

func TestComment_PrepareUntrusted(t *testing.T) {
	comment := Comment{
		Text:        `blah`,
		User:        User{ID: "username"},
		ParentID:    "p123",
		ID:          "123",
		Locator:     Locator{SiteID: "site", URL: "url"},
		Score:       10,
		Pin:         true,
		Deleted:     true,
		Timestamp:   time.Date(2018, 1, 1, 9, 30, 0, 0, time.UTC),
		Votes:       map[string]bool{"uu": true},
		Controversy: 123,
		Imported:    true,
	}

	comment.PrepareUntrusted()
	assert.Equal(t, "", comment.ID)
	assert.Equal(t, "p123", comment.ParentID)
	assert.Equal(t, "blah", comment.Text)
	assert.Equal(t, 0, comment.Score)
	assert.Equal(t, false, comment.Pin)
	assert.Equal(t, time.Time{}, comment.Timestamp)
	assert.Equal(t, false, comment.Deleted)
	assert.Equal(t, make(map[string]bool), comment.Votes)
	assert.Equal(t, make(map[string]VotedIPInfo), comment.VotedIPs)
	assert.Equal(t, User{ID: "username"}, comment.User)
	assert.Equal(t, 0., comment.Controversy)
	assert.Equal(t, false, comment.Imported)
}

func TestComment_SetDeleted(t *testing.T) {
	comment := Comment{
		Text:      `blah`,
		User:      User{ID: "userid", Name: "username", IP: "123", Picture: "pic"},
		ParentID:  "p123",
		ID:        "123",
		Locator:   Locator{SiteID: "site", URL: "url"},
		Score:     10,
		Deleted:   false,
		Timestamp: time.Date(2018, 1, 1, 9, 30, 0, 0, time.UTC),
		Votes:     map[string]bool{"uu": true},
		Pin:       true,
		PostTitle: "post title",
	}

	comment.SetDeleted(SoftDelete)

	assert.Equal(t, "", comment.Text)
	assert.Equal(t, "", comment.Orig)
	assert.Equal(t, "", comment.PostTitle)
	assert.Equal(t, map[string]bool{}, comment.Votes)
	assert.Equal(t, map[string]VotedIPInfo{}, comment.VotedIPs)
	assert.Equal(t, 0, comment.Score)
	assert.True(t, comment.Deleted)
	assert.Nil(t, comment.Edit)
	assert.False(t, comment.Pin)
	assert.Equal(t, User{Name: "username", ID: "userid", Picture: "pic", Admin: false, Blocked: false, IP: "123"}, comment.User)
}

func TestComment_SetDeletedHard(t *testing.T) {
	comment := Comment{
		Text:      `blah`,
		User:      User{ID: "userid", Name: "username", IP: "123", Picture: "pic"},
		ParentID:  "p123",
		ID:        "123",
		Locator:   Locator{SiteID: "site", URL: "url"},
		Score:     10,
		Deleted:   false,
		Timestamp: time.Date(2018, 1, 1, 9, 30, 0, 0, time.UTC),
		Votes:     map[string]bool{"uu": true},
		Pin:       true,
		PostTitle: "post title",
	}

	comment.SetDeleted(HardDelete)

	assert.Equal(t, "", comment.Text)
	assert.Equal(t, "", comment.Orig)
	assert.Equal(t, "", comment.PostTitle)
	assert.Equal(t, map[string]bool{}, comment.Votes)
	assert.Equal(t, map[string]VotedIPInfo{}, comment.VotedIPs)
	assert.Equal(t, 0, comment.Score)
	assert.True(t, comment.Deleted)
	assert.Nil(t, comment.Edit)
	assert.False(t, comment.Pin)
	assert.Equal(t, User{Name: "deleted", ID: "deleted", Picture: "", Admin: false, Blocked: false, IP: ""}, comment.User)
}

func TestComment_Snippet(t *testing.T) {
	tbl := []struct {
		limit int
		inp   string
		out   string
	}{
		{0, "", ""},
		{-1, "test\nblah", "test blah"},
		{5, "test\nblah", "test ..."},
		{5, "xyz12345 xxx", "xyz12..."},
		{10, "xyz12345 xxx\ntest 123456", "xyz12345 ..."},
		{0, "<p>some <b>bold</b>, text.</p>\n<p>and more</p>\n", "some bold, text. and more"},
		{0, "<p>one</p><p>two</p><ul><li>three</li><li>four</li></ul>five<br/>six", "one two three four five six"},
		{0, "un<em>believ</em>able <code>co</code>de", "unbelievable code"},
		{0, "<section>one</section><section>two</section><summary>three</summary>four", "one two three four"},
		{0, `<p>1 &lt; 2 &amp; <a href="https://example.com">a "link"</a></p>`, "1 &lt; 2 &amp; a &#34;link&#34;"},
		{0, "<p>quoted</blockquote></p><script>alert(1)</script>", "quoted"},
		{14, `<p>picture <img src="https://example.com/p.png" alt="a b c"/> and text</p>`, "picture and ..."},
		{4, "<p>&lt;&lt;&lt;&lt;&lt;&lt;</p>", "&lt;&lt;&lt;&lt;..."},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c := Comment{Text: tt.inp}
			out := c.Snippet(tt.limit)
			assert.Equal(t, tt.out, out)
		})
	}
}

func TestComment_sanitizeAsURL(t *testing.T) {
	tbl := []struct {
		inp, out string
	}{
		{
			"/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085",
			"/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085",
		},
		{
			"https://radio-t.com/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085",
			"https://radio-t.com/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085",
		},
		{
			"javascript:alert(document.domain)//",
			"",
		},
		{
			"<script>alert()</script>",
			"%3Cscript%3Ealert%28%29%3C/script%3E",
		},
		{
			"<a href=javascript:alert(document.domain)//>xxx</a>",
			"",
		},
		{
			`https://radio-t.com/p/2021/03/23/prep-747/?ref=\`,
			`https://radio-t.com/p/2021/03/23/prep-747/?ref=\`,
		},
		{
			`https://radio-t.com/p/2021/03/23/prep-747/?ref=\\`, // as stored by older versions for a single backslash
			`https://radio-t.com/p/2021/03/23/prep-747/?ref=\\`,
		},
		{
			`https://radio-t.com/p/2021/03/23/prep\747/#a\b`,
			"https://radio-t.com/p/2021/03/23/prep%5C747/#a%5Cb",
		},
		{
			`https://radio-t.com/p/2021/03/23/prep-747/?a="b"&c='d'&amp;e=f`,
			`https://radio-t.com/p/2021/03/23/prep-747/?a="b"&c='d'&amp;e=f`,
		},
		{
			"https://radio-t.com/p/2021/03/23/prep-747/?ref=\u200b",
			"https://radio-t.com/p/2021/03/23/prep-747/?ref=\u200b",
		},
		{
			"https://radio-t.com/p/2021/03/23/prep-747/?a=b\nc",
			"",
		},
		{
			"https://radio-t.com/p/2021/03/23/prep-747/?a=b\x01",
			"",
		},
		{
			"https://radio-t.com/p/2021/03/23/prep-747/?ref=\u00a0#",
			"https://radio-t.com/p/2021/03/23/prep-747/?ref=",
		},
		// white space before the url is dropped, whatever kind it is
		{"\thttps://radio-t.com/p/", "https://radio-t.com/p/"},
		{"\n\u00a0 https://radio-t.com/p/", "https://radio-t.com/p/"},
		// the html parser replaces NUL; a control character in the fragment, or one past ASCII, is kept escaped
		{"https://radio-t.com\x00/p/", "https://radio-t.com%EF%BF%BD/p/"},
		{"https://radio-t.com/p#a\x01b", "https://radio-t.com/p#a%01b"},
		{"https://radio-t.com/p/\u0085x", "https://radio-t.com/p/%C2%85x"},
		// a relative path starting with an encoded slash is kept while it can be written back as it came
		{"/%2Ffolder/file", "/%2Ffolder/file"},
		{"//radio-t.com/p/", "//radio-t.com/p/"},
		// and refused once it would come out starting with "//", which the next pass would read as a host:
		// a url that a second pass would change is not accepted
		{"/%2F<x", ""},
		{`/%2Ffolder\file`, ""},
		{`%2f%2f\`, ""},
		{"%2F/\u00a0=]22@@=", ""},
		{"http://[fe80::1%25en%5B0]/p", ""}, // an escaped bracket in the zone of an address is written back unescaped
		// a relative path that would come out as "//host/..." is refused although a second pass keeps it
		{"/%2Fradio-t.com/p/<", ""},
		{"%2F/radio-t.com/p/<", ""},
		{"/%2Fdocs/caf\u00e9", ""},
	}

	for i, tt := range tbl {
		c := Comment{}
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			out := c.SanitizeAsURL(tt.inp)
			assert.Equal(t, tt.out, out)
			// a stored url is sanitized again on every read, the second pass must not change it
			assert.Equal(t, out, c.SanitizeAsURL(out), "not idempotent")
		})
	}
}

func TestComment_sanitizeText(t *testing.T) {
	tbl := []struct {
		inp, out string
	}{
		{
			"/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085",
			"/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085",
		},
		{
			"https://radio-t.com/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085",
			"https://radio-t.com/p/2021/03/23/prep-747/#remark42__comment-1b365913-7056-4920-b9ad-01304bdda085",
		},
		{
			"<script>alert()</script>something",
			"something",
		},
		{
			"<a href=javascript:alert(document.domain)//>xxx</a>",
			"xxx",
		},
	}

	for i, tt := range tbl {
		c := Comment{}
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.out, c.SanitizeText(tt.inp))
		})
	}
}

func TestComment_sanitizeSharedPolicy(t *testing.T) {
	// the policies are built once, not per call: building a UGC policy alone takes 800+ allocations
	c := Comment{}
	assert.Less(t, testing.AllocsPerRun(10, func() { c.SanitizeAsURL("https://radio-t.com/p/2021/03/23/prep-747/") }), 200.0)
	assert.Less(t, testing.AllocsPerRun(10, func() { c.SanitizeText("<i>prep</i> 747") }), 35.0)
	assert.Less(t, testing.AllocsPerRun(10, func() {
		cc := Comment{Text: "<p>blah</p>", User: User{ID: "id", Name: "name", Picture: "https://example.com/pic.png"},
			Locator: Locator{URL: "https://radio-t.com/p/2021/03/23/prep-747/"}, PostTitle: "prep 747"}
		cc.Sanitize()
	}), 600.0)

	// the shared policies are used by many requests at once, the race detector checks it is safe
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			cc := Comment{Text: `<pre class="chroma">blah</pre><script>alert()</script>`, User: User{Name: "<b>name</b>"},
				Locator: Locator{URL: "javascript:alert(document.domain)//"}, PostTitle: "<i>prep</i> 747"}
			cc.Sanitize()
			assert.Equal(t, `<pre class="chroma">blah</pre>`, cc.Text)
			assert.Equal(t, "name", cc.User.Name)
			assert.Empty(t, cc.Locator.URL)
			assert.Equal(t, "prep 747", cc.PostTitle)
		})
	}
	wg.Wait()
}
