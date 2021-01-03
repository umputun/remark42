// Package lgr provides a simple logger with some extras. Primary way to log is Logf method.
// The logger's output can be customized in 2 ways:
//   - by setting individual formatting flags, i.e. lgr.New(lgr.Msec, lgr.CallerFunc)
//   - by passing formatting template, i.e. lgr.New(lgr.Format(lgr.Short))
// Leveled output works for messages based on text prefix, i.e. Logf("INFO some message") means INFO level.
// Debug and trace levels can be filtered based on lgr.Trace and lgr.Debug options.
// ERROR, FATAL and PANIC levels send to err as well. FATAL terminate caller application with os.Exit(1)
// and PANIC also prints stack trace.
package lgr

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"
)

var levels = []string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR", "PANIC", "FATAL"}

const (
	// Short logging format
	Short = `{{.DT.Format "2006/01/02 15:04:05"}} {{.Level}} {{.Message}}`
	// WithMsec is a logging format with milliseconds
	WithMsec = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} {{.Message}}`
	// WithPkg is WithMsec logging format with caller package
	WithPkg = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} ({{.CallerPkg}}) {{.Message}}`
	// ShortDebug is WithMsec logging format with caller file and line
	ShortDebug = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} ({{.CallerFile}}:{{.CallerLine}}) {{.Message}}`
	// FuncDebug is WithMsec logging format with caller function
	FuncDebug = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} ({{.CallerFunc}}) {{.Message}}`
	// FullDebug is WithMsec logging format with caller file, line and function
	FullDebug = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} ({{.CallerFile}}:{{.CallerLine}} {{.CallerFunc}}) {{.Message}}`
)

var secretReplacement = []byte("******")

var (
	reTraceDefault = regexp.MustCompile(`.*/lgr/logger\.go.*\n`)
	reTraceStd     = regexp.MustCompile(`.*/log/log\.go.*\n`)
)

// Logger provided simple logger with basic support of levels. Thread safe
type Logger struct {
	// set with Option calls
	stdout, stderr io.Writer // destination writes for out and err
	dbg            bool      // allows reporting for DEBUG level
	trace          bool      // allows reporting for TRACE and DEBUG levels
	callerFile     bool      // reports caller file with line number, i.e. foo/bar.go:89
	callerFunc     bool      // reports caller function name, i.e. bar.myFunc
	callerPkg      bool      // reports caller package name
	levelBraces    bool      // encloses level with [], i.e. [INFO]
	callerDepth    int       // how many stack frames to skip, relative to the real (reported) frame
	format         string    // layout template
	secrets        [][]byte  // sub-strings to secrets by matching
	mapper         Mapper    // map (alter) output based on levels

	// internal use
	now           nowFn
	fatal         panicFn
	msec          bool
	lock          sync.Mutex
	callerOn      bool
	levelBracesOn bool
	errorDump     bool
	templ         *template.Template
	reTrace       *regexp.Regexp
}

// can be redefined internally for testing
type nowFn func() time.Time
type panicFn func()

// layout holds all parts to construct the final message with template or with individual flags
type layout struct {
	DT         time.Time
	Level      string
	Message    string
	CallerPkg  string
	CallerFile string
	CallerFunc string
	CallerLine int
}

// New makes new leveled logger. By default writes to stdout/stderr.
// default format: 2018/01/07 13:02:34.123 DEBUG some message 123
func New(options ...Option) *Logger {

	res := Logger{
		now:         time.Now,
		fatal:       func() { os.Exit(1) },
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		callerDepth: 0,
		mapper:      nopMapper,
		reTrace:     reTraceDefault,
	}
	for _, opt := range options {
		opt(&res)
	}

	if res.format != "" {
		// formatter defined
		var err error
		res.templ, err = template.New("lgr").Parse(res.format)
		if err != nil {
			fmt.Printf("invalid template %s, error %v. switched to %s\n", res.format, err, Short)
			res.format = Short
			res.templ = template.Must(template.New("lgrDefault").Parse(Short))
		}

		buf := bytes.Buffer{}
		if err = res.templ.Execute(&buf, layout{}); err != nil {
			fmt.Printf("failed to execute template %s, error %v. switched to %s\n", res.format, err, Short)
			res.format = Short
			res.templ = template.Must(template.New("lgrDefault").Parse(Short))
		}
	}

	// set *On flags once for optimization on multiple Logf calls
	res.callerOn = strings.Contains(res.format, "{{.Caller") || res.callerFile || res.callerFunc || res.callerPkg
	res.levelBracesOn = strings.Contains(res.format, "[{{.Level}}]") || res.levelBraces

	return &res
}

// Logf implements L interface to output with printf style.
// DEBUG and TRACE filtered out by dbg and trace flags.
// ERROR and FATAL also send the same line to err writer.
// FATAL and PANIC adds runtime stack and os.exit(1), like panic.
func (l *Logger) Logf(format string, args ...interface{}) {
	// to align call depth between (*Logger).Logf() and, for example, Printf()
	l.logf(format, args...)
}

//nolint gocyclo
func (l *Logger) logf(format string, args ...interface{}) {

	var lv, msg string
	if len(args) == 0 {
		lv, msg = l.extractLevel(format)
	} else {
		lv, msg = l.extractLevel(fmt.Sprintf(format, args...))
	}

	if lv == "DEBUG" && !l.dbg {
		return
	}
	if lv == "TRACE" && !l.trace {
		return
	}

	var ci callerInfo
	if l.callerOn { // optimization to avoid expensive caller evaluation if caller info not in the template
		ci = l.reportCaller(l.callerDepth)
	}

	elems := layout{
		DT:         l.now(),
		Level:      l.formatLevel(lv),
		Message:    strings.TrimSuffix(msg, "\n"), // output adds EOL, trim from the message if passed
		CallerFunc: ci.FuncName,
		CallerFile: ci.File,
		CallerPkg:  ci.Pkg,
		CallerLine: ci.Line,
	}

	var data []byte
	if l.format == "" {
		data = []byte(l.formatWithOptions(elems))
	} else {
		buf := bytes.Buffer{}
		err := l.templ.Execute(&buf, elems) // once constructed, a template may be executed safely in parallel.
		if err != nil {
			fmt.Printf("failed to execute template, %v\n", err) // should never happen
		}
		data = buf.Bytes()
	}
	data = append(data, '\n')

	if l.levelBracesOn { // rearrange space in short levels
		data = bytes.Replace(data, []byte("[WARN ]"), []byte("[WARN] "), 1)
		data = bytes.Replace(data, []byte("[INFO ]"), []byte("[INFO] "), 1)
	}
	data = l.hideSecrets(data)

	l.lock.Lock()
	_, _ = l.stdout.Write(data)

	// write to err as well for high levels, exit(1) on fatal and panic and dump stack on panic level
	switch lv {
	case "ERROR":
		if l.stderr != l.stdout {
			_, _ = l.stderr.Write(data)
		}
		if l.errorDump {
			stackInfo := make([]byte, 1024*1024)
			if stackSize := runtime.Stack(stackInfo, false); stackSize > 0 {
				traceLines := l.reTrace.Split(string(stackInfo[:stackSize]), -1)
				if len(traceLines) > 0 {
					_, _ = l.stdout.Write([]byte(">>> stack trace:\n" + traceLines[len(traceLines)-1]))
				}
			}
		}
	case "FATAL":
		if l.stderr != l.stdout {
			_, _ = l.stderr.Write(data)
		}
		l.fatal()
	case "PANIC":
		if l.stderr != l.stdout {
			_, _ = l.stderr.Write(data)
		}
		_, _ = l.stderr.Write(getDump())
		l.fatal()
	}

	l.lock.Unlock()
}

func (l *Logger) hideSecrets(data []byte) []byte {
	for _, h := range l.secrets {
		data = bytes.Replace(data, h, secretReplacement, -1)
	}
	return data
}

type callerInfo struct {
	File     string
	Line     int
	FuncName string
	Pkg      string
}

// calldepth 0 identifying the caller of reportCaller()
func (l *Logger) reportCaller(calldepth int) (res callerInfo) {

	// caller gets file, line number abd function name via runtime.Callers
	// file looks like /go/src/github.com/go-pkgz/lgr/logger.go
	// file is an empty string if not known.
	// funcName looks like:
	//   main.Test
	//   foo/bar.Test
	//   foo/bar.Test.func1
	//   foo/bar.(*Bar).Test
	//   foo/bar.glob..func1
	// funcName is an empty string if not known.
	// line is a zero if not known.
	caller := func(calldepth int) (file string, line int, funcName string) {
		pcs := make([]uintptr, 1)
		n := runtime.Callers(calldepth, pcs)
		if n != 1 {
			return "", 0, ""
		}

		frame, _ := runtime.CallersFrames(pcs).Next()

		return frame.File, frame.Line, frame.Function
	}

	// add 5 to adjust stack level because it was called from 3 nested functions added by lgr, i.e. caller,
	// reportCaller and logf, plus 2 frames by runtime
	filePath, line, funcName := caller(calldepth + 2 + 3)
	if (filePath == "") || (line <= 0) || (funcName == "") {
		return callerInfo{}
	}

	_, pkgInfo := path.Split(path.Dir(filePath))
	res.Pkg = strings.Split(pkgInfo, "@")[0] // remove version from package name

	res.File = filePath
	if pathElems := strings.Split(filePath, "/"); len(pathElems) > 2 {
		res.File = strings.Join(pathElems[len(pathElems)-2:], "/")
	}
	res.Line = line

	funcNameElems := strings.Split(funcName, "/")
	res.FuncName = funcNameElems[len(funcNameElems)-1]

	return res
}

// speed-optimized version of formatter, used with individual options only, i.e. without Format call
func (l *Logger) formatWithOptions(elems layout) (res string) {

	orElse := func(flag bool, fnTrue func() string, fnFalse func() string) string {
		if flag {
			return fnTrue()
		}
		return fnFalse()
	}
	nothing := func() string { return "" }

	parts := make([]string, 0, 4)

	parts = append(
		parts,
		l.mapper.TimeFunc(orElse(l.msec,
			func() string { return elems.DT.Format("2006/01/02 15:04:05.000") },
			func() string { return elems.DT.Format("2006/01/02 15:04:05") },
		)),
		l.levelMapper(elems.Level)(orElse(l.levelBraces,
			func() string { return `[` + elems.Level + `]` },
			func() string { return elems.Level },
		)),
	)

	if l.callerFile || l.callerFunc || l.callerPkg {
		var callerParts []string
		v := orElse(l.callerFile, func() string { return elems.CallerFile + ":" + strconv.Itoa(elems.CallerLine) }, nothing)
		if v != "" {
			callerParts = append(callerParts, v)
		}
		if v := orElse(l.callerFunc, func() string { return elems.CallerFunc }, nothing); v != "" {
			callerParts = append(callerParts, v)
		}
		if v := orElse(l.callerPkg, func() string { return elems.CallerPkg }, nothing); v != "" {
			callerParts = append(callerParts, v)
		}

		caller := "{" + strings.Join(callerParts, " ") + "}"
		if l.mapper.CallerFunc != nil {
			caller = l.mapper.CallerFunc(caller)
		}
		parts = append(parts, caller)
	}

	msg := elems.Message
	if l.mapper.MessageFunc != nil {
		msg = l.mapper.MessageFunc(elems.Message)
	}

	parts = append(parts, l.levelMapper(elems.Level)(msg))
	return strings.Join(parts, " ")
}

// formatLevel aligns level to 5 chars
func (l *Logger) formatLevel(lv string) string {

	spaces := ""
	if len(lv) == 4 {
		spaces = " "
	}
	return lv + spaces
}

// extractLevel parses messages with optional level prefix and returns level and the message with stripped level
func (l *Logger) extractLevel(line string) (level, msg string) {
	for _, lv := range levels {
		if strings.HasPrefix(line, lv) {
			return lv, strings.TrimSpace(line[len(lv):])
		}
		if strings.HasPrefix(line, "["+lv+"]") {
			return lv, strings.TrimSpace(line[len("["+lv+"]"):])
		}
	}
	return "INFO", line
}

func (l *Logger) levelMapper(level string) mapFunc {

	nop := func(s string) string {
		return s
	}

	switch level {
	case "TRACE", "DEBUG":
		if l.mapper.DebugFunc == nil {
			return nop
		}
		return l.mapper.DebugFunc
	case "INFO ":
		if l.mapper.InfoFunc == nil {
			return nop
		}
		return l.mapper.InfoFunc
	case "WARN ":
		if l.mapper.WarnFunc == nil {
			return nop
		}
		return l.mapper.WarnFunc
	case "ERROR", "PANIC", "FATAL":
		if l.mapper.ErrorFunc == nil {
			return nop
		}
		return l.mapper.ErrorFunc
	}
	return func(s string) string { return s }
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
