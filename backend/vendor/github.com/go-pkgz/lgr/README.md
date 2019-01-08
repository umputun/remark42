# lgr - simple logger with some extras [![Build Status](https://travis-ci.org/go-pkgz/lgr.svg?branch=master)](https://travis-ci.org/go-pkgz/lgr) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/lgr/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/lgr?branch=master) [![godoc](https://godoc.org/github.com/go-pkgz/lgr?status.svg)](https://godoc.org/github.com/go-pkgz/lgr)

## install

`go get github/go-pkgz/lgr`

## usage

```go
    l := lgr.New(lgr.Debug, lgr.Caller) // allow debug and caller info
    l.Logf("INFO some important err message, %v", err)
    l.Logf("DEBUG some less important err message, %v", err)
```

output looks like this:
```
2018/01/07 13:02:34.000 INFO  {svc/handler.go:101 h.MyFunc1} some important err message, can't open file`
2018/01/07 13:02:34.015 DEBUG {svc/handler.go:155 h.MyFunc2} some less important err message, file is too small`
```

_Without `lgr.Caller` it will drop `{caller}` part_

## details

### interfaces and default loggers

- `lgr` package provides a single interface `lgr.L` with a single method `Logf(format string, args ...interface{})`. Function wrapper `lgr.Func` allows to make `lgr.L` from a function directly.
- Default logger functionality can be used without `lgr.New`, but just `lgr.Printf`
- Two predefined loggers available: `lgr.NoOp` (do-nothing logger) and `lgr.Std` (passing directly to stdlib log)

### options

`lgr.New` call accepts functional options:

- `lgr.Debug` - turn debug mode on. This allows messages with "DEBUG" level (filtered overwise)
- `lgr.Caller` - adds the caller info each message
- `lgr.Out(io.Writer)` - sets the output writer, default `os.Stdout`
- `lgr.Err(io.Writer)` - sets the error writer, default `os.Stderr`

### levels

`lgr.Logf` recognizes prefixes like "INFO" or "[INFO]" as levels. The full list of supported levels - "DEBUG", "INFO", "WARN", "ERROR", "PANIC" and "FATAL"

- `DEBUG` will be filtered unless `lgr.Debug` option defined
- `INFO` and `WARN` don't have any special behavior attached
- `ERROR` sends messages to both out and err writers
- "PANIC" and "FATAL" send messages to both out and err writers. In addition sends dump of callers and runtime info to err only, and call `os.Exit(1)`.
  