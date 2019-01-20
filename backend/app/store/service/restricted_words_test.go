package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMatcher_Tokenize(t *testing.T) {
	matcher := NewRestrictedWordsMatcher(StaticRestrictedWordsLister{})
	input := "   word0 word1 word2, word3,,, !word4 !word5? word6-word7 word8"
	expectedTokens := []string{"word0", "word1", "word2", "word3", "word4", "word5", "word6", "word7", "word8"}
	actualTokens := matcher.tokenize(input)

	assert.Equal(t, expectedTokens, actualTokens)
}

func TestMatcher_TokenizeLanguages(t *testing.T) {
	matcher := NewRestrictedWordsMatcher(StaticRestrictedWordsLister{})
	input := "русский 中文 française ไทย"
	expectedTokens := []string{"русский", "中文", "française", "ไทย"}
	actualTokens := matcher.tokenize(input)

	assert.Equal(t, expectedTokens, actualTokens)
}

func TestMatcher_TokenizeSingleWord(t *testing.T) {
	matcher := NewRestrictedWordsMatcher(StaticRestrictedWordsLister{})
	expectedTokens := []string{"word"}
	actualTokens := matcher.tokenize("word")
	assert.Equal(t, 1, len(actualTokens))
	assert.Equal(t, expectedTokens, actualTokens)
}

func TestMatcher_TokenizeEmptyString(t *testing.T) {
	matcher := NewRestrictedWordsMatcher(StaticRestrictedWordsLister{})
	actualTokens := matcher.tokenize("")
	assert.Equal(t, 0, len(actualTokens))
}

func TestMatcher_TokenizeSemanticallyEmptyString(t *testing.T) {
	matcher := NewRestrictedWordsMatcher(StaticRestrictedWordsLister{})
	actualTokens := matcher.tokenize("\t\t\n\t \n\t \r\n    \t ,,, !#$%^&*()")
	assert.Equal(t, 0, len(actualTokens))
}

func TestMatcher_TokenizeEmoji(t *testing.T) {
	matcher := NewRestrictedWordsMatcher(StaticRestrictedWordsLister{})
	actualTokens := matcher.tokenize("👍")
	assert.Equal(t, 0, len(actualTokens))
}

func TestMatcher_MatchIfContainsRestrictedWords(t *testing.T) {
	matcher := NewRestrictedWordsMatcher(StaticRestrictedWordsLister{[]string{"duck"}})
	text := "What the duck it that?"
	assert.True(t, matcher.Match("fakeID", text))
}

func TestMatcher_DoNotMatchIfNoRestrictedWords(t *testing.T) {
	matcher := NewRestrictedWordsMatcher(StaticRestrictedWordsLister{[]string{"quack"}})
	text := "What the duck it that?"
	assert.False(t, matcher.Match("fakeID", text))
}

func TestWildcardTrie_Check(t *testing.T) {
	n := newWildcardTrie("abc", "abb", "aab")

	shouldMatch(t, n, "abc", "abb", "aab")
	shouldNotMatch(t, n, "aaa", "aaaa", "a", "ab")
}

func TestWildcardTrie_CheckPrefixWildcards(t *testing.T) {
	n := newWildcardTrie("abc", "*ck", "?olor", "+itch", "!unt")

	shouldMatch(t, n, "abc", "duck", "quack", "ck", "color", "olor", "witch", "skitch", "aunt")
	shouldNotMatch(t, n, "quacker", "itch", "unt", "stunt")
}

func TestWildcardTrie_CheckSuffixWildcards(t *testing.T) {
	n := newWildcardTrie("abc", "du*", "colo?", "pitch+", "aun!")

	shouldMatch(t, n, "abc", "duck", "dungeon", "du", "color", "colo", "pitcher", "pitch1", "aunt")
	shouldNotMatch(t, n, "bbc", "pitch", "aun", "auntie")
}

func TestWildcardTrie_CheckPrefixAndSuffixWildcards(t *testing.T) {
	n := newWildcardTrie("abc", "*uc*", "?olo?", "+itc+", "!un!")

	shouldMatch(t, n, "abc", "duck", "stuck", "uc", "color", "olo", "pitch", "thewitcher", "aunt")
	shouldNotMatch(t, n, "bbc", "trololo", "itc", "itcher", "pitc", "un", "stunt", "auntie")
}

func TestWildcardTrie_CheckInnerWildcards(t *testing.T) {
	n := newWildcardTrie("abc", "d*k", "colo?r", "pi+h", "au!t")

	shouldMatch(t, n, "abc", "duck", "dk", "color", "colour", "pitch", "pipestash", "aunt")
	shouldNotMatch(t, n, "quack", "colouur", "pih", "aut", "august")
}

func TestWildcardTrie_CheckInnerAndOuterWildcards(t *testing.T) {
	n := newWildcardTrie("abc", "*a*a*", "?o?o?", "+p+p+", "!i!i!")

	shouldMatch(t, n, "abc", "safari", "banana", "aa", "oo", "olo", "color", "xpxpx", "xyzpxyzpxyz", "wiwiw")
	shouldNotMatch(t, n, "car", "colour", "pp", "xpxp", "pxpx", "ii", "wwwiwwwiwww")
}

func shouldMatch(t *testing.T, n *wildcardTrie, tokens ...string) {
	for _, token := range tokens {
		assert.True(t, n.check(token), "should match token '%s'", token)
	}
}

func shouldNotMatch(t *testing.T, n *wildcardTrie, tokens ...string) {
	for _, token := range tokens {
		assert.False(t, n.check(token), "should not match token '%s'", token)
	}
}
