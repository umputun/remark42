package rest

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/rest"
	"github.com/go-pkgz/rest/logger"

	"github.com/umputun/remark42/backend/app/store"
	"github.com/umputun/remark42/backend/app/templates"
)

// All error codes for UI mapping and translation
const (
	ErrInternal             = 0  // any internal error
	ErrCommentNotFound      = 1  // can't find comment
	ErrDecode               = 2  // failed to unmarshal incoming request
	ErrNoAccess             = 3  // rejected by auth
	ErrCommentValidation    = 4  // validation failed
	ErrPostNotFound         = 5  // can't find post
	ErrSiteNotFound         = 6  // can't find site
	ErrUserBlocked          = 7  // user blocked
	ErrReadOnly             = 8  // write failed on read only
	ErrCommentRejected      = 9  // general error on rejected comment change
	ErrCommentEditExpired   = 10 // too late for edit
	ErrCommentEditChanged   = 11 // parent comment cannot be changed
	ErrVoteRejected         = 12 // general error on vote rejected
	ErrVoteSelf             = 13 // vote for own comment
	ErrVoteDbl              = 14 // already voted for the comment
	ErrVoteMax              = 15 // too many votes for the comment
	ErrVoteMinScore         = 16 // min score reached for the comment
	ErrActionRejected       = 17 // general error for rejected actions
	ErrAssetNotFound        = 18 // requested file not found
	ErrCommentRestrictWords = 19 // restricted words in a comment
	ErrImgNotFound          = 20 // posted image not found in the storage
)

// errTmplData store data for error message
type errTmplData struct {
	Error   string
	Details string
}

// SendErrorHTML makes html body from error_response.html.tmpl template and responds with provided http status code,
// error code is not included in render as it is intended for UI developers and not for the users
func SendErrorHTML(w http.ResponseWriter, r *http.Request, httpStatusCode int, err error, details string, errCode int) {
	log.Printf("[WARN] %s", errDetailsMsg(r, httpStatusCode, err, details, errCode, 0))
	page, e := RenderTemplate("error", "error_response.html.tmpl", errTmplData{Error: err.Error(), Details: details})
	if e != nil {
		log.Printf("[WARN] can't make error page, %v", e)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	HTMLResponse(w, httpStatusCode, page)
}

// RenderTemplate executes html template file under the given name with data. The file is read from the
// working directory when present there, so an operator can override it, and from the embedded copy
// otherwise. An override may wrap its content in {{define "<name>"}}, so the name is kept independent of
// the file name.
func RenderTemplate(name, file string, data any) (string, error) {
	content, err := templates.Read(file)
	if err != nil {
		return "", fmt.Errorf("can't read template %s: %w", file, err)
	}
	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("can't parse template %s: %w", file, err)
	}
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("can't execute template %s: %w", file, err)
	}
	return buf.String(), nil
}

// SendErrorJSON makes {error: blah, details: blah, code: 42} json body and responds with error code
func SendErrorJSON(w http.ResponseWriter, r *http.Request, httpStatusCode int, err error, details string, errCode int) {
	log.Printf("[WARN] %s", errDetailsMsg(r, httpStatusCode, err, details, errCode, 0))
	writeErrorJSON(w, httpStatusCode, err, details, errCode)
}

// SendErrorJSONAs is SendErrorJSON for an error whose text is for the server log only, such as one naming
// a file on the server: err is logged in full and the body carries sent in its place
func SendErrorJSONAs(w http.ResponseWriter, r *http.Request, httpStatusCode int, err, sent error, details string, errCode int) {
	log.Printf("[WARN] %s", errDetailsMsg(r, httpStatusCode, err, details, errCode, 0))
	writeErrorJSON(w, httpStatusCode, sent, details, errCode)
}

// SendErrorJSONFromHelper is SendErrorJSON for a helper which sends errors for its callers: the logged
// source line is where the helper was called, not the line inside the helper
func SendErrorJSONFromHelper(w http.ResponseWriter, r *http.Request, httpStatusCode int, err error, details string, errCode int) {
	log.Printf("[WARN] %s", errDetailsMsg(r, httpStatusCode, err, details, errCode, 1))
	writeErrorJSON(w, httpStatusCode, err, details, errCode)
}

func writeErrorJSON(w http.ResponseWriter, httpStatusCode int, err error, details string, errCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)
	rest.RenderJSON(w, rest.JSON{"error": err.Error(), "details": details, "code": errCode})
}

// HTMLResponse writes HTML content with the given status code
func HTMLResponse(w http.ResponseWriter, status int, html string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(html))
}

// PlainTextResponse writes plain text content with the given status code
func PlainTextResponse(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(text))
}

// errDetailsMsg makes the log line of an error sent to the client. It names the source line which called
// the function calling errDetailsMsg, or the one skip frames further up the stack.
func errDetailsMsg(r *http.Request, httpStatusCode int, err error, details string, errCode, skip int) string {
	uinfoStr := ""
	if user, e := GetUserInfo(r); e == nil {
		uinfoStr = user.Name + "/" + user.ID + " - "
	}
	q := r.URL.String()
	if qun, e := url.QueryUnescape(q); e == nil {
		q = qun
	}

	srcFileInfo := ""
	if pc, file, line, ok := runtime.Caller(2 + skip); ok {
		fnameElems := strings.Split(file, "/")
		funcNameElems := strings.Split(runtime.FuncForPC(pc).Name(), "/")
		srcFileInfo = fmt.Sprintf("[%s:%d %s]", strings.Join(fnameElems[len(fnameElems)-3:], "/"),
			line, funcNameElems[len(funcNameElems)-1])
	}

	// the url and the error can carry the user's email address, as the email login and subscription requests do
	return store.MaskForLog(fmt.Sprintf("%s - %v - %d (%d) - %s%s - %s",
		details, err, httpStatusCode, errCode, uinfoStr, q, srcFileInfo))
}

// MaskedLog is a log backend passing every line through store.MaskForLog, for logs that record user input
type MaskedLog struct {
	L logger.Backend
}

// Logf formats the line and writes it to L with email addresses and tokens hidden
func (m MaskedLog) Logf(format string, args ...any) {
	m.L.Logf("%s", store.MaskForLog(fmt.Sprintf(format, args...)))
}
