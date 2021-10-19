---
title: Configure with Reproxy
---

## How to configure remark42 with [Reproxy](https://reproxy.io)

Example of Reproxy configuration (reverse proxy) running remark42 service on remark42.example.com with docker compose. Reproxy handles SSL termination with LE and gzip all the responses.

```yaml
version: "3.4"

services:
  reproxy:
    image: umputun/reproxy:master
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
    image: umputun/remark42:master
    container_name: "remark42"
    hostname: "remark42"
    restart: always
    logging: *default_logging
    environment:
      - MHOST
      - SECRET=some-secret-thing
      - USER=app
      - REMARK_URL=https://remark42.example.com
      - STORE_BOLT_PATH=/srv/var
      - BACKUP_PATH=/srv/var/backup
      - CACHE_MAX_VALUE=10000000
      - IMAGE_PROXY_HTTP2HTTPS=true
      - AVATAR_RESIZE=48
      - ADMIN_SHARED_ID=github_ef0f706a79cc24b112345
      - ADMIN_SHARED_NAME=myname,anothername
      - ADMIN_SHARED_EMAIL=admin@example.com
      - AUTH_TWITTER_CID=12345678
      - AUTH_TWITTER_CSEC=asdfghjkl
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
    ports:
      - "8080"
    volumes:
      - ./var/remark42:/srv/var
    labels:
      reproxy.server: remark42.example.com
      reproxy.port: "8080"
      reproxy.route: "^/(.*)"
      reproxy.dest: "/$$1"
      reproxy.ping: "/ping"
```
