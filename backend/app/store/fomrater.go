package store

import (
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	blackfriday "gopkg.in/russross/blackfriday.v2"
)

// CommentFormater implements all generic formatings ops on comment
type CommentFormater struct {
	converter CommentConverter
}

// CommentConverter defines interface to convert some parts of commentHTML
// Passed at creation time and does client-defined convertions, like image proxy link change
type CommentConverter interface {
	Convert(commentHTML string) string
}

// NewCommentFormater makes CommentFormater
func NewCommentFormater(converter CommentConverter) *CommentFormater {
	return &CommentFormater{converter: converter}
}

// Format comment fields
func (f *CommentFormater) Format(c Comment) Comment {
	c.Text = f.FormatText(c.Text)
	return c
}

// FormatText formatting line
func (f *CommentFormater) FormatText(txt string) (res string) {
	mdExt := blackfriday.NoIntraEmphasis | blackfriday.Tables | blackfriday.FencedCode |
		blackfriday.Strikethrough | blackfriday.SpaceHeadings | blackfriday.HardLineBreak |
		blackfriday.BackslashLineBreak | blackfriday.Autolink
	res = string(blackfriday.Run([]byte(txt), blackfriday.WithExtensions(mdExt)))
	if f.converter != nil {
		res = f.converter.Convert(res)
	}
	res = f.shortenAutoLinks(res, shortURLLen)
	return res
}

// Shortens all the automatic links in HTML: auto link has equal "href" and "text" attributes.
func (f *CommentFormater) shortenAutoLinks(commentHTML string, max int) (resHTML string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(commentHTML))
	if err != nil {
		return commentHTML
	}
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			if href != s.Text() || len(href) < max+3 || max < 3 {
				return
			}
			url, e := url.Parse(href)
			if e != nil {
				return
			}
			url.Path, url.RawQuery, url.Fragment = "", "", ""
			host := url.String()
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
