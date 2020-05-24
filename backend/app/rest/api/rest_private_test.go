package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/render"
	"github.com/go-pkgz/auth/token"
	"github.com/go-pkgz/lgr"
	R "github.com/go-pkgz/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/notify"
	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/store/image"
)

// gopher png for test, from https://golang.org/src/image/png/example_test.go
const gopher = `iVBORw0KGgoAAAANSUhEUgAAAEsAAAA8CAAAAAALAhhPAAAFfUlEQVRYw62XeWwUVRzHf2+OPbo9d7tsWyiyaZti6eWGAhISoIGKECEKCAiJJkYTiUgTMYSIosYYBBIUIxoSPIINEBDi2VhwkQrVsj1ESgu9doHWdrul7ba73WNm3vOPtsseM9MdwvvrzTs+8/t95ze/33sI5BqiabU6m9En8oNjduLnAEDLUsQXFF8tQ5oxK3vmnNmDSMtrncks9Hhtt/qeWZapHb1ha3UqYSWVl2ZmpWgaXMXGohQAvmeop3bjTRtv6SgaK/Pb9/bFzUrYslbFAmHPp+3WhAYdr+7GN/YnpN46Opv55VDsJkoEpMrY/vO2BIYQ6LLvm0ThY3MzDzzeSJeeWNyTkgnIE5ePKsvKlcg/0T9QMzXalwXMlj54z4c0rh/mzEfr+FgWEz2w6uk8dkzFAgcARAgNp1ZYef8bH2AgvuStbc2/i6CiWGj98y2tw2l4FAXKkQBIf+exyRnteY83LfEwDQAYCoK+P6bxkZm/0966LxcAAILHB56kgD95PPxltuYcMtFTWw/FKkY/6Opf3GGd9ZF+Qp6mzJxzuRSractOmJrH1u8XTvWFHINNkLQLMR+XHXvfPPHw967raE1xxwtA36IMRfkAAG29/7mLuQcb2WOnsJReZGfpiHsSBX81cvMKywYZHhX5hFPtOqPGWZCXnhWGAu6lX91ElKXSalcLXu3UaOXVay57ZSe5f6Gpx7J2MXAsi7EqSp09b/MirKSyJfnfEEgeDjl8FgDAfvewP03zZ+AJ0m9aFRM8eEHBDRKjfcreDXnZdQuAxXpT2NRJ7xl3UkLBhuVGU16gZiGOgZmrSbRdqkILuL/yYoSXHHkl9KXgqNu3PB8oRg0geC5vFmLjad6mUyTKLmF3OtraWDIfACyXqmephaDABawfpi6tqqBZytfQMqOz6S09iWXhktrRaB8Xz4Yi/8gyABDm5NVe6qq/3VzPrcjELWrebVuyY2T7ar4zQyybUCtsQ5Es1FGaZVrRVQwAgHGW2ZCRZshI5bGQi7HesyE972pOSeMM0dSktlzxRdrlqb3Osa6CCS8IJoQQQgBAbTAa5l5epO34rJszibJI8rxLfGzcp1dRosutGeb2VDNgqYrwTiPNsLxXiPi3dz7LiS1WBRBDBOnqEjyy3aQb+/bLiJzz9dIkscVBBLxMfSEac7kO4Fpkngi0ruNBeSOal+u8jgOuqPz12nryMLCniEjtOOOmpt+KEIqsEdocJjYXwrh9OZqWJQyPCTo67LNS/TdxLAv6R5ZNK9npEjbYdT33gRo4o5oTqR34R+OmaSzDBWsAIPhuRcgyoteNi9gF0KzNYWVItPf2TLoXEg+7isNC7uJkgo1iQWOfRSP9NR11RtbZZ3OMG/VhL6jvx+J1m87+RCfJChAtEBQkSBX2PnSiihc/Twh3j0h7qdYQAoRVsRGmq7HU2QRbaxVGa1D6nIOqaIWRjyRZpHMQKWKpZM5feA+lzC4ZFultV8S6T0mzQGhQohi5I8iw+CsqBSxhFMuwyLgSwbghGb0AiIKkSDmGZVmJSiKihsiyOAUs70UkywooYP0bii9GdH4sfr1UNysd3fUyLLMQN+rsmo3grHl9VNJHbbwxoa47Vw5gupIqrZcjPh9R4Nye3nRDk199V+aetmvVtDRE8/+cbgAAgMIWGb3UA0MGLE9SCbWX670TDy1y98c3D27eppUjsZ6fql3jcd5rUe7+ZIlLNQny3Rd+E5Tct3WVhTM5RBCEdiEK0b6B+/ca2gYU393nFj/n1AygRQxPIUA043M42u85+z2SnssKrPl8Mx76NL3E6eXc3be7OD+H4WHbJkKI8AU8irbITQjZ+0hQcPEgId/Fn/pl9crKH02+5o2b9T/eMx7pKoskYgAAAABJRU5ErkJggg==`

func gopherPNG() io.Reader { return base64.NewDecoder(base64.StdEncoding, strings.NewReader(gopher)) }

func TestRest_Create(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	resp, err := post(t, ts.URL+"/api/v1/comment",
		`{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`)
	assert.NoError(t, err)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(b))

	c := R.JSON{}
	err = json.Unmarshal(b, &c)
	assert.NoError(t, err)
	loc := c["locator"].(map[string]interface{})
	assert.Equal(t, "remark42", loc["site"])
	assert.Equal(t, "https://radio-t.com/blah1", loc["url"])
	assert.True(t, len(c["id"].(string)) > 8)
}

func TestRest_CreateOldPost(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	// make old, but not too old comment
	old := store.Comment{Text: "test test old", ParentID: "", Timestamp: time.Now().AddDate(0, 0, -5),
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}, User: store.User{ID: "u1"}}
	_, err := srv.DataService.Create(old)
	assert.NoError(t, err)

	comments, err := srv.DataService.Find(store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}, "time", store.User{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(comments))

	// try to add new comment to the same old post
	resp, err := post(t, ts.URL+"/api/v1/comment",
		`{"text": "test 123", "locator":{"site": "remark42","url": "https://radio-t.com/blah1"}}`)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	assert.NoError(t, srv.DataService.DeleteAll("remark42"))
	// make too old comment
	old = store.Comment{Text: "test test old", ParentID: "", Timestamp: time.Now().AddDate(0, 0, -15),
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}, User: store.User{ID: "u1"}}
	_, err = srv.DataService.Create(old)
	assert.NoError(t, err)

	resp, err = post(t, ts.URL+"/api/v1/comment",
		`{"text": "test 123", "locator":{"site": "remark42","url": "https://radio-t.com/blah1"}}`)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestRest_CreateTooBig(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	longComment := fmt.Sprintf(`{"text": "%4001s", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, "Щ")

	resp, err := post(t, ts.URL+"/api/v1/comment", longComment)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	c := R.JSON{}
	err = json.Unmarshal(b, &c)
	assert.NoError(t, err)
	assert.Equal(t, "comment text exceeded max allowed size 4000 (4001)", c["error"])
	assert.Equal(t, "invalid comment", c["details"])

	veryLongComment := fmt.Sprintf(`{"text": "%70000s", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, "Щ")
	resp, err = post(t, ts.URL+"/api/v1/comment", veryLongComment)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	b, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	c = R.JSON{}
	err = json.Unmarshal(b, &c)
	assert.NoError(t, err)
	assert.Equal(t, "http: request body too large", c["error"])
	assert.Equal(t, "can't bind comment", c["details"])
}

func TestRest_CreateWithRestrictedWord(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	badComment := `{"text": "What the duck is that?", "locator":{"url": "https://radio-t.com/blah1",
"site": "remark42"}}`

	resp, err := post(t, ts.URL+"/api/v1/comment", badComment)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	c := R.JSON{}
	err = json.Unmarshal(b, &c)
	assert.NoError(t, err)
	assert.Equal(t, "comment contains restricted words", c["error"])
	assert.Equal(t, "invalid comment", c["details"])
}

func TestRest_CreateRejected(t *testing.T) {

	ts, _, teardown := startupT(t)
	defer teardown()
	body := `{"text": "test 123", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`

	// try to create without auth
	resp, err := http.Post(ts.URL+"/api/v1/comment", "", strings.NewReader(body))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 401, resp.StatusCode)

	// try with wrong aud
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/comment", strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Add("X-JWT", devTokenBadAud)
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusForbidden, resp.StatusCode, "reject wrong aud")
}

func TestRest_CreateAndGet(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	// create comment
	resp, err := post(t, ts.URL+"/api/v1/comment",
		`{"text": "**test** *123*\n\n http://radio-t.com", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	c := R.JSON{}
	err = json.Unmarshal(b, &c)
	assert.NoError(t, err)

	id := c["id"].(string)

	// get created comment by id as admin
	res, code := getWithAdminAuth(t, fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah1", ts.URL, id))
	assert.Equal(t, 200, code)
	comment := store.Comment{}
	err = json.Unmarshal([]byte(res), &comment)
	assert.NoError(t, err)
	assert.Equal(t, "<p><strong>test</strong> <em>123</em></p>\n\n<p><a href=\"http://radio-t.com\" rel=\"nofollow\">http://radio-t.com</a></p>\n", comment.Text)
	assert.Equal(t, "**test** *123*\n\n http://radio-t.com", comment.Orig)
	assert.Equal(t, store.User{Name: "admin", ID: "admin", Admin: true, Blocked: false,
		IP: "dbc7c999343f003f189f70aaf52cc04443f90790"},
		comment.User)

	// get created comment by id as non-admin
	res, code = getWithDevAuth(t, fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah1", ts.URL, id))
	assert.Equal(t, 200, code)
	comment = store.Comment{}
	err = json.Unmarshal([]byte(res), &comment)
	assert.NoError(t, err)
	assert.Equal(t, store.User{Name: "admin", ID: "admin", Admin: true, Blocked: false, IP: ""}, comment.User, "no ip")
}

func TestRest_Update(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id := addComment(t, c1, ts)

	client := http.Client{}
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/comment/"+id+"?site=remark42&url=https://radio-t.com/blah1",
		strings.NewReader(`{"text":"updated text", "summary":"my edit"}`))
	assert.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	b, err := client.Do(req)
	assert.NoError(t, err)
	body, err := ioutil.ReadAll(b.Body)
	assert.NoError(t, err)
	assert.Equal(t, 200, b.StatusCode, string(body))

	// comments returned by update
	c2 := store.Comment{}
	err = json.Unmarshal(body, &c2)
	assert.NoError(t, err)
	assert.Equal(t, id, c2.ID)
	assert.Equal(t, "<p>updated text</p>\n", c2.Text)
	assert.Equal(t, "updated text", c2.Orig)
	assert.Equal(t, "my edit", c2.Edit.Summary)
	assert.True(t, time.Since(c2.Edit.Timestamp) < 1*time.Second)

	// read updated comment
	res, code := getWithAdminAuth(t, fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah1", ts.URL, id))
	assert.Equal(t, 200, code)
	c3 := store.Comment{}
	err = json.Unmarshal([]byte(res), &c3)
	assert.NoError(t, err)
	assert.Equal(t, c2, c3, "same as response from update")
}

func TestRest_UpdateDelete(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id := addComment(t, c1, ts)

	// check multi count updated
	resp, err := post(t, ts.URL+"/api/v1/counts?site=remark42", `["https://radio-t.com/blah1","https://radio-t.com/blah2"]`)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	bb, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	j := []store.PostInfo{}
	err = json.Unmarshal(bb, &j)
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah1", Count: 1},
		{URL: "https://radio-t.com/blah2", Count: 0}}, j)

	// delete a comment
	client := http.Client{}
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/comment/"+id+"?site=remark42&url=https://radio-t.com/blah1",
		strings.NewReader(`{"delete": true, "summary":"removed by user"}`))
	require.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	b, err := client.Do(req)
	require.NoError(t, err)
	body, err := ioutil.ReadAll(b.Body)
	require.NoError(t, err)
	assert.Equal(t, 200, b.StatusCode, string(body))

	// comments returned by update
	c2 := store.Comment{}
	err = json.Unmarshal(body, &c2)
	require.NoError(t, err)
	assert.Equal(t, id, c2.ID)
	assert.True(t, c2.Deleted)

	// read updated comment
	res, code := getWithDevAuth(t, fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah1", ts.URL, id))
	assert.Equal(t, 200, code)
	c3 := store.Comment{}
	err = json.Unmarshal([]byte(res), &c3)
	assert.NoError(t, err)
	assert.Equal(t, "", c3.Text)
	assert.Equal(t, "", c3.Orig)
	assert.True(t, c3.Deleted)

	// check multi count updated
	resp, err = post(t, ts.URL+"/api/v1/counts?site=remark42", `["https://radio-t.com/blah1","https://radio-t.com/blah2"]`)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	bb, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	j = []store.PostInfo{}
	err = json.Unmarshal(bb, &j)
	require.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "https://radio-t.com/blah1", Count: 0},
		{URL: "https://radio-t.com/blah2", Count: 0}}, j)
}

func TestRest_UpdateNotOwner(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}, User: store.User{ID: "xyz"}}
	id1, err := srv.DataService.Create(c1)
	assert.NoError(t, err)

	client := http.Client{}
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/comment/"+id1+
		"?site=remark42&url=https://radio-t.com/blah1", strings.NewReader(`{"text":"updated text", "summary":"my edit"}`))
	assert.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	b, err := client.Do(req)
	assert.NoError(t, err)
	body, err := ioutil.ReadAll(b.Body)
	assert.NoError(t, err)
	assert.NoError(t, b.Body.Close())
	assert.Equal(t, 403, b.StatusCode, string(body), "update from non-owner")
	assert.Equal(t, `{"code":3,"details":"can not edit comments for other users","error":"rejected"}`+"\n", string(body))

	client = http.Client{}
	req, err = http.NewRequest(http.MethodPut, ts.URL+"/api/v1/comment/"+id1+
		"?site=remark42&url=https://radio-t.com/blah1", strings.NewReader(`ERRR "text":"updated text", "summary":"my"}`))
	assert.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	b, err = client.Do(req)
	assert.NoError(t, err)
	assert.NoError(t, b.Body.Close())
	assert.Equal(t, 400, b.StatusCode, string(body), "update is not json")
}

func TestRest_UpdateWrongAud(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id := addComment(t, c1, ts)

	client := http.Client{}
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/comment/"+id+"?site=remark42&url=https://radio-t.com/blah1",
		strings.NewReader(`{"text":"updated text", "summary":"my edit"}`))
	assert.NoError(t, err)
	req.Header.Add("X-JWT", devTokenBadAud)
	b, err := client.Do(req)
	assert.NoError(t, err)
	assert.NoError(t, b.Body.Close())
	assert.Equal(t, http.StatusForbidden, b.StatusCode, "reject update with wrong aut in jwt")
}

func TestRest_UpdateWithRestrictedWords(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "What the quack is that?", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah1"}}
	id := addComment(t, c1, ts)

	client := http.Client{}
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/comment/"+id+"?site=remark42&url=https://radio-t.com/blah1",
		strings.NewReader(`{"text":"What the duck is that?", "summary":"my edit"}`))
	assert.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	b, err := client.Do(req)
	assert.NoError(t, err)
	body, err := ioutil.ReadAll(b.Body)
	assert.NoError(t, err)
	c := R.JSON{}
	err = json.Unmarshal(body, &c)
	assert.NoError(t, err)
	assert.Equal(t, 400, b.StatusCode, string(body))
	assert.Equal(t, "comment contains restricted words", c["error"])
	assert.Equal(t, "invalid comment", c["details"])
}

func TestRest_Vote(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}

	id1 := addComment(t, c1, ts)
	addComment(t, c2, ts)

	vote := func(val int) int {
		client := http.Client{}
		req, err := http.NewRequest(http.MethodPut,
			fmt.Sprintf("%s/api/v1/vote/%s?site=remark42&url=https://radio-t.com/blah&vote=%d", ts.URL, id1, val), nil)
		assert.NoError(t, err)
		req.Header.Add("X-JWT", devToken)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.NoError(t, resp.Body.Close())
		return resp.StatusCode
	}

	assert.Equal(t, 200, vote(1), "first vote allowed")
	assert.Equal(t, 400, vote(1), "second vote rejected")
	body, code := getWithDevAuth(t, fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1))
	assert.Equal(t, 200, code)
	cr := store.Comment{}
	err := json.Unmarshal([]byte(body), &cr)
	assert.NoError(t, err)
	assert.Equal(t, 1, cr.Score)
	assert.Equal(t, 1, cr.Vote)
	assert.Equal(t, map[string]bool(nil), cr.Votes)

	assert.Equal(t, 200, vote(-1), "opposite vote allowed")
	body, code = getWithDevAuth(t, fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1))
	assert.Equal(t, 200, code)
	cr = store.Comment{}
	err = json.Unmarshal([]byte(body), &cr)
	assert.NoError(t, err)
	assert.Equal(t, 0, cr.Score)
	assert.Equal(t, 0, cr.Vote)

	assert.Equal(t, 200, vote(-1), "opposite vote allowed one more time")
	body, code = getWithDevAuth(t, fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1))
	assert.Equal(t, 200, code)
	cr = store.Comment{}
	err = json.Unmarshal([]byte(body), &cr)
	assert.NoError(t, err)
	assert.Equal(t, -1, cr.Score)
	assert.Equal(t, -1, cr.Vote)

	assert.Equal(t, 400, vote(-1), "dbl vote not allowed")
	body, code = getWithDevAuth(t, fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1))
	assert.Equal(t, 200, code)
	cr = store.Comment{}
	err = json.Unmarshal([]byte(body), &cr)
	assert.NoError(t, err)
	assert.Equal(t, -1, cr.Score)
	assert.Equal(t, -1, cr.Vote)

	body, code = get(t, fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1))
	assert.Equal(t, 200, code)
	cr = store.Comment{}
	err = json.Unmarshal([]byte(body), &cr)
	assert.NoError(t, err)
	assert.Equal(t, -1, cr.Score)
	assert.Equal(t, 0, cr.Vote, "no vote info for not authed user")
	assert.Equal(t, map[string]bool(nil), cr.Votes)

	req, err := http.NewRequest("GET",
		fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1), nil)
	assert.NoError(t, err)
	resp, err := sendReq(t, req, adminUmputunToken)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	cr = store.Comment{}
	err = json.NewDecoder(resp.Body).Decode(&cr)
	assert.NoError(t, err)
	assert.Equal(t, -1, cr.Score)
	assert.Equal(t, 0, cr.Vote, "no vote info for different user")
	assert.Equal(t, map[string]bool(nil), cr.Votes)
}

func TestRest_AnonVote(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	c1 := store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}
	c2 := store.Comment{Text: "test test #2", ParentID: "p1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}

	id1 := addComment(t, c1, ts)
	addComment(t, c2, ts)

	vote := func(val int) int {
		client := http.Client{}
		req, err := http.NewRequest(http.MethodPut,
			fmt.Sprintf("%s/api/v1/vote/%s?site=remark42&url=https://radio-t.com/blah&vote=%d", ts.URL, id1, val), nil)
		assert.NoError(t, err)
		req.Header.Add("X-JWT", anonToken)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.NoError(t, resp.Body.Close())
		return resp.StatusCode
	}

	getWithAnonAuth := func(url string) (body string, code int) {
		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("GET", url, nil)
		require.NoError(t, err)
		req.Header.Add("X-JWT", anonToken)
		r, err := client.Do(req)
		require.NoError(t, err)
		defer r.Body.Close()
		b, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		return string(b), r.StatusCode
	}

	assert.Equal(t, 403, vote(1), "vote is disallowed with anonVote false")
	srv.privRest.anonVote = true
	assert.Equal(t, 200, vote(1), "first vote allowed")
	assert.Equal(t, 400, vote(1), "second vote rejected")
	body, code := getWithAnonAuth(fmt.Sprintf("%s/api/v1/id/%s?site=remark42&url=https://radio-t.com/blah", ts.URL, id1))

	assert.Equal(t, 200, code)
	cr := store.Comment{}
	err := json.Unmarshal([]byte(body), &cr)
	assert.NoError(t, err)
	assert.Equal(t, 1, cr.Score)
	assert.Equal(t, 1, cr.Vote)
	assert.Equal(t, map[string]bool(nil), cr.Votes)
}

type MockFS struct{}

func (fs *MockFS) ReadFile(path string) ([]byte, error) {
	return []byte(fmt.Sprintf("template %s", path)), nil
}

func TestRest_Email(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	srv.privRest.templates = &MockFS{}

	// issue good token
	claims := token.Claims{
		Handshake: &token.Handshake{ID: "dev::good@example.com"},
		StandardClaims: jwt.StandardClaims{
			Audience:  "remark42",
			ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
			NotBefore: time.Now().Add(-1 * time.Minute).Unix(),
			Issuer:    "remark42",
		},
	}
	tkn, err := srv.Authenticator.TokenService().Token(claims)
	require.NoError(t, err)
	goodToken := tkn

	var testData = []struct {
		description  string
		url          string
		method       string
		responseCode int
		noAuth       bool
		cookieEmail  string
	}{
		{description: "issue delete request without auth", url: "/api/v1/email", method: http.MethodDelete, responseCode: http.StatusUnauthorized, noAuth: true},
		{description: "issue delete request without site_id", url: "/api/v1/email", method: http.MethodDelete, responseCode: http.StatusBadRequest},
		{description: "delete non-existent user email", url: "/api/v1/email?site=remark42", method: http.MethodDelete, responseCode: http.StatusOK},
		{description: "set user email, token not set", url: "/api/v1/email/confirm?site=remark42", method: http.MethodPost, responseCode: http.StatusBadRequest},
		{description: "send confirmation without address", url: "/api/v1/email/subscribe?site=remark42", method: http.MethodPost, responseCode: http.StatusBadRequest},
		{description: "send confirmation", url: "/api/v1/email/subscribe?site=remark42&address=good@example.com", method: http.MethodPost, responseCode: http.StatusOK},
		{description: "set user email, token is good", url: fmt.Sprintf("/api/v1/email/confirm?site=remark42&tkn=%s", goodToken), method: http.MethodPost, responseCode: http.StatusOK, cookieEmail: "good@example.com"},
		{description: "send confirmation with same address", url: "/api/v1/email/subscribe?site=remark42&address=good@example.com", method: http.MethodPost, responseCode: http.StatusConflict},
		{description: "get user email", url: "/api/v1/email?site=remark42", method: http.MethodGet, responseCode: http.StatusOK},
		{description: "delete user email", url: "/api/v1/email?site=remark42", method: http.MethodDelete, responseCode: http.StatusOK},
		{description: "send another confirmation", url: "/api/v1/email/subscribe?site=remark42&address=good@example.com", method: http.MethodPost, responseCode: http.StatusOK},
		{description: "set user email, token is good", url: fmt.Sprintf("/api/v1/email/confirm?site=remark42&tkn=%s", goodToken), method: http.MethodPost, responseCode: http.StatusOK, cookieEmail: "good@example.com"},
		{description: "unsubscribe user, no token", url: "/email/unsubscribe.html?site=remark42", method: http.MethodPost, responseCode: http.StatusBadRequest},
		{description: "unsubscribe user, wrong token", url: "/email/unsubscribe.html?site=remark42&tkn=jwt", method: http.MethodGet, responseCode: http.StatusForbidden},
		{description: "unsubscribe user, good token", url: fmt.Sprintf("/email/unsubscribe.html?site=remark42&tkn=%s", goodToken), method: http.MethodPost, responseCode: http.StatusOK},
		{description: "unsubscribe user second time, good token", url: fmt.Sprintf("/email/unsubscribe.html?site=remark42&tkn=%s", goodToken), method: http.MethodPost, responseCode: http.StatusConflict},
	}
	client := http.Client{}
	for _, x := range testData {
		x := x
		t.Run(x.description, func(t *testing.T) {
			req, err := http.NewRequest(x.method, ts.URL+x.url, nil)
			require.NoError(t, err)
			if !x.noAuth {
				req.Header.Add("X-JWT", devToken)
			}
			resp, err := client.Do(req)
			require.NoError(t, err)
			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			// read User.Email from the token in the cookie
			for _, c := range resp.Cookies() {
				if c.Name == "JWT" {
					claims, err := srv.Authenticator.TokenService().Parse(c.Value)
					require.NoError(t, err)
					assert.Equal(t, x.cookieEmail, claims.User.Email, "cookie email check failed")
				}
			}
			assert.Equal(t, x.responseCode, resp.StatusCode, string(body))
		})
	}
}

func TestRest_EmailNotification(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	mockDestination := &notify.MockDest{}
	srv.privRest.notifyService = notify.NewService(srv.DataService, 1, mockDestination)
	defer srv.privRest.notifyService.Close()

	client := http.Client{}

	// create new comment from dev user
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/comment", strings.NewReader(
		`{"text": "test 123",
"user": {"name": "dev::good@example.com"},
"locator":{"url": "https://radio-t.com/blah1",
"site": "remark42"}}`))
	assert.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(body))
	parentComment := store.Comment{}
	require.NoError(t, render.DecodeJSON(strings.NewReader(string(body)), &parentComment))
	// wait for mock notification Submit to kick off
	time.Sleep(time.Millisecond * 30)
	require.Equal(t, 2, len(mockDestination.Get()))
	assert.Empty(t, mockDestination.Get()[0].Email)
	assert.Equal(t, "admin@example.org", mockDestination.Get()[1].Email)

	// create child comment from another user, email notification only to admin expected
	req, err = http.NewRequest("POST", ts.URL+"/api/v1/comment", strings.NewReader(fmt.Sprintf(
		`{"text": "test 456",
	"pid": "%s",
	"user": {"name": "other_user"},
	"locator":{"url": "https://radio-t.com/blah1",
	"site": "remark42"}}`, parentComment.ID)))
	assert.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(body))
	// wait for mock notification Submit to kick off
	time.Sleep(time.Millisecond * 30)
	require.Equal(t, 4, len(mockDestination.Get()))
	assert.Empty(t, mockDestination.Get()[2].Email)
	assert.Equal(t, "admin@example.org", mockDestination.Get()[3].Email)

	// send confirmation token for email
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/v1/email/subscribe?site=remark42&address=good@example.com", nil)
	require.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err = client.Do(req)
	require.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	// wait for mock notification Submit to kick off
	time.Sleep(time.Millisecond * 30)
	require.Equal(t, 5, len(mockDestination.Get()))
	require.NotEmpty(t, mockDestination.Get()[4].Verification)
	verificationToken := mockDestination.Get()[4].Verification.Token

	// verify email
	req, err = http.NewRequest(http.MethodPost, ts.URL+fmt.Sprintf("/api/v1/email/confirm?site=remark42&tkn=%s", verificationToken), nil)
	require.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err = client.Do(req)
	require.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	// get user information to verify the subscription
	req, err = http.NewRequest(http.MethodGet, ts.URL+"/api/v1/user?site=remark42", nil)
	require.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err = client.Do(req)
	require.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	var user store.User
	err = json.Unmarshal(body, &user)
	assert.NoError(t, err)
	assert.Equal(t, store.User{Name: "developer one", ID: "dev", EmailSubscription: true,
		Picture: "http://example.com/pic.png", IP: "127.0.0.1", SiteID: "remark42"}, user)

	// create child comment from another user, email notification expected
	req, err = http.NewRequest("POST", ts.URL+"/api/v1/comment", strings.NewReader(fmt.Sprintf(
		`{"text": "test 789",
	"pid": "%s",
	"user": {"name": "other_user"},
	"locator":{"url": "https://radio-t.com/blah1",
	"site": "remark42"}}`, parentComment.ID)))
	assert.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(body))
	// wait for mock notification Submit to kick off
	time.Sleep(time.Millisecond * 30)
	require.Equal(t, 7, len(mockDestination.Get()))
	assert.Equal(t, "good@example.com", mockDestination.Get()[5].Email)

	// delete user's email
	req, err = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/email?site=remark42", nil)
	require.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err = client.Do(req)
	require.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	// create child comment from another user, no email notification expected except for admin
	req, err = http.NewRequest("POST", ts.URL+"/api/v1/comment", strings.NewReader(
		`{"text": "test 321",
	"user": {"name": "other_user"},
	"locator":{"url": "https://radio-t.com/blah1",
	"site": "remark42"}}`))
	assert.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	body, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(body))
	// wait for mock notification Submit to kick off
	time.Sleep(time.Millisecond * 30)
	require.Equal(t, 9, len(mockDestination.Get()))
	assert.Empty(t, mockDestination.Get()[7].Email)
}

func TestRest_UserAllData(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	// write 3 comments
	user := store.User{ID: "dev", Name: "user name 1"}
	c1 := store.Comment{User: user, Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 10, 0, time.Local)}
	c2 := store.Comment{User: user, Text: "test test #2", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 20, 0, time.Local)}
	c3 := store.Comment{User: user, Text: "test test #3", ParentID: "p1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 25, 0, time.Local)}
	_, err := srv.DataService.Create(c1)
	require.NoError(t, err, "%+v", err)
	_, err = srv.DataService.Create(c2)
	require.NoError(t, err)
	_, err = srv.DataService.Create(c3)
	require.NoError(t, err)

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("GET", ts.URL+"/api/v1/userdata?site=remark42", nil)
	require.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))

	ungzReader, err := gzip.NewReader(resp.Body)
	assert.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	ungzBody, err := ioutil.ReadAll(ungzReader)
	assert.NoError(t, err)
	strUungzBody := string(ungzBody)
	assert.True(t, strings.HasPrefix(strUungzBody,
		`{"info": {"name":"developer one","id":"dev","picture":"http://example.com/pic.png","ip":"127.0.0.1","admin":false,"site_id":"remark42"}, "comments":[{`))
	assert.Equal(t, 3, strings.Count(strUungzBody, `"text":`), "3 comments inside")

	parsed := struct {
		Info     store.User      `json:"info"`
		Comments []store.Comment `json:"comments"`
	}{}

	err = json.Unmarshal(ungzBody, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, store.User{Name: "developer one", ID: "dev",
		Picture: "http://example.com/pic.png", IP: "127.0.0.1", SiteID: "remark42"}, parsed.Info)
	assert.Equal(t, 3, len(parsed.Comments))

	req, err = http.NewRequest("GET", ts.URL+"/api/v1/userdata?site=remark42", nil)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, 401, resp.StatusCode)
}

func TestRest_UserAllDataManyComments(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	user := store.User{ID: "dev", Name: "user name 1"}
	c := store.Comment{User: user, Text: "test test #1", Locator: store.Locator{SiteID: "remark42",
		URL: "https://radio-t.com/blah1"}, Timestamp: time.Date(2018, 5, 27, 1, 14, 10, 0, time.Local)}

	for i := 0; i < 51; i++ {
		c.ID = fmt.Sprintf("id-%03d", i)
		c.Timestamp = c.Timestamp.Add(time.Second)
		_, err := srv.DataService.Create(c)
		require.NoError(t, err)
	}
	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest("GET", ts.URL+"/api/v1/userdata?site=remark42", nil)
	require.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.Equal(t, "application/gzip", resp.Header.Get("Content-Type"))

	ungzReader, err := gzip.NewReader(resp.Body)
	assert.NoError(t, err)
	ungzBody, err := ioutil.ReadAll(ungzReader)
	assert.NoError(t, err)
	strUngzBody := string(ungzBody)
	assert.True(t, strings.HasPrefix(strUngzBody,
		`{"info": {"name":"developer one","id":"dev","picture":"http://example.com/pic.png","ip":"127.0.0.1","admin":false,"site_id":"remark42"}, "comments":[{`))
	assert.Equal(t, 51, strings.Count(strUngzBody, `"text":`), "51 comments inside")
}

func TestRest_DeleteMe(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	client := http.Client{}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/deleteme?site=remark42", ts.URL), nil)
	assert.NoError(t, err)
	req.Header.Add("X-JWT", devToken)
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, resp.Body.Close())
	assert.NoError(t, err)

	m := map[string]string{}
	err = json.Unmarshal(body, &m)
	assert.NoError(t, err)
	assert.Equal(t, "remark42", m["site"])
	assert.Equal(t, "dev", m["user_id"])

	tkn := m["token"]
	claims, err := srv.Authenticator.TokenService().Parse(tkn)
	assert.NoError(t, err)
	assert.Equal(t, "dev", claims.User.ID)
	assert.Equal(t, "https://demo.remark42.com/web/deleteme.html?token="+tkn, m["link"])

	req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/deleteme?site=remark42", ts.URL), nil)
	assert.NoError(t, err)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
	assert.NoError(t, resp.Body.Close())
}

func TestRest_SavePictureCtrl(t *testing.T) {
	ts, _, teardown := startupT(t)
	defer teardown()

	// save picture
	savePic := func(name string) (id string) {
		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		fileWriter, err := bodyWriter.CreateFormFile("file", name)
		require.NoError(t, err)
		_, err = io.Copy(fileWriter, gopherPNG())
		require.NoError(t, err)
		contentType := bodyWriter.FormDataContentType()
		require.NoError(t, bodyWriter.Close())

		client := http.Client{}
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/picture", ts.URL), bodyBuf)
		require.NoError(t, err)
		req.Header.Add("Content-Type", contentType)
		req.Header.Add("X-JWT", devToken)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		m := map[string]string{}
		err = json.Unmarshal(body, &m)
		assert.NoError(t, err)
		assert.True(t, m["id"] != "")
		return m["id"]
	}

	id := savePic("picture.png")
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/picture/%s", ts.URL, id))
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 1462, len(body))
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))

	id = savePic("picture.gif")
	resp, err = http.Get(fmt.Sprintf("%s/api/v1/picture/%s", ts.URL, id))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))

	id = savePic("picture.jpg")
	resp, err = http.Get(fmt.Sprintf("%s/api/v1/picture/%s", ts.URL, id))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))

	id = savePic("picture.blah")
	resp, err = http.Get(fmt.Sprintf("%s/api/v1/picture/%s", ts.URL, id))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "image/png", resp.Header.Get("Content-Type"))

	resp, err = http.Get(fmt.Sprintf("%s/api/v1/picture/blah/pic.blah", ts.URL))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, 400, resp.StatusCode)
}

func TestRest_CreateWithPictures(t *testing.T) {
	ts, svc, teardown := startupT(t)
	defer func() {
		teardown()
		os.RemoveAll("/tmp/remark42")
	}()
	lgr.Setup(lgr.Debug, lgr.CallerFile, lgr.CallerFunc)

	imageService := image.NewService(&image.FileSystem{
		Staging:  "/tmp/remark42/images.staging",
		Location: "/tmp/remark42/images",
	}, image.ServiceParams{
		EditDuration: 100 * time.Millisecond,
		MaxSize:      2000,
	})
	defer imageService.Close(context.Background())

	svc.privRest.imageService = imageService
	svc.ImageService = imageService

	dataService := svc.DataService
	dataService.ImageService = svc.ImageService
	svc.privRest.dataService = dataService

	uploadPicture := func(file string) (id string) {
		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		fileWriter, err := bodyWriter.CreateFormFile("file", file)
		require.NoError(t, err)
		_, err = io.Copy(fileWriter, gopherPNG())
		require.NoError(t, err)
		contentType := bodyWriter.FormDataContentType()
		require.NoError(t, bodyWriter.Close())
		client := http.Client{}
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/picture", ts.URL), bodyBuf)
		require.NoError(t, err)
		req.Header.Add("Content-Type", contentType)
		req.Header.Add("X-JWT", devToken)
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		m := map[string]string{}
		err = json.Unmarshal(body, &m)
		assert.NoError(t, err)
		return m["id"]
	}

	var ids [3]string

	for i := range ids {
		ids[i] = uploadPicture(fmt.Sprintf("pic%d.png", i))
	}

	text := fmt.Sprintf(`text 123  ![](/api/v1/picture/%s) *xxx* ![](/api/v1/picture/%s) ![](/api/v1/picture/%s)`, ids[0], ids[1], ids[2])
	body := fmt.Sprintf(`{"text": "%s", "locator":{"url": "https://radio-t.com/blah1", "site": "remark42"}}`, text)

	resp, err := post(t, ts.URL+"/api/v1/comment", body)
	assert.NoError(t, err)
	b, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(b))

	for i := range ids {
		_, err = os.Stat("/tmp/remark42/images/" + ids[i])
		assert.Error(t, err, "picture %d not moved from staging yet", i)
	}

	time.Sleep(1500 * time.Millisecond)

	for i := range ids {
		_, err = os.Stat("/tmp/remark42/images/" + ids[i])
		assert.NoError(t, err, "picture %d moved from staging and available in permanent location", i)
	}
}
