---
title: Configure with Reproxy
---

## How to configure remark42 with [Reproxy](https://reproxy.io)

Example of Reproxy configuration (reverse proxy) running remark42 service on remark42.example.com with Docker Compose. Reproxy handles SSL termination with LE and gzip all the responses.

```yaml
services:
  reproxy:
    image: umputun/reproxy:latest
    restart: always
    hostname: reproxy
    container_name: reproxy
    logging: &default_logging
      driver: json-file
      options:
        max-size: "10m"
        max-file: "5"
    ports:
      - "80:8080"
      - "443:8443"
    environment:
      - TZ=America/Chicago
      - DOCKER_ENABLED=true
      - SSL_TYPE=auto
      - SSL_ACME_EMAIL=admin@example.com
      - SSL_ACME_FQDN=remark42.example.com
      - SSL_ACME_LOCATION=/srv/var/ssl
      - GZIP=true
      - MAX_SIZE=10M
      - LOGGER_ENABLED=true
      - LOGGER_FILE=/srv/var/logs/access.log
      - LOGGER_STDOUT=true
      - ASSETS_CACHE=30d,text/html:30s
      - HEADER=X-XSS-Protection:1;mode=block;,X-Content-Type-Options:nosniff

    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./var/ssl:/srv/var/ssl
      - ./var/logs:/srv/var/logs

  remark42:
    image: ghcr.io/umputun/remark42:latest
    container_name: "remark42"
    hostname: "remark42"
    restart: always
    logging: *default_logging
    environment:
      - SECRET=some-secret-thing
      - REMARK_URL=https://remark42.example.com
      - SITE=remark
      - CACHE_MAX_VALUE=10000000
      - IMAGE_PROXY_HTTP2HTTPS=true
      - AVATAR_RESIZE=48
      - ADMIN_SHARED_ID=github_ef0f706a79cc24b112345
      - ADMIN_SHARED_EMAIL=admin@example.com
      - AUTH_ANON=true
      - AUTH_EMAIL_ENABLE=true
      - AUTH_EMAIL_FROM=confirmation@example.com
      - SMTP_HOST=smtp.mailgun.org
      - SMTP_PORT=465
      - SMTP_TLS=true
      - SMTP_USERNAME=postmaster@mg.example.com
      - SMTP_PASSWORD=thepassword
      - IMAGE_MAX_SIZE=5000000
      - EMOJI=true
    volumes:
      - ./var/remark42:/srv/var
    labels:
      reproxy.server: remark42.example.com
      reproxy.port: "8080"
      reproxy.route: "^/(.*)"
      reproxy.dest: "/$$1"
      reproxy.ping: "/ping"
```

Reproxy accepts request bodies up to 64K by default and refuses larger ones, which stops picture uploads and imports through it. `MAX_SIZE=10M` raises the limit; keep it above `IMAGE_MAX_SIZE` and above the uncompressed size of the largest backup or export you restore or import through Reproxy, since the `restore` and `import` commands send a `.gz` file unpacked.

If Docker rejects Reproxy's default API version (1.24), set `DOCKER_API_VERSION` to a version your engine supports (`docker version` shows the range), for example `1.44`.

To make per-IP rate limiting and vote de-duplication use the real client IP, set [`--trusted-proxy`](../../configuration/parameters/#trusted-proxies-and-client-ip) to the network Reproxy connects from (the Docker network for a same-compose Reproxy, e.g. `172.16.0.0/12`). By default Reproxy sets `X-Real-IP` to the address of the connection and overwrites any value the client sent, so a client connecting from a public address cannot choose its IP; one connecting from a private address still can, as [Trusted proxies and client IP](../../configuration/parameters/#trusted-proxies-and-client-ip) describes. If another proxy is in front of Reproxy, set Reproxy's `TRUSTED_PROXIES` to that proxy's address and have the proxy append the client address to `X-Forwarded-For` (for nginx, `proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for`), as Reproxy's README describes under [client address and forwarded headers](https://github.com/umputun/reproxy#client-address-and-forwarded-headers); without that, every visitor arrives from the proxy's address and shares one rate limit. Keep the `remark42` service's port unpublished, as in the example: Reproxy is the entry point, and a published port makes Remark42 reachable directly, so that external traffic appears as a trusted Docker peer.
