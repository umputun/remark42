package lgr

import (
	stdlog "log"
)

var def = newShared() // default logger doesn't allow DEBUG and doesn't add caller info

// sharedSecrets are hidden by the default logger and by the logger made by SetupStdLogger
var sharedSecrets secretSet

// L defines minimal interface used to log things
type L interface {
	Logf(format string, args ...interface{})
}

// Func type is an adapter to allow the use of ordinary functions as Logger.
type Func func(format string, args ...interface{})

// Logf calls f(format, args...)
func (f Func) Logf(format string, args ...interface{}) { f(format, args...) }

// NoOp logger
var NoOp = Func(func(format string, args ...interface{}) {}) //nolint:revive

// Std logger sends to std default logger directly
var Std = Func(func(format string, args ...interface{}) { stdlog.Printf(format, args...) })

// Printf simplifies replacement of std logger
func Printf(format string, args ...interface{}) {
	def.logf(format, args...)
}

// Print simplifies replacement of std logger
func Print(line string) {
	def.logf(line) //nolint:govet
}

// Fatalf simplifies replacement of std logger
func Fatalf(format string, args ...interface{}) {
	def.logf(format, args...)
	def.fatal()
}

// Setup default logger with options
func Setup(opts ...Option) {
	def = newShared(opts...)
}

// AddSecrets adds values to hide in the output of the default logger and the std logger made by
// SetupStdLogger. They stay in effect after Setup calls. Loggers made by New are not affected.
func AddSecrets(vals ...string) {
	sharedSecrets.add(vals...)
}

// RemoveSecrets stops hiding values added by AddSecrets. A value a logger got with Secret option
// stays hidden by that logger.
func RemoveSecrets(vals ...string) {
	sharedSecrets.remove(vals...)
}

// newShared makes a logger hiding the package-level secrets in addition to its own.
// The link is set here, before the logger is in use, so logging reads it without a lock.
func newShared(opts ...Option) *Logger {
	l := New(opts...)
	l.shared = &sharedSecrets
	return l
}

// Default returns pre-constructed def logger (debug off, callers disabled)
func Default() L { return def }
