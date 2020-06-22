package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/service"
)

func TestMigrator_Import(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	r := strings.NewReader(`{"version":1} {"id":"2aa0478c-df1b-46b1-b561-03d507cf482c","pid":"","text":"<p>test test #1</p>",
"user":{"name":"developer one","id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com",
"admin":true,"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"remark42","url":"https://radio-t.com/blah1"},
"score":0,"votes":{},"time":"2018-04-30T01:37:00.849053725-05:00"}
	{"id":"83fd97fd-ff64-48d1-9fb7-ca7769c77037","pid":"p1","text":"<p>test test #2</p>","user":{"name":"developer one",
"id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,
"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"remark42","url":"https://radio-t.com/blah2"},"score":0,
"votes":{},"time":"2018-04-30T01:37:00.861387771-05:00"}`)

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=remark42&provider=native", r)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	assert.NoError(t, err)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "{\"status\":\"import request accepted\"}\n", string(b))

	waitForMigrationCompletion(t, ts)
}

func TestMigrator_ImportForm(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	r := strings.NewReader(`{"version":1} {"id":"2aa0478c-df1b-46b1-b561-03d507cf482c","pid":"","text":"<p>test test #1</p>",
"user":{"name":"developer one","id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com",
"admin":true,"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"remark42","url":"https://radio-t.com/blah1"},
"score":0,"votes":{},"time":"2018-04-30T01:37:00.849053725-05:00"}
	{"id":"83fd97fd-ff64-48d1-9fb7-ca7769c77037","pid":"p1","text":"<p>test test #2</p>","user":{"name":"developer one",
"id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,
"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"remark42","url":"https://radio-t.com/blah2"},"score":0,
"votes":{},"time":"2018-04-30T01:37:00.861387771-05:00"}`)

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("file", "import.json")
	require.NoError(t, err)
	_, err = io.Copy(fileWriter, r)
	require.NoError(t, err)
	contentType := bodyWriter.FormDataContentType()
	require.NoError(t, bodyWriter.Close())

	authts := strings.Replace(ts.URL, "http://", "http://admin:password@", 1)
	resp, err := http.Post(authts+"/api/v1/admin/import/form?site=remark42&provider=native", contentType, bodyBuf)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "{\"status\":\"import request accepted\"}\n", string(b))

	waitForMigrationCompletion(t, ts)
}

func TestMigrator_ImportFromWP(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	r := strings.NewReader(strings.Replace(xmlTestWP, "'", "`", -1))

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=remark42&provider=wordpress", r)
	assert.NoError(t, err)
	req.Header.Add("Content-Type", "application/xml; charset=utf-8")
	req.SetBasicAuth("admin", "password")
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, "{\"status\":\"import request accepted\"}\n", string(b))

	waitForMigrationCompletion(t, ts)
}

func TestMigrator_ImportRejected(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	r := strings.NewReader(`{"version":1} {"id":"2aa0478c-df1b-46b1-b561-03d507cf482c","pid":"","text":"<p>test test #1</p>",
"user":{"name":"developer one","id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com",
"admin":true,"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"remark42","url":"https://radio-t.com/blah1"},
"score":0,"votes":{},"time":"2018-04-30T01:37:00.849053725-05:00"}
	{"id":"83fd97fd-ff64-48d1-9fb7-ca7769c77037","pid":"p1","text":"<p>test test #2</p>","user":{"name":"developer one",
"id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,
"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"remark42","url":"https://radio-t.com/blah2"},"score":0,
"votes":{},"time":"2018-04-30T01:37:00.861387771-05:00"}`)

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=remark42&provider=native&secret=XYZ", r)
	assert.NoError(t, err)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMigrator_ImportDouble(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	tmpl := `{"id":"%d","pid":"","text":"<p>test test #1</p>","user":{"name":"developer one","id":"dev",
"picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,
"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"remark42","url":"https://radio-t.com/blah1"},"score":0,
"votes":{},"time":"2018-04-30T01:37:00.849053725-05:00"}`
	recs := []string{}
	for i := 0; i < 50; i++ {
		recs = append(recs, fmt.Sprintf(tmpl, i))
	}
	r := strings.NewReader(`{"version":1}` + strings.Join(recs, "\n")) // reader with 10k records
	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=remark42&provider=native", r)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	assert.NoError(t, err)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	client = &http.Client{Timeout: 5 * time.Second}
	req, err = http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=remark42&provider=native", r)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	assert.NoError(t, err)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	waitForMigrationCompletion(t, ts)
}

func TestMigrator_ImportWaitExpired(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	tmpl := `{"id":"%d","pid":"","text":"<p>test test #1</p>","user":{"name":"developer one","id":"dev",
"picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,
"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"remark42","url":"https://radio-t.com/blah1"},"score":0,
"votes":{},"time":"2018-04-30T01:37:00.849053725-05:00"}`
	nRecs := 50
	recs := make([]string, 0, nRecs)
	for i := 0; i < nRecs; i++ {
		recs = append(recs, fmt.Sprintf(tmpl, i))
	}
	r := strings.NewReader(`{"version":1}` + strings.Join(recs, "\n")) // reader with `nRecs` records
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=remark42&provider=native", r)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	require.NoError(t, err)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	client = &http.Client{Timeout: 5 * time.Second}
	req, err = http.NewRequest("GET", ts.URL+"/api/v1/admin/wait?site=remark42&timeout=5ms", nil)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	assert.NoError(t, err)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)

	waitForMigrationCompletion(t, ts)
}

func TestMigrator_Export(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	r := strings.NewReader(`{"version":1} {"id":"2aa0478c-df1b-46b1-b561-03d507cf482c","pid":"","text":"<p>test test #1</p>",
"user":{"name":"developer one","id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com",
"admin":true,"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"remark42","url":"https://radio-t.com/blah1"},
"score":0,"votes":{},"time":"2018-04-30T01:37:00.849053725-05:00"}
	{"id":"83fd97fd-ff64-48d1-9fb7-ca7769c77037","pid":"p1","text":"<p>test test #2</p>","user":{"name":"developer one",
"id":"dev","picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,
"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"remark42","url":"https://radio-t.com/blah2"},"score":0,
"votes":{},"time":"2018-04-30T01:37:00.861387771-05:00"}`)

	// import comments first
	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=remark42&provider=native", r)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	waitForMigrationCompletion(t, ts)

	// check file mode
	req, err = http.NewRequest("GET", ts.URL+"/api/v1/admin/export?mode=file&site=remark42", nil)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))

	ungzReader, err := gzip.NewReader(resp.Body)
	assert.NoError(t, err)
	ungzBody, err := ioutil.ReadAll(ungzReader)
	assert.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 3, strings.Count(string(ungzBody), "\n"))
	assert.Equal(t, 2, strings.Count(string(ungzBody), "\"text\""))
	t.Logf("%s", string(ungzBody))

	// check stream mode
	req, err = http.NewRequest("GET", ts.URL+"/api/v1/admin/export?mode=stream&site=remark42", nil)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 3, strings.Count(string(body), "\n"))
	assert.Equal(t, 2, strings.Count(string(body), "\"text\""))
	t.Logf("%s", string(body))

	req, err = http.NewRequest("GET", ts.URL+"/api/v1/admin/export?site=remark42", nil)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMigrator_Remap(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	// create 2 comments in https://remark42.com/demo/
	c1 := store.Comment{Text: "first comment", Timestamp: time.Now(),
		Locator: store.Locator{SiteID: "remark42", URL: "https://remark42.com/demo/"}, User: store.User{ID: "u1"}}
	_, err := srv.DataService.Create(c1)
	require.NoError(t, err)
	c2 := store.Comment{Text: "second comment", Timestamp: time.Now(),
		Locator: store.Locator{SiteID: "remark42", URL: "https://remark42.com/demo/"}, User: store.User{ID: "u2"}}
	_, err = srv.DataService.Create(c2)
	require.NoError(t, err)

	// create 1 comment in https://remark42.com/demo-another/
	c3 := store.Comment{Text: "third comment", Timestamp: time.Now(),
		Locator: store.Locator{SiteID: "remark42", URL: "https://remark42.com/demo-another/"}, User: store.User{ID: "u3"}}
	_, err = srv.DataService.Create(c3)
	require.NoError(t, err)

	// set url https://remark42.com/demo-another/ to be readonly
	err = srv.DataService.SetMetas("remark42", []service.UserMetaData{}, []service.PostMetaData{{
		URL:      "https://remark42.com/demo-another/",
		ReadOnly: true,
	}})
	require.NoError(t, err)

	// check that comments created as expected
	res, code := get(t, ts.URL+"/api/v1/find?site=remark42&url=https://remark42.com/demo/")
	require.Equal(t, 200, code)
	comments := commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	require.NoError(t, err)
	require.Equal(t, 2, comments.Info.Count)
	require.False(t, comments.Info.ReadOnly)

	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://remark42.com/demo-another/")
	require.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	require.NoError(t, err)
	require.Equal(t, 1, comments.Info.Count)
	require.True(t, comments.Info.ReadOnly)

	// we want remap urls to another domain - www.remark42.com
	rules := "https://remark42.com/* https://www.remark42.com/*"
	resp, err := post(t, ts.URL+"/api/v1/admin/remap?site=remark42", rules) // auth as admin
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	waitForMigrationCompletion(t, ts)

	// after remap finished we should find comments from new urls
	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://www.remark42.com/demo/")
	require.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	require.NoError(t, err)
	require.Equal(t, 2, comments.Info.Count)
	require.False(t, comments.Info.ReadOnly)

	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://www.remark42.com/demo-another/")
	require.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	require.NoError(t, err)
	require.Equal(t, 1, comments.Info.Count)
	require.True(t, comments.Info.ReadOnly)

	// should find nothing from previous url
	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://remark42.com/demo/")
	require.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	require.NoError(t, err)
	require.Equal(t, 0, comments.Info.Count)

	res, code = get(t, ts.URL+"/api/v1/find?site=remark42&url=https://remark42.com/demo-another/")
	require.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	require.NoError(t, err)
	require.Equal(t, 0, comments.Info.Count)
}

func TestMigrator_RemapReject(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	// without admin credentials
	client := &http.Client{Timeout: 1 * time.Second}
	rules := strings.NewReader(`https://remark42.com/* https://www.remark42.com/*`)
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/remap?site=remark42", rules)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func waitForMigrationCompletion(t *testing.T, ts *httptest.Server) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", ts.URL+"/api/v1/admin/wait?site=remark42", nil)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, "{\"site_id\":\"remark42\",\"status\":\"completed\"}\n", string(b))
}

var xmlTestWP = `
<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0"
	xmlns:excerpt="http://wordpress.org/export/1.2/excerpt/"
	xmlns:content="http://purl.org/rss/1.0/modules/content/"
	xmlns:wfw="http://wellformedweb.org/CommentAPI/"
	xmlns:dc="http://purl.org/dc/elements/1.1/"
	xmlns:wp="http://wordpress.org/export/1.2/"
>

<channel>
	<title>Real Men Wear Dress.es</title>
	<link>https://realmenweardress.es</link>
	<description>SuperAdmin&#039;s gaming and technological musings</description>
	<pubDate>Mon, 23 Jul 2018 10:21:47 +0000</pubDate>
	<language>en-US</language>
	<wp:wxr_version>1.2</wp:wxr_version>
	<wp:base_site_url>https://realmenweardress.es</wp:base_site_url>
	<wp:base_blog_url>https://realmenweardress.es</wp:base_blog_url>

	<wp:author><wp:author_id>2</wp:author_id><wp:author_login><![CDATA[SuperAdmin]]></wp:author_login><wp:author_email><![CDATA[superadmin@super.eu]]></wp:author_email><wp:author_display_name><![CDATA[SuperAdmin]]></wp:author_display_name><wp:author_first_name><![CDATA[SuperAdmin]]></wp:author_first_name><wp:author_last_name><![CDATA[superadmin]]></wp:author_last_name></wp:author>
	<wp:author><wp:author_id>1</wp:author_id><wp:author_login><![CDATA[admin]]></wp:author_login><wp:author_email><![CDATA[superadmin@superadmin.co.uk]]></wp:author_email><wp:author_display_name><![CDATA[admin]]></wp:author_display_name><wp:author_first_name><![CDATA[]]></wp:author_first_name><wp:author_last_name><![CDATA[]]></wp:author_last_name></wp:author>

	<wp:category>
		<wp:term_id>25</wp:term_id>
		<wp:category_nicename><![CDATA[cataclysm]]></wp:category_nicename>
		<wp:category_parent><![CDATA[]]></wp:category_parent>
		<wp:cat_name><![CDATA[Cataclysm]]></wp:cat_name>
	</wp:category>

	<wp:tag>
		<wp:term_id>39</wp:term_id>
		<wp:tag_slug><![CDATA[addons]]></wp:tag_slug>
		<wp:tag_name><![CDATA[addons]]></wp:tag_name>
	</wp:tag>

	<generator>https://wordpress.org/?v=4.8.1</generator>

	<item>
		<title>Post without comments</title>
		<link>https://realmenweardress.es/2010/06/hello-world/screenshot_013110_200413/</link>
		<pubDate>Sat, 19 Jun 2010 08:34:13 +0000</pubDate>
		<dc:creator><![CDATA[admin]]></dc:creator>
		<guid isPermaLink="false">http://realmenweardress.es/wp-content/uploads/2010/06/ScreenShot_013110_200413.jpeg</guid>
		<description></description>
		<content:encoded><![CDATA[So you can actually fly into the well it appears and if your lucky you stay mounted. I imagine it terrifies the poor rats.]]></content:encoded>
		<excerpt:encoded><![CDATA[]]></excerpt:encoded>
		<wp:post_id>6</wp:post_id>
		<wp:post_date><![CDATA[2010-06-19 08:34:13]]></wp:post_date>
		<wp:post_date_gmt><![CDATA[2010-06-19 08:34:13]]></wp:post_date_gmt>
		<wp:comment_status><![CDATA[open]]></wp:comment_status>
		<wp:ping_status><![CDATA[open]]></wp:ping_status>
		<wp:post_name><![CDATA[screenshot_013110_200413]]></wp:post_name>
		<wp:status><![CDATA[inherit]]></wp:status>
		<wp:post_parent>1</wp:post_parent>
		<wp:menu_order>0</wp:menu_order>
		<wp:post_type><![CDATA[attachment]]></wp:post_type>
		<wp:post_password><![CDATA[]]></wp:post_password>
		<wp:is_sticky>0</wp:is_sticky>
		<wp:attachment_url><![CDATA[https://realmenweardress.es/wp-content/uploads/2010/06/ScreenShot_013110_200413-e1277214413194.jpeg]]></wp:attachment_url>
		<wp:postmeta>
			<wp:meta_key><![CDATA[_wp_attached_file]]></wp:meta_key>
			<wp:meta_value><![CDATA[2010/06/ScreenShot_013110_200413-e1277214413194.jpeg]]></wp:meta_value>
		</wp:postmeta>
	</item>
	<item>
		<title>Post with comments. One is not approved</title>
		<link>https://realmenweardress.es/2010/07/do-you-rp/</link>
		<pubDate>Mon, 19 Jul 2010 14:24:22 +0000</pubDate>
		<dc:creator><![CDATA[SuperAdmin]]></dc:creator>
		<guid isPermaLink="false">http://realmenweardress.es/?p=100</guid>
		<description></description>
		<content:encoded><![CDATA[<a href="http://realmenweardress.es/wp-content/uploads/2010/07/ScreenShot_071410_230307-e1279546180886.jpeg"><img class="size-thumbnail wp-image-102 alignleft" title="I need to stand on things else I can't reach" src="http://realmenweardress.es/wp-content/uploads/2010/07/ScreenShot_071410_230307-e1279546270587-120x120.jpg" alt="I need to stand on things else I can't reach" width="120" height="120" /></a>Meet Grokknomel?]]></content:encoded>
		<excerpt:encoded><![CDATA[]]></excerpt:encoded>
		<wp:post_id>100</wp:post_id>
		<wp:post_date><![CDATA[2010-07-19 14:24:22]]></wp:post_date>
		<wp:post_date_gmt><![CDATA[2010-07-19 14:24:22]]></wp:post_date_gmt>
		<wp:comment_status><![CDATA[open]]></wp:comment_status>
		<wp:ping_status><![CDATA[open]]></wp:ping_status>
		<wp:post_name><![CDATA[do-you-rp]]></wp:post_name>
		<wp:status><![CDATA[publish]]></wp:status>
		<wp:post_parent>0</wp:post_parent>
		<wp:menu_order>0</wp:menu_order>
		<wp:post_type><![CDATA[post]]></wp:post_type>
		<wp:post_password><![CDATA[]]></wp:post_password>
		<wp:is_sticky>0</wp:is_sticky>
		<category domain="post_tag" nicename="alts"><![CDATA[alts]]></category>
		<category domain="post_tag" nicename="role-playing"><![CDATA[role playing]]></category>
		<category domain="category" nicename="stuff"><![CDATA[Stuff]]></category>
		<category domain="post_tag" nicename="weird-in-a-cant-quite-help-myself-way"><![CDATA[weird in a can't quite help myself way]]></category>
		<wp:postmeta>
			<wp:meta_key><![CDATA[_edit_last]]></wp:meta_key>
			<wp:meta_value><![CDATA[2]]></wp:meta_value>
		</wp:postmeta>
		<wp:comment>
			<wp:comment_id>8</wp:comment_id>
			<wp:comment_author><![CDATA[SuperUser1]]></wp:comment_author>
			<wp:comment_author_email><![CDATA[superuser1@aol.com]]></wp:comment_author_email>
			<wp:comment_author_url>http://superuser1.blogspot.com</wp:comment_author_url>
			<wp:comment_author_IP><![CDATA[79.141.141.73]]></wp:comment_author_IP>
			<wp:comment_date><![CDATA[2010-07-20 12:08:08]]></wp:comment_date>
			<wp:comment_date_gmt><![CDATA[2010-07-20 12:08:08]]></wp:comment_date_gmt>
			<wp:comment_content><![CDATA[I do catch myself]]></wp:comment_content>
			<wp:comment_approved><![CDATA[1]]></wp:comment_approved>
			<wp:comment_type><![CDATA[]]></wp:comment_type>
			<wp:comment_parent>0</wp:comment_parent>
			<wp:comment_user_id>0</wp:comment_user_id>
		</wp:comment>
		<wp:comment>
			<wp:comment_id>9</wp:comment_id>
			<wp:comment_author><![CDATA[SuperUser2]]></wp:comment_author>
			<wp:comment_author_email><![CDATA[superuser2@gmail.com]]></wp:comment_author_email>
			<wp:comment_author_url>http://thewowstorm.wordpress.com</wp:comment_author_url>
			<wp:comment_author_IP><![CDATA[97.36.113.1]]></wp:comment_author_IP>
			<wp:comment_date><![CDATA[2010-07-20 13:09:25]]></wp:comment_date>
			<wp:comment_date_gmt><![CDATA[2010-07-20 13:09:25]]></wp:comment_date_gmt>
			<wp:comment_content><![CDATA[I think it us inherent in the game to start seeing your character as a personality]]></wp:comment_content>
			<wp:comment_approved><![CDATA[0]]></wp:comment_approved>
			<wp:comment_type><![CDATA[]]></wp:comment_type>
			<wp:comment_parent>0</wp:comment_parent>
			<wp:comment_user_id>0</wp:comment_user_id>
		</wp:comment>
		<wp:comment>
			<wp:comment_id>13</wp:comment_id>
			<wp:comment_author><![CDATA[Wednesday Reading &laquo; Cynwise&#039;s Battlefield Manual]]></wp:comment_author>
			<wp:comment_author_email><![CDATA[]]></wp:comment_author_email>
			<wp:comment_author_url>http://cynwise.wordpress.com/2010/07/21/wednesday-reading-8/</wp:comment_author_url>
			<wp:comment_author_IP><![CDATA[74.200.244.101]]></wp:comment_author_IP>
			<wp:comment_date><![CDATA[2010-07-21 14:02:08]]></wp:comment_date>
			<wp:comment_date_gmt><![CDATA[2010-07-21 14:02:08]]></wp:comment_date_gmt>
			<wp:comment_content><![CDATA[[...] I know I&#8217;m a bit loony with my attachment to my bankers.  I&#8217;m glad I&#8217;m not the only one. [...]]]></wp:comment_content>
			<wp:comment_approved><![CDATA[1]]></wp:comment_approved>
			<wp:comment_type><![CDATA[pingback]]></wp:comment_type>
			<wp:comment_parent>0</wp:comment_parent>
			<wp:comment_user_id>0</wp:comment_user_id>
		</wp:comment>
		<wp:comment>
			<wp:comment_id>14</wp:comment_id>
			<wp:comment_author><![CDATA[SuperUser3]]></wp:comment_author>
			<wp:comment_author_email><![CDATA[blablah@gmail.com]]></wp:comment_author_email>
			<wp:comment_author_url>http://realmenweardress.es</wp:comment_author_url>
			<wp:comment_author_IP><![CDATA[128.243.253.117]]></wp:comment_author_IP>
			<wp:comment_date><![CDATA[2010-08-18 15:19:14]]></wp:comment_date>
			<wp:comment_date_gmt><![CDATA[2010-08-18 15:19:14]]></wp:comment_date_gmt>
			<wp:comment_content><![CDATA[Looks like http://releases.rancher.com/os/latest is no longer hosted - installs using this 'base-url' are failing.

I switched to Github with success:

'''
set base-url https://github.com/rancher/os/releases/download/v1.1.1-rc1
'''

Thanks for the article!]]></wp:comment_content>
			<wp:comment_approved><![CDATA[1]]></wp:comment_approved>
			<wp:comment_type><![CDATA[]]></wp:comment_type>
			<wp:comment_parent>13</wp:comment_parent>
			<wp:comment_user_id>2</wp:comment_user_id>
		</wp:comment>
	</item>
	</channel>
</rss>
`
