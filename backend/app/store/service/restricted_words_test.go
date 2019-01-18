package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWhitespaceTokenizer_Tokenize(t *testing.T) {
	tokenizer := WhitespaceTokenizer{}
	input := "   word0 word1 word2, word3,,, !word4 !word5? word6-word7 word8"
	expectedTokens := []string{"word0", "word1", "word2", "word3", "word4", "word5", "word6", "word7", "word8"}
	actualTokens := tokenizer.Tokenize(input)

	assert.Equal(t, expectedTokens, actualTokens)
}

func TestWhitespaceTokenizer_TokenizeLanguages(t *testing.T) {
	tokenizer := WhitespaceTokenizer{}
	input := "русский 中文 française ไทย"
	expectedTokens := []string{"русский", "中文", "française", "ไทย"}
	actualTokens := tokenizer.Tokenize(input)

	assert.Equal(t, expectedTokens, actualTokens)
}

func TestWhitespaceTokenizer_TokenizeSingleWord(t *testing.T) {
	tokenizer := WhitespaceTokenizer{}
	expectedTokens := []string{"word"}
	actualTokens := tokenizer.Tokenize("word")
	assert.Equal(t, 1, len(actualTokens))
	assert.Equal(t, expectedTokens, actualTokens)
}

func TestWhitespaceTokenizer_TokenizeEmptyString(t *testing.T) {
	tokenizer := WhitespaceTokenizer{}
	actualTokens := tokenizer.Tokenize("")
	assert.Equal(t, 0, len(actualTokens))
}

func TestWhitespaceTokenizer_TokenizeSemanticallyEmptyString(t *testing.T) {
	tokenizer := WhitespaceTokenizer{}
	actualTokens := tokenizer.Tokenize("\t\t\n\t \n\t \r\n    \t ,,, !#$%^&*()")
	assert.Equal(t, 0, len(actualTokens))
}

func TestWhitespaceTokenizer_TokenizeEmoji(t *testing.T) {
	tokenizer := WhitespaceTokenizer{}
	actualTokens := tokenizer.Tokenize("👍")
	assert.Equal(t, 0, len(actualTokens))
}
