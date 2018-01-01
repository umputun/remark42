package common

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/render"
)

// SendErrorJSON makes {error: blah, details: blah} json body and responds with error code
func SendErrorJSON(w http.ResponseWriter, r *http.Request, code int, err error, details string) {
	logDetails(r, code, err, details)
	render.Status(r, code)
	render.JSON(w, r, map[string]interface{}{"error": err.Error(), "details": details})
}

// SendErrorText with simple text body and responds with error code
func SendErrorText(w http.ResponseWriter, r *http.Request, code int, text string) {
	render.Status(r, code)
	render.PlainText(w, r, text)
}

func logDetails(r *http.Request, code int, err error, details string) {
	uinfoStr := ""
	if user, е := GetUserInfo(r); е == nil {
		uinfoStr = user.Name + "/" + user.ID + " - "
	}

	q := r.URL.String()
	if qun, err := url.QueryUnescape(q); err == nil {
		q = qun
	}

	log.Printf("[DEBUG] %s - %v - %d - %s%s - %s", details, err, code, uinfoStr, strings.Split(r.RemoteAddr, ":")[0], q)
}
