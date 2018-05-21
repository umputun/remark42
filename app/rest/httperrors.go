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
	log.Printf("[DEBUG] %s", errDetailsMsg(r, code, err, details))
	render.Status(r, code)
	render.JSON(w, r, map[string]interface{}{"error": err.Error(), "details": details})
}

func errDetailsMsg(r *http.Request, code int, err error, details string) string {
	uinfoStr := ""
	if user, e := GetUserInfo(r); e == nil {
		uinfoStr = user.Name + "/" + user.ID + " - "
	}

	q := r.URL.String()
	if qun, e := url.QueryUnescape(q); e == nil {
		q = qun
	}

	srcFileInfo := ""
	if _, file, line, ok := runtime.Caller(2); ok {
		fnameElems := strings.Split(file, "/")
		srcFileInfo = fmt.Sprintf(" [caused by %s:%d]", strings.Join(fnameElems[len(fnameElems)-3:], "/"), line)
	}

	return fmt.Sprintf("%s - %v - %d - %s%s - %s%s",
		details, err, code, uinfoStr, strings.Split(r.RemoteAddr, ":")[0], q, srcFileInfo)
}
