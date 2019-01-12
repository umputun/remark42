package lgr

import (
	stdlog "log"
	"os"
)

var def = New() // default logger doesn't allow DEBUG and doesn't add caller info

// L defines minimal interface used to log things
type L interface {
	Logf(format string, args ...interface{})
}

// Func type is an adapter to allow the use of ordinary functions as Logger.
type Func func(format string, args ...interface{})

// Logf calls f(id)
func (f Func) Logf(format string, args ...interface{}) { f(format, args...) }

// NoOp logger
var NoOp = Func(func(format string, args ...interface{}) {})

// Std logger sends to std default logger directly
var Std = Func(func(format string, args ...interface{}) { stdlog.Printf(format, args...) })

// Printf simplifies replacement of std logger
func Printf(format string, args ...interface{}) {
	def.Logf(format, args...)
}

// Print simplifies replacement of std logger
func Print(line string) {
	def.Logf(line)
}

// Fatalf simplifies replacement of std logger
func Fatalf(format string, args ...interface{}) {
	def.Logf(format, args...)
	os.Exit(1)
}

// Setup default logger with options
func Setup(opts ...Option) {
	def = New(opts...)
	def.skipCallers = 2
}

// Default returns pre-constructed def logger (debug on, callers disabled)
func Default() L { return def }
