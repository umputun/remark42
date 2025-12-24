package lgr

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"
)

// ToSlogHandler converts lgr.L to slog.Handler
func ToSlogHandler(l L) slog.Handler {
	return &lgrSlogHandler{lgr: l}
}

// FromSlogHandler creates lgr.L wrapper around slog.Handler
func FromSlogHandler(h slog.Handler) L {
	return &slogLgrAdapter{handler: h}
}

// SetupWithSlog sets up the global logger with a slog logger
func SetupWithSlog(logger *slog.Logger) {
	options := []Option{SlogHandler(logger.Handler())}
	
	// check if the slog handler is enabled for debug level
	// if so, enable debug mode in lgr to prevent filtering
	if logger.Handler().Enabled(context.Background(), slog.LevelDebug) {
		options = append(options, Debug)
	}
	
	Setup(options...)
}

// lgrSlogHandler implements slog.Handler using lgr.L
type lgrSlogHandler struct {
	lgr    L
	attrs  []slog.Attr
	groups []string
}

// Enabled implements slog.Handler
func (h *lgrSlogHandler) Enabled(_ context.Context, level slog.Level) bool {
	switch {
	case level < slog.LevelInfo: // debug, Trace
		// check if underlying lgr logger is configured to show debug
		// since we can't directly query lgr's debug status, we assume enabled
		return true
	default:
		return true
	}
}

// Handle implements slog.Handler
func (h *lgrSlogHandler) Handle(_ context.Context, record slog.Record) error {
	level := levelToString(record.Level)

	// build message with attributes
	msg := record.Message

	// add time if record has it, otherwise current time is used by lgr
	var timeStr string
	if !record.Time.IsZero() {
		timeStr = record.Time.Format("2006/01/02 15:04:05.000 ")
	}

	// format attributes as key=value pairs
	var attrs strings.Builder
	if len(h.attrs) > 0 || record.NumAttrs() > 0 {
		attrs.WriteString(" ")
	}

	// add pre-defined attributes
	for _, attr := range h.attrs {
		attrs.WriteString(formatAttr(attr, h.groups))
	}

	// add record attributes
	record.Attrs(func(attr slog.Attr) bool {
		attrs.WriteString(formatAttr(attr, h.groups))
		return true
	})

	// combine everything into final message
	logMsg := fmt.Sprintf("%s%s %s%s", timeStr, level, msg, attrs.String())
	h.lgr.Logf(logMsg)
	return nil
}

// WithAttrs implements slog.Handler
func (h *lgrSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := &lgrSlogHandler{
		lgr:    h.lgr,
		attrs:  append(h.attrs, attrs...),
		groups: h.groups,
	}
	return newHandler
}

// WithGroup implements slog.Handler
func (h *lgrSlogHandler) WithGroup(name string) slog.Handler {
	newHandler := &lgrSlogHandler{
		lgr:    h.lgr,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
	return newHandler
}

// slogLgrAdapter implements lgr.L using slog.Handler
type slogLgrAdapter struct {
	handler slog.Handler
}

// Logf implements lgr.L interface
func (a *slogLgrAdapter) Logf(format string, args ...interface{}) {
	// parse log level from the beginning of the message
	msg := fmt.Sprintf(format, args...)
	level, msg := extractLevel(msg)

	// create a record with caller information
	// skip level is critical:
	// - 0 = this line
	// - 1 = this function (Logf)
	// - 2 = caller of Logf (user code)
	//
	// note: We use PC=0 to ensure slog.Record.PC() returns 0,
	// which causes slog to skip obtaining the caller info itself
	record := slog.NewRecord(time.Now(), stringToLevel(level), msg, 2)

	// we need to manually add the source information ourselves, since
	// slog.Handler might have AddSource=true but won't get the caller
	// right due to how we're adapting lgr → slog
	pc, file, line, ok := runtime.Caller(2) // skip to caller of Logf
	if ok {
		// only add source info if we can find it
		funcName := runtime.FuncForPC(pc).Name()
		record.AddAttrs(
			slog.Group("source",
				slog.String("function", funcName),
				slog.String("file", file),
				slog.Int("line", line),
			),
		)
	}

	// handle the record
	if err := a.handler.Handle(context.Background(), record); err != nil {
		// if handling fails, fallback to stderr
		fmt.Fprintf(os.Stderr, "slog handler error: %v\n", err)
	}
}

// Helper functions

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

// extractLevel parses lgr-style log message to extract level prefix
func extractLevel(msg string) (level, message string) {
	for _, lvl := range levels {
		prefix := lvl + " "
		bracketPrefix := "[" + lvl + "] "

		if strings.HasPrefix(msg, prefix) {
			return lvl, strings.TrimPrefix(msg, prefix)
		}
		if strings.HasPrefix(msg, bracketPrefix) {
			return lvl, strings.TrimPrefix(msg, bracketPrefix)
		}
	}

	return "INFO", msg
}

// formatAttr converts slog.Attr to string representation
func formatAttr(attr slog.Attr, groups []string) string {
	if attr.Equal(slog.Attr{}) {
		return ""
	}

	key := attr.Key
	if len(groups) > 0 {
		key = strings.Join(groups, ".") + "." + key
	}

	val := attr.Value.String()

	// handle string values specially by quoting them
	if attr.Value.Kind() == slog.KindString {
		val = fmt.Sprintf("%q", attr.Value.String())
	}

	return fmt.Sprintf("%s=%s ", key, val)
}
