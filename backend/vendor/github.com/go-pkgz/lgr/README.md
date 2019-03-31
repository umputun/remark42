# lgr - simple logger with some extras [![Build Status](https://travis-ci.org/go-pkgz/lgr.svg?branch=master)](https://travis-ci.org/go-pkgz/lgr) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/lgr/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/lgr?branch=master) [![godoc](https://godoc.org/github.com/go-pkgz/lgr?status.svg)](https://godoc.org/github.com/go-pkgz/lgr)

## install

`go get github.com/go-pkgz/lgr`

## usage

```go
    l := lgr.New(lgr.Debug, lgr.CallerFile) // allow debug and caller file info
    l.Logf("INFO some important message, %v", err)
    l.Logf("DEBUG some less important message, %v", err)
```

output looks like this:
```
2018/01/07 13:02:34.000 INFO  {svc/handler.go:101 h.MyFunc1} some important message, can't open file`
2018/01/07 13:02:34.015 DEBUG {svc/handler.go:155 h.MyFunc2} some less important message, file is too small`
```

_Without `lgr.Caller*` it will drop `{caller}` part_

## details

### interfaces and default loggers

- `lgr` package provides a single interface `lgr.L` with a single method `Logf(format string, args ...interface{})`. Function wrapper `lgr.Func` allows to make `lgr.L` from a function directly.
- Default logger functionality can be used without `lgr.New`, but just `lgr.Printf`
- Two predefined loggers available: `lgr.NoOp` (do-nothing logger) and `lgr.Std` (passing directly to stdlib log)

### options

`lgr.New` call accepts functional options:

- `lgr.Debug` - turn debug mode on to allow messages with "DEBUG" level (filtered overwise)
- `lgr.Out(io.Writer)` - sets the output writer, default `os.Stdout`
- `lgr.Err(io.Writer)` - sets the error writer, default `os.Stderr`
- `lgr.CallerFile` - adds the caller file info
- `lgr.CallerFunc` - adds the caller function info
- `lgr.CallerPkg` - adds the caller package
- `lgr.LevelBraces` - wraps levels with "[" and "]"
- `lgr.Msec` - adds milliseconds to timestamp
- `lgr.Format` - sets custom template, overwrite all other formatting modifiers.

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
    lgr.Format(`{{.Level}} - {{.DT.Format "2006-01-02T15:04:05Z07:00") - {{.CallerPkg}} - {{.Message}}`)
```
)
    
### levels

`lgr.Logf` recognizes prefixes like "INFO" or "[INFO]" as levels. The full list of supported levels - "TRACE", "DEBUG", "INFO", "WARN", "ERROR", "PANIC" and "FATAL"

- `TRACE` will be filtered unless `lgr.Trace` option defined
- `DEBUG` will be filtered unless `lgr.Debug` or `lgr.Trace` options defined
- `INFO` and `WARN` don't have any special behavior attached
- `ERROR` sends messages to both out and err writers
- `PANIC` and `FATAL` send messages to both out and err writers. In addition sends dump of callers and runtime info to err only, and calls `os.Exit(1)`.

### adaptors

`lgr` logger can be converted to `io.Writer` or `*log.Logger`

- `lgr.ToWriter(l lgr.L, level string) io.Writer` - makes io.Writer forwarding write ops to underlying `lgr.L`
- `lgr.ToStdLogger(l lgr.L, level string) *log.Logger` - makes standard logger on top of `lgr.L`

_`level` parameter is optional, if defined will enforce the level._    
  
### global logger

Users **should avoid** global logger and pass the concrete logger as a dependency. However, in some cases a global logger may be needed, for example migration from stdlib `log` to `lgr`. For such cases `log "github.com/go-pkgz/lgr"` can be imported instead of `log` package.

Global logger provides `lgr.Printf`, `lgr.Print` and `lgr.Fatalf` functions. User can customize the logger by calling `lgr.Setup(options ...)`. The instance of this logger can be retrieved with `lgr.Default()`
 
