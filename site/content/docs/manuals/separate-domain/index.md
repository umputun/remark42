---
title: Configure Instance on a different domain
---

## How to configure a single Remark42 instance for multiple domains

### What doesn't work so far?

Firefox with "block all third-party cookies" turned on: it keeps no cookie for an embedded widget, partitioned or not, whatever the configuration. OAuth sign-in fails there at once, and Email, Telegram and anonymous sign-in last only until the next page load. In every other browser and mode measured below, OAuth sign-in on a page under another registrable domain works over https with no setting, and Email, Telegram and anonymous sign-in work with `AUTH_SEND_JWT_HEADER=true`, as the setup below explains.

Discussion [#1139](https://github.com/umputun/remark42/discussions/1139) has the history of cross-domain OAuth.

### Setup

Set `ALLOWED_HOSTS="'self',https://example1.org,https://example2.org"` with your domain names, and `AUTH_SEND_JWT_HEADER=true` if you enable Email, Telegram or anonymous authorisation. OAuth providers need neither of the two cookie settings below; the reasons follow.

`AUTH_SEND_JWT_HEADER` is what keeps a reader who signed in by Email, Telegram or anonymously signed in across a page reload on Safari, in Chrome Incognito, and in Chrome configured to block third-party cookies. The one setting it does not survive is Firefox's "block all third-party cookies", which discards partitioned cookies as well, and which no configuration survives. Read [its security considerations](../../configuration/parameters/#security-considerations-for-authsend-jwt-header) before enabling it: it hands the token to the widget's script in a response header, which costs XSS exposure that a session kept only in a server-set `HttpOnly` cookie does not.

**For Email, Telegram and anonymous authorisation, `AUTH_SAME_SITE=none` is not needed alongside it**, which reverses what this page recommended for years, so it is worth showing the measurement instead of asserting it. Signing in anonymously from an embedded widget over https on Chromium with its default third-party cookie policy, and dumping the browser's cookie jar, gives with the setting:

| name | httpOnly | partition key | written by |
| --- | --- | --- | --- |
| `JWT` | yes | none | the server |
| `XSRF-TOKEN` | no | none | the server |
| `JWT` | yes | the embedding site | the server, for the embedded widget |
| `XSRF-TOKEN` | no | the embedding site | the server, for the embedded widget |

and without it, only the partitioned pair. Sign-in, posting and the reload all work either way, in a permissive browser and in one blocking third-party cookies. What the setting adds is the unpartitioned `HttpOnly` `JWT` in the first row, delivered as a third-party cookie to every listed domain wherever the browser still permits that. Email, Telegram and anonymous authorisation do not need it, so with only those enabled, leaving it at the default is the smaller exposure.

**OAuth needs neither setting.** OAuth completes in a popup, which is a top-level page of Remark42's own site, so the session it sets belongs to that site. When the widget is embedded on another site, the popup hands the sign-in over to the widget: it tells the widget which sign-in is under way before it goes to the provider, the provider's callback records which account signed in, and the widget then either uses Remark42's own session, where the browser lets the frame see it, or receives a session of its own. That session is a second pair of cookies, `__Host-JWT-EMBEDDED` (`HttpOnly`) and `__Host-XSRF-TOKEN-EMBEDDED`, set by the server with `SameSite=None; Secure; Partitioned`, so the browser keeps it for the embedding site. A `__Host-REMARK42-EMBEDDED` cookie tells the server the widget uses that pair; from then on the widget's requests are answered from it alone, and signing out in the widget leaves Remark42's own site signed in. This works with `AUTH_SAME_SITE` left at `default` or set to `lax` or `none`, whatever `AUTH_SEND_JWT_HEADER` says, and needs https, since a partitioned cookie has to be `Secure`. `strict` breaks OAuth sign-in: the provider sends the reader back to Remark42's callback from its own site, a browser does not send a `strict` cookie with that cross-site request, and the sign-in fails. Apple returns the reader with a cross-site POST, which carries no `lax` cookie either, so with Apple sign-in enabled set `AUTH_SAME_SITE=none`.

For Email, Telegram and anonymous authorisation, keep `AUTH_SAME_SITE=none` if you are *not* setting `AUTH_SEND_JWT_HEADER`. Then the server's cookies are the only ones there are, and this is what allows them to be set off-domain at all, for as long as the reader's browser still accepts unpartitioned third-party cookies.

The `'self'` in `ALLOWED_HOSTS` value means "domain where Remark42 is installed on" and needed if you want `remark42.example.com/web/` to work in case you want to test something with it.

### Technical details

`ALLOWED_HOSTS` has three uses. It sets CSP [frame-ancestors](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/frame-ancestors), which limits the hosts allowed to embed Remark42; when it is empty, any host can. Its plain host entries, with the host of `REMARK_URL`, also limit the address a sign-in returns the reader to; `'self'` and entries with `*` do not count there. Page titles are fetched only from addresses on the listed domains and their subdomains, and on the domain formed by the last two labels of the `REMARK_URL` host name and its subdomains; every redirect target is checked against the same domains.

`AUTH_SAME_SITE` sets the [SameSite](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite) attribute on the cookies the server sets. `none` lets those cookies be set on any domain where Remark42 is shown.

The `default` setting does not mean `Lax`. It means Remark42 emits no `SameSite` attribute at all and leaves the choice to the browser, and browsers differ: Chromium treats a missing attribute as `Lax` and so refuses the cookie cross-site, while Firefox accepts it. That difference is not a detail, because it decides which of the two cookie pairs below a reader actually ends up with.

`SameSite=None` is not sufficient on its own, and for Email, Telegram and anonymous authorisation with `AUTH_SEND_JWT_HEADER` it is not necessary either. A browser that blocks third-party cookies drops a cookie set by Remark42 for a reader on another domain no matter what its `SameSite` value is, unless the cookie is explicitly marked [`Partitioned`](https://developer.mozilla.org/en-US/docs/Web/Privacy/Privacy_sandbox/Partitioned_cookies), and the server-set cookies are not. `AUTH_SEND_JWT_HEADER=true` is what closes that: the widget tells the server that it is embedded on another host, and the server then sets its two cookies a second time, marked `SameSite=None; Secure; Partitioned`, in every response that sets them, so the browser keeps that pair for the embedding site and sends it back after a reload. The partitioned pair carries `SameSite=None` whatever `AUTH_SAME_SITE` says, which reaches only the ordinary pair. The token also comes back in an `X-JWT` response header, which the widget keeps in memory for as long as the page stays open. The exception is Firefox's "block all third-party cookies", which discards partitioned cookies too, so the flag closes nothing there and no configuration does.

A browser that refuses a cross-site `Set-Cookie` lacking `SameSite=None` refuses it outright, so with the setting left at its default the server's pair is absent from the jar, not present with a stricter attribute.

Note that this applies to Email, Telegram and anonymous authorisation, which the widget performs from inside the frame. OAuth completes in a popup that is a top-level page of its own, so the cookie set there is an ordinary unpartitioned cookie of the Remark42 domain; the handoff described above is what carries the sign-in over to the frame.

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

**OAuth has a table of its own.** The table above is for Email, Telegram and anonymous authorisation. OAuth was measured in the same cross-site arrangement, with a test provider on a third registrable domain behind the custom OAuth2 provider, and with Google. A cell works when, after the popup sign-in, the widget shows the account that signed in, a comment posted through it appears, and both are still there after a reload. Chrome 158, a release build with its stock flags, ran in three modes: default settings, third-party cookies blocked in settings, and an Incognito window. Firefox 160, also a release build, ran in four: default settings, a private window, strict tracking protection, and "block all third-party cookies". Safari 27.2 ran with its own settings, with neither variable set.

| configuration | Chrome, default | Chrome, third-party cookies blocked or Incognito | Firefox, default, private or strict | Firefox, "block all third-party" | Safari |
| --- | --- | --- | --- | --- | --- |
| neither set | works | works | works | fails | works |
| `AUTH_SEND_JWT_HEADER` only | works | works | works | fails | not measured |
| `AUTH_SAME_SITE=none` only | works | works | works | fails | not measured |
| both | works | works | works | fails | not measured |

A provider that cuts the sign-in window off from the page that opened it, with `Cross-Origin-Opener-Policy: same-origin`, changed no cell, and neither did serving Remark42 below a path. Which session the widget ends up with depends on the browser. Chrome with its default settings and `AUTH_SAME_SITE=none`, and Firefox, which lets a frame use its own site's cookies after a sign-in window the frame opened, show the widget Remark42's own session, so the widget and Remark42's site are one session there: signing in as somebody else on one signs the other in as them too, and signing out of one signs out of both. Everywhere else the widget gets the session of its own described in the setup above.

If the browser blocks the sign-in window, the widget says so and asks to allow pop-ups for the page.

With Remark42 and the host page on two subdomains of one registrable domain, OAuth works in every cell measured, as it does on Remark42's own pages: the widget there always sees the session the sign-in window set.

Here is what the common combinations of `ALLOWED_HOSTS`, `AUTH_SAME_SITE` and `AUTH_SEND_JWT_HEADER` give:

- Default setup with unaltered variables: comments are shown on any domain. OAuth works everywhere over https except in Firefox's "block all third-party cookies". Email, Telegram and anonymous authorisation work only on the same domain Remark42 is installed on and other hosts under the same registrable domain, and on Firefox with its default settings. Firefox stores the server's attribute-less cookie in a per-site partition instead of refusing it, so Email, Telegram and anonymous authorisation survive a reload cross-domain there with neither variable set. Every other browser measured above refuses that cookie cross-site.
- `ALLOWED_HOSTS` set to a set of domains: comments are shown only on listed domains, with the same authorisation as above.
- `AUTH_SAME_SITE` set to `None`: comments are shown on any domain. Email, Telegram and anonymous authorisation work on browsers that still permit third-party cookies, and stop working on the ones that block them. OAuth works as with the default setup.
- `ALLOWED_HOSTS` set to a set of domains and `AUTH_SAME_SITE` set to `None`: comments are shown on listed domains, with the same authorisation caveat.
- `ALLOWED_HOSTS` and `AUTH_SEND_JWT_HEADER=true`, with `AUTH_SAME_SITE` left alone: comments are shown on listed domains, and Email, Telegram and anonymous authorisation survives a reload under every third-party cookie policy except Firefox's "block all third-party cookies", which no configuration survives. OAuth works as with the default setup. This is the recommended arrangement. Adding `AUTH_SAME_SITE=none` on top changes nothing about whether a reader who used one of those three methods stays signed in; it only adds the server's unpartitioned cookies where the browser still takes them.
- `ALLOWED_HOSTS`, `AUTH_SEND_JWT_HEADER=true` and `AUTH_SAME_SITE=none`: the same, plus the server's unpartitioned cookies wherever the browser still takes them, which OAuth does not need either.
