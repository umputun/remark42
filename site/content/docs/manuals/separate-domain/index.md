---
title: Configure Instance on a different domain
---

## How to configure a single Remark42 instance for multiple domains

### What doesn't work so far?

Unless discussion [#1139](https://github.com/umputun/remark42/discussions/1139) has a marked answer, authorisation using oAuth like GitHub or Google is impossible on domains other than the original one. Telegram, Email and anonymous auth work on allowed HTTPS embedding domains when `AUTH_SEND_JWT_HEADER=true`.

### Setup

Set `ALLOWED_HOSTS="'self',https://example1.org,https://example2.org"` with your domain names, and `AUTH_SEND_JWT_HEADER=true`.

`AUTH_SEND_JWT_HEADER` is what keeps a reader signed in across a page reload on Safari, in Chrome Incognito, and in Chrome configured to block third-party cookies. The one setting it does not survive is Firefox's "block all third-party cookies", which discards partitioned cookies as well, and which no configuration survives. Read [its security considerations](../../configuration/parameters/#security-considerations-for-authsend-jwt-header) before enabling it: it puts the token in a cookie JavaScript can read, which costs XSS exposure that a server-set `HttpOnly` cookie does not.

**`AUTH_SAME_SITE=none` is not needed alongside it**, which reverses what this page recommended for years, so it is worth showing the measurement instead of asserting it. Signing in anonymously from an embedded widget over https on Chromium with its default third-party cookie policy, and dumping the browser's cookie jar, gives with the setting:

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

`ALLOWED_HOSTS` sets CSP [frame-ancestors](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/frame-ancestors), which, once enabled, limits the domains where Remark42 would work. The default value is `*` so that it would work on any domain.

`AUTH_SAME_SITE` sets the [SameSite](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite) attribute on the cookies the server sets. `none` lets those cookies be set on any domain where Remark42 is shown.

The `default` setting does not mean `Lax`. It means Remark42 emits no `SameSite` attribute at all and leaves the choice to the browser, and browsers differ: Chromium treats a missing attribute as `Lax` and so refuses the cookie cross-site, while Firefox accepts it. That difference is not a detail, because it decides which of the two cookie pairs below a reader actually ends up with.

`SameSite=None` is not sufficient on its own, and with `AUTH_SEND_JWT_HEADER` it is not necessary either. A browser that blocks third-party cookies drops a cookie set by Remark42 for a reader on another domain no matter what its `SameSite` value is, unless the cookie is explicitly marked [`Partitioned`](https://developer.mozilla.org/en-US/docs/Web/Privacy/Privacy_sandbox/Partitioned_cookies), and the server-set cookies are not. `AUTH_SEND_JWT_HEADER=true` is what closes that: the token comes back in an `X-JWT` response header and the widget stores it in its own cookie, written from inside the embedded frame and marked `SameSite=None; Secure; Partitioned`, so the browser keeps it for that embedding site and sends it back after a reload. That cookie is the widget's own doing and owes nothing to `AUTH_SAME_SITE`, which reaches only the pair the server sets. The exception is Firefox's "block all third-party cookies", which discards partitioned cookies too, so the flag closes nothing there and no configuration does.

A browser that refuses a cross-site `Set-Cookie` lacking `SameSite=None` refuses it outright, so with the setting left at its default the server's pair is absent from the jar, not present with a stricter attribute.

Note that this applies to Email, Telegram and anonymous authorisation, which the widget performs from inside the frame. It does not rescue oAuth, which completes in a popup that is a top-level page of its own, so the cookie set there belongs to the Remark42 domain and the embedded frame never sees it.

### What each browser actually does

Measured on real domains over real certificates, with Remark42 on one registrable domain and the
host page on another, signing in and then reloading. Chrome and Firefox were driven through
Playwright, Safari 27 through its own WebDriver, so the Safari column is Safari itself and not an
approximation of it. Every "blocked" column below was verified with a control cookie: an ordinary
third-party cookie written from inside the widget frame has to be dropped, or the run is not
blocking anything and proves nothing.

| configuration | Chrome, default | Chrome, third-party cookies blocked | Firefox, default | Firefox, "block all third-party" | Safari |
| --- | --- | --- | --- | --- | --- |
| neither set | fails | fails | works | fails | fails |
| `AUTH_SEND_JWT_HEADER` only | works | works | works | fails | works |
| `AUTH_SAME_SITE=none` only | works | fails | works | fails | fails |
| both | works | works | works | fails | works |

Four things in that table are worth spelling out.

**Firefox keeps a reader signed in with nothing configured at all.** Its Total Cookie Protection files the server's attribute-less cookie under the embedding site's partition instead of refusing it, so Email, Telegram and anonymous authorisation survive a reload cross-domain on a stock Firefox with neither variable set. No other browser measured here does that: Chrome sends no `SameSite` attribute by default and treats the cookie as `Lax`, and Safari blocks it as third-party. Do not read that row as a recommendation, since it holds on one engine and breaks the moment the reader turns on "block all third-party cookies".

**Safari needs no configuring to break the old recipe.** It blocks third-party cookies out of the
box while still honouring `Partitioned`, so `AUTH_SAME_SITE=none` on its own has already stopped
working there for every reader. This is not a future deprecation to plan for. With the header flag
the widget's own partitioned cookie is readable in the frame and the session survives the reload.

**Firefox reaches "works" by a different route, and a weaker one.** Chrome and Safari refuse the
server's cookie when it carries no `SameSite` attribute, which leaves the field clear for the
widget to write its own partitioned pair. Firefox accepts that cookie, and because the server's
`JWT` is `HttpOnly`, the browser then refuses to let the widget's script overwrite it: a cookie set
by JavaScript may not replace an `HttpOnly` one of the same name. So on Firefox the session rides
on an ordinary unpartitioned third-party cookie even with the header flag on, and it disappears the
moment the reader blocks those.

**No configuration survives Firefox's "block all third-party cookies" setting.** That mode discards
partitioned cookies too, so the `Partitioned` escape hatch does not apply. A reader who has turned
it on cannot stay signed in on an embedded widget, and nothing in Remark42 can change that.

Here are all possible combinations of these two:

- Default setup with unaltered variables: comments are shown on any domain, and the authorisation wouldn't work anywhere, except on the same domain Remark42 is installed on and subdomains of it, and on Firefox with its default settings. Firefox stores the server's attribute-less cookie in a per-site partition instead of refusing it, so Email, Telegram and anonymous authorisation survive a reload cross-domain there with neither variable set. Every other browser measured below refuses that cookie cross-site.
- `ALLOWED_HOSTS` set to a set of domains: comments are shown only on listed domains, and authorisation wouldn't work anywhere, except on the same domain Remark42 is installed on and subdomains of it, and on Firefox with its default settings, for the reason above.
- `AUTH_SAME_SITE` set to `None`: comments are shown on any domain. Authorisation works on browsers that still permit third-party cookies, and stops working on the ones that block them.
- `ALLOWED_HOSTS` set to a set of domains and `AUTH_SAME_SITE` set to `None`: comments are shown on listed domains, with the same authorisation caveat.
- `ALLOWED_HOSTS` and `AUTH_SEND_JWT_HEADER=true`, with `AUTH_SAME_SITE` left alone: comments are shown on listed domains, and Email, Telegram and anonymous authorisation survives a reload under every third-party cookie policy except Firefox's "block all third-party cookies", which no configuration survives. This is the recommended arrangement. Adding `AUTH_SAME_SITE=none` on top changes nothing about whether a reader stays signed in; it only adds the server's unpartitioned cookies where the browser still takes them.
