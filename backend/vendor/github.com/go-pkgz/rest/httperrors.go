package rest

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strings"

	"github.com/go-pkgz/rest/logger"
)

// SendErrorJSON sends {error: msg} with error code and logging error and caller
func SendErrorJSON(w http.ResponseWriter, r *http.Request, l logger.Backend, code int, err error, msg string) {
	if l != nil {
		l.Logf("%s", errDetailsMsg(r, code, err, msg))
	}
	w.WriteHeader(code)
	RenderJSON(w, r, JSON{"error": msg})
}

func errDetailsMsg(r *http.Request, code int, err error, msg string) string {

	q := r.URL.String()
	if qun, e := url.QueryUnescape(q); e == nil {
		q = qun
	}

	srcFileInfo := ""
	if pc, file, line, ok := runtime.Caller(2); ok {
		fnameElems := strings.Split(file, "/")
		funcNameElems := strings.Split(runtime.FuncForPC(pc).Name(), "/")
		srcFileInfo = fmt.Sprintf(" [caused by %s:%d %s]", strings.Join(fnameElems[len(fnameElems)-3:], "/"),
			line, funcNameElems[len(funcNameElems)-1])
	}

	remoteIP := r.RemoteAddr
	if pos := strings.Index(remoteIP, ":"); pos >= 0 {
		remoteIP = remoteIP[:pos]
	}
	if err == nil {
		err = errors.New("no error")
	}
	return fmt.Sprintf("%s - %v - %d - %s - %s%s", msg, err, code, remoteIP, q, srcFileInfo)
}
