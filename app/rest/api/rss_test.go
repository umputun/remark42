package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServer_Rss(t *testing.T) {
	srv, port := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(srv)

	// add one more comment
	r := strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`)
	resp, err := http.Post(fmt.Sprintf("http://dev:password@127.0.0.1:%d/api/v1/comment", port), "application/json", r)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	pubDate := time.Now().Format(time.RFC1123Z)

	// get created comment by id
	res, code := get(t, fmt.Sprintf("http://dev:password@127.0.0.1:%d/api/v1/rss/post?site=radio-t&url=https://radio-t.com/blah1", port))
	assert.Equal(t, 200, code)

	assert.Nil(t, err)
	t.Log(res)

	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
        <channel>
            <title>Remark42 comments</title>
            <link>https://radio-t.com/blah1</link>
            <description>comment updates</description>
            <pubDate>%s</pubDate>
            <item>
		      <title>developer one</title>
		      <link>https://radio-t.com/blah1</link>
		      <description>&lt;p&gt;test 123&lt;/p&gt;&#xA;</description>
		      <author>developer one</author>
		      <pubDate>%s</pubDate>
            </item>
         </channel>
	</rss>`, pubDate, pubDate)

	// clean formatting, i.e. multiple spaces, \t, \n
	expected = strings.Replace(expected, "\n", " ", -1)
	expected = strings.Replace(expected, "\t", " ", -1)
	res = strings.Replace(res, "\n", " ", -1)
	reSpaces := regexp.MustCompile(`[\s\p{Zs}]{2,}`)
	expected = reSpaces.ReplaceAllString(expected, " ")
	res = reSpaces.ReplaceAllString(res, " ")

	assert.Equal(t, expected, res)

}
