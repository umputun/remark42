# Threat model

## What this project does and where untrusted input enters

Remark42 is a self-hosted comment engine. A Go server (`backend/`) stores comments in BoltDB and serves a REST API; a Preact widget (`frontend/apps/remark42/`) is embedded into third-party sites through an iframe and renders comments in the Remark42 origin. One server can host several sites (`--site`), each with its own comments and admins.

Untrusted input comes from:

- anonymous visitors: every route in the open groups of `backend/app/rest/api/rest.go`, including `/api/v1/img`, RSS, counts and the unsubscribe page
- authenticated commenters: comment create, edit and preview, votes, image upload, email and Telegram subscriptions, account deletion. Login is cheap: OAuth providers, email confirmation, Telegram, and an optional anonymous login, so treat "authenticated" as "any internet user"
- OAuth providers and user-supplied avatar URLs
- remote servers the backend fetches from: the image proxy (`rest/proxy`) and the page title extractor (`store/service/title.go`)
- the host page that embeds the widget, through `postMessage` (`app/utils/post-message.ts`, `app/embed.ts`, `app/remark.tsx`)

Site admins are trusted for their own site but not for other sites on the same server. The operator who sets the command-line flags is fully trusted.

## Components that matter most

- Comment rendering: markdown to HTML in `store/formatter.go`, sanitizing with bluemonday in `store/comment.go`, and every place that re-emits comment text: the widget, the last-comments widget, RSS, and email, Telegram, Slack and webhook notifications (`notify/`). Past reports were stored XSS and HTML injection through these paths, including the browser re-parsing text the server had already sanitized.
- Image proxy and title extractor: SSRF, protection bypass of `safehttp` (private, reserved and encoded address forms), responses that the browser can interpret as HTML in the Remark42 origin, and use of the server as an open relay.
- Authentication and authorization: JWT cookies and the XSRF header (go-pkgz/auth), `--secret` and per-site secrets from the RPC admin store (`--admin.rpc.secret_per_site`), the anonymous and email providers, admin checks on `/api/v1/admin`, and isolation between sites (a user or admin of one site affecting another).
- Client IP handling: rate limits, vote de-duplication and anonymous identities depend on the client IP, so check how forwarded headers are trusted.
- Widget: iframe messaging origin checks, URL and locator handling, raw HTML insertion, clipboard and other DOM sinks.

Lower priority: admin import, export and remap parsers (`rest/api/migrator.go`, `migrator/`), which only admins reach, and the RPC store and admin backends, which talk to operator-configured servers.

## How to exercise it

The image contains `/src/backend` and `/src/frontend/apps/remark42` only, with no `.git`, docs, site or e2e directories. Go dependencies are vendored and work offline; widget dependencies are installed in `node_modules`.

- Backend tests: `cd /src/backend/app && go test ./...`. Tests in `rest/api` start the full server and are the easiest base for an end-to-end reproducer.
- Widget tests: `cd /src/frontend/apps/remark42 && pnpm test` (jest) or `pnpm test:unit`.
- Running server: `/usr/local/bin/remark42` is built with the widget embedded. A minimal local instance: `SECRET=test REMARK_URL=http://127.0.0.1:8080 remark42 server --site=remark --auth.anon --admin-passwd=password --store.bolt.path=/tmp/r42 --backup=/tmp/r42`, then use the API with `curl`. Admin basic auth is user `admin` with that password.

## How to rate severity

- Critical: unauthenticated code execution, authentication bypass, reading or forging another user's session, admin takeover.
- High: stored XSS that runs without unusual victim interaction, SSRF reaching internal or metadata addresses, crossing site isolation, privilege escalation from commenter to admin.
- Medium: reflected or interaction-heavy XSS, HTML injection into notifications, rate limit or vote manipulation, information disclosure about the server or other users.
- Low: issues that require a non-default insecure flag, and resource exhaustion of a single self-hosted instance.

A report should name the attacker position (anonymous, commenter, admin of another site), the flags it needs, and include a reproducer.

## Anything to leave alone

- Code in `backend/vendor` and `node_modules`; report it only if Remark42 calls it in an unsafe way.
- The `--auth.dev` provider, which exists for local development.
- Missing hardening headers or cookie flags without a working attack.
- Behavior controlled entirely by the operator's own configuration.
