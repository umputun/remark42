package api

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/remark/backend/app/store"
)

func TestServer_RssPost(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts, srv)

	waitOnSecChange()

	c1 := store.Comment{
		Text:    "test 123",
		Locator: store.Locator{URL: "https://radio-t.com/blah1", SiteID: "radio-t"},
	}
	id1 := addComment(t, c1, ts)
	pubDate := time.Now().Format(time.RFC1123Z)

	res, code := get(t, ts.URL+"/api/v1/rss/post?site=radio-t&url=https://radio-t.com/blah1")
	assert.Equal(t, 200, code)
	t.Log(res)

	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
        <channel>
            <title>Remark42 comments</title>
            <link>https://radio-t.com/blah1</link>
            <description>comment updates</description>
            <pubDate>%s</pubDate>
            <item>
		      <title>developer one</title>
		      <link>https://radio-t.com/blah1#remark42__comment-%s</link>
		      <description>&lt;p&gt;test 123&lt;/p&gt;&#xA;</description>
		      <author>developer one</author>
		      <pubDate>%s</pubDate>
            </item>
         </channel>
	</rss>`, pubDate, id1, pubDate)

	expected, res = cleanRssFormatting(expected, res)
	assert.Equal(t, expected, res)

	_, code = get(t, ts.URL+"/api/v1/rss/post?site=radio-t-bad&url=https://radio-t.com/blah1")
	assert.Equal(t, 400, code)
}

func TestServer_RssSite(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts, srv)

	waitOnSecChange()

	pubDate := time.Now().Format(time.RFC1123Z)

	c1 := store.Comment{
		Text:    "test 123",
		Locator: store.Locator{URL: "https://radio-t.com/blah10", SiteID: "radio-t"},
	}
	c2 := store.Comment{
		Text:    "xyz test",
		Locator: store.Locator{URL: "https://radio-t.com/blah11", SiteID: "radio-t"},
	}
	id1 := addComment(t, c1, ts)
	id2 := addComment(t, c2, ts)

	res, code := get(t, ts.URL+"/api/v1/rss/site?site=radio-t")
	assert.Equal(t, 200, code)
	t.Log(res)

	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
		  <channel>
		    <title>Remark42 comments</title>
		    <link>radio-t</link>
		    <description>comment updates</description>
		    <pubDate>%s</pubDate>
		    <item>
		      <title>developer one</title>
		      <link>https://radio-t.com/blah11#remark42__comment-%s</link>
		      <description>&lt;p&gt;xyz test&lt;/p&gt;&#xA;</description>
		      <author>developer one</author>
		      <pubDate>%s</pubDate>
		    </item>
		    <item>
		      <title>developer one</title>
		      <link>https://radio-t.com/blah10#remark42__comment-%s</link>
		      <description>&lt;p&gt;test 123&lt;/p&gt;&#xA;</description>
		      <author>developer one</author>
		      <pubDate>%s</pubDate>
		    </item>
		  </channel>
		</rss>`, pubDate, id2, pubDate, id1, pubDate)

	expected, res = cleanRssFormatting(expected, res)
	assert.Equal(t, expected, res)

	_, code = get(t, ts.URL+"/api/v1/rss/site?site=bad-radio-t")
	assert.Equal(t, 400, code)
}

func TestServer_RssWithReply(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts, srv)

	waitOnSecChange()

	pubDate := time.Now().Format(time.RFC1123Z)

	c1 := store.Comment{
		Text:    "test 123",
		Locator: store.Locator{URL: "https://radio-t.com/blah10", SiteID: "radio-t"},
	}
	c2 := store.Comment{
		Text:    "xyz test",
		Locator: store.Locator{URL: "https://radio-t.com/blah10", SiteID: "radio-t"},
	}
	id1 := addComment(t, c1, ts)
	c2.ParentID = id1
	id2 := addComment(t, c2, ts)

	res, code := get(t, ts.URL+"/api/v1/rss/post?site=radio-t&url=https://radio-t.com/blah10")
	assert.Equal(t, 200, code)
	t.Log(res)

	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
		  <channel>
		    <title>Remark42 comments</title>
		    <link>https://radio-t.com/blah10</link>
		    <description>comment updates</description>
		    <pubDate>%s</pubDate>
		    <item>
		      <title>developer one &gt; developer one</title>
		      <link>https://radio-t.com/blah10#remark42__comment-%s</link>
		      <description>&lt;p&gt;xyz test&lt;/p&gt;&#xA;</description>
		      <author>developer one</author>
		      <pubDate>%s</pubDate>
		    </item>
		    <item>
		      <title>developer one</title>
		      <link>https://radio-t.com/blah10#remark42__comment-%s</link>
		      <description>&lt;p&gt;test 123&lt;/p&gt;&#xA;</description>
		      <author>developer one</author>
		      <pubDate>%s</pubDate>
		    </item>
		  </channel>
		</rss>`, pubDate, id2, pubDate, id1, pubDate)

	expected, res = cleanRssFormatting(expected, res)
	assert.Equal(t, expected, res)
}

func TestServer_RssReplies(t *testing.T) {
	srv, ts := prep(t)
	assert.NotNil(t, srv)
	defer cleanup(ts, srv)

	waitOnSecChange()

	pubDate := time.Now().Format(time.RFC1123Z)

	c1 := store.Comment{
		Text:    "c1",
		Locator: store.Locator{URL: "https://radio-t.com/blah1", SiteID: "radio-t"},
		User:    store.User{ID: "user1", Name: "user1"},
	}
	id1, err := srv.DataService.Create(c1)
	assert.Nil(t, err)
	c2 := store.Comment{
		Text:     "reply to c1 from user2",
		ParentID: id1,
		Locator:  store.Locator{URL: "https://radio-t.com/blah1", SiteID: "radio-t"},
		User:     store.User{ID: "user2", Name: "user2"},
	}
	id2, err := srv.DataService.Create(c2)
	assert.Nil(t, err)
	c3 := store.Comment{
		Text:     "reply to c1 from user3",
		ParentID: id1,
		Locator:  store.Locator{URL: "https://radio-t.com/blah1", SiteID: "radio-t"},
		User:     store.User{ID: "user3", Name: "user3"},
	}
	id3, err := srv.DataService.Create(c3)
	assert.Nil(t, err)
	c4 := store.Comment{
		Text:     "reply to c2 from developer one",
		ParentID: id2,
		Locator:  store.Locator{URL: "https://radio-t.com/blah1", SiteID: "radio-t"},
	}
	addComment(t, c4, ts)
	c5 := store.Comment{
		Text:    "developer one",
		Locator: store.Locator{URL: "https://radio-t.com/blah1", SiteID: "radio-t"},
	}
	addComment(t, c5, ts)

	// replies to c1 (user1). Must be [c3, c2]
	res, code := get(t, ts.URL+"/api/v1/rss/reply?user=user1&site=radio-t")
	assert.Equal(t, 200, code)
	t.Log(res)
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
	    <channel>
	        <title>Remark42 comments</title>
	        <link>radio-t</link>
	        <description>comment updates</description>
	        <pubDate>%s</pubDate>
	        <item>
		      <title>user3 &gt; user1</title>
		      <link>https://radio-t.com/blah1#remark42__comment-%s</link>
		      <description>reply to c1 from user3</description>
		      <author>user3</author>
		      <pubDate>%s</pubDate>
			</item>
			<item>
		      <title>user2 &gt; user1</title>
		      <link>https://radio-t.com/blah1#remark42__comment-%s</link>
		      <description>reply to c1 from user2</description>
		      <author>user2</author>
		      <pubDate>%s</pubDate>
			</item>
	     </channel>
	</rss>`, pubDate, id3, pubDate, id2, pubDate)
	expected, res = cleanRssFormatting(expected, res)
	assert.Equal(t, expected, res)

	_, code = get(t, ts.URL+"/api/v1/rss/reply?user=user1&site=radio-t-bad")
	assert.Equal(t, 400, code)
}

func waitOnSecChange() {
	for {
		if time.Now().Nanosecond() < 100000000 {
			break
		}
		time.Sleep(10 * time.Nanosecond)
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
