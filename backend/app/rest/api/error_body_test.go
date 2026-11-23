package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-pkgz/lgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark42/backend/app/store"
)

// errorResponse is the JSON body rest.SendErrorJSON writes
type errorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details"`
	Code    int    `json:"code"`
}

// captureLog sends the log of the test's server to the returned buffer
func captureLog(t *testing.T) *lockedBuffer {
	t.Helper()
	logs := &lockedBuffer{}
	lgr.Setup(lgr.Out(logs), lgr.Err(logs))
	t.Cleanup(func() { lgr.Setup() })
	return logs
}

// readErrorResponse reads and closes resp, returning its raw body and the decoded error
func readErrorResponse(t *testing.T, resp *http.Response) (body string, e errorResponse) {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return string(b), decodeError(t, string(b))
}

// decodeError decodes the error an answer of the api carries in body
func decodeError(t *testing.T, body string) (e errorResponse) {
	t.Helper()
	require.NoError(t, json.Unmarshal([]byte(body), &e), body)
	return e
}

// serverDir is the directory a server started by startupT keeps its files in
func serverDir(srv *Rest) string { return filepath.Dir(srv.WebRoot) }

func TestRest_SavePictureStoreError(t *testing.T) {
	ts, srv := startupT(t)
	logs := captureLog(t)

	// a file where the staging directory belongs makes the file system store fail to save
	require.NoError(t, os.MkdirAll(serverDir(srv)+"/pics-remark42", 0o700))
	require.NoError(t, os.WriteFile(serverDir(srv)+"/pics-remark42/staging", []byte("x"), 0o600))

	resp := uploadPicture(t, ts.URL, gopherPNG())
	body, e := readErrorResponse(t, resp)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, errorResponse{Error: "image store error", Details: "can't save image", Code: 0}, e,
		"the store error is replaced in the body")
	assert.NotContains(t, body, serverDir(srv), "no server path in the body")
	assert.Contains(t, logs.String(), serverDir(srv)+"/pics-remark42/staging", "the store error, with the path, is still logged")
}

// an image the service refuses keeps the reason in the body, as API users tell the cases apart by it
func TestRest_SavePictureRejectedImage(t *testing.T) {
	ts, _ := startupT(t)

	tbl := []struct {
		name, img, want string
	}{
		{"too large", "\x89PNG\r\n\x1a\n" + strings.Repeat("x", 20000), "can't load image: file is too large (limit=10000)"},
		{"not an image", "plain text, not an image", "can't load image: file format not allowed"},
		{"malformed", "\x89PNG\r\n\x1a\n" + strings.Repeat("x", 100), "image rejected: malformed or exceeds 16777216-pixel safe limit"},
	}
	for _, tc := range tbl {
		t.Run(tc.name, func(t *testing.T) {
			resp := uploadPicture(t, ts.URL, strings.NewReader(tc.img))
			_, e := readErrorResponse(t, resp)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Equal(t, errorResponse{Error: tc.want, Details: "can't save image", Code: 0}, e)
		})
	}
}

// an upload over the in-memory limit spills to a temp file; a failure there keeps the temp path out of the body
func TestRest_SavePictureUploadSpillError(t *testing.T) {
	ts, _ := startupT(t)
	logs := captureLog(t)
	missing := filepath.Join(t.TempDir(), "missing")
	t.Setenv("TMPDIR", missing)

	resp := uploadPicture(t, ts.URL, bytes.NewReader(make([]byte, 6*1024*1024)))
	body, e := readErrorResponse(t, resp)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, errorResponse{Error: "upload storage error", Details: "can't parse multipart form", Code: 2}, e,
		"the file error is replaced in the body")
	assert.NotContains(t, body, missing, "no server path in the body")
	assert.Contains(t, logs.String(), missing, "the file error, with the path, is still logged")
}

// a request error from the multipart parser carries no path and keeps its text
func TestRest_SavePictureNotMultipart(t *testing.T) {
	ts, _ := startupT(t)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/picture?site=remark42", strings.NewReader(`{}`))
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")
	resp, err := sendReq(req, devToken)
	require.NoError(t, err)
	_, e := readErrorResponse(t, resp)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, errorResponse{Error: "request Content-Type isn't multipart/form-data", Details: "can't parse multipart form",
		Code: 2}, e, "the parser error is sent as it is")
}

// uploadPicture posts img as the "file" field of POST /picture, signed in as the dev user
func uploadPicture(t *testing.T, url string, img io.Reader) *http.Response {
	t.Helper()
	return uploadPictureAs(t, url, devToken, img)
}

// uploadPictureAs posts img as the "file" field of POST /picture with the session token tkn
func uploadPictureAs(t *testing.T, url, tkn string, img io.Reader) *http.Response {
	t.Helper()
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("file", "picture.png")
	require.NoError(t, err)
	_, err = io.Copy(fileWriter, img)
	require.NoError(t, err)
	require.NoError(t, bodyWriter.Close())

	req, err := http.NewRequest(http.MethodPost, url+"/api/v1/picture?site=remark42", bodyBuf)
	require.NoError(t, err)
	req.Header.Add("Content-Type", bodyWriter.FormDataContentType())
	resp, err := sendReq(req, tkn)
	require.NoError(t, err)
	return resp
}

func TestRest_VoteSameIP(t *testing.T) {
	ts, srv := startupT(t, func(srv *Rest) { srv.DataService.RestrictSameIPVotes.Enabled = true })

	id := addComment(t, store.Comment{Text: "test test #1",
		Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"}}, ts)

	voteURL := fmt.Sprintf("%s/api/v1/vote/%s?site=remark42&url=https://radio-t.com/blah&vote=1", ts.URL, id)
	_, code := call(t, http.MethodPut, voteURL, "", dev2Token)
	require.Equal(t, http.StatusOK, code, "first vote from the address accepted")

	// another user from the same address
	body, code := call(t, http.MethodPut, voteURL, "", emailUserToken)
	e := decodeError(t, body)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Equal(t, errorResponse{Error: "the same ip already voted for " + id, Details: "can't vote for comment",
		Code: 14}, e, "rejected as a repeated vote, with no address hash")
	secret, err := srv.DataService.AdminStore.Key("remark42")
	require.NoError(t, err)
	assert.NotContains(t, body, store.HashValue("127.0.0.1", secret), "no address hash in the body")
}

func TestAdmin_DeleteUserAvatarErrorBody(t *testing.T) {
	ts, srv := startupT(t)
	logs := captureLog(t)

	c := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"},
		User: store.User{Name: "user1", ID: "user1", Picture: "https://demo.remark42.com/api/v1/avatar/pic.image"}}
	_, err := srv.DataService.Create(c)
	require.NoError(t, err)
	avatarFile := blockAvatarRemoval(t, srv)

	body, code := call(t, http.MethodDelete, ts.URL+"/api/v1/admin/user/user1?site=remark42", "", adminPasswd)
	e := decodeError(t, body)

	assert.Equal(t, http.StatusInternalServerError, code)
	assert.Equal(t, errorResponse{Error: "avatar store error", Details: "can't delete user's avatar", Code: 0}, e,
		"the store error is replaced in the body")
	assert.NotContains(t, body, serverDir(srv), "no server path in the body")
	assert.Contains(t, logs.String(), avatarFile, "the store error, with the path, is still logged")
}

func TestAdmin_DeleteMeRequestAvatarErrorBody(t *testing.T) {
	ts, srv := startupT(t)
	logs := captureLog(t)

	c := store.Comment{Text: "test test #1", Locator: store.Locator{SiteID: "remark42", URL: "https://radio-t.com/blah"},
		User: store.User{Name: "user1", ID: "user1"}}
	_, err := srv.DataService.Create(c)
	require.NoError(t, err)
	avatarFile := blockAvatarRemoval(t, srv)

	tkn := deleteMeToken(t, srv, "remark42", "user1", "https://demo.remark42.com/api/v1/avatar/pic.image")
	body, code := call(t, http.MethodGet, ts.URL+"/api/v1/admin/deleteme?token="+tkn, "", adminPasswd)
	e := decodeError(t, body)

	assert.Equal(t, http.StatusInternalServerError, code)
	assert.Equal(t, errorResponse{Error: "avatar store error", Details: "can't delete user's avatar", Code: 0}, e,
		"the store error is replaced in the body")
	assert.NotContains(t, body, serverDir(srv), "no server path in the body")
	assert.Contains(t, logs.String(), avatarFile, "the store error, with the path, is still logged")
}

// blockAvatarRemoval puts a non-empty directory where the avatar pic.image belongs, so removing it fails
// with an error other than not found, and returns that path
func blockAvatarRemoval(t *testing.T, srv *Rest) string {
	t.Helper()
	avatarFile := testAvatarDir(srv) + "/42/pic.image" // "pic" hashes to partition 42
	require.NoError(t, os.MkdirAll(avatarFile, 0o700))
	require.NoError(t, os.WriteFile(avatarFile+"/child", []byte("x"), 0o600))
	return avatarFile
}

func TestMigrator_ImportTempFileError(t *testing.T) {
	ts, _ := startupT(t)
	logs := captureLog(t)
	missing := filepath.Join(t.TempDir(), "missing")
	t.Setenv("TMPDIR", missing)

	authts := strings.Replace(ts.URL, "http://", "http://admin:password@", 1)

	t.Run("body", func(t *testing.T) {
		resp, err := http.Post(authts+"/api/v1/admin/import?site=remark42&provider=native", "application/json",
			strings.NewReader(`{"version":1}`))
		require.NoError(t, err)
		body, e := readErrorResponse(t, resp)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Equal(t, errorResponse{Error: "temp file error", Details: "can't save request to temp file", Code: 0}, e,
			"the file error is replaced in the body")
		assert.NotContains(t, body, missing, "no server path in the body")
	})

	t.Run("form", func(t *testing.T) {
		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		fileWriter, err := bodyWriter.CreateFormFile("file", "import.json")
		require.NoError(t, err)
		_, err = fileWriter.Write([]byte(`{"version":1}`))
		require.NoError(t, err)
		require.NoError(t, bodyWriter.Close())
		resp, err := http.Post(authts+"/api/v1/admin/import/form?site=remark42&provider=native",
			bodyWriter.FormDataContentType(), bodyBuf)
		require.NoError(t, err)
		body, e := readErrorResponse(t, resp)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		assert.Equal(t, errorResponse{Error: "temp file error", Details: "can't save request to temp file", Code: 0}, e,
			"the file error is replaced in the body")
		assert.NotContains(t, body, missing, "no server path in the body")
	})

	assert.Contains(t, logs.String(), missing, "the file error, with the path, is still logged")
}
