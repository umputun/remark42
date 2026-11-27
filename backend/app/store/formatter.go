package store

import (
	"bytes"
	"html/template"
	"io"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/Depado/bfchroma/v2"
	"github.com/PuerkitoBio/goquery"
	"github.com/alecthomas/chroma/v2/formatters/html"
	log "github.com/go-pkgz/lgr"
	bf "github.com/russross/blackfriday/v2"
)

// CommentFormatter implements all generic formatting ops on comment
type CommentFormatter struct {
	converters []CommentConverter
}

// CommentConverter defines interface to convert some parts of commentHTML
// Passed at creation time and does client-defined conversions, like image proxy link change
type CommentConverter interface {
	Convert(text string) string
}

// CommentConverterFunc functional struct implementing CommentConverter
type CommentConverterFunc func(text string) string

// Convert calls func for given text
func (f CommentConverterFunc) Convert(text string) string {
	return f(text)
}

// NewCommentFormatter makes CommentFormatter
func NewCommentFormatter(converters ...CommentConverter) *CommentFormatter {
	return &CommentFormatter{converters: converters}
}

// Format comment fields
func (f *CommentFormatter) Format(c Comment, raw bool) Comment {
	c.Text = f.FormatText(c.Text, raw)
	return c
}

// FormatText converts text with markdown processor, applies external converters and shortens links
//
// raw=true disables SmartyPants for HTML rendering (replacement of quotes, dashes, fractions, etc).
//
// A code block that can't be highlighted is rendered as plain code. Text the markdown processor
// panics on is returned HTML-escaped, as a paragraph with line breaks.
func (f *CommentFormatter) FormatText(txt string, raw bool) (res string) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("[WARN] can't format comment text, returned it escaped, %v", rec)
			res = "<p>" + strings.ReplaceAll(template.HTMLEscapeString(txt), "\n", "<br/>\n") + "</p>\n"
		}
	}()
	mdExt, rend := GetMdExtensionsAndRenderer(raw)
	res = string(bf.Run([]byte(txt), bf.WithExtensions(mdExt), bf.WithRenderer(plainCodeOnPanic{rend})))
	return f.FormatHTML(f.unEscape(res))
}

// FormatHTML shortens the links of text that is already HTML, sanitizes it, applies external converters and
// marks its images lazy.
//
// The converters and lazyImage find images with an HTML parser, while the sanitizer reads tags one at a time,
// and the two can disagree: an <img> inside an HTML comment within an SVG <title> is comment text to the parser
// and a tag to the sanitizer. Sanitizing first gives the converters the tags that are stored. Links are
// shortened before that, as the sanitizer can re-encode a link's href so that it no longer equals its text.
func (f *CommentFormatter) FormatHTML(commentHTML string) (res string) {
	res = f.shortenAutoLinks(commentHTML, shortURLLen)
	res = sanitizeBalanced(res)
	for _, conv := range f.converters {
		res = conv.Convert(res)
	}
	res = f.lazyImage(res)
	return res
}

// Shortens all the automatic links in HTML: auto link has equal "href" and "text" attributes.
func (f *CommentFormatter) shortenAutoLinks(commentHTML string, maximum int) (resHTML string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(commentHTML))
	if err != nil {
		return commentHTML
	}
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			// the link is cut by runes below, so its length is counted in runes too
			if href != s.Text() || utf8.RuneCountInString(href) < maximum+3 || maximum < 3 {
				return
			}
			commentURL, e := url.Parse(href)
			if e != nil {
				return
			}
			commentURL.Path, commentURL.RawQuery, commentURL.Fragment = "", "", ""
			host := commentURL.String()
			if host == "" {
				return
			}

			short := string([]rune(href)[:maximum-3])
			if len(short) < len(host) {
				short = host
			}
			s.SetText(short + "...")
		}
	})
	resHTML, err = doc.Find("body").Html()
	if err != nil {
		return commentHTML
	}
	return resHTML
}

func (f *CommentFormatter) unEscape(txt string) (res string) {
	elems := []struct {
		from, to string
	}{
		{`&amp;mdash;`, "—"},
	}
	res = txt
	for _, e := range elems {
		res = strings.ReplaceAll(res, e.from, e.to)
	}
	return res
}

// lazyImage adds loading=“lazy” attribute to all images
func (f *CommentFormatter) lazyImage(commentHTML string) (resHTML string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(commentHTML))
	if err != nil {
		return commentHTML
	}
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		s.SetAttr("loading", "lazy")
	})
	resHTML, err = doc.Find("body").Html()
	if err != nil {
		return commentHTML
	}
	return resHTML
}

// GetMdExtensionsAndRenderer returns blackfriday extensions and renderer used for rendering markdown
// within store module.
//
// raw=true disables SmartyPants for HTML rendering (replacement of quotes, dashes, fractions, etc).
func GetMdExtensionsAndRenderer(raw bool) (bf.Extensions, *bfchroma.Renderer) {
	mdExt := bf.NoIntraEmphasis | bf.Tables | bf.FencedCode |
		bf.Strikethrough | bf.SpaceHeadings | bf.HardLineBreak |
		bf.BackslashLineBreak | bf.Autolink

	flags := bf.HTMLFlags(0)
	if !raw {
		flags = bf.Smartypants | bf.SmartypantsFractions | bf.SmartypantsDashes | bf.SmartypantsAngledQuotes
	}

	rend := bf.NewHTMLRenderer(bf.HTMLRendererParameters{Flags: flags})

	extRend := bfchroma.NewRenderer(bfchroma.Extend(rend), bfchroma.ChromaOptions(html.WithClasses(true)))
	return mdExt, extRend
}

// plainCodeOnPanic renders a code block as plain code when its highlighting panics, the same way
// bfchroma renders one whose highlighting fails. The lexer is picked by the language the commenter
// names after the fence, and some of chroma's lexers panic on some input.
type plainCodeOnPanic struct {
	*bfchroma.Renderer
}

// RenderNode satisfies the Renderer interface
func (r plainCodeOnPanic) RenderNode(w io.Writer, node *bf.Node, entering bool) bf.WalkStatus {
	if node.Type != bf.CodeBlock {
		return r.Renderer.RenderNode(w, node, entering)
	}
	var buf bytes.Buffer // nothing reaches w if the highlighting panics halfway
	if !r.highlight(&buf, node) {
		return r.Base.RenderNode(w, node, entering)
	}
	_, _ = buf.WriteTo(w)
	return bf.SkipChildren
}

// highlight writes the code block highlighted by chroma to w and reports whether it succeeded
func (r plainCodeOnPanic) highlight(w io.Writer, node *bf.Node) (ok bool) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("[WARN] can't highlight code block, rendered it plain, %v", rec)
			ok = false
		}
	}()
	return r.RenderWithChroma(w, node.Literal, node.CodeBlockData) == nil
}
