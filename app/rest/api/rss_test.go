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

func TestServer_RssPost(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	waitOnMinChange()

	// add one more comment
	r := strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "radio-t"}}`)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/comment", r)
	assert.Nil(t, err)
	withBasicAuth(req, "dev", "password")

	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	pubDate := time.Now().Format(time.RFC1123Z)

	res, code := get(t, ts.URL+"/api/v1/rss/post?site=radio-t&url=https://radio-t.com/blah1")
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

	expected, res = cleanRssFormatting(expected, res)
	assert.Equal(t, expected, res)
}

func TestServer_RssSite(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts)

	waitOnMinChange()

	pubDate := time.Now().Format(time.RFC1123Z)

	client := &http.Client{Timeout: 5 * time.Second}

	r := strings.NewReader(`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah10", "site": "radio-t"}}`)
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/comment", r)
	assert.Nil(t, err)
	withBasicAuth(req, "dev", "password")
	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	r = strings.NewReader(`{"text": "xyz test", "locator":{"url": "https://radio-t.com/blah11", "site": "radio-t"}}`)
	req, err = http.NewRequest("POST", ts.URL+"/api/v1/comment", r)
	assert.Nil(t, err)
	withBasicAuth(req, "dev", "password")
	resp, err = client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	res, code := get(t, ts.URL+"/api/v1/rss/site?site=radio-t")
	assert.Equal(t, 200, code)

	assert.Nil(t, err)
	t.Log(res)

	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
		  <channel>
		    <title>Remark42 comments</title>
		    <link>radio-t</link>
		    <description>comment updates</description>
		    <pubDate>%s</pubDate>
		    <item>
		      <title>developer one</title>
		      <link>https://radio-t.com/blah11</link>
		      <description>&lt;p&gt;xyz test&lt;/p&gt;&#xA;</description>
		      <author>developer one</author>
		      <pubDate>%s</pubDate>
		    </item>
		    <item>
		      <title>developer one</title>
		      <link>https://radio-t.com/blah10</link>
		      <description>&lt;p&gt;test 123&lt;/p&gt;&#xA;</description>
		      <author>developer one</author>
		      <pubDate>%s</pubDate>
		    </item>
		  </channel>
		</rss>`, pubDate, pubDate, pubDate)

	expected, res = cleanRssFormatting(expected, res)
	assert.Equal(t, expected, res)
}

func waitOnMinChange() {
	if time.Now().Second() == 59 {
		time.Sleep(1001 * time.Millisecond)
	}
}

// clean formatting, i.e. multiple spaces, \t, \n
func cleanRssFormatting(expected, actual string) (string, string) {
	reSpaces := regexp.MustCompile(`[\s\p{Zs}]{2,}`)

	expected = strings.Replace(expected, "\n", " ", -1)
	expected = strings.Replace(expected, "\t", " ", -1)
	expected = reSpaces.ReplaceAllString(expected, " ")

	actual = strings.Replace(actual, "\n", " ", -1)
	actual = reSpaces.ReplaceAllString(actual, " ")
	return expected, actual
}
