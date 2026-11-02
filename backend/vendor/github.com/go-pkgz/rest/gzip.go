package rest

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var gzDefaultContentTypes = []string{
	"text/css",
	"text/javascript",
	"text/xml",
	"text/html",
	"text/plain",
	"application/javascript",
	"application/x-javascript",
	"application/json",
}

var gzPool = sync.Pool{
	New: func() any { return gzip.NewWriter(io.Discard) },
}

// gzipResponseWriter defers the compression decision until the response content type is known,
// either from the header the handler set or sniffed from the first chunk of the body.
//
// One consequence is worth knowing about: when the handler calls WriteHeader without a Content-Type,
// the status cannot be sent yet, because the body has to be sniffed first. Headers changed between
// that call and the first Write therefore still reach the client, where net/http would have ignored
// them. Handlers that mutate headers after WriteHeader are relying on a no-op, so this is more
// permissive rather than wrong, but it is a difference from the bare ResponseWriter.
type gzipResponseWriter struct {
	http.ResponseWriter

	gzCts       []string
	gz          *gzip.Writer
	status      int
	statusSet   bool
	decided     bool
	wroteHeader bool
	hijacked    bool
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	// 1xx are interim responses, they pass straight through and the final status still follows.
	// 101 is the exception, it hands the connection to another protocol and is final.
	if status >= 100 && status < 200 && status != http.StatusSwitchingProtocols {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	if w.statusSet {
		return // net/http keeps the first final status, so later calls are ignored here too
	}
	w.statusSet = true
	w.status = status

	// with a content type in hand the decision can be made right away, otherwise it waits for the
	// first Write so the body can be sniffed. 101 cannot wait: the upgrade sequence carries no body
	// and usually no content type, and the Hijack that follows would leave the status unsent
	if ctype := w.Header().Get("Content-Type"); ctype != "" || status == http.StatusSwitchingProtocols {
		w.decide(ctype)
		w.commit()
	}
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.decided {
		ctype := w.Header().Get("Content-Type")
		// net/http suppresses sniffing for an already encoded body, guessing a type from
		// compressed bytes would only mislabel it
		if ctype == "" && w.Header().Get("Content-Encoding") == "" {
			ctype = http.DetectContentType(b)
			w.Header().Set("Content-Type", ctype)
		}
		w.decide(ctype)
	}
	if !w.wroteHeader {
		w.commit()
	}
	if w.gz != nil {
		return w.gz.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// decide turns compression on if the response content type is one of the configured types
func (w *gzipResponseWriter) decide(ctype string) {
	w.decided = true

	switch w.status {
	case http.StatusSwitchingProtocols, http.StatusNoContent, http.StatusResetContent, http.StatusNotModified:
		return // these carry no body to compress
	}
	if w.Header().Get("Content-Encoding") != "" {
		return // the handler encoded the body itself, wrapping it again would mislabel the result
	}
	if w.status == http.StatusPartialContent || w.Header().Get("Content-Range") != "" {
		return // the range metadata describes the identity representation
	}

	for _, c := range w.gzCts {
		if !strings.HasPrefix(strings.ToLower(ctype), strings.ToLower(c)) {
			continue
		}
		gz := gzPool.Get().(*gzip.Writer)
		gz.Reset(w.ResponseWriter)
		w.gz = gz
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length") // the handler's length describes the uncompressed body
		return
	}
}

func (w *gzipResponseWriter) commit() {
	w.wroteHeader = true
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.ResponseWriter.WriteHeader(w.status)
}

// close finishes the gzip stream and makes sure the status reaches the client even if the handler
// wrote no body at all. finished reports whether the handler returned normally: when it panicked
// instead, an uncommitted response is left alone so a recoverer upstream can still make it a 500.
func (w *gzipResponseWriter) close(finished bool) {
	if w.hijacked {
		return // the handler owns the connection now, nothing may be written to it
	}
	if w.wroteHeader || finished {
		if !w.decided {
			w.decide(w.Header().Get("Content-Type"))
		}
		if !w.wroteHeader {
			w.commit()
		}
	}
	if w.gz == nil {
		return
	}
	_ = w.gz.Close()
	gzPool.Put(w.gz)
	w.gz = nil
}

// flush pushes buffered data out, keeping streaming responses working through the compressor
func (w *gzipResponseWriter) flush() {
	if w.hijacked {
		return
	}
	// decide before the headers leave, otherwise a later write could be compressed after the client
	// was already told the body is identity
	if !w.decided {
		w.decide(w.Header().Get("Content-Type"))
	}
	if !w.wroteHeader {
		w.commit()
	}
	if w.gz != nil {
		_ = w.gz.Flush()
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// hijack passes through to the underlying writer for protocol upgrades
func (w *gzipResponseWriter) hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("http.Hijacker not supported")
	}
	// finish the stream first, whatever the handler already wrote has to reach the wire before the
	// connection changes hands, and a failure there means truncated output rather than something to
	// swallow behind a successful hijack
	if w.gz != nil {
		if err := w.gz.Close(); err != nil {
			return nil, nil, fmt.Errorf("finish gzip stream before hijack: %w", err)
		}
	}

	conn, rw, err := h.Hijack()
	if err != nil {
		// the connection was not taken over, so the writer stays attached and closed: a handler that
		// carries on writing now gets an error instead of appending raw bytes to a body already
		// advertised as gzip, and the deferred close still returns the writer to the pool
		return nil, nil, err
	}

	if w.gz != nil {
		gzPool.Put(w.gz)
		w.gz = nil
	}
	w.hijacked = true
	return conn, rw, nil
}

// the wrapper must offer exactly the optional interfaces the underlying writer has, otherwise a
// handler's type assertion succeeds and the call then does nothing, which is how http.TimeoutHandler
// (offering neither) would silently lose a Flush
type gzipFlusher struct{ *gzipResponseWriter }

func (w gzipFlusher) Flush() { w.flush() }

type gzipHijacker struct{ *gzipResponseWriter }

func (w gzipHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) { return w.hijack() }

type gzipFlushHijacker struct{ *gzipResponseWriter }

func (w gzipFlushHijacker) Flush() { w.flush() }

func (w gzipFlushHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) { return w.hijack() }

// wrapGzipWriter picks the variant matching the capabilities of the writer underneath
func wrapGzipWriter(gw *gzipResponseWriter) http.ResponseWriter {
	_, isFlusher := gw.ResponseWriter.(http.Flusher)
	_, isHijacker := gw.ResponseWriter.(http.Hijacker)

	switch {
	case isFlusher && isHijacker:
		return gzipFlushHijacker{gw}
	case isFlusher:
		return gzipFlusher{gw}
	case isHijacker:
		return gzipHijacker{gw}
	}
	return gw
}

// Unwrap exposes the underlying writer to http.ResponseController
func (w *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// Gzip is a middleware compressing response. The decision is made on the response content type,
// so it applies to what the handler actually produced. Content types default to the common textual
// ones and can be overridden by the caller.
func Gzip(contentTypes ...string) func(http.Handler) http.Handler {

	gzCts := gzDefaultContentTypes
	if len(contentTypes) > 0 {
		gzCts = contentTypes
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// the representation depends on Accept-Encoding, caches must key on it even when not compressing
			w.Header().Add("Vary", "Accept-Encoding")

			if !acceptsGzip(r.Header.Values("Accept-Encoding")) {
				next.ServeHTTP(w, r)
				return
			}

			gw := &gzipResponseWriter{ResponseWriter: w, gzCts: gzCts}
			finished := false
			defer func() { gw.close(finished) }()

			next.ServeHTTP(wrapGzipWriter(gw), r)
			finished = true
		})
	}
}

// acceptsGzip reports whether the client accepts gzip, honoring an explicit q=0 rejection.
// A named gzip entry decides the answer on its own, as it takes precedence over the "*" wildcard.
// Repeated Accept-Encoding fields form a single list, so every field is examined.
func acceptsGzip(headers []string) bool {
	var wildcard, wildcardSeen bool

	for _, header := range headers {
		for enc := range strings.SplitSeq(header, ",") {
			name, params, _ := strings.Cut(strings.TrimSpace(enc), ";")
			n := strings.ToLower(strings.TrimSpace(name))
			if n != "gzip" && n != "*" {
				continue
			}
			acceptable := !rejectedByQuality(params)
			if n == "gzip" {
				return acceptable
			}
			if !wildcardSeen {
				wildcard, wildcardSeen = acceptable, true
			}
		}
	}
	return wildcard
}

// rejectedByQuality reports whether the parameters of an Accept-Encoding entry carry q=0
func rejectedByQuality(params string) bool {
	q, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(params)), "q=")
	if !ok {
		return false
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(q), 64)
	return err == nil && v == 0
}
