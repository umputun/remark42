package rest

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	"github.com/go-chi/render"
)

// SendErrorJSON makes {error: blah, details: blah} json body and responds with error code
func SendErrorJSON(w http.ResponseWriter, r *http.Request, code int, err error, details string) {
	logDetails(r, code, err, details)
	render.Status(r, code)
	render.JSON(w, r, map[string]interface{}{"error": err.Error(), "details": details})
}

func logDetails(r *http.Request, code int, err error, details string) {
	uinfoStr := ""
	if user, е := GetUserInfo(r); е == nil {
		uinfoStr = user.Name + "/" + user.ID + " - "
	}

	q := r.URL.String()
	if qun, е := url.QueryUnescape(q); е == nil {
		q = qun
	}

	srcFileInfo := ""
	if _, file, line, ok := runtime.Caller(2); ok {
		fnameElems := strings.Split(file, "/")
		srcFileInfo = fmt.Sprintf(" [caused by %s:%d]", strings.Join(fnameElems[len(fnameElems)-3:], "/"), line)
	}

	log.Printf("[DEBUG] %s - %v - %d - %s%s - %s%s",
		details, err, code, uinfoStr, strings.Split(r.RemoteAddr, ":")[0], q, srcFileInfo)
}
