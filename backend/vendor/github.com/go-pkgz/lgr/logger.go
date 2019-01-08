package lgr

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

var levels = []string{"DEBUG", "INFO", "WARN", "ERROR", "PANIC", "FATAL"}

// Logger provided simple logger with basic support of levels. Thread safe
type Logger struct {
	stdout, stderr io.Writer
	dbg            bool
	lock           sync.Mutex
	callers        bool
	now            nowFn
	fatal          panicFn
	skipCallers    int
}

type nowFn func() time.Time
type panicFn func()

// New makes new leveled logger. Accepts dbg flag turing on info about the caller and allowing DEBUG messages/
// Two writers can be passed optionally - first for out and second for err
func New(options ...Option) *Logger {
	res := Logger{
		now:         time.Now,
		fatal:       func() { os.Exit(1) },
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		skipCallers: 1,
	}
	for _, opt := range options {
		opt(&res)
	}
	return &res
}

// Logf implements L interface to output with printf style.
// Each line prefixed with ts, level and optionally (dbg mode only) by caller info.
// ERROR and FATAL also send the same line to err writer.
// FATAL adds runtime stack and os.exit(1), like panic.
func (l *Logger) Logf(format string, args ...interface{}) {

	lv, msg := l.extractLevel(fmt.Sprintf(format, args...))
	var bld strings.Builder
	bld.WriteString(l.now().Format("2006/01/02 15:04:05.000 "))
	bld.WriteString(lv)

	if l.dbg && l.callers {
		if pc, file, line, ok := runtime.Caller(l.skipCallers); ok {
			fnameElems := strings.Split(file, "/")
			funcNameElems := strings.Split(runtime.FuncForPC(pc).Name(), "/")
			srcFileInfo := fmt.Sprintf("{%s:%d %s} ", strings.Join(fnameElems[len(fnameElems)-2:], "/"),
				line, funcNameElems[len(funcNameElems)-1])
			bld.WriteString(srcFileInfo)
		}
	}

	if lv == "DEBUG " && !l.dbg {
		return
	}
	bld.WriteString(msg)  //nolint
	bld.WriteString("\n") //nolint

	l.lock.Lock()
	msgb := []byte(bld.String())
	l.stdout.Write(msgb) //nolint

	switch lv {
	case "PANIC ", "FATAL ":
		l.stderr.Write(msgb)      //nolint
		l.stderr.Write(getDump()) //nolint
		l.fatal()
	case "ERROR ":
		l.stderr.Write(msgb) //nolint
	}

	l.lock.Unlock()
}

func (l *Logger) extractLevel(line string) (level, msg string) {
	spaces := " "
	for _, lv := range levels {
		if strings.HasPrefix(line, lv) {
			if len(lv) == 4 {
				spaces = "  "
			}
			return lv + spaces, line[len(lv)+1:]
		}
		if strings.HasPrefix(line, "["+lv+"]") {
			if len(lv) == 4 {
				spaces = "  "
			}
			return lv + spaces, line[len(lv)+3:]
		}
	}
	return "", line
}

// getDump reads runtime stack and returns as a string
func getDump() []byte {
	maxSize := 5 * 1024 * 1024
	stacktrace := make([]byte, maxSize)
	length := runtime.Stack(stacktrace, true)
	if length > maxSize {
		length = maxSize
	}
	return stacktrace[:length]
}

// Option func type
type Option func(l *Logger)

// Out sets out writer
func Out(w io.Writer) Option {
	return func(l *Logger) {
		l.stdout = w
	}
}

// Err sets error writer
func Err(w io.Writer) Option {
	return func(l *Logger) {
		l.stderr = w
	}
}

// Debug turn on dbg mode
func Debug(l *Logger) {
	l.dbg = true
}

// Caller adds caller info with func, file, and line number
func Caller(l *Logger) {
	l.callers = true
}
