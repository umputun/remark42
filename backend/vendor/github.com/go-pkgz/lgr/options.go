package lgr

import "io"

// Option func type
type Option func(l *Logger)

// Out sets out writer, stdout by default
func Out(w io.Writer) Option {
	return func(l *Logger) {
		l.stdout = w
	}
}

// Err sets error writer, stderr by default
func Err(w io.Writer) Option {
	return func(l *Logger) {
		l.stderr = w
	}
}

// Debug turn on dbg mode
func Debug(l *Logger) {
	l.dbg = true
}

// Trace turn on trace + dbg mode
func Trace(l *Logger) {
	l.dbg = true
	l.trace = true
}

// CallerDepth sets number of stack frame skipped for caller reporting, 0 by default
func CallerDepth(n int) Option {
	return func(l *Logger) {
		l.callerDepth = n
	}
}

// Format sets output layout, overwrites all options for individual parts, i.e. Caller*, Msec and LevelBraces
func Format(f string) Option {
	return func(l *Logger) {
		l.format = f
	}
}

// CallerFunc adds caller info with function name. Ignored if Format option used.
func CallerFunc(l *Logger) {
	l.callerFunc = true
}

// CallerPkg adds caller's package name. Ignored if Format option used.
func CallerPkg(l *Logger) {
	l.callerPkg = true
}

// LevelBraces surrounds level with [], i.e. [INFO]. Ignored if Format option used.
func LevelBraces(l *Logger) {
	l.levelBraces = true
}

// CallerFile adds caller info with file, and line number. Ignored if Format option used.
func CallerFile(l *Logger) {
	l.callerFile = true
}

// Msec adds .msec to timestamp. Ignored if Format option used.
func Msec(l *Logger) {
	l.msec = true
}

// Secret sets list of substring to be hidden, i.e. replaced by "******"
// Useful to prevent passwords or other sensitive tokens to be logged.
func Secret(vals ...string) Option {
	return func(l *Logger) {
		for _, v := range vals {
			l.secrets = append(l.secrets, []byte(v))
		}
	}
}

// Map sets mapper functions to change elements of the logged message based on levels.
func Map(m Mapper) Option {
	return func(l *Logger) {
		l.mapper = m
	}
}

// StackTraceOnError turns on stack trace for ERROR level.
func StackTraceOnError(l *Logger) {
	l.errorDump = true
}
