package migrator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUrlMapper(t *testing.T) {
	// want remap urls from https://radio-t.com to https://www.radio-t.com
	// also map individual urls
	rules := strings.NewReader(`
https://radio-t.com* https://www.radio-t.com*
https://radio-t.com/p/2018/09/22////podcast-616/ https://www.radio-t.com/p/2018/09/22/podcast-616/
https://radio-t.com/p/2018/09/22/podcast-616/?with_query=1 https://www.radio-t.com/p/2018/09/22/podcast-616/
`)

	mapper := UrlMapper{}
	err := mapper.LoadRules(rules)
	assert.NoError(t, err)

	// if url not matched mapper should return given url
	assert.Equal(t, "https://any.com/post/1/", mapper.URL("https://any.com/post/1/"))
	assert.Equal(t, "https://radio-t.co", mapper.URL("https://radio-t.co"))

	// check strict matching
	assert.Equal(t, "https://www.radio-t.com/p/2018/09/22/podcast-616/", mapper.URL("https://radio-t.com/p/2018/09/22////podcast-616/"))
	assert.Equal(t, "https://www.radio-t.com/p/2018/09/22/podcast-616/", mapper.URL("https://radio-t.com/p/2018/09/22/podcast-616/?with_query=1"))

	// check pattern matching (by prefix)
	assert.Equal(t, "https://www.radio-t.com/p/post/123/", mapper.URL("https://radio-t.com/p/post/123/"))
}

func TestUrlMapper_BadInput(t *testing.T) {
	mapper := UrlMapper{}
	assert.Error(t, mapper.LoadRules(strings.NewReader("https://radio-t.com ")))
	assert.Error(t, mapper.LoadRules(strings.NewReader("https://radio-t.com https://radio-t.com https://radio-t.com")))
	assert.Error(t, mapper.LoadRules(strings.NewReader("https://radio-t.com https://radio-t.com\n https://radio-t.com")))
	assert.Error(t, mapper.LoadRules(strings.NewReader("https://radio-t.com   \n https://radio-t.com https://radio-t.com")))
}
