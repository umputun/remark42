package notify

import (
	"strings"

	"golang.org/x/net/html"
)

// pruneHTML shortens HTML to at most maxLength bytes of visible text and maxMarkup bytes of tags, and
// returns it well nested whatever it is given, as Telegram refuses a message whose tags are not. An end
// tag closes its element together with the elements opened inside it, an end tag with nothing to close
// is dropped, self-closing tags are dropped as Telegram has no void elements, and the elements still open
// at the end are closed. The end tag of an element counts against maxMarkup when the element is opened,
// so a cut can always close what is open. Text cut short ends in "...".
func pruneHTML(htmlText string, maxLength, maxMarkup int) string {
	var result strings.Builder
	var open []string // names of the open elements, innermost last
	visibleLen, markupLen := 0, 0

	const suffix = "..."
	suffixLen := len(suffix)

	closeFrom := func(i int) {
		for j := len(open) - 1; j >= i; j-- {
			result.WriteString("</" + open[j] + ">")
		}
		open = open[:i]
	}
	cut := func() string {
		result.WriteString(suffix)
		closeFrom(0)
		return result.String()
	}

	tokenizer := html.NewTokenizer(strings.NewReader(htmlText))
	for {
		if tokenizer.Next() == html.ErrorToken {
			closeFrom(0)
			return result.String()
		}
		token := tokenizer.Token()

		switch token.Type {
		case html.StartTagToken:
			start := token.String()
			if markupLen+len(start)+len("</"+token.Data+">") > maxMarkup {
				return cut()
			}
			markupLen += len(start) + len("</"+token.Data+">")
			result.WriteString(start)
			open = append(open, token.Data)

		case html.EndTagToken:
			for i := len(open) - 1; i >= 0; i-- {
				if open[i] == token.Data {
					closeFrom(i)
					break
				}
			}

		case html.TextToken:
			text := token.String()
			if visibleLen+len(text)+suffixLen > maxLength {
				result.WriteString(pruneStringToWord(text, maxLength-visibleLen-suffixLen))
				return cut()
			}
			visibleLen += len(text)
			result.WriteString(text)

		default: // comments, doctype and self-closing tags
			continue
		}
	}
}

// pruneStringToWord prunes string to specified length respecting word boundaries
func pruneStringToWord(text string, maxLength int) string {
	if maxLength <= 0 {
		return ""
	}
	if len(text) <= maxLength {
		return text
	}

	// find last space at or before maxLength to cut at word boundary
	lastSpace := strings.LastIndex(text[:maxLength+1], " ")
	if lastSpace <= 0 {
		return ""
	}
	return text[:lastSpace]
}
