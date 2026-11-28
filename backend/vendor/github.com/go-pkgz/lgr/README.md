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
2018/01/07 13:02:34.015 DEBUG {svc/handler.go:155 h.MyFunc2} some less important message, file is too small
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
- `lgr.Trace` - turn trace mode on to allow messages with "TRACE" and "DEBUG" levels both (filtered otherwise)
- `lgr.Out(io.Writer)` - sets the output writer, default `os.Stdout`
- `lgr.Err(io.Writer)` - sets the error writer, default `os.Stderr`
- `lgr.CallerFile` - adds the caller file info (only affects lgr's native text format, not slog output)
- `lgr.CallerFunc` - adds the caller function info (only affects lgr's native text format, not slog output)
- `lgr.CallerPkg` - adds the caller package (only affects lgr's native text format, not slog output)
- `lgr.CallerDepth(n)` - skips n more stack frames when reporting the caller; use it when lgr is called through a wrapper (1 for `lgr.ToWriter`, 3 for `lgr.ToStdLogger`). `lgr.SetupStdLogger` sets 3 itself, and a `lgr.CallerDepth` passed to it replaces that value. `lgr.ToSlogHandler` wrapping a `*lgr.Logger` takes the caller from the slog record and needs none; a type that embeds `*lgr.Logger` and logs through its promoted `Logf` needs 3 when a `slog.Logger` calls the handler, and a handler or bridge in between adds frames of its own.
- `lgr.LevelBraces` - wraps levels with "[" and "]"
- `lgr.Msec` - adds milliseconds to timestamp
- `lgr.Format` - sets a custom template, overwrites all other formatting modifiers and the mapper set with `lgr.Map`. A template that fails to parse, or to execute with empty values, is replaced by `lgr.Short`, and a notice about it is printed to the process stdout, not to the `lgr.Out` writer.
- `lgr.Secret(secret ...)` - sets list of the secrets to hide from the logging outputs, see "secrets".
- `lgr.Map(mapper)` - sets mapper functions to change elements of the logging output based on levels.
- `lgr.StackTraceOnError` - turns on stack trace for ERROR level. The trace is written to the out writer after the line, not to the err writer.
- `lgr.SlogHandler(h slog.Handler)` - delegates logging to the provided slog handler. Of the other options, only these still apply: `lgr.Debug` and `lgr.Trace` (lgr filters messages before the handler applies its own level), `lgr.Secret` for the message, `lgr.CallerDepth` for the source position passed to the handler, and the exit after `FATAL` and `PANIC`, with `PANIC` also writing `*** PANIC: <message>` and the goroutine dump to the err writer. All other options are ignored, and `ERROR` and `FATAL` lines are not copied to the err writer.

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

_Note: formatter (predefined or custom) adds measurable overhead - the cost will depend on the version of Go. You can validate this in your environment via benchmarks: `go test -bench=. -run=Bench`_

### secrets

`lgr.Secret(secret ...)` sets the values to replace with `******` when the logger is created. The list can be changed later, while the logger is in use and other goroutines log:

- `l.AddSecrets(secret ...)` and `l.RemoveSecrets(secret ...)` change the list of a logger made by `lgr.New`.
- `lgr.AddSecrets(secret ...)` and `lgr.RemoveSecrets(secret ...)` change the package-level list. It is hidden by the global logger and by the std logger set with `lgr.SetupStdLogger`, in addition to the secrets each of them got with `lgr.Secret`, and it stays in effect after `lgr.Setup`, `lgr.SetupStdLogger` and `lgr.SetupWithSlog` calls. Loggers made by `lgr.New` don't use the package-level list.

```go
lgr.Setup(lgr.Secret(dbPassword))
lgr.SetupStdLogger(lgr.Secret(dbPassword))
// later, for example on config reload
lgr.AddSecrets(newToken, newPassword) // hidden in both lgr.Printf and std log.Printf output
```

Things to know:

- Empty and whitespace-only values are ignored. Adding a value twice is the same as adding it once, and a single remove drops it.
- The list of a logger and the package-level list are independent. A value present in both stays hidden until removed from both.
- A line being logged at the moment of the change may still be written with the previous list.
- Longer values are replaced first, so a secret containing another secret is hidden as a whole. Secrets that only partially overlap are replaced one after another, and a part of one of them may stay visible.
- Secrets are hidden in the message passed to the logger. With `lgr.ToSlogHandler` wrapping a `*lgr.Logger`, or a type that embeds one, attributes are part of that message, and string values are masked before they are quoted. Any other `lgr.L`, such as an `lgr.Func` around a masking logger, gets string values already quoted, so a secret that Go quoting changes, such as one containing a quote, a backslash, a newline or a tab, may not be masked there. `lgr.FromSlogHandler` adds no masking of its own: the wrapped handler can redact the record itself, for example with `ReplaceAttr`, or pass it to a masking logger. Attributes added by a slog handler set with `lgr.SlogHandler` are not processed.

### levels

`Logf` recognizes prefixes like `INFO` or `[INFO]` as levels. A bare level counts only when followed by whitespace (space, tab, CR or LF), a colon or the end of the line, so `WARNING: disk full` is not `WARN`; a bracketed level may be followed by anything. A line without a recognized prefix is `INFO`. The full list of supported levels - `TRACE`, `DEBUG`, `INFO`, `WARN`, `ERROR`, `PANIC` and `FATAL`.

- `TRACE` will be filtered unless `lgr.Trace` option defined
- `DEBUG` will be filtered unless `lgr.Debug` or `lgr.Trace` options defined
- `INFO` and `WARN` don't have any special behavior attached
- `ERROR` sends messages to both out and err writers, once when both are the same file or writer
- `FATAL` sends messages to both out and err writers and calls `os.Exit(1)`
- `PANIC` does the same as `FATAL` but in addition sends dump of callers and runtime info to err.

### mapper

Elements of the output can be altered with a set of user defined function passed as `lgr.Map` options. Such a mapper changes
the value of an element (i.e. timestamp, level, message, caller) and has separate functions for each level. Note: both level 
and messages elements handled by the same function for a given level. 

- `MessageFunc` maps the message on every level, before the function of the level.
- `TRACE` uses `DebugFunc`, `PANIC` and `FATAL` use `ErrorFunc`.
- The mapper is ignored when `lgr.Format` or `lgr.SlogHandler` is set.
- A second `lgr.Map` option replaces the first one.

_A typical use-case is to produce colorful output with a user-defined colorization library._

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

- `lgr.ToWriter(l lgr.L, level string) *lgr.Writer` - makes io.Writer forwarding write ops to underlying `lgr.L`
- `lgr.ToStdLogger(l lgr.L, level string) *log.Logger` - makes standard logger on top of `lgr.L`
- `lgr.ToSlogHandler(l lgr.L) slog.Handler` - converts lgr.L to a slog.Handler for use with slog. Attributes are written as `key=value` pairs after the message, with group names joined to the keys by dots, as the slog text handler does. When `l` is a `*lgr.Logger`, the `Caller*` options report the code that called the slog method.

_`level` parameter is optional, if defined (non-empty) will enforce the level._

- `lgr.SetupStdLogger(opts ...Option)` initializes std global logger (`log.std`) with lgr logger and given options. 
All standard methods like `log.Print`, `log.Println`, `log.Fatal` and so on will be forwarded to lgr. Their lines are leveled by prefix like any other, so `log.Fatalf("no prefix")` is logged at `INFO` before the program exits.
- `lgr.SetupWithSlog(logger *slog.Logger)` sets up the global logger with a slog logger. It turns on `lgr.Debug` or `lgr.Trace` when the handler accepts `slog.LevelDebug` or `slog.LevelDebug-4`.

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

Messages below the handler's level are dropped; with `nil` options, as above, `DEBUG` and `TRACE` are not written. `FATAL` and `PANIC` are passed to the handler as `ERROR` records, and then the program exits with status 1, as it does with `lgr.New`; a handler that does not accept `ERROR` drops the record, and the program exits without logging the message. `PANIC` writes a heading and the stacks of all goroutines to stderr first, leaving the message to the handler. Levels are parsed as described in "levels"; only the space or tab after the level prefix is removed, and the rest of the text reaches the handler as it is, including the newline the standard `log` package adds when it writes through `lgr.ToStdLogger`, so a handler that redacts the message sees the whole value. As with `lgr.New`, `Logf` called without arguments uses the text as it is, so `%%` stays `%%`.

#### Using slog directly in lgr

```go
// Create a logger that uses slog directly
jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
logger := lgr.New(lgr.SlogHandler(jsonHandler))

// Use lgr API with slog backend
logger.Logf("INFO message")
// Output: {"time":"2023-09-15T10:34:56.789Z","level":"INFO","msg":"message"}
```

lgr's own level filter runs before the handler: pass `lgr.Debug` or `lgr.Trace` to `lgr.New` as well to let `DEBUG` or `TRACE` messages reach it. Then the handler drops the messages below its own level.

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

Users **should avoid** global logger and pass the concrete logger as a dependency. However, in some cases a global logger may be needed, for example migration from stdlib `log` to `lgr`. For such cases `log "github.com/go-pkgz/lgr"` can be imported instead of `log` package, as long as the code calls only the three functions the global logger provides: `Printf`, `Print`, which takes a single string, and `Fatalf`. For code using the rest of the `log` API, such as `log.Println` or `log.Fatal`, keep the standard `log` import and call `lgr.SetupStdLogger` to forward its output to lgr.

Global logger provides `lgr.Printf`, `lgr.Print(line string)` and `lgr.Fatalf` functions. `lgr.Fatalf` logs a message without a level prefix at `FATAL`, and exits after any message. User can customize the logger by calling `lgr.Setup(options ...)`. The instance of this logger can be retrieved with `lgr.Default()`

