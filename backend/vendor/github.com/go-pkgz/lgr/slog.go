package lgr

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ToSlogHandler converts lgr.L to slog.Handler. Attributes are written as key=value pairs after the message,
// string values quoted, with group names joined to keys by dots as the slog text handler does.
// For *Logger and types embedding it, secrets in string values are hidden before quoting. For *Logger,
// the caller reported by the Caller* options is the one in the slog record, so CallerDepth is not needed.
func ToSlogHandler(l L) slog.Handler {
	return &lgrSlogHandler{lgr: l}
}

// FromSlogHandler creates lgr.L wrapper around slog.Handler. Records below the level the handler accepts are dropped.
// FATAL and PANIC are passed as ERROR records and then exit with os.Exit(1), as Logger does;
// PANIC writes a fixed heading and the stacks of all goroutines to stderr first; the message goes to the handler only.
func FromSlogHandler(h slog.Handler) L {
	return &slogLgrAdapter{handler: h, stderr: os.Stderr, fatal: func() { os.Exit(1) }}
}

// SetupWithSlog sets up the global logger with a slog logger.
// Debug or Trace is turned on when the handler accepts slog.LevelDebug or slog.LevelDebug-4.
func SetupWithSlog(logger *slog.Logger) {
	options := []Option{SlogHandler(logger.Handler())}

	// check the lowest level the slog handler is enabled for
	// and allow it in lgr too, to prevent filtering
	switch {
	case logger.Handler().Enabled(context.Background(), slog.LevelDebug-4):
		options = append(options, Trace)
	case logger.Handler().Enabled(context.Background(), slog.LevelDebug):
		options = append(options, Debug)
	}

	Setup(options...)
}

// lgrSlogHandler implements slog.Handler using lgr.L
type lgrSlogHandler struct {
	lgr    L
	attrs  []groupedAttr // attributes added by WithAttrs, with the groups open at the time
	prefix string        // groups opened by WithGroup, each followed by a dot
}

// groupedAttr is an attribute with the group prefix it was added under
type groupedAttr struct {
	prefix string
	attr   slog.Attr
}

// Enabled implements slog.Handler
func (h *lgrSlogHandler) Enabled(_ context.Context, level slog.Level) bool {
	switch {
	case level < slog.LevelInfo: // debug, Trace
		// an lgr.L has no level to query, so every level is reported enabled,
		// and Handle drops what a wrapped *Logger filters out
		return true
	default:
		return true
	}
}

// Handle implements slog.Handler
func (h *lgrSlogHandler) Handle(_ context.Context, record slog.Record) error {
	level := levelToString(record.Level)
	// a record the wrapped logger drops returns before resolving attributes runs LogValue for nothing
	if l, isLogger := h.lgr.(*Logger); isLogger && l.skips(level) {
		return nil
	}

	// build message with attributes
	msg := record.Message

	// string values are quoted, so secrets are hidden before quoting changes how they are spelled.
	// the interface also matches types that embed *Logger; a nil pointer to such a type may still
	// log through its own Logf, while the promoted hideSecrets would dereference it
	hide := func(s string) string { return s }
	if hl, ok := h.lgr.(interface{ hideSecrets(string) string }); ok && !h.isNilPointer() {
		hide = hl.hideSecrets
	}

	// format attributes as key=value pairs
	var attrs strings.Builder

	// add pre-defined attributes
	for _, a := range h.attrs {
		h.writeAttr(&attrs, a.prefix, a.attr, hide)
	}

	// add record attributes
	record.Attrs(func(attr slog.Attr) bool {
		h.writeAttr(&attrs, h.prefix, attr, hide)
		return true
	})

	// combine level prefix and message; lgr.Logf adds its own timestamp and level formatting.
	// the text is passed as an argument, so a % in it is not taken for a formatting verb
	logMsg := level + " " + msg
	if attrs.Len() > 0 {
		logMsg += " " + attrs.String()
	}
	if l, isLogger := h.lgr.(*Logger); isLogger && record.PC != 0 {
		l.logf(record.PC, "%s", logMsg) // report the caller slog recorded, not this method
		return nil
	}
	h.lgr.Logf("%s", logMsg)
	return nil
}

// WithAttrs implements slog.Handler
func (h *lgrSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	// a new slice each time, so handlers derived from one parent never share a backing array
	res := make([]groupedAttr, 0, len(h.attrs)+len(attrs))
	res = append(res, h.attrs...)
	for _, a := range attrs {
		res = append(res, groupedAttr{prefix: h.prefix, attr: a})
	}
	return &lgrSlogHandler{lgr: h.lgr, attrs: res, prefix: h.prefix}
}

// WithGroup implements slog.Handler. Attributes added before the group keep their keys.
func (h *lgrSlogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &lgrSlogHandler{lgr: h.lgr, attrs: h.attrs, prefix: h.prefix + name + "."}
}

// slogLgrAdapter implements lgr.L using slog.Handler
type slogLgrAdapter struct {
	handler slog.Handler
	stderr  io.Writer // handler errors and the PANIC dump
	fatal   func()    // called after FATAL and PANIC, os.Exit(1) outside of tests
}

// Logf implements lgr.L interface
func (a *slogLgrAdapter) Logf(format string, args ...interface{}) {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	lv, msg, found := extractLevel(msg)
	if found && msg != "" && (msg[0] == ' ' || msg[0] == '\t') {
		msg = msg[1:] // only the separator goes, the rest reaches the handler as it is, so its redaction still matches
	}
	level := stringToLevel(lv)

	if a.handler.Enabled(context.Background(), level) {
		// get the caller's PC so slog handlers can resolve source info when AddSource is enabled
		var pcs [1]uintptr
		runtime.Callers(2, pcs[:]) // skip runtime.Callers and Logf
		record := slog.NewRecord(time.Now(), level, msg, pcs[0])

		if err := a.handler.Handle(context.Background(), record); err != nil {
			_, _ = fmt.Fprintf(a.stderr, "slog handler error: %v\n", err)
		}
	}

	// FATAL and PANIC stop the program after the record is handled, as they do in Logger.
	// the message is left to the handler, which may mask it, so stderr gets only the heading and the dump
	if lv == "FATAL" || lv == "PANIC" {
		if lv == "PANIC" {
			_, _ = fmt.Fprintf(a.stderr, "\n*** PANIC\n\n%s", getDump())
		}
		a.fatal()
	}
}

// Helper functions

// isNilPointer reports whether the wrapped logger is a nil pointer
func (h *lgrSlogHandler) isNilPointer() bool {
	rv := reflect.ValueOf(h.lgr)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

// levelToString converts slog.Level to string representation used by lgr
func levelToString(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		if level <= slog.LevelDebug-4 {
			return "TRACE"
		}
		return "DEBUG"
	case level < slog.LevelWarn:
		return "INFO"
	case level < slog.LevelError:
		return "WARN"
	default:
		return "ERROR"
	}
}

// stringToLevel converts lgr level string to slog.Level
func stringToLevel(level string) slog.Level {
	switch level {
	case "TRACE":
		return slog.LevelDebug - 4
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR", "PANIC", "FATAL":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// writeAttr writes attr to b as a key=value pair followed by a space, with the group prefix added to the key.
// Group values are written as one pair per member, their keys joined by dots, and empty groups are skipped.
// String values are quoted, after hide has replaced the secrets in them.
func (h *lgrSlogHandler) writeAttr(b *strings.Builder, prefix string, attr slog.Attr, hide func(string) string) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}

	switch attr.Value.Kind() {
	case slog.KindGroup:
		if attr.Key != "" { // a group without a key adds its members at the current level
			prefix += attr.Key + "."
		}
		for _, member := range attr.Value.Group() {
			h.writeAttr(b, prefix, member, hide)
		}
	case slog.KindString:
		b.WriteString(prefix + attr.Key + "=" + strconv.Quote(hide(attr.Value.String())) + " ")
	default:
		b.WriteString(prefix + attr.Key + "=" + attr.Value.String() + " ")
	}
}
