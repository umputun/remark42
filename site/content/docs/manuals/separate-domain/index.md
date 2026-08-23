---
title: Configure Instance on a different domain
---

## How to configure a single Remark42 instance for multiple domains

### What doesn't work so far?

Unless discussion [#1139](https://github.com/umputun/remark42/discussions/1139) has a marked answer, authorisation using oAuth like GitHub or Google is impossible on domains other than the original one. Telegram, Email and anonymous auth would work everywhere.

### Setup

Set `ALLOWED_HOSTS="'self',https://example1.org,https://example2.org"` with your domain names, and `AUTH_SEND_JWT_HEADER=true`.

`AUTH_SEND_JWT_HEADER` is what keeps a reader signed in across a page reload on Safari, in Chrome Incognito, and in any browser configured to block third-party cookies. Read [its security considerations](../../configuration/parameters/#security-considerations-for-authsend-jwt-header) before enabling it: it puts the token in a cookie JavaScript can read, which costs XSS exposure that a server-set `HttpOnly` cookie does not.

**`AUTH_SAME_SITE=none` is not needed alongside it**, which reverses what this page recommended for years, so it is worth showing the measurement instead of asserting it. Signing in anonymously from an embedded widget over https and dumping the browser's cookie jar gives, with the setting:

| name | httpOnly | partition key | written by |
| --- | --- | --- | --- |
| `JWT` | yes | none | the server |
| `XSRF-TOKEN` | no | none | the server |
| `JWT` | no | the embedding site | the widget |
| `XSRF-TOKEN` | no | the embedding site | the widget |

and without it, only the widget's own partitioned pair. Sign-in, posting and the reload all work either way, in a permissive browser and in one blocking third-party cookies. What the setting adds is the unpartitioned `HttpOnly` `JWT` in the first row, delivered as a third-party cookie to every listed domain wherever the browser still permits that. Nothing needs it, so leaving it at the default is the smaller exposure.

Keep `AUTH_SAME_SITE=none` if you are *not* setting `AUTH_SEND_JWT_HEADER`. Then the server's cookies are the only ones there are, and this is what allows them to be set off-domain at all, for as long as the reader's browser still accepts unpartitioned third-party cookies.

The `'self'` in `ALLOWED_HOSTS` value means "domain where Remark42 is installed on" and needed if you want `remark42.example.com/web/` to work in case you want to test something with it.

### Technical details

`ALLOWED_HOSTS` sets CSP [frame-ancestors](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/frame-ancestors), which, once enabled, limits the domains where Remark42 would work. The default value is `*` so that it would work on any domain`.

`AUTH_SAME_SITE` sets the [SAME_SITE](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite) attribute for authorisation cookies, allowing Remark42 either on the original domain and subdomains there (default value, which equals to `Lax`) or allows setting authorisation cookies on any domain where remark42 is shown (`None` setting).

`SameSite=None` is not sufficient on its own, and with `AUTH_SEND_JWT_HEADER` it is not necessary either. A browser that blocks third-party cookies drops a cookie set by Remark42 for a reader on another domain no matter what its `SameSite` value is, unless the cookie is explicitly marked [`Partitioned`](https://developer.mozilla.org/en-US/docs/Web/Privacy/Privacy_sandbox/Partitioned_cookies), and the server-set cookies are not. `AUTH_SEND_JWT_HEADER=true` is what closes that: the token comes back in an `X-JWT` response header and the widget stores it in its own cookie, written from inside the embedded frame and marked `SameSite=None; Secure; Partitioned`, so the browser keeps it for that embedding site and sends it back after a reload. That cookie is the widget's own doing and owes nothing to `AUTH_SAME_SITE`, which reaches only the pair the server sets.

A browser that refuses a cross-site `Set-Cookie` lacking `SameSite=None` refuses it outright, so with the setting left at its default the server's pair is absent from the jar, not present with a stricter attribute.

Note that this applies to Email, Telegram and anonymous authorisation, which the widget performs from inside the frame. It does not rescue oAuth, which completes in a popup that is a top-level page of its own, so the cookie set there belongs to the Remark42 domain and the embedded frame never sees it.

Here are all possible combinations of these two:

- Default setup with unaltered variables: comments are shown on any domain, but the authorisation wouldn't work anywhere, except on the same domain Remark42 is installed on and subdomains of it.
- `ALLOWED_HOSTS` set to a set of domains: comments are shown only on listed domains, and authorisation wouldn't work anywhere, expect on the same domain Remark42 is installed on and subdomains of it.
- `AUTH_SAME_SITE` set to `None`: comments are shown on any domain. Authorisation works on browsers that still permit third-party cookies, and stops working on the ones that block them.
- `ALLOWED_HOSTS` set to a set of domains and `AUTH_SAME_SITE` set to `None`: comments are shown on listed domains, with the same authorisation caveat.
- `ALLOWED_HOSTS` and `AUTH_SEND_JWT_HEADER=true`, with `AUTH_SAME_SITE` left alone: comments are shown on listed domains, and Email, Telegram and anonymous authorisation survives a reload whatever the browser's third-party cookie policy. This is the recommended arrangement. Adding `AUTH_SAME_SITE=none` on top changes nothing about whether a reader stays signed in; it only adds the server's unpartitioned cookies where the browser still takes them.
