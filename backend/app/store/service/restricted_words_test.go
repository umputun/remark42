package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type DummyLister struct {
	Words []string
}

func (l DummyLister) List(siteID string) (restricted []string, err error) {
	return l.Words, nil
}

func TestMatcher_Tokenize(t *testing.T) {
	matcher := NewMatcher(DummyLister{})
	input := "   word0 word1 word2, word3,,, !word4 !word5? word6-word7 word8"
	expectedTokens := []string{"word0", "word1", "word2", "word3", "word4", "word5", "word6", "word7", "word8"}
	actualTokens := matcher.tokenize(input)

	assert.Equal(t, expectedTokens, actualTokens)
}

func TestMatcher_TokenizeLanguages(t *testing.T) {
	matcher := NewMatcher(DummyLister{})
	input := "русский 中文 française ไทย"
	expectedTokens := []string{"русский", "中文", "française", "ไทย"}
	actualTokens := matcher.tokenize(input)

	assert.Equal(t, expectedTokens, actualTokens)
}

func TestMatcher_TokenizeSingleWord(t *testing.T) {
	matcher := NewMatcher(DummyLister{})
	expectedTokens := []string{"word"}
	actualTokens := matcher.tokenize("word")
	assert.Equal(t, 1, len(actualTokens))
	assert.Equal(t, expectedTokens, actualTokens)
}

func TestMatcher_TokenizeEmptyString(t *testing.T) {
	matcher := NewMatcher(DummyLister{})
	actualTokens := matcher.tokenize("")
	assert.Equal(t, 0, len(actualTokens))
}

func TestMatcher_TokenizeSemanticallyEmptyString(t *testing.T) {
	matcher := NewMatcher(DummyLister{})
	actualTokens := matcher.tokenize("\t\t\n\t \n\t \r\n    \t ,,, !#$%^&*()")
	assert.Equal(t, 0, len(actualTokens))
}

func TestMatcher_TokenizeEmoji(t *testing.T) {
	matcher := NewMatcher(DummyLister{})
	actualTokens := matcher.tokenize("👍")
	assert.Equal(t, 0, len(actualTokens))
}

func TestMatcher_MatchIfContainsRestrictedWords(t *testing.T) {
	matcher := NewMatcher(DummyLister{[]string{"duck"}})
	text := "What the duck it that?"
	assert.True(t, matcher.Match("fakeID", text))
}

func TestMatcher_DoNotMatchIfNoRestrictedWords(t *testing.T) {
	matcher := NewMatcher(DummyLister{[]string{"quack"}})
	text := "What the duck it that?"
	assert.False(t, matcher.Match("fakeID", text))
}
