package store

import (
	"net/url"
	"strings"

	"github.com/Depado/bfchroma/v2"
	"github.com/PuerkitoBio/goquery"
	"github.com/alecthomas/chroma/v2/formatters/html"
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
func (f *CommentFormatter) FormatText(txt string, raw bool) (res string) {
	mdExt, rend := GetMdExtensionsAndRenderer(raw)
	res = string(bf.Run([]byte(txt), bf.WithExtensions(mdExt), bf.WithRenderer(rend)))
	res = f.unEscape(res)

	for _, conv := range f.converters {
		res = conv.Convert(res)
	}
	res = f.shortenAutoLinks(res, shortURLLen)
	res = f.lazyImage(res)
	return res
}

// Shortens all the automatic links in HTML: auto link has equal "href" and "text" attributes.
func (f *CommentFormatter) shortenAutoLinks(commentHTML string, max int) (resHTML string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(commentHTML))
	if err != nil {
		return commentHTML
	}
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			if href != s.Text() || len(href) < max+3 || max < 3 {
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

			short := string([]rune(href)[:max-3])
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
		res = strings.Replace(res, e.from, e.to, -1)
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
