package logger

import "log"

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
var Std = Func(func(format string, args ...interface{}) { log.Printf(format, args...) })
