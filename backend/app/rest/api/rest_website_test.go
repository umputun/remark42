package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

func TestRest_UserWebsite(t *testing.T) {
	ts, srv, teardown := startupT(t)
	defer teardown()

	send := func(method, url, body, tkn string, basicAuth bool) (int, string) {
		t.Helper()
		reqBody := io.Reader(http.NoBody)
		if body != "" {
			reqBody = strings.NewReader(body)
		}
		req, err := http.NewRequest(method, ts.URL+url, reqBody)
		require.NoError(t, err)
		if basicAuth {
			req.SetBasicAuth("admin", "password")
		}
		resp, err := sendReq(req, tkn)
		require.NoError(t, err)
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		return resp.StatusCode, string(b)
	}

	// feature disabled by default
	code, _ := send(http.MethodGet, "/api/v1/website?site=remark42", "", devToken, false)
	assert.Equal(t, http.StatusForbidden, code)
	code, _ = send(http.MethodPut, "/api/v1/website?site=remark42", `{"website":"https://example.com"}`, devToken, false)
	assert.Equal(t, http.StatusForbidden, code)
	res, code := get(t, ts.URL+"/api/v1/config?site=remark42")
	require.Equal(t, http.StatusOK, code)
	assert.Contains(t, res, `"user_website":false`)

	srv.DataService.UserWebsite = true
	srv.privRest.userWebsite = true
	srv.adminRest.userWebsite = true
	res, _ = get(t, ts.URL+"/api/v1/config?site=remark42")
	assert.Contains(t, res, `"user_website":true`)

	// a comment by the dev user, cached before the website is set
	id := addComment(t, store.Comment{Text: "hello", Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}, ts)
	websiteOf := func() string {
		body, status := get(t, ts.URL+"/api/v1/id/"+id+"?site=remark42&url=https://radio-t.com/blah")
		require.Equal(t, http.StatusOK, status, body)
		c := store.Comment{}
		require.NoError(t, json.Unmarshal([]byte(body), &c))
		return c.User.Website
	}
	assert.Empty(t, websiteOf())

	// without auth and invalid values
	code, _ = send(http.MethodPut, "/api/v1/website?site=remark42", `{"website":"https://example.com"}`, "", false)
	assert.Equal(t, http.StatusUnauthorized, code)
	code, _ = send(http.MethodPut, "/api/v1/website?site=remark42", `{"website":"javascript:alert(1)"}`, devToken, false)
	assert.Equal(t, http.StatusBadRequest, code)
	code, _ = send(http.MethodPut, "/api/v1/website?site=remark42", `not json`, devToken, false)
	assert.Equal(t, http.StatusBadRequest, code)

	// set, read and see it on comments
	code, body := send(http.MethodPut, "/api/v1/website?site=remark42", `{"website":"dev.example.com/blog"}`, devToken, false)
	require.Equal(t, http.StatusOK, code, body)
	assert.JSONEq(t, `{"website":"https://dev.example.com/blog"}`, body)
	code, body = send(http.MethodGet, "/api/v1/website?site=remark42", "", devToken, false)
	require.Equal(t, http.StatusOK, code)
	assert.JSONEq(t, `{"website":"https://dev.example.com/blog"}`, body)
	assert.Equal(t, "https://dev.example.com/blog", websiteOf(), "cache flushed after the change")

	// an empty value removes it
	code, _ = send(http.MethodPut, "/api/v1/website?site=remark42", `{"website":""}`, devToken, false)
	require.Equal(t, http.StatusOK, code)
	assert.Empty(t, websiteOf())

	// admin sets and removes a website of any user
	code, _ = send(http.MethodPut, "/api/v1/admin/website/provider1_dev?site=remark42", `{"website":"https://imported.example.com"}`, devToken, false)
	assert.Equal(t, http.StatusForbidden, code, "not an admin")
	code, body = send(http.MethodPut, "/api/v1/admin/website/provider1_dev?site=remark42", `{"website":"https://imported.example.com"}`, "", true)
	require.Equal(t, http.StatusOK, code, body)
	assert.Equal(t, "https://imported.example.com", websiteOf())
	code, _ = send(http.MethodPut, "/api/v1/admin/website/provider1_dev?site=remark42", `{"website":"ftp://x.example.com"}`, "", true)
	assert.Equal(t, http.StatusBadRequest, code)
	code, _ = send(http.MethodDelete, "/api/v1/admin/website/provider1_dev?site=remark42", "", "", true)
	require.Equal(t, http.StatusOK, code)
	assert.Empty(t, websiteOf())
}
