package tollbooth_chi

import (
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"net/http"
)

func LimitHandler(lmt *limiter.Limiter) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		wrapper := &limiterWrapper{
			lmt: lmt,
		}

		wrapper.handler = handler
		return wrapper
	}
}

type limiterWrapper struct {
	lmt     *limiter.Limiter
	handler http.Handler
}

func (l *limiterWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	select {
	case <-ctx.Done():
		http.Error(w, "Context was canceled", http.StatusServiceUnavailable)
		return

	default:
		httpError := tollbooth.LimitByRequest(l.lmt, w, r)
		if httpError != nil {
			w.Header().Add("Content-Type", l.lmt.GetMessageContentType())
			w.WriteHeader(httpError.StatusCode)
			w.Write([]byte(httpError.Message))
			return
		}

		l.handler.ServeHTTP(w, r)
	}
}
