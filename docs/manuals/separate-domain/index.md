---
title: Configure Instance on a different domain
---

## How to configure a single Remark42 instance for multiple domains

### What doesn't work so far?

Unless discussion [#1139](https://github.com/umputun/remark42/discussions/1139) has a marked answer, authorisation using oAuth like GitHub or Google is impossible on domains other than the original one. Telegram, Email and anonymous auth would work everywhere.

### Setup

Set `ALLOWED_HOSTS='self',example1.org,example2.org` with your domain names and `AUTH_SAME_SITE=none`. The `'self'` value means "domain which Remark42 is installed on" so you don't need to write it twice.

### Technical details

`ALLOWED_HOSTS` sets CSP [frame-ancestors](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Security-Policy/frame-ancestors), which, once enabled, limits the domains where Remark42 would work. The default value is not set so that it would work on any domain.

`AUTH_SAME_SITE` sets the [SAME_SITE](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie/SameSite) attribute for authorisation cookies, allowing Remark42 either on the original domain and subdomains there  (default value, not set which equals to `Lax`) or allows setting authorisation cookies on any domain where remark42 is shown (`None` setting).

Here are all possible combinations of these two:

- Default setup with unaltered variables: comments are shown on any domain, but the authorisation wouldn't work anywhere, but on the same domain Remark42 is installed on and subdomains of it.
- `ALLOWED_HOSTS` set to a set of domains: comments are shown only on listed domains, authorisation wouldn't work anywhere, but on the same domain Remark42 is installed on and subdomains of it.
- `AUTH_SAME_SITE` set to `None`: comments are shown on any domain. The authorisation would work anywhere.
- `ALLOWED_HOSTS` set to a set of domains and `AUTH_SAME_SITE` set to `None`: comments are shown on listed domains. The authorisation would work on all of them.
