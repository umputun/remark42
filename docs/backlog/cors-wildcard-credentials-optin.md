---
worth: yes
where: backend/app/rest/api/middleware.go:127
added: 2026-08-18
---
# corsMiddleware panics at boot once go-pkgz/rest is bumped past v1.22.0

`corsMiddleware` passes `R.CorsAllowedOrigins("*")` together with `R.CorsAllowCredentials(true)`
(lines 128 and 132). go-pkgz/rest#52 makes `rest.CORS` panic at construction on exactly that
combination, so the next bump of `github.com/go-pkgz/rest` past the pinned `v1.22.0`
(`backend/go.mod:16`) crashes remark42 on startup, not at request time.

The fix is one option:

```go
R.CorsAllowedOrigins("*"),
R.CorsAllowCredentials(true),
R.CorsUnsafeAnyOriginWithCredentials(true),
```

It cannot be added before the bump, since the option does not exist in v1.22.0. So this is bump and
edit in one commit, in either order within that commit, and it will not compile split across two.

Why the wildcard stays rather than the origins being enumerated: remark42 serves a comment widget
embedded on arbitrary third-party sites, so the set of origins is not knowable. The upstream default
is right for a normal service and wrong here, which is why the escape hatch was asked for instead of
accepting the panic. The named consequence stands and is worth re-reading when this is touched: any
site a signed-in user visits can read authenticated responses, so state-changing requests must keep
being protected by something other than the origin (`X-XSRF-Token` today).

Surfaced reviewing go-pkgz/rest#52. The unconditional panic in the original version of that PR was
pushed back on precisely because remark42 had no way to comply; `CorsUnsafeAnyOriginWithCredentials`
exists as a result.
