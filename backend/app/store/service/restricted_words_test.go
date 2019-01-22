package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMatcher_Tokenize(t *testing.T) {

	matcher := NewRestrictedWordsMatcher(StaticRestrictedWordsLister{})

	tbl := []struct {
		input  string
		output []string
	}{
		{
			"   word0 word1 word2, word3,,, !word4 !word5? word6-word7 word8",
			[]string{"word0", "word1", "word2", "word3", "word4", "word5", "word6", "word7", "word8"},
		},
		{"русский 中文 française ไทย", []string{"русский", "中文", "française", "ไทย"}},
		{"word", []string{"word"}},
		{"", []string{}},
		{"\t\t\n\t \n\t \r\n    \t ,,, !#$%^&*()", []string{}},
		{"👍", []string{}},
	}

	for _, td := range tbl {
		tokens := matcher.tokenize(td.input)
		assert.Equal(t, td.output, tokens, "unexpected result for input '%v'", td.input)
	}
}

func TestWildcardTrie_Check(t *testing.T) {

	tbl := []struct {
		input   []string
		match   []string
		nomatch []string
	}{
		{[]string{"abc", "abb", "aab"}, []string{"abc", "abb", "aab"}, []string{"aaa", "aaaa", "a", "ab"}},
		{[]string{"abc", "*ck", "*z"}, []string{"abc", "duck", "quack", "ck", "xyz"}, []string{"quacker", "buzzer"}},
		{[]string{"abc", "du*", "c*"}, []string{"abc", "duck", "dungeon", "du", "cup"}, []string{"bbc", "ddu", "scuba"}},
		{[]string{"abc", "*uc*", "*x*"}, []string{"abc", "duck", "stuck", "uc", "wwxww", "xww", "wwx"}, []string{"bbc", "duke"}},
		{[]string{"abc", "d*k", "st*ck"}, []string{"abc", "duck", "dk", "stck", "stuck", "stiiick"}, []string{"bbc", "adka", "st", "ck"}},
		{[]string{"abc", "*a*a*"}, []string{"abc", "safari", "banana", "aa"}, []string{"bbc", "car", "a"}},
		{
			[]string{"ложить", "при*", "*ий", "*бег*", "про*жа", "*ไ*ย*", "*請*请*"},
			[]string{"ложить", "приклад", "ихний", "прибегать", "пропажа", "ไทย", "ทไย", "ไยท", "請問请问"},
			[]string{"положить", "гранпри", "бийск", "請", "ยไท"},
		},
	}

	for _, td := range tbl {
		n := newWildcardTrie(td.input...)

		for _, token := range td.match {
			assert.True(t, n.check(token), "should match token '%s' for restricted words '%v'", token, td.input)
		}

		for _, token := range td.nomatch {
			assert.False(t, n.check(token), "should not match token '%s' for restricted words '%v'", token, td.input)
		}
	}
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
