# AGENTS.md

This file provides guidance to AI coding agents (Claude Code and others) when working with code in this repository.

## Project

`github.com/xdg-go/stringprep`: a Go implementation of RFC-3454 stringprep (with errata as of 2018-02) and the RFC-4013 SASLprep profile. Single package, no subpackages. The only dependency is `golang.org/x/text` (for NFKC normalization). This is a stable v1 library; recent releases only bump `golang.org/x/text` for CVEs.

## Commands

```sh
go test -race ./...                 # what CI runs
go test -run TestSASLprep ./...     # single test
go vet ./...
```

CI (`.github/workflows/test.yml`) tests on Ubuntu with the Go version from `go.mod` (1.18), plus `oldstable` and `stable`. Keep code compatible with Go 1.18 (no newer stdlib APIs) unless the minimum version is deliberately raised.

## Architecture

The algorithm lives in `Profile.Prepare` (`profile.go`), which runs the RFC-3454 steps in order:

1. **Map:** each rune goes through `Profile.Mappings` in order; the first `Mapping` that contains the rune wins, and unmapped runes pass through.
2. **Normalize:** NFKC via `golang.org/x/text/unicode/norm` when `Normalize` is true.
3. **Prohibit:** any rune in any `Profile.Prohibits` set returns an `Error`.
4. **BiDi:** when `CheckBiDi` is true, `passesBiDiRules` (`bidi.go`) applies RFC-3454 section 6 using tables C.8, D.1, and D.2. Empty strings pass.

Data types:

- **`Set`** (`set.go`): a slice of closed `RuneRange` intervals. `Contains` uses binary search, so every `Set` table must stay sorted and non-overlapping.
- **`Mapping`** (`map.go`): `map[rune][]rune`; an empty slice means "map to nothing" (table B.1).
- **`Error`** (`error.go`): carries the message and the offending rune. Error messages are unexported package vars, and tests compare whole `Error` values with `reflect.DeepEqual`, so changing a message or the offending rune breaks tests.

`tables.go` (~3200 lines) holds every RFC-3454 appendix table. Each table is an unexported var (`tableC2_1`) re-exported as `TableC2_1`; underscores replace dots in the table numbers. No generator script exists in the repo, so edit tables by hand and verify against the RFC text.

`saslprep.go` defines the `SASLprep` profile as a composition of these tables plus a local non-ASCII-space mapping. It operates only in "stored strings" mode (prohibits unassigned code points via table A.1); the doc comment explains why.

## Conventions

- Tests are table-driven in the `stringprep` package (white-box). `examples_test.go` holds the godoc examples.
- `CHANGELOG.md` follows the git-chglog config in `.chglog/` (commit types `feat`, `fix`, `perf`, `refactor`, `task` → "Maintenance"), but recent entries were edited by hand.
- Source files carry the Apache 2.0 copyright header.
