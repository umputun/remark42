package notify

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

const commentTextLengthLimit = 100

type stringArr struct {
	data []string
	len  int
}

// Push adds element to the end
func (s *stringArr) Push(v string) {
	s.data = append(s.data, v)
	s.len += len(v)
}

// Pop removes element from end and returns it
func (s *stringArr) Pop() string {
	l := len(s.data)
	newData, v := s.data[:l-1], s.data[l-1]
	s.data = newData
	s.len -= len(v)
	return v
}

// Unshift adds element to the start
func (s *stringArr) Unshift(v string) {
	s.data = append([]string{v}, s.data...)
	s.len += len(v)
}

// Shift removes element from start and returns it
func (s *stringArr) Shift() string {
	v, newData := s.data[0], s.data[1:]
	s.data = newData
	s.len -= len(v)
	return v
}

// String returns all strings concatenated
func (s stringArr) String() string {
	return strings.Join(s.data, "")
}

// Len returns total length of all strings concatenated
func (s stringArr) Len() int {
	return s.len
}

// pruneHTML prunes string keeping HTML closing tags
func pruneHTML(htmlText string, maxLength int) string {
	result := stringArr{}
	endTokens := stringArr{}

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
			// skip tokens without content
			continue

		case html.StartTagToken:
			// <token></token>
			// len(token) * 2 + len("<></>")
			totalLenToAppend := len(token.Data)*2 + 5

			lengthAfterChange := result.Len() + totalLenToAppend + endTokens.Len() + suffixLen

			if lengthAfterChange > maxLength {
				return result.String() + suffix + endTokens.String()
			}

			endTokens.Unshift(fmt.Sprintf("</%s>", token.Data))

		case html.EndTagToken:
			endTokens.Shift()

		case html.TextToken, html.SelfClosingTagToken:
			lengthAfterChange := result.Len() + len(token.String()) + endTokens.Len() + suffixLen

			if lengthAfterChange > maxLength {
				text := pruneStringToWord(token.String(), maxLength-result.Len()-endTokens.Len()-suffixLen)
				return result.String() + text + suffix + endTokens.String()
			}
		}

		result.Push((token.String()))
	}
}

// pruneStringToWord prunes string to specified length respecting words
func pruneStringToWord(text string, maxLength int) string {
	if maxLength <= 0 {
		return ""
	}

	result := ""

	arr := strings.Split(text, " ")
	for _, s := range arr {
		if len(result)+len(s) >= maxLength {
			return strings.TrimRight(result, " ")
		}
		result += s + " "
	}

	return text
}
