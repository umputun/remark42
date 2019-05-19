package api

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store"
)

func TestServer_RssPost(t *testing.T) {
	ts, rst, teardown := startupT(t)
	defer teardown()

	waitOnSecChange()

	c1 := store.Comment{
		ID:      "1234567890",
		Text:    "test 123",
		Locator: store.Locator{URL: "https://radio-t.com/blah1", SiteID: "radio-t"},
		User:    store.User{ID: "u1", Name: "developer one"},
	}
	id1, err := rst.DataService.Create(c1)
	require.NoError(t, err)
	assert.Equal(t, "1234567890", id1)
	pubDate := time.Now().Format(time.RFC1123Z)

	res, code := get(t, ts.URL+"/api/v1/rss/post?site=radio-t&url=https://radio-t.com/blah1")
	assert.Equal(t, 200, code)
	t.Log(res)

	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
        <channel>
            <title>Remark42 comments</title>
            <link>https://radio-t.com/blah1</link>
            <description>post comments for https://radio-t.com/blah1</description>
            <pubDate>%s</pubDate>
            <item>
		      <title>developer one</title>
		      <link>https://radio-t.com/blah1#remark42__comment-1234567890</link>
		      <description>test 123</description>
		      <author>developer one</author>
              <guid>1234567890</guid>
		      <pubDate>%s</pubDate>
            </item>
         </channel>
	</rss>`, pubDate, pubDate)

	expected, res = cleanRssFormatting(expected, res)
	assert.Equal(t, expected, res)

	_, code = get(t, ts.URL+"/api/v1/rss/post?site=radio-t-bad&url=https://radio-t.com/blah1")
	assert.Equal(t, 400, code)
}

func TestServer_RssSite(t *testing.T) {
	ts, rst, teardown := startupT(t)
	defer teardown()

	waitOnSecChange()

	pubDate := time.Now().Format(time.RFC1123Z)

	c1 := store.Comment{
		ID:      "comment-id-1",
		Text:    "test 123",
		Locator: store.Locator{URL: "https://radio-t.com/blah10", SiteID: "radio-t"},
		User:    store.User{ID: "u1", Name: "developer one"},
	}
	c2 := store.Comment{
		ID:      "comment-id-2",
		Text:    "xyz test",
		Locator: store.Locator{URL: "https://radio-t.com/blah11", SiteID: "radio-t"},
		User:    store.User{ID: "u1", Name: "developer one"},
	}

	_, err := rst.DataService.Create(c1)
	require.NoError(t, err)
	_, err = rst.DataService.Create(c2)
	require.NoError(t, err)

	require.NoError(t, err)
	res, code := get(t, ts.URL+"/api/v1/rss/site?site=radio-t")
	assert.Equal(t, 200, code)
	t.Log(res)

	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
		  <channel>
		    <title>Remark42 comments</title>
		    <link>radio-t</link>
		    <description>site comment for radio-t</description>
		    <pubDate>%s</pubDate>
		    <item>
		      <title>developer one</title>
		      <link>https://radio-t.com/blah11#remark42__comment-comment-id-2</link>
		      <description>xyz test</description>
		      <author>developer one</author>
              <guid>comment-id-2</guid>
		      <pubDate>%s</pubDate>
		    </item>
		    <item>
		      <title>developer one</title>
		      <link>https://radio-t.com/blah10#remark42__comment-comment-id-1</link>
		      <description>test 123</description>
		      <author>developer one</author>
              <guid>comment-id-1</guid>
		      <pubDate>%s</pubDate>
		    </item>
		  </channel>
		</rss>`, pubDate, pubDate, pubDate)

	expected, res = cleanRssFormatting(expected, res)
	assert.Equal(t, expected, res)

	_, code = get(t, ts.URL+"/api/v1/rss/site?site=bad-radio-t")
	assert.Equal(t, 400, code)
}

func TestServer_RssWithReply(t *testing.T) {
	ts, rst, teardown := startupT(t)
	defer teardown()

	waitOnSecChange()

	pubDate := time.Now().Format(time.RFC1123Z)

	c1 := store.Comment{
		ID:      "comment-id-1",
		Text:    "test 123",
		Locator: store.Locator{URL: "https://radio-t.com/blah10", SiteID: "radio-t"},
		User:    store.User{ID: "u1", Name: "developer one"},
	}
	c2 := store.Comment{
		ID:       "comment-id-2",
		ParentID: "comment-id-1",
		Text:     "xyz test",
		Locator:  store.Locator{URL: "https://radio-t.com/blah10", SiteID: "radio-t"},
		User:     store.User{ID: "u1", Name: "developer one"},
	}

	_, err := rst.DataService.Create(c1)
	require.NoError(t, err)
	_, err = rst.DataService.Create(c2)
	require.NoError(t, err)

	res, code := get(t, ts.URL+"/api/v1/rss/post?site=radio-t&url=https://radio-t.com/blah10")
	assert.Equal(t, 200, code)
	t.Log(res)

	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
		  <channel>
		    <title>Remark42 comments</title>
		    <link>https://radio-t.com/blah10</link>
		    <description>post comments for https://radio-t.com/blah10</description>
		    <pubDate>%s</pubDate>
		    <item>
		      <title>developer one &gt; developer one</title>
		      <link>https://radio-t.com/blah10#remark42__comment-comment-id-2</link>
		      <description>xyz test</description>
		      <author>developer one</author>
              <guid>comment-id-2</guid>
		      <pubDate>%s</pubDate>
		    </item>
		    <item>
		      <title>developer one</title>
		      <link>https://radio-t.com/blah10#remark42__comment-comment-id-1</link>
		      <description>test 123</description>
		      <author>developer one</author>
              <guid>comment-id-1</guid>
		      <pubDate>%s</pubDate>
		    </item>
		  </channel>
		</rss>`, pubDate, pubDate, pubDate)

	expected, res = cleanRssFormatting(expected, res)
	assert.Equal(t, expected, res)
}

func TestServer_RssReplies(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	waitOnSecChange()

	pubDate := time.Now().Format(time.RFC1123Z)

	c1 := store.Comment{
		ID:      "comment-1",
		Text:    "c1",
		Locator: store.Locator{URL: "https://radio-t.com/blah1", SiteID: "radio-t"},
		User:    store.User{ID: "user1", Name: "user1"},
	}
	c2 := store.Comment{
		ID:       "comment-2",
		Text:     "reply to c1 from user2",
		ParentID: "comment-1",
		Locator:  store.Locator{URL: "https://radio-t.com/blah1", SiteID: "radio-t"},
		User:     store.User{ID: "user2", Name: "user2"},
	}
	c3 := store.Comment{
		ID:       "comment-3",
		Text:     "reply to c1 from user3",
		ParentID: "comment-1",
		Locator:  store.Locator{URL: "https://radio-t.com/blah1", SiteID: "radio-t"},
		User:     store.User{ID: "user3", Name: "user3"},
	}
	c4 := store.Comment{
		ID:       "comment-4",
		Text:     "reply to c2 from developer one",
		ParentID: "comment-2",
		Locator:  store.Locator{URL: "https://radio-t.com/blah1", SiteID: "radio-t"},
		User:     store.User{ID: "dev", Name: "developer one"},
	}
	c5 := store.Comment{
		ID:      "comment-5",
		Text:    "developer one",
		Locator: store.Locator{URL: "https://radio-t.com/blah1", SiteID: "radio-t"},
		User:    store.User{ID: "dev", Name: "developer one"},
	}

	_, err := srv.DataService.Create(c1)
	require.NoError(t, err)
	_, err = srv.DataService.Create(c2)
	require.NoError(t, err)
	_, err = srv.DataService.Create(c3)
	require.NoError(t, err)
	_, err = srv.DataService.Create(c4)
	require.NoError(t, err)
	_, err = srv.DataService.Create(c5)
	require.NoError(t, err)

	// replies to c1 (user1). Must be [c3, c2]
	res, code := get(t, ts.URL+"/api/v1/rss/reply?user=user1&site=radio-t")
	assert.Equal(t, 200, code)
	t.Log(res)
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
	    <channel>
	        <title>Remark42 comments</title>
	        <link>radio-t</link>
	        <description>replies to user1</description>
	        <pubDate>%s</pubDate>
	        <item>
		      <title>user3 &gt; user1</title>
		      <link>https://radio-t.com/blah1#remark42__comment-comment-3</link>
		      <description>reply to c1 from user3</description>
		      <author>user3</author>
              <guid>comment-3</guid> 
		      <pubDate>%s</pubDate>
			</item>
			<item>
		      <title>user2 &gt; user1</title>
		      <link>https://radio-t.com/blah1#remark42__comment-comment-2</link>
		      <description>reply to c1 from user2</description>
		      <author>user2</author>
              <guid>comment-2</guid> 
		      <pubDate>%s</pubDate>
			</item>
	     </channel>
	</rss>`, pubDate, pubDate, pubDate)
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
