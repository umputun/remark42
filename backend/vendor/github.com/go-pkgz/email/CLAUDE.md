# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`github.com/go-pkgz/email` is a single-package library wrapping the stdlib `net/smtp` to simplify sending
messages (alerts, notifications, password-reset mails). It has no runtime dependencies — only `testify` for
tests. Not designed for high-volume/low-latency bulk sending. Go 1.19.

## Commands

- Test: `go test -race ./...` — CI runs with `TZ=America/Chicago`; the Date header uses RFC1123Z with an
  injectable clock, so timezone can affect assertions. Match CI: `TZ=America/Chicago go test -race ./...`.
- Single test: `go test -run TestEmail_Send`
- Lint: `golangci-lint run` from repo root (config is golangci-lint **v2** format; CI pins v2.6)
- Regenerate mocks: `go generate ./...` (requires `moq`; do not hand-edit files under `mocks/`)

## Architecture

Three source files, one package:

- `email.go` — `Sender` type, `Send`, and MIME message construction (`buildMessage`).
- `options.go` — functional options (`Option = func(*Sender)`), all applied in `NewSender`.
- `auth.go` — custom LOGIN SASL auth mechanism.

Key design points that span files:

- **Functional options.** `Sender` fields are unexported and set only through `Option` funcs in `options.go`,
  applied in `NewSender` after defaults. Add a new config knob = new `Option` func + field, nothing else.

- **`SMTPClient` interface is the seam.** It's a consumer-side subset of `net/smtp.Client` (Mail/Auth/Rcpt/
  Data/Quit/Close). If the caller injects one via the `SMTP()` option it's reused; otherwise `Send` builds a
  fresh `net/smtp` client per call via `em.client()`. This interface is what the moq mock implements, so all
  `Send` tests run without a real SMTP server.

- **`em.client()` handles the three transport modes:** plain (dial + optional STARTTLS), and implicit `TLS`
  (dial over TLS, port 465). `STARTTLS` upgrades a plain connection (port 587). `InsecureSkipVerify` feeds the
  `tls.Config`.

- **`buildMessage` is the intricate part.** It assembles headers manually, then bodies: `multipart/mixed` for
  attachments, `multipart/related` for inline images, quoted-printable for the text body. Inline images get an
  auto `Content-ID` equal to their filename. Attachment/inline type is sniffed from the first 512 bytes via
  `http.DetectContentType`, then the file is re-seeked to 0 and base64-encoded. Changes here are easy to break
  silently — the tests assert on the exact serialized message string.

- **LOGIN auth (`auth.go`) exists because stdlib only ships PLAIN.** Needed for Office 365 / Outlook.com.
  Enabled with the `LoginAuth()` option; it refuses to send credentials over an unencrypted, non-localhost
  connection. `Sender.auth()` picks PLAIN vs LOGIN and returns nil (no auth) when username/password are empty.

- **Envelope vs. headers.** `extractEmailAddress` (via `net/mail`) strips a display name so `MAIL FROM` /
  `RCPT TO` get a bare address, while the `From`/`To` *headers* keep the full `"Name" <addr>` form. Falls back
  to the raw string if parsing fails.

- **`timeNow` field** is an injectable clock (`func() time.Time`, defaults to `time.Now`) so tests can pin the
  Date header deterministically — set `s.timeNow` directly in tests.

## Testing conventions

- Mocks are moq-generated into `mocks/` from `//go:generate` directives at the top of `email.go` for the
  `SMTPClient` and `Logger` interfaces. Use the mock's `*Calls()` accessors to assert interactions.
- `testdata/` holds attachment fixtures (`1.txt`, `2.txt`, `image.jpg`, `nullfile` for the empty-file path).
- Coverage in CI strips `mocks`/`_mock.go` lines before submitting to coveralls.
