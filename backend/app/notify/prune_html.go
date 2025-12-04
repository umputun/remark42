package notify

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// pruneHTML prunes string keeping HTML closing tags.
// maxLength applies to visible text only, not HTML tags.
func pruneHTML(htmlText string, maxLength int) string {
	var result strings.Builder
	var endTokens []string
	visibleLen := 0

	suffix := "..."
	suffixLen := len(suffix)

	tokenizer := html.NewTokenizer(strings.NewReader(htmlText))
	for {
		if tokenizer.Next() == html.ErrorToken {
			return result.String()
		}
		token := tokenizer.Token()

		switch token.Type {
		case html.CommentToken, html.DoctypeToken:
			continue

		case html.StartTagToken:
			endTokens = append([]string{fmt.Sprintf("</%s>", token.Data)}, endTokens...)
			result.WriteString(token.String())

		case html.EndTagToken:
			if len(endTokens) > 0 {
				endTokens = endTokens[1:]
			}
			result.WriteString(token.String())

		case html.SelfClosingTagToken:
			result.WriteString(token.String())

		case html.TextToken:
			text := token.String()
			if visibleLen+len(text)+suffixLen > maxLength {
				remaining := maxLength - visibleLen - suffixLen
				text = pruneStringToWord(text, remaining)
				result.WriteString(text)
				result.WriteString(suffix)
				for _, endTag := range endTokens {
					result.WriteString(endTag)
				}
				return result.String()
			}
			visibleLen += len(text)
			result.WriteString(text)
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
