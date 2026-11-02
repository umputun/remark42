// Package logger defines interface for logging. Implementation should be passed by user.
// Also provides NoOp (do-nothing) and Std (redirect to std log) predefined loggers.
package logger

import "log"

// L defined logger interface used everywhere in the package
type L interface {
	Logf(format string, args ...any)
}

// Func type is an adapter to allow the use of ordinary functions as Logger.
type Func func(format string, args ...any)

// Logf calls f(format, args...).
func (f Func) Logf(format string, args ...any) { f(format, args...) }

// NoOp logger
var NoOp = Func(func(string, ...any) {})

// Std logger sends to std default logger directly
var Std = Func(func(format string, args ...any) { log.Printf(format, args...) })
