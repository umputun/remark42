---
title: Configure Instance on a different domain
---

## How to configure a single Remark42 instance for multiple domains

### What doesn't work so far?

When Remark42 and the embedding page are on different registrable domains, authorisation using OAuth like GitHub or Google works in one case only: of the browsers measured, in Chrome with its default settings, and only when `AUTH_SAME_SITE=none` is set. It fails in Chrome with third-party cookies blocked or in Incognito, in Firefox and in Safari, whatever the configuration. Discussion [#1139](https://github.com/umputun/remark42/discussions/1139) is where this is tracked. Telegram, Email and anonymous auth work on allowed HTTPS embedding domains when `AUTH_SEND_JWT_HEADER=true`.

OAuth works in every browser measured when Remark42 and the page share a registrable domain, for example Remark42 on `remark42.example.com` and the page on `www.example.com`. One instance has one `REMARK_URL`, so that covers one site and not the other domains it is embedded on.

### Setup

Set `ALLOWED_HOSTS="'self',https://example1.org,https://example2.org"` with your domain names, and `AUTH_SEND_JWT_HEADER=true`. If you enable OAuth providers, set `AUTH_SAME_SITE=none` as well; the reasons follow.

`AUTH_SEND_JWT_HEADER` is what keeps a reader who signed in by Email, Telegram or anonymously signed in across a page reload on Safari, in Chrome Incognito, and in Chrome configured to block third-party cookies. The one setting it does not survive is Firefox's "block all third-party cookies", which discards partitioned cookies as well, and which no configuration survives. Read [its security considerations](../../configuration/parameters/#security-considerations-for-authsend-jwt-header) before enabling it: it hands the token to the widget's script in a response header, which costs XSS exposure that a session kept only in a server-set `HttpOnly` cookie does not.

**For Email, Telegram and anonymous authorisation, `AUTH_SAME_SITE=none` is not needed alongside it**, which reverses what this page recommended for years, so it is worth showing the measurement instead of asserting it. Signing in anonymously from an embedded widget over https on Chromium with its default third-party cookie policy, and dumping the browser's cookie jar, gives with the setting:

| name | httpOnly | partition key | written by |
| --- | --- | --- | --- |
| `JWT` | yes | none | the server |
| `XSRF-TOKEN` | no | none | the server |
| `JWT` | yes | the embedding site | the server, for the embedded widget |
| `XSRF-TOKEN` | no | the embedding site | the server, for the embedded widget |

and without it, only the partitioned pair. Sign-in, posting and the reload all work either way, in a permissive browser and in one blocking third-party cookies. What the setting adds is the unpartitioned `HttpOnly` `JWT` in the first row, delivered as a third-party cookie to every listed domain wherever the browser still permits that. Email, Telegram and anonymous authorisation do not need it, so with only those enabled, leaving it at the default is the smaller exposure.

**OAuth does need it.** OAuth completes in a popup, which is a top-level page of its own, so the response that sets the session does not answer the embedded widget and carries no partitioned pair. The frame can then be signed in only by the server's `JWT` cookie from the first row. With `AUTH_SAME_SITE` left at `default` that cookie carries no `SameSite` attribute, Chrome treats it as `Lax` and does not send it from a frame on another site, so `AUTH_SEND_JWT_HEADER=true` on its own leaves cross-site OAuth failing in every browser measured. If you enable OAuth providers, keep `AUTH_SAME_SITE=none` next to `AUTH_SEND_JWT_HEADER=true`, at the cost described above. That makes cross-site OAuth work in Chrome with its default settings and in no other browser or mode measured; the OAuth table below has the rest.

Also keep `AUTH_SAME_SITE=none` if you are *not* setting `AUTH_SEND_JWT_HEADER`. Then the server's cookies are the only ones there are, and this is what allows them to be set off-domain at all, for as long as the reader's browser still accepts unpartitioned third-party cookies.

The `'self'` in `ALLOWED_HOSTS` value means "domain where Remark42 is installed on" and needed if you want `remark42.example.com/web/` to work in case you want to test something with it.

### Technical details

`ALLOWED_HOSTS` sets CSP [frame-ancestors](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/frame-ancestors), which, once enabled, limits the domains where Remark42 would work. The default value is `*` so that it would work on any domain.

`AUTH_SAME_SITE` sets the [SameSite](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite) attribute on the cookies the server sets. `none` lets those cookies be set on any domain where Remark42 is shown.

The `default` setting does not mean `Lax`. It means Remark42 emits no `SameSite` attribute at all and leaves the choice to the browser, and browsers differ: Chromium treats a missing attribute as `Lax` and so refuses the cookie cross-site, while Firefox accepts it. That difference is not a detail, because it decides which of the two cookie pairs below a reader actually ends up with.

`SameSite=None` is not sufficient on its own, and for Email, Telegram and anonymous authorisation with `AUTH_SEND_JWT_HEADER` it is not necessary either. A browser that blocks third-party cookies drops a cookie set by Remark42 for a reader on another domain no matter what its `SameSite` value is, unless the cookie is explicitly marked [`Partitioned`](https://developer.mozilla.org/en-US/docs/Web/Privacy/Privacy_sandbox/Partitioned_cookies), and the server-set cookies are not. `AUTH_SEND_JWT_HEADER=true` is what closes that: the widget tells the server that it is embedded on another host, and the server then sets its two cookies a second time, marked `SameSite=None; Secure; Partitioned`, in every response that sets them, so the browser keeps that pair for the embedding site and sends it back after a reload. The partitioned pair carries `SameSite=None` whatever `AUTH_SAME_SITE` says, which reaches only the ordinary pair. The token also comes back in an `X-JWT` response header, which the widget keeps in memory for as long as the page stays open. The exception is Firefox's "block all third-party cookies", which discards partitioned cookies too, so the flag closes nothing there and no configuration does.

A browser that refuses a cross-site `Set-Cookie` lacking `SameSite=None` refuses it outright, so with the setting left at its default the server's pair is absent from the jar, not present with a stricter attribute.

Note that this applies to Email, Telegram and anonymous authorisation, which the widget performs from inside the frame. It does not rescue OAuth, which completes in a popup that is a top-level page of its own, so the cookie set there is an ordinary unpartitioned cookie of the Remark42 domain. On a page under another registrable domain, the embedded frame sees it only where the browser still sends unpartitioned third-party cookies and the cookie is `SameSite=None`, which of the browsers measured means Chrome with its default settings and `AUTH_SAME_SITE=none`.

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
the partitioned pair is kept for the frame and the session survives the reload.

**Firefox reaches "works" with both pairs.** Chrome and Safari refuse the server's ordinary cookie
when it carries no `SameSite` attribute, so there the session rests on the partitioned pair alone.
Firefox accepts the ordinary cookie as well and files it under the embedding site, so with the
header flag on a Firefox reader holds both pairs and sends both from the frame.

**No configuration survives Firefox's "block all third-party cookies" setting.** That mode discards
partitioned cookies too, so the `Partitioned` escape hatch does not apply. A reader who has turned
it on cannot stay signed in on an embedded widget, and nothing in Remark42 can change that.

**OAuth has a table of its own.** The table above is for Email, Telegram and anonymous authorisation. OAuth was measured separately in the same cross-site arrangement, on Remark42 v1.18.0 (released 11 October 2026) with a test provider on a third registrable domain behind the custom OAuth2 provider. A cell works when `/api/v1/user`, requested from inside the widget frame, answers for the signed-in user after the popup sign-in and again after a reload. Chrome 157, a release build with its stock flags, ran in three modes: default settings, third-party cookies blocked in settings, and an Incognito window. Firefox 154, also a release build, ran in four: default settings, a private window, strict tracking protection, and "block all third-party cookies". Safari 27.2 was driven by hand; all four of its configurations failed straight after sign-in, and the reload was recorded for the first only. Every Chrome and Firefox sign-in was matched against the backend log, and blocking was confirmed with the same control cookie as above.

| configuration | Chrome, default | Chrome, third-party cookies blocked or Incognito | Firefox, all four modes | Safari |
| --- | --- | --- | --- | --- |
| neither set | fails | fails | fails | fails |
| `AUTH_SEND_JWT_HEADER` only | fails | fails | fails | fails |
| `AUTH_SAME_SITE=none` only | works | fails | fails | fails |
| both | works | fails | fails | fails |

The two cells that work are Chrome with its default settings and `AUTH_SAME_SITE=none`, where the popup's cookie is `SameSite=None` and Chrome still sends it from the frame as an unpartitioned third-party cookie. `AUTH_SEND_JWT_HEADER` changes no cell, because the header never reaches the widget. In every other cell the frame does not get the popup's cookie, and no configuration changes that.

With Remark42 and the host page on two subdomains of one registrable domain, OAuth worked in every cell measured: all four configurations in each Chrome and Firefox mode above, and Safari with neither variable set. The same run repeated the first table with anonymous sign-in, and every cell agreed with it.

Here is what the common combinations of `ALLOWED_HOSTS`, `AUTH_SAME_SITE` and `AUTH_SEND_JWT_HEADER` give:

- Default setup with unaltered variables: comments are shown on any domain, and the authorisation wouldn't work anywhere, except on the same domain Remark42 is installed on and other hosts under the same registrable domain, where it works, OAuth included, and on Firefox with its default settings, for Email, Telegram and anonymous authorisation only. Firefox stores the server's attribute-less cookie in a per-site partition instead of refusing it, so Email, Telegram and anonymous authorisation survive a reload cross-domain there with neither variable set. Every other browser measured above refuses that cookie cross-site.
- `ALLOWED_HOSTS` set to a set of domains: comments are shown only on listed domains, and authorisation wouldn't work anywhere, with the same two exceptions as above.
- `AUTH_SAME_SITE` set to `None`: comments are shown on any domain. Authorisation works on browsers that still permit third-party cookies, and stops working on the ones that block them. OAuth across registrable domains is narrower: of the browsers measured it works only in Chrome with its default settings, and not in Firefox, where the other methods do work.
- `ALLOWED_HOSTS` set to a set of domains and `AUTH_SAME_SITE` set to `None`: comments are shown on listed domains, with the same authorisation caveat.
- `ALLOWED_HOSTS` and `AUTH_SEND_JWT_HEADER=true`, with `AUTH_SAME_SITE` left alone: comments are shown on listed domains, and Email, Telegram and anonymous authorisation survives a reload under every third-party cookie policy except Firefox's "block all third-party cookies", which no configuration survives. OAuth does not work across registrable domains in this arrangement. This is the recommended arrangement when no OAuth provider is enabled. Adding `AUTH_SAME_SITE=none` on top changes nothing about whether a reader who used one of those three methods stays signed in; it only adds the server's unpartitioned cookies where the browser still takes them.
- `ALLOWED_HOSTS`, `AUTH_SEND_JWT_HEADER=true` and `AUTH_SAME_SITE=none`: the same, plus OAuth in Chrome with its default settings, the only browser measured where OAuth works across registrable domains. This is the arrangement to use when OAuth providers are enabled.
