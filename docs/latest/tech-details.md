---
title: Technical details
---

# Technical details

* Data stored in [boltdb](https://github.com/coreos/bbolt) (embedded key/value database) files under `STORE_BOLT_PATH`
* Each site stored in a separate boltbd file.
* In order to migrate/move remark42 to another host boltbd files as well as avatars directory `AVATAR_FS_PATH` should be transferred. Optionally, boltdb can be used to store avatars as well.
* Automatic backup process runs every 24h and exports all content in json-like format to `backup-remark-YYYYMMDD.gz`.
* Authentication implemented with [go-pkgz/auth](https://github.com/go-pkgz/auth) stored in a cookie. It uses HttpOnly, secure cookies.
* All heavy REST calls cached internally in LRU cache limited by `CACHE_MAX_ITEMS` and `CACHE_MAX_SIZE` with [go-pkgz/rest](https://github.com/go-pkgz/rest)
* User's activity throttled globally (up to 1000 simultaneous requests) and limited locally (per user, usually up to 10 req/sec)
* Request timeout set to 60sec
* Admin authentication (`--admin-password` set) allows to hit remark42 API without social login and with admin privileges. Adds basic-auth for username: `admin`, password: `${ADMIN_PASSWD}`.
* User can vote for the comment multiple times but only to change the vote. Double-voting not allowed.
* User can edit comments in 5 mins (configurable) window after creation.
* User ID hashed and prefixed by oauth provider name to avoid collisions and potential abuse.
* All avatars resized and cached locally to prevent rate limiters from oauth providers, part of [go-pkgz/auth](https://github.com/go-pkgz/auth) functionality.
* Images can be proxied (`IMAGE_PROXY_HTTP2HTTPS=true`) to prevent mixed http/https.
* All images can be proxied and saved (`IMAGE_PROXY_CACHE_EXTERNAL=true`) instead of serving from original location. Beware, images which are posted with this parameter enabled will be served from proxy even after it will be disabled.
* Docker build uses [publicly available](https://github.com/umputun/baseimage) base images.
