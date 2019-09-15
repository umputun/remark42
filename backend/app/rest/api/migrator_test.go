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
	assert.Nil(t, err)
	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	b, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "{\"status\":\"import request accepted\"}\n", string(b))

	waitForImportCompletion(t, ts)
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
	assert.Nil(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	b, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "{\"status\":\"import request accepted\"}\n", string(b))

	waitForImportCompletion(t, ts)
}

func TestMigrator_ImportFromWP(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	r := strings.NewReader(strings.Replace(xmlTestWP, "'", "`", -1))

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=remark42&provider=wordpress", r)
	assert.Nil(t, err)
	req.Header.Add("Content-Type", "application/xml; charset=utf-8")
	req.SetBasicAuth("admin", "password")
	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	b, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "{\"status\":\"import request accepted\"}\n", string(b))

	waitForImportCompletion(t, ts)
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
	assert.Nil(t, err)
	resp, err := client.Do(req)
	assert.Nil(t, err)
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
	for i := 0; i < 150; i++ {
		recs = append(recs, fmt.Sprintf(tmpl, i))
	}
	r := strings.NewReader(`{"version":1}` + strings.Join(recs, "\n")) // reader with 10k records
	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=remark42&provider=native", r)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	assert.Nil(t, err)
	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	client = &http.Client{Timeout: 1 * time.Second}
	req, err = http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=remark42&provider=native", r)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	assert.Nil(t, err)
	resp, err = client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	waitForImportCompletion(t, ts)
}

func TestMigrator_ImportWaitExpired(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	tmpl := `{"id":"%d","pid":"","text":"<p>test test #1</p>","user":{"name":"developer one","id":"dev",
"picture":"/api/v1/avatar/remark.image","profile":"https://remark42.com","admin":true,
"ip":"ae12fe3b5f129b5cc4cdd2b136b7b7947c4d2741"},"locator":{"site":"remark42","url":"https://radio-t.com/blah1"},"score":0,
"votes":{},"time":"2018-04-30T01:37:00.849053725-05:00"}`
	recs := []string{}
	for i := 0; i < 150; i++ {
		recs = append(recs, fmt.Sprintf(tmpl, i))
	}
	r := strings.NewReader(`{"version":1}` + strings.Join(recs, "\n")) // reader with 10k records
	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=remark42&provider=native", r)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	require.Nil(t, err)
	resp, err := client.Do(req)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	client = &http.Client{Timeout: 10 * time.Second}
	req, err = http.NewRequest("GET", ts.URL+"/api/v1/admin/import/wait?site=remark42&timeout=100ms", nil)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	assert.NoError(t, err)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)

	waitForImportCompletion(t, ts)
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
	require.Nil(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	waitForImportCompletion(t, ts)

	// check file mode
	req, err = http.NewRequest("GET", ts.URL+"/api/v1/admin/export?mode=file&site=remark42", nil)
	require.Nil(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err = client.Do(req)
	require.Nil(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))

	ungzReader, err := gzip.NewReader(resp.Body)
	assert.NoError(t, err)
	ungzBody, err := ioutil.ReadAll(ungzReader)
	assert.NoError(t, err)
	assert.Equal(t, 3, strings.Count(string(ungzBody), "\n"))
	assert.Equal(t, 2, strings.Count(string(ungzBody), "\"text\""))
	t.Logf("%s", string(ungzBody))

	// check stream mode
	req, err = http.NewRequest("GET", ts.URL+"/api/v1/admin/export?mode=stream&site=remark42", nil)
	require.Nil(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err = client.Do(req)
	require.Nil(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, 3, strings.Count(string(body), "\n"))
	assert.Equal(t, 2, strings.Count(string(body), "\"text\""))
	t.Logf("%s", string(body))

	req, err = http.NewRequest("GET", ts.URL+"/api/v1/admin/export?site=remark42", nil)
	require.Nil(t, err)
	resp, err = client.Do(req)
	require.Nil(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMigrator_Convert(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	// test case
	// we have several comments from one site `radio-t` belong to two urls.
	// one url is set readonly.
	//[
	//	{
	//		"url": "https://radio-t.com/demo/",
	//		"count": 5
	//	},
	//	{
	//		"url": "https://radio-t.com/demo-another/",   - readonly!
	//		"count": 1
	//	}
	//]
	// import test case first
	s := `{"version":1,"users":[{"id":"blocked_user","blocked":{"status":true,"until":"2019-09-21T07:18:32.2346858-05:00"},"verified":false},{"id":"verified_user","blocked":{"status":false,"until":"0001-01-01T00:00:00Z"},"verified":true}],"posts":[{"url":"https://radio-t.com/demo-another/","read_only":true}]}
{"id":"25a18d59-aee9-45ab-86f5-c3fa31ef22c9","pid":"","text":"<p>comment to another post</p>\n","orig":"comment to another post","user":{"name":"admin","id":"admin","picture":"","ip":"1539deba4a54fc7862f0adf8d27192b19a27b1ed","admin":true},"locator":{"site":"radio-t","url":"https://radio-t.com/demo-another/"},"score":0,"vote":0,"time":"2019-09-14T07:26:23.4121277-05:00"}
{"id":"b814a90b-5b60-4e2b-b6e9-7058266c7706","pid":"","text":"<p>first comment from dev_user</p>\n","orig":"first comment from dev_user","user":{"name":"dev_user","id":"dev_user","picture":"http://127.0.0.1:8080/api/v1/avatar/ccfa2abd01667605b4e1fc4fcb91b1e1af323240.image","ip":"1539deba4a54fc7862f0adf8d27192b19a27b1ed","admin":true,"site_id":"radio-t"},"locator":{"site":"radio-t","url":"https://radio-t.com/demo/"},"score":1,"voted_ips":{"1539deba4a54fc7862f0adf8d27192b19a27b1ed":{"Timestamp":"2019-09-14T07:16:57.9319874-05:00","Value":true}},"vote":0,"time":"2019-09-14T07:16:18.0986736-05:00","title":"radio-t demo page"}
{"id":"145e3285-5dfd-4a4c-b8b0-3c6b5164473c","pid":"b814a90b-5b60-4e2b-b6e9-7058266c7706","text":"<p>reply to first message from any_user</p>\n","orig":"reply to first message from any_user","user":{"name":"any_user","id":"any_user","picture":"http://127.0.0.1:8080/api/v1/avatar/05ac5abbad12297e7a3578106fc0306f4fd73171.image","ip":"1539deba4a54fc7862f0adf8d27192b19a27b1ed","admin":false,"site_id":"radio-t"},"locator":{"site":"radio-t","url":"https://radio-t.com/demo/"},"score":0,"vote":0,"time":"2019-09-14T07:16:55.2362843-05:00","title":"radio-t demo page"}
{"id":"9beeb568-52b2-466d-b012-cd0d4dcdb854","pid":"","text":"<p>I want to be verified</p>\n","orig":"I want to be verified","user":{"name":"verified_user","id":"verified_user","picture":"http://127.0.0.1:8080/api/v1/avatar/7be676cbf4b5d7c0ae4da5f8143de927b12cfb42.image","ip":"1539deba4a54fc7862f0adf8d27192b19a27b1ed","admin":false,"verified":true,"site_id":"radio-t"},"locator":{"site":"radio-t","url":"https://radio-t.com/demo/"},"score":1,"voted_ips":{"1539deba4a54fc7862f0adf8d27192b19a27b1ed":{"Timestamp":"2019-09-14T07:22:01.0384052-05:00","Value":true}},"vote":0,"time":"2019-09-14T07:17:26.1825625-05:00","title":"radio-t demo page"}
{"id":"28e3b25a-d13b-4c0e-9179-5de9aad4a196","pid":"","text":"<p>I want to be blocked</p>\n","orig":"I want to be blocked","user":{"name":"blocked_user","id":"blocked_user","picture":"http://127.0.0.1:8080/api/v1/avatar/b4570b63a82ff5b5e188c9cb1820362ec13ad361.image","ip":"1539deba4a54fc7862f0adf8d27192b19a27b1ed","admin":false,"block":true,"site_id":"radio-t"},"locator":{"site":"radio-t","url":"https://radio-t.com/demo/"},"score":0,"vote":0,"time":"2019-09-14T07:18:05.8465267-05:00","title":"radio-t demo page"}
{"id":"09328137-9ac6-4388-ab75-e50113874f45","pid":"145e3285-5dfd-4a4c-b8b0-3c6b5164473c","text":"<p>reply from admin</p>\n","orig":"reply from admin","user":{"name":"dev_user","id":"dev_user","picture":"http://127.0.0.1:8080/api/v1/avatar/ccfa2abd01667605b4e1fc4fcb91b1e1af323240.image","ip":"1539deba4a54fc7862f0adf8d27192b19a27b1ed","admin":true,"site_id":"radio-t"},"locator":{"site":"radio-t","url":"https://radio-t.com/demo/"},"score":0,"vote":0,"time":"2019-09-14T07:24:17.5763304-05:00","title":"radio-t demo page"}`
	r := strings.NewReader(s)

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=radio-t&provider=native", r)
	require.Nil(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	waitForImportCompletion(t, ts)

	// import finished
	// check that comments imported as expected
	res, code := get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/demo/")
	require.Equal(t, 200, code)
	comments := commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	require.Nil(t, err)
	require.Equal(t, 5, comments.Info.Count)

	res, code = get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/demo-another/")
	require.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	require.Nil(t, err)
	require.Equal(t, 1, comments.Info.Count)

	// we want remap urls to another domain - www.radio-t.com
	rules := strings.NewReader(`https://radio-t.com/* https://www.radio-t.com/*
https://radio-t.com/demo-another/ https://www.radio-t.com/demo-another/`)
	req, err = http.NewRequest("POST", ts.URL+"/api/v1/admin/convert?site=radio-t", rules)
	require.Nil(t, err)
	req.SetBasicAuth("admin", "password")
	resp, err = client.Do(req)
	require.Nil(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	waitForImportCompletion(t, ts)

	// after convert finished we should find comments from new urls
	res, code = get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://www.radio-t.com/demo/")
	require.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	require.Nil(t, err)
	require.Equal(t, 5, comments.Info.Count)
	require.False(t, comments.Info.ReadOnly)

	res, code = get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://www.radio-t.com/demo-another/")
	require.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	require.Nil(t, err)
	require.Equal(t, 1, comments.Info.Count)
	require.True(t, comments.Info.ReadOnly)

	// should find nothing from previous url
	res, code = get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/demo/")
	require.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	require.Nil(t, err)
	require.Equal(t, 0, comments.Info.Count)

	res, code = get(t, ts.URL+"/api/v1/find?site=radio-t&url=https://radio-t.com/demo-another/")
	require.Equal(t, 200, code)
	comments = commentsWithInfo{}
	err = json.Unmarshal([]byte(res), &comments)
	require.Nil(t, err)
	require.Equal(t, 0, comments.Info.Count)
}

func TestMigrator_ConvertReject(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	s := `{"version":1,"users":[{"id":"blocked_user","blocked":{"status":true,"until":"2019-09-21T07:18:32.2346858-05:00"},"verified":false},{"id":"verified_user","blocked":{"status":false,"until":"0001-01-01T00:00:00Z"},"verified":true}],"posts":[{"url":"https://radio-t.com/demo-another/","read_only":true}]}
{"id":"25a18d59-aee9-45ab-86f5-c3fa31ef22c9","pid":"","text":"<p>comment to another post</p>\n","orig":"comment to another post","user":{"name":"admin","id":"admin","picture":"","ip":"1539deba4a54fc7862f0adf8d27192b19a27b1ed","admin":true},"locator":{"site":"radio-t","url":"https://radio-t.com/demo-another/"},"score":0,"vote":0,"time":"2019-09-14T07:26:23.4121277-05:00"}
{"id":"b814a90b-5b60-4e2b-b6e9-7058266c7706","pid":"","text":"<p>first comment from dev_user</p>\n","orig":"first comment from dev_user","user":{"name":"dev_user","id":"dev_user","picture":"http://127.0.0.1:8080/api/v1/avatar/ccfa2abd01667605b4e1fc4fcb91b1e1af323240.image","ip":"1539deba4a54fc7862f0adf8d27192b19a27b1ed","admin":true,"site_id":"radio-t"},"locator":{"site":"radio-t","url":"https://radio-t.com/demo/"},"score":1,"voted_ips":{"1539deba4a54fc7862f0adf8d27192b19a27b1ed":{"Timestamp":"2019-09-14T07:16:57.9319874-05:00","Value":true}},"vote":0,"time":"2019-09-14T07:16:18.0986736-05:00","title":"radio-t demo page"}
{"id":"145e3285-5dfd-4a4c-b8b0-3c6b5164473c","pid":"b814a90b-5b60-4e2b-b6e9-7058266c7706","text":"<p>reply to first message from any_user</p>\n","orig":"reply to first message from any_user","user":{"name":"any_user","id":"any_user","picture":"http://127.0.0.1:8080/api/v1/avatar/05ac5abbad12297e7a3578106fc0306f4fd73171.image","ip":"1539deba4a54fc7862f0adf8d27192b19a27b1ed","admin":false,"site_id":"radio-t"},"locator":{"site":"radio-t","url":"https://radio-t.com/demo/"},"score":0,"vote":0,"time":"2019-09-14T07:16:55.2362843-05:00","title":"radio-t demo page"}
{"id":"9beeb568-52b2-466d-b012-cd0d4dcdb854","pid":"","text":"<p>I want to be verified</p>\n","orig":"I want to be verified","user":{"name":"verified_user","id":"verified_user","picture":"http://127.0.0.1:8080/api/v1/avatar/7be676cbf4b5d7c0ae4da5f8143de927b12cfb42.image","ip":"1539deba4a54fc7862f0adf8d27192b19a27b1ed","admin":false,"verified":true,"site_id":"radio-t"},"locator":{"site":"radio-t","url":"https://radio-t.com/demo/"},"score":1,"voted_ips":{"1539deba4a54fc7862f0adf8d27192b19a27b1ed":{"Timestamp":"2019-09-14T07:22:01.0384052-05:00","Value":true}},"vote":0,"time":"2019-09-14T07:17:26.1825625-05:00","title":"radio-t demo page"}
{"id":"28e3b25a-d13b-4c0e-9179-5de9aad4a196","pid":"","text":"<p>I want to be blocked</p>\n","orig":"I want to be blocked","user":{"name":"blocked_user","id":"blocked_user","picture":"http://127.0.0.1:8080/api/v1/avatar/b4570b63a82ff5b5e188c9cb1820362ec13ad361.image","ip":"1539deba4a54fc7862f0adf8d27192b19a27b1ed","admin":false,"block":true,"site_id":"radio-t"},"locator":{"site":"radio-t","url":"https://radio-t.com/demo/"},"score":0,"vote":0,"time":"2019-09-14T07:18:05.8465267-05:00","title":"radio-t demo page"}
{"id":"09328137-9ac6-4388-ab75-e50113874f45","pid":"145e3285-5dfd-4a4c-b8b0-3c6b5164473c","text":"<p>reply from admin</p>\n","orig":"reply from admin","user":{"name":"dev_user","id":"dev_user","picture":"http://127.0.0.1:8080/api/v1/avatar/ccfa2abd01667605b4e1fc4fcb91b1e1af323240.image","ip":"1539deba4a54fc7862f0adf8d27192b19a27b1ed","admin":true,"site_id":"radio-t"},"locator":{"site":"radio-t","url":"https://radio-t.com/demo/"},"score":0,"vote":0,"time":"2019-09-14T07:24:17.5763304-05:00","title":"radio-t demo page"}`
	r := strings.NewReader(s)

	// without admin credentials
	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/admin/import?site=radio-t&provider=native", r)
	require.Nil(t, err)
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func waitForImportCompletion(t *testing.T, ts *httptest.Server) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", ts.URL+"/api/v1/admin/import/wait?site=remark42", nil)
	require.NoError(t, err)
	req.SetBasicAuth("admin", "password")
	assert.NoError(t, err)
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
		<category domain="post_tag" nicename="wierd-in-a-cant-quite-help-myself-way"><![CDATA[wierd in a can't quite help myself way]]></category>
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
