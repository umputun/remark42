package migrator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUrlMapper_URL(t *testing.T) {
	// want remap urls from https://radio-t.com to https://www.radio-t.com
	// also map individual urls
	rules := strings.NewReader(`
https://radio-t.com* https://www.radio-t.com*
https://radio-t.com/p/2018/09/22////podcast-616/ https://www.radio-t.com/p/2018/09/22/podcast-616/
https://radio-t.com/p/2018/09/22/podcast-616/?with_query=1 https://www.radio-t.com/p/2018/09/22/podcast-616/
`)

	mapper, err := NewURLMapper(rules)
	assert.NoError(t, err)

	// if url not matched mapper should return given url
	assert.Equal(t, "https://any.com/post/1/", mapper.URL("https://any.com/post/1/"))
	assert.Equal(t, "https://radio-t.co", mapper.URL("https://radio-t.co"))
	// check strict matching
	assert.Equal(t, "https://www.radio-t.com/p/2018/09/22/podcast-616/", mapper.URL("https://radio-t.com/p/2018/09/22////podcast-616/"))
	assert.Equal(t, "https://www.radio-t.com/p/2018/09/22/podcast-616/", mapper.URL("https://radio-t.com/p/2018/09/22/podcast-616/?with_query=1"))
	// check pattern matching (by prefix)
	assert.Equal(t, "https://www.radio-t.com/p/post/123/", mapper.URL("https://radio-t.com/p/post/123/"))

	// want remap from http to https
	rules = strings.NewReader(`http://anysite.com/p/123 https://anysite.com/p/321`)
	mapper, err = NewURLMapper(rules)
	assert.NoError(t, err)
	assert.Equal(t, "https://anysite.com/p/321", mapper.URL("http://anysite.com/p/123"))
	assert.Equal(t, "https://notexist", mapper.URL("https://notexist"))
	assert.Equal(t, "https://anysite.com/", mapper.URL("https://anysite.com/")) // not exist

	// want remap from http to https by pattern
	rules = strings.NewReader(`http://anysite.com* https://anysite.com*`)
	mapper, err = NewURLMapper(rules)
	assert.NoError(t, err)
	assert.Equal(t, "https://anysite.com/p/1", mapper.URL("http://anysite.com/p/1"))
	assert.Equal(t, "https://anysite.com/", mapper.URL("http://anysite.com/"))
	assert.Equal(t, "https://notexist", mapper.URL("https://notexist"))
}

func TestUrlMapper_URLPrefersLongestPrefix(t *testing.T) {
	rules := strings.NewReader(`
https://a.com/* https://x.com/*
https://a.com/blog/* https://y.com/*
`)
	mapper, err := NewURLMapper(rules)
	require.NoError(t, err)

	// the rules sit in a map, so a mapper that picks by iteration order is right only some of the time
	for range 200 {
		require.Equal(t, "https://y.com/1", mapper.URL("https://a.com/blog/1"), "the longer prefix wins")
		require.Equal(t, "https://x.com/about", mapper.URL("https://a.com/about"))
	}

	// a bare asterisk is a prefix rule for every url
	mapper, err = NewURLMapper(strings.NewReader("* https://z.com/*"))
	require.NoError(t, err)
	assert.Equal(t, "https://z.com/https://a.com/1", mapper.URL("https://a.com/1"))
}

func TestUrlMapper_New(t *testing.T) {
	cases := []struct {
		rules       string
		expectError bool
	}{
		// bad input, expect error
		{
			rules:       "https://radio-t.com ",
			expectError: true,
		},
		{
			rules:       "https://radio-t.com https://radio-t.com https://radio-t.com",
			expectError: true,
		},
		{
			rules:       "https://radio-t.com https://radio-t.com\n https://radio-t.com",
			expectError: true,
		},
		{
			rules:       "https://radio-t.com   \n https://radio-t.com https://radio-t.com",
			expectError: true,
		},

		// valid input, no error
		{
			rules: "https://radio-t.com* https://www.radio-t.com*",
		},
		{
			rules: "https://radio-t.com/p/2018/09/22/podcast-616/?with_query=1 https://www.radio-t.com/p/2018/09/22/podcast-616/",
		},
		{
			rules: "https://any.com/p/111 https://any.com/p/222   \n https://any.com/p/333 https://any.com/p/222   \n",
		},
	}
	for _, c := range cases {
		_, err := NewURLMapper(strings.NewReader(c.rules))
		if c.expectError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}
