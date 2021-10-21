---
title: Technical Details
---

Data stored in [boltdb](https://github.com/etcd-io/bbolt) (embedded key/value database) files under `STORE_BOLT_PATH`. Each site is stored in a separate boltdb file.

To migrate/move Remark42 to another host boltdb files as well as avatars directory `AVATAR_FS_PATH` should be transferred. Optionally, boltdb can be used to store avatars as well.

Automatic backup process runs every 24h and exports all content in JSON-like format to `backup-remark-YYYYMMDD.gz`.

Authentication implemented with [go-pkgz/auth](https://github.com/go-pkgz/auth) stored in a cookie. It uses HttpOnly, secure cookies.

All heavy REST calls cached internally in LRU cache limited by `CACHE_MAX_ITEMS` and `CACHE_MAX_SIZE` with [go-pkgz/rest](https://github.com/go-pkgz/rest).

User's activity throttled globally (up to 1000 simultaneous requests) and limited locally (per user, usually up to 10 req/sec).

Request timeout set to 60sec.

Admin authentication (`--admin-password` set) allows to hit Remark42 API without social login and with admin privileges. Adds basic-auth for username: `admin`, password: `${ADMIN_PASSWD}`.

User can vote for the comment multiple times but only to change the vote. Double voting is not allowed.

User can edit comments in 5 mins (configurable) window after creation.

User ID hashed and prefixed by OAuth provider name to avoid collisions and potential abuse.

All avatars resized and cached locally to prevent rate limiters from OAuth providers, part of [go-pkgz/auth](https://github.com/go-pkgz/auth) functionality.

Images served over HTTP can be proxied to HTTPS (`IMAGE_PROXY_HTTP2HTTPS=true`) to prevent mixed HTTP/HTTPS.

All images can be proxied and saved locally (`IMAGE_PROXY_CACHE_EXTERNAL=true`) instead of serving from the original location. Beware, images that are posted with this parameter enabled will be served from proxy even after it will be disabled.

Docker build uses [publicly available](https://github.com/umputun/baseimage) base images.
