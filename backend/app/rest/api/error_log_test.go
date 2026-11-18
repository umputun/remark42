package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-pkgz/lgr"
	"github.com/stretchr/testify/assert"
)

// a picture request rejected through sendPictureError logs the handler's line, not the helper's
func TestRest_ErrorLogNamesHandler(t *testing.T) {
	logs := &lockedBuffer{}
	lgr.Setup(lgr.Out(logs), lgr.Err(logs))
	t.Cleanup(func() { lgr.Setup() })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/picture/user/..", http.NoBody)
	req.SetPathValue("user", "user")
	req.SetPathValue("id", "..")
	rec := httptest.NewRecorder()
	(&public{}).loadPictureCtrl(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, logs.String(), "invalid picture id")
	assert.Contains(t, logs.String(), " api.(*public).loadPictureCtrl]")
	assert.NotContains(t, logs.String(), "sendPictureError")
}
