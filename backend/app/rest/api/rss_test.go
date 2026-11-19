package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/rest"
	"github.com/umputun/remark42/backend/app/store"
)

// rssPubTime returns a second-aligned base timestamp and formats it the way the feed does, so
// comment pubDates are pinned rather than dependent on when in the second the test runs.
func rssPubTime() (base time.Time, pubDate string) {
	base = time.Now().Truncate(time.Second)
	return base, base.Format(time.RFC1123Z)
}

func TestServer_RssPost(t *testing.T) {
	ts, rst, teardown := startupT(t)
	defer teardown()

	base, pubDate := rssPubTime()

	c1 := store.Comment{
		ID:        "1234567890",
		Text:      "test 123",
		Timestamp: base,
		Locator:   store.Locator{URL: "https://radio-t.com/blah1", SiteID: "remark42"},
		User:      store.User{ID: "u1", Name: "developer one"},
	}
	id1, err := rst.DataService.Create(c1)
	require.NoError(t, err)
	assert.Equal(t, "1234567890", id1)

	res, code := get(t, ts.URL+"/api/v1/rss/post?site=remark42&url=https://radio-t.com/blah1")
	assert.Equal(t, http.StatusOK, code)
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

	_, code = get(t, ts.URL+"/api/v1/rss/post?site=remark42-bad&url=https://radio-t.com/blah1")
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestServer_RssSite(t *testing.T) {
	ts, rst, teardown := startupT(t)
	defer teardown()

	base, pubDate := rssPubTime()

	c1 := store.Comment{
		ID:        "comment-id-1",
		Text:      "test 123",
		Timestamp: base,
		Locator:   store.Locator{URL: "https://radio-t.com/blah10", SiteID: "remark42"},
		User:      store.User{ID: "u1", Name: "developer one"},
	}
	c2 := store.Comment{
		ID:        "comment-id-2",
		Text:      "xyz test",
		Timestamp: base.Add(time.Millisecond),
		Locator:   store.Locator{URL: "https://radio-t.com/blah11", SiteID: "remark42"},
		User:      store.User{ID: "u1", Name: "developer one"},
	}

	_, err := rst.DataService.Create(c1)
	require.NoError(t, err)
	_, err = rst.DataService.Create(c2)
	require.NoError(t, err)

	require.NoError(t, err)
	res, code := get(t, ts.URL+"/api/v1/rss/site?site=remark42")
	assert.Equal(t, http.StatusOK, code)
	t.Log(res)

	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
		  <channel>
		    <title>Remark42 comments</title>
		    <link>remark42</link>
		    <description>site comment for remark42</description>
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
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestServer_RssWithReply(t *testing.T) {
	ts, rst, teardown := startupT(t)
	defer teardown()

	base, pubDate := rssPubTime()

	c1 := store.Comment{
		ID:        "comment-id-1",
		Text:      "test 123",
		Timestamp: base,
		Locator:   store.Locator{URL: "https://radio-t.com/blah10", SiteID: "remark42"},
		User:      store.User{ID: "u1", Name: "developer one"},
	}
	c2 := store.Comment{
		ID:        "comment-id-2",
		ParentID:  "comment-id-1",
		Text:      "xyz test",
		Timestamp: base.Add(time.Millisecond),
		Locator:   store.Locator{URL: "https://radio-t.com/blah10", SiteID: "remark42"},
		User:      store.User{ID: "u1", Name: "developer one"},
	}

	_, err := rst.DataService.Create(c1)
	require.NoError(t, err)
	_, err = rst.DataService.Create(c2)
	require.NoError(t, err)

	res, code := get(t, ts.URL+"/api/v1/rss/post?site=remark42&url=https://radio-t.com/blah10")
	assert.Equal(t, http.StatusOK, code)
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
              <description>xyz test&lt;blockquote&gt;&lt;p&gt;test 123&lt;/p&gt;&lt;/blockquote&gt;</description>
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

	base, pubDate := rssPubTime()

	c1 := store.Comment{
		ID:        "comment-1",
		Text:      "c1",
		Timestamp: base,
		Locator:   store.Locator{URL: "https://radio-t.com/blah1", SiteID: "remark42"},
		User:      store.User{ID: "user1", Name: "user1"},
	}
	c2 := store.Comment{
		ID:        "comment-2",
		Text:      "reply to c1 from user2",
		ParentID:  "comment-1",
		Timestamp: base.Add(time.Millisecond),
		Locator:   store.Locator{URL: "https://radio-t.com/blah1", SiteID: "remark42"},
		User:      store.User{ID: "user2", Name: "user2"},
	}
	c3 := store.Comment{
		ID:        "comment-3",
		Text:      "reply to c1 from user3",
		ParentID:  "comment-1",
		Timestamp: base.Add(2 * time.Millisecond),
		Locator:   store.Locator{URL: "https://radio-t.com/blah1", SiteID: "remark42"},
		User:      store.User{ID: "user3", Name: "user3"},
	}
	c4 := store.Comment{
		ID:        "comment-4",
		Text:      "reply to c2 from developer one",
		ParentID:  "comment-2",
		Timestamp: base.Add(3 * time.Millisecond),
		Locator:   store.Locator{URL: "https://radio-t.com/blah1", SiteID: "remark42"},
		User:      store.User{ID: "dev", Name: "developer one"},
	}
	c5 := store.Comment{
		ID:        "comment-5",
		Text:      "developer one",
		Timestamp: base.Add(4 * time.Millisecond),
		Locator:   store.Locator{URL: "https://radio-t.com/blah1", SiteID: "remark42"},
		User:      store.User{ID: "dev", Name: "developer one"},
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
	res, code := get(t, ts.URL+"/api/v1/rss/reply?user=user1&site=remark42")
	assert.Equal(t, http.StatusOK, code)
	t.Log(res)
	expected := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
	    <channel>
	        <title>Remark42 comments</title>
	        <link>remark42</link>
	        <description>replies to user1</description>
	        <pubDate>%s</pubDate>
	        <item>
		      <title>user3 &gt; user1</title>
		      <link>https://radio-t.com/blah1#remark42__comment-comment-3</link>
		      <description>reply to c1 from user3&lt;blockquote&gt;&lt;p&gt;c1&lt;/p&gt;&lt;/blockquote&gt;</description>
		      <author>user3</author>
              <guid>comment-3</guid> 
		      <pubDate>%s</pubDate>
			</item>
			<item>
		      <title>user2 &gt; user1</title>
		      <link>https://radio-t.com/blah1#remark42__comment-comment-2</link>
		      <description>reply to c1 from user2&lt;blockquote&gt;&lt;p&gt;c1&lt;/p&gt;&lt;/blockquote&gt;</description>
		      <author>user2</author>
              <guid>comment-2</guid> 
		      <pubDate>%s</pubDate>
			</item>
	     </channel>
	</rss>`, pubDate, pubDate, pubDate)
	expected, res = cleanRssFormatting(expected, res)
	assert.Equal(t, expected, res)

	_, code = get(t, ts.URL+"/api/v1/rss/reply?user=user1&site=remark42-bad")
	assert.Equal(t, http.StatusBadRequest, code)
}

// clean formatting, i.e. multiple spaces, \t, \n
func cleanRssFormatting(expected, actual string) (cleanExp, cleanAct string) {
	reSpaces := regexp.MustCompile(`[\s\p{Zs}]{2,}`)

	expected = strings.ReplaceAll(expected, "\n", " ")
	expected = strings.ReplaceAll(expected, "\t", " ")
	expected = reSpaces.ReplaceAllString(expected, " ")

	actual = strings.ReplaceAll(actual, "\n", " ")
	actual = reSpaces.ReplaceAllString(actual, " ")
	return expected, actual
}

func TestServer_RssPostLimitsItems(t *testing.T) {
	ts, rst, teardown := startupT(t)
	defer teardown()

	base, _ := rssPubTime()
	for i := range maxRssItems + 5 {
		_, err := rst.DataService.Create(store.Comment{
			ID:        fmt.Sprintf("id-%02d", i),
			Text:      fmt.Sprintf("comment %d", i),
			Timestamp: base.Add(time.Duration(i) * time.Second),
			Locator:   store.Locator{URL: "https://radio-t.com/many", SiteID: "remark42"},
			User:      store.User{ID: "u1", Name: "developer one"},
		})
		require.NoError(t, err)
	}

	res, code := get(t, ts.URL+"/api/v1/rss/post?site=remark42&url=https://radio-t.com/many")
	require.Equal(t, http.StatusOK, code)
	assert.Equal(t, maxRssItems, strings.Count(res, "<item>"), "post feed has at most maxRssItems items")
}

// TestServer_RssResponses pins what every feed answers besides the feed itself: the content type on
// success and the status, details and code of the JSON error for an unknown site
func TestServer_RssResponses(t *testing.T) {
	ts, rst, teardown := startupT(t)
	defer teardown()

	_, err := rst.DataService.Create(store.Comment{ID: "c1", Text: "test 123", Timestamp: time.Now(),
		Locator: store.Locator{URL: "https://radio-t.com/blah1", SiteID: "remark42"}, User: store.User{ID: "u1", Name: "user one"}})
	require.NoError(t, err)

	tbl := []struct {
		name, path, badPath string
		details             string
		errCode             int
	}{
		{"post", "/api/v1/rss/post?site=remark42&url=https://radio-t.com/blah1", "/api/v1/rss/post?site=bad&url=https://radio-t.com/blah1",
			"can't find comments", rest.ErrPostNotFound},
		{"site", "/api/v1/rss/site?site=remark42", "/api/v1/rss/site?site=bad", "can't get last comments", rest.ErrSiteNotFound},
		{"reply", "/api/v1/rss/reply?user=u1&site=remark42", "/api/v1/rss/reply?user=u1&site=bad", "can't get replies", rest.ErrSiteNotFound},
	}
	client := http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Get(ts.URL + tt.path)
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, "application/xml; charset=utf-8", resp.Header.Get("Content-Type"))
			assert.Contains(t, string(body), "<title>Remark42 comments</title>")

			resp, err = client.Get(ts.URL + tt.badPath)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			j := struct {
				Details string `json:"details"`
				Code    int    `json:"code"`
			}{}
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&j))
			assert.Equal(t, tt.details, j.Details)
			assert.Equal(t, tt.errCode, j.Code)
		})
	}
}
