# lgr - simple logger with some extras 
[![Build Status](https://github.com/go-pkgz/lgr/workflows/build/badge.svg)](https://github.com/go-pkgz/lgr/actions) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/lgr/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/lgr?branch=master) [![godoc](https://godoc.org/github.com/go-pkgz/lgr?status.svg)](https://godoc.org/github.com/go-pkgz/lgr)

## install

`go get github.com/go-pkgz/lgr`

## usage

```go
    l := lgr.New(lgr.Msec, lgr.Debug, lgr.CallerFile, lgr.CallerFunc) // allow debug and caller info, timestamp with milliseconds
    l.Logf("INFO some important message, %v", err)
    l.Logf("DEBUG some less important message, %v", err)
```

output looks like this:
```
2018/01/07 13:02:34.000 INFO  {svc/handler.go:101 h.MyFunc1} some important message, can't open file myfile.xyz
2018/01/07 13:02:34.015 DEBUG {svc/handler.go:155 h.MyFunc2} some less important message, file is too small`
```

_Without `lgr.Caller*` it will drop `{caller}` part_

## details

### interfaces and default loggers

- `lgr` package provides a single interface `lgr.L` with a single method `Logf(format string, args ...interface{})`. Function wrapper `lgr.Func` allows making `lgr.L` from a function directly.
- Default logger functionality can be used without `lgr.New` (see "global logger")
- Two predefined loggers available: `lgr.NoOp` (do-nothing logger) and `lgr.Std` (passing directly to stdlib log)

### options

`lgr.New` call accepts functional options:

- `lgr.Debug` - turn debug mode on to allow messages with "DEBUG" level (filtered otherwise)
- `lgr.Trace` - turn trace mode on to allow messages with "TRACE" abd "DEBUG" levels both (filtered otherwise)
- `lgr.Out(io.Writer)` - sets the output writer, default `os.Stdout`
- `lgr.Err(io.Writer)` - sets the error writer, default `os.Stderr`
- `lgr.CallerFile` - adds the caller file info (only affects lgr's native text format, not slog output)
- `lgr.CallerFunc` - adds the caller function info (only affects lgr's native text format, not slog output)
- `lgr.CallerPkg` - adds the caller package (only affects lgr's native text format, not slog output)
- `lgr.LevelBraces` - wraps levels with "[" and "]"
- `lgr.Msec` - adds milliseconds to timestamp
- `lgr.Format` - sets a custom template, overwrite all other formatting modifiers.
- `lgr.Secret(secret ...)` - sets list of the secrets to hide from the logging outputs.
- `lgr.Map(mapper)` - sets mapper functions to change elements of the logging output based on levels.
- `lgr.StackTraceOnError` - turns on stack trace for ERROR level.
- `lgr.SlogHandler(h slog.Handler)` - delegates logging to the provided slog handler.

example: `l := lgr.New(lgr.Debug, lgr.Msec)`

#### formatting templates:

Several predefined templates provided and can be passed directly to `lgr.Format`, i.e. `lgr.Format(lgr.WithMsec)`

```
	Short      = `{{.DT.Format "2006/01/02 15:04:05"}} {{.Level}} {{.Message}}`
	WithMsec   = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} {{.Message}}`
	WithPkg    = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} ({{.CallerPkg}}) {{.Message}}`
	ShortDebug = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} ({{.CallerFile}}:{{.CallerLine}}) {{.Message}}`
	FuncDebug  = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} ({{.CallerFunc}}) {{.Message}}`
	FullDebug  = `{{.DT.Format "2006/01/02 15:04:05.000"}} {{.Level}} ({{.CallerFile}}:{{.CallerLine}} {{.CallerFunc}}) {{.Message}}`
```

User can make a custom template and pass it directly to `lgr.Format`. For example:

```go
    lgr.Format(`{{.Level}} - {{.DT.Format "2006-01-02T15:04:05Z07:00"}} - {{.CallerPkg}} - {{.Message}}`)
```

_Note: formatter (predefined or custom) adds measurable overhead - the cost will depend on the version of Go, but is between 30
 and 50% in recent tests with 1.12. You can validate this in your environment via benchmarks: `go test -bench=. -run=Bench`_

### levels

`lgr.Logf` recognize prefixes like `INFO` or `[INFO]` as levels. The full list of supported levels - `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`, `PANIC` and `FATAL`.

- `TRACE` will be filtered unless `lgr.Trace` option defined
- `DEBUG` will be filtered unless `lgr.Debug` or `lgr.Trace` options defined
- `INFO` and `WARN` don't have any special behavior attached
- `ERROR` sends messages to both out and err writers
- `FATAL` and send messages to both out and err writers and exit(1)
- `PANIC` does the same as `FATAL` but in addition sends dump of callers and runtime info to err.

### mapper

Elements of the output can be altered with a set of user defined function passed as `lgr.Map` options. Such a mapper changes
the value of an element (i.e. timestamp, level, message, caller) and has separate functions for each level. Note: both level 
and messages elements handled by the same function for a given level. 

_A typical use-case is to produce colorful output with a user-define colorization library._

example with [fatih/color](https://github.com/fatih/color):

```go
	colorizer := lgr.Mapper{
		ErrorFunc:  func(s string) string { return color.New(color.FgHiRed).Sprint(s) },
		WarnFunc:   func(s string) string { return color.New(color.FgHiYellow).Sprint(s) },
		InfoFunc:   func(s string) string { return color.New(color.FgHiWhite).Sprint(s) },
		DebugFunc:  func(s string) string { return color.New(color.FgWhite).Sprint(s) },
		CallerFunc: func(s string) string { return color.New(color.FgBlue).Sprint(s) },
		TimeFunc:   func(s string) string { return color.New(color.FgCyan).Sprint(s) },
	}

	logOpts := []lgr.Option{lgr.Msec, lgr.LevelBraces, lgr.Map(colorizer)}
```
### adaptors

`lgr` logger can be converted to `io.Writer`, `*log.Logger`, or `slog.Handler`

- `lgr.ToWriter(l lgr.L, level string) io.Writer` - makes io.Writer forwarding write ops to underlying `lgr.L`
- `lgr.ToStdLogger(l lgr.L, level string) *log.Logger` - makes standard logger on top of `lgr.L`
- `lgr.ToSlogHandler(l lgr.L) slog.Handler` - converts lgr.L to a slog.Handler for use with slog

_`level` parameter is optional, if defined (non-empty) will enforce the level._

- `lgr.SetupStdLogger(opts ...Option)` initializes std global logger (`log.std`) with lgr logger and given options. 
All standard methods like `log.Print`, `log.Println`, `log.Fatal` and so on will be forwarder to lgr.
- `lgr.SetupWithSlog(logger *slog.Logger)` sets up the global logger with a slog logger.

### slog integration

In addition to the standard logger interface, lgr provides seamless integration with Go's `log/slog` package:

#### Using lgr with slog

```go
// Create lgr logger
lgrLogger := lgr.New(lgr.Debug, lgr.Msec)

// Convert to slog handler and create slog logger
handler := lgr.ToSlogHandler(lgrLogger)
logger := slog.New(handler)

// Use standard slog API with lgr formatting
logger.Info("message", "key1", "value1")
// Output: 2023/09/15 10:34:56.789 INFO  message key1="value1" 
```

#### Using slog with lgr interface

```go
// Create slog handler
jsonHandler := slog.NewJSONHandler(os.Stdout, nil)

// Wrap it with lgr interface
logger := lgr.FromSlogHandler(jsonHandler)

// Use lgr API with slog backend
logger.Logf("INFO message with %s", "structured data")
// Output: {"time":"2023-09-15T10:34:56.789Z","level":"INFO","msg":"message with structured data"}
```

#### Using slog directly in lgr

```go
// Create a logger that uses slog directly
jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
logger := lgr.New(lgr.SlogHandler(jsonHandler))

// Use lgr API with slog backend
logger.Logf("INFO message")
// Output: {"time":"2023-09-15T10:34:56.789Z","level":"INFO","msg":"message"}
```

#### JSON output with caller information

To get caller information in JSON output when using slog handlers, create the handler with `AddSource: true`:

```go
// Create JSON handler with source information (caller info)
jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    AddSource: true,  // This enables caller information in JSON output
})

// Use handler with lgr
logger := lgr.New(lgr.SlogHandler(jsonHandler))

logger.Logf("INFO message with caller info")
// Output will include source file, line and function in JSON
```

Note: The lgr caller options (`lgr.CallerFile`, `lgr.CallerFunc`, `lgr.CallerPkg`) only work with lgr's native text format
and don't affect JSON output from slog handlers. To include caller information in JSON logs:

1. For slog JSON handlers: Create the handler with `AddSource: true` as shown above
2. For text-based logs: Use lgr's native caller options without slog integration

This behavior is designed to respect each logging system's conventions for representing caller information.

### global logger

Users **should avoid** global logger and pass the concrete logger as a dependency. However, in some cases a global logger may be needed, for example migration from stdlib `log` to `lgr`. For such cases `log "github.com/go-pkgz/lgr"` can be imported instead of `log` package.

Global logger provides `lgr.Printf`, `lgr.Print` and `lgr.Fatalf` functions. User can customize the logger by calling `lgr.Setup(options ...)`. The instance of this logger can be retrieved with `lgr.Default()`

