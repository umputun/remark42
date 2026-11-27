package notify

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func TestPruneHTML(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		maxLength int
		maxMarkup int
		expected  string
	}{
		{"within limit", "<p>Hello</p>", 20, 100, "<p>Hello</p>"},
		{"exceeds limit", "<p>Hello world, this is a long text</p>", 15, 100, "<p>Hello world,...</p>"},
		{"nested tags", "<div><p>Hello world</p><p>More text</p></div>", 20, 100, "<div><p>Hello world</p><p>More...</p></div>"},
		{"html comment stripped", "<!-- comment --><p>Hello</p>", 20, 100, "<p>Hello</p>"},
		{"self-closing tag dropped", "<p>Hello<br/>World</p>", 8, 100, "<p>Hello...</p>"},
		{"end tag closing nothing dropped", "</i>Hello</b> world", 20, 100, "Hello world"},
		{"open tags closed at the end", "<b>Hello <i>world", 20, 100, "<b>Hello <i>world</i></b>"},
		{"misnested tags", "<del>a <ins>b</del> c</ins>", 20, 100, "<del>a <ins>b</ins></del> c"},
		{"end tag closes the elements inside", "<b><i><s>a</b>c", 20, 100, "<b><i><s>a</s></i></b>c"},
		{"same element nested", "<b>a<b>b</b>c</b>d", 20, 100, "<b>a<b>b</b>c</b>d"},
		{"markup limit reached", "<b>a<i>b<s>c</s></i></b>", 20, len("<b></b><i></i>"), "<b>a<i>b...</i></b>"},
		{"markup limit counts attributes", `<a href="https://example.com/long/path">link</a> text`, 20, 20, "..."},
		{"closing tags of repeated elements counted", strings.Repeat("<b>", 50) + "x", 20, 70, strings.Repeat("<b>", 10) + "..." + strings.Repeat("</b>", 10)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := pruneHTML(tt.html, tt.maxLength, tt.maxMarkup)
			assert.Equal(t, tt.expected, res)
			assert.Empty(t, nestingProblems(res))
		})
	}
}

// TestPruneHTML_Bounded feeds markup a comment can carry and checks the two promises of pruneHTML for each:
// tags that nest properly, and no more than the given bytes of them
func TestPruneHTML_Bounded(t *testing.T) {
	inputs := []string{
		"</i>Buy now",
		"<b>left open",
		"<del>a <ins>b</del> c</ins>",
		"<del/>Buy <ins/>",
		"nice post <b/>",
		strings.Repeat("<b>", 5000) + strings.Repeat(".</b>", 40),
		strings.Repeat("<b><i>.</i></b>", 2000),
		`<a href="https://example.com/` + strings.Repeat("x", 5000) + `">link</a>`,
		strings.Repeat("</b>", 1000) + "text" + strings.Repeat("<i>", 1000),
	}
	for _, maxMarkup := range []int{0, 7, 100, 4096} {
		for _, in := range inputs {
			res := pruneHTML(in, 100, maxMarkup)
			assert.Empty(t, nestingProblems(res), "input %.60q", in)
			assert.LessOrEqual(t, markupLen(res), maxMarkup, "input %.60q", in)
			assert.LessOrEqual(t, len(res)-markupLen(res), 100, "input %.60q", in)
		}
	}
}

// nestingProblems lists what is wrong with the nesting of the tags in an HTML text: end tags that do not
// close the innermost open element, self-closing tags and elements left open
func nestingProblems(s string) []string {
	var probs, open []string
	z := html.NewTokenizer(strings.NewReader(s))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		name, _ := z.TagName()
		switch tt {
		case html.StartTagToken:
			open = append(open, string(name))
		case html.SelfClosingTagToken:
			probs = append(probs, "self-closing "+string(name))
		case html.EndTagToken:
			if len(open) == 0 || open[len(open)-1] != string(name) {
				probs = append(probs, fmt.Sprintf("end tag %s with %v open", name, open))
				continue
			}
			open = open[:len(open)-1]
		}
	}
	if len(open) > 0 {
		probs = append(probs, fmt.Sprintf("left open %v", open))
	}
	return probs
}

// markupLen counts the bytes of tags in an HTML text
func markupLen(s string) int {
	res := 0
	z := html.NewTokenizer(strings.NewReader(s))
	for {
		switch z.Next() {
		case html.ErrorToken:
			return res
		case html.StartTagToken, html.EndTagToken, html.SelfClosingTagToken:
			res += len(z.Raw())
		}
	}
}

func TestPruneStringToWord(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLength int
		expected  string
	}{
		{"within limit", "hello world", 15, "hello world"},
		{"cut at word boundary", "hello world and more", 11, "hello world"},
		{"zero length", "hello", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, pruneStringToWord(tt.text, tt.maxLength))
		})
	}
}
