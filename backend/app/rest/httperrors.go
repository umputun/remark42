package rest

import (
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	"github.com/go-chi/render"
	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
)

// All error codes for UI mapping and translation
const (
	ErrInternal              = 0  // any internal error
	ErrCommentNotFound       = 1  // can't find comment
	ErrDecode                = 2  // failed to unmarshal incoming request
	ErrNoAccess              = 3  // rejected by auth
	ErrCommentValidation     = 4  // validation failed
	ErrPostNotFound          = 5  // can't find post
	ErrSiteNotFound          = 6  // can't find site
	ErrUserBlocked           = 7  // user blocked
	ErrReadOnly              = 8  // write failed on read only
	ErrCommentRejected       = 9  // general error on rejected comment change
	ErrCommentEditExpired    = 10 // too late for edit
	ErrCommentEditChanged    = 11 // parent comment cannot be changed
	ErrVoteRejected          = 12 // general error on vote rejected
	ErrVoteSelf              = 13 // vote for own comment
	ErrVoteDbl               = 14 // already voted for the comment
	ErrVoteMax               = 15 // too many votes for the comment
	ErrVoteMinScore          = 16 // min score reached for the comment
	ErrActionRejected        = 17 // general error for rejected actions
	ErrAssetNotFound         = 18 // requested file not found
	ErrNotificationsDisabled = 19 // required notification method is not enabled
)

// SendErrorJSON makes {error: blah, details: blah} json body and responds with error code
func SendErrorJSON(w http.ResponseWriter, r *http.Request, httpStatusCode int, err error, details string, errCode int) {
	log.Printf("[WARN] %s", errDetailsMsg(r, httpStatusCode, err, details, errCode))
	render.Status(r, httpStatusCode)
	render.JSON(w, r, rest.JSON{"error": err.Error(), "details": details, "code": errCode})
}

func errDetailsMsg(r *http.Request, httpStatusCode int, err error, details string, errCode int) string {
	uinfoStr := ""
	if user, e := GetUserInfo(r); e == nil {
		uinfoStr = user.Name + "/" + user.ID + " - "
	}
	q := r.URL.String()
	if qun, e := url.QueryUnescape(q); e == nil {
		q = qun
	}

	srcFileInfo := ""
	if pc, file, line, ok := runtime.Caller(2); ok {
		fnameElems := strings.Split(file, "/")
		funcNameElems := strings.Split(runtime.FuncForPC(pc).Name(), "/")
		srcFileInfo = fmt.Sprintf("[%s:%d %s]", strings.Join(fnameElems[len(fnameElems)-3:], "/"),
			line, funcNameElems[len(funcNameElems)-1])
	}

	return fmt.Sprintf("%s - %v - %d (%d) - %s%s - %s",
		details, err, httpStatusCode, errCode, uinfoStr, q, srcFileInfo)
}
