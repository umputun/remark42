package store

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	blackfriday "gopkg.in/russross/blackfriday.v2"
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
func (f *CommentFormatter) Format(c Comment) Comment {
	c.Text = f.FormatText(c.Text)
	return c
}

// FormatText formatting line
func (f *CommentFormatter) FormatText(txt string) (res string) {
	mdExt := blackfriday.NoIntraEmphasis | blackfriday.Tables | blackfriday.FencedCode |
		blackfriday.Strikethrough | blackfriday.SpaceHeadings | blackfriday.HardLineBreak |
		blackfriday.BackslashLineBreak | blackfriday.Autolink
	res = string(blackfriday.Run([]byte(txt), blackfriday.WithExtensions(mdExt)))
	for _, conv := range f.converters {
		res = conv.Convert(res)

	}
	res = f.shortenAutoLinks(res, shortURLLen)
	return res
}

// Shortens all the automatic links in HTML: auto link has equal "href" and "text" attributes.
func (f *CommentFormatter) shortenAutoLinks(commentHTML string, max int) (resHTML string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(commentHTML))
	if err != nil {
		return commentHTML
	}
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
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
			short := href[:max-3]
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
