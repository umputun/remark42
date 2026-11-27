---
title: Command-Line Interface parameters
---

## Required parameters

Only two parameters are required to start the server:

1. `SECRET` - secret key, can be any long and hard-to-guess string
2. `REMARK_URL` - URL pointing to your Remark42 server, i.e., `https://demo.remark42.com`

To let users sign in and comment, enable at least one [authentication method](https://remark42.com/docs/configuration/authorization/), such as OAuth2, email (`AUTH_EMAIL_ENABLE=true`) or anonymous login (`AUTH_ANON=true`). OAuth2 is not required when using email or anonymous login.

A small `docker-compose.yml` with the required parameters and a few optional ones:

```yaml
services:
  remark42:
    image: ghcr.io/umputun/remark42:latest
    restart: always
    container_name: "remark42"
    environment:
      - REMARK_URL=https://demo.remark42.com # URL pointing to your Remark42 server
      - SITE=YOUR_SITE_ID # site ID, same as used for `site_id`, see "Setup on your website"
      - SECRET=abcd-123456-xyz-$%^& # secret key
      - AUTH_ANON=true # enable anonymous commenting
      - AUTH_GITHUB_CID=12345667890 # OAuth2 client ID
      - AUTH_GITHUB_CSEC=abcdefg12345678 # OAuth2 client secret
    volumes:
      - ./var:/srv/var # persistent volume to store all Remark42 data
```

## Complete parameters list

| Command line                   | Environment                    | Default                 | Description                                              |
|--------------------------------|--------------------------------|-------------------------|----------------------------------------------------------|
| url                            | REMARK_URL                     |                         | URL to Remark42 server, _required_                       |
| secret                         | SECRET                         |                         | the shared secret key used to sign JWT, should be a random, long, hard-to-guess string, _required_. With the default `admin.type=shared` it also keys the stored IP hashes and the IDs of anonymous users, so changing it signs everyone out, gives each anonymous user a new ID at the next sign-in and invalidates the unsubscribe and data deletion links already sent |
| site                           | SITE                           | `remark`                | site name(s), _multi_                                    |
| store.type                     | STORE_TYPE                     | `bolt`                  | type of storage, `bolt` or `rpc`                         |
| store.bolt.path                | STORE_BOLT_PATH                | `./var`                 | parent directory for the bolt files                      |
| store.bolt.timeout             | STORE_BOLT_TIMEOUT             | `30s`                   | boltdb access timeout                                    |
| store.rpc.api                  | STORE_RPC_API                  |                         | rpc extension api url                                    |
| store.rpc.timeout              | STORE_RPC_TIMEOUT              | `5s`                    | http timeout |
| store.rpc.auth_user            | STORE_RPC_AUTH_USER            |                         | basic auth user name                                     |
| store.rpc.auth_passwd          | STORE_RPC_AUTH_PASSWD          |                         | basic auth user password                                 |
| admin.type                     | ADMIN_TYPE                     | `shared`                | type of admin store, `shared` or `rpc`                   |
| admin.rpc.api                  | ADMIN_RPC_API                  |                         | rpc extension api url                                    |
| admin.rpc.timeout              | ADMIN_RPC_TIMEOUT              | `5s`                    | http timeout |
| admin.rpc.auth_user            | ADMIN_RPC_AUTH_USER            |                         | basic auth user name                                     |
| admin.rpc.auth_passwd          | ADMIN_RPC_AUTH_PASSWD          |                         | basic auth user password                                 |
| admin.rpc.secret_per_site      | ADMIN_RPC_SECRET_PER_SITE      |                         | enable JWT secret retrieval per aud, which is site_id in this case |
| admin.shared.id                | ADMIN_SHARED_ID                |                         | admin IDs (list of user IDs), _multi_                    |
| admin.shared.email             | ADMIN_SHARED_EMAIL             | `admin@` + host of `REMARK_URL`, port included | admin emails, _multi_. The first one is public (`admin_email` in `/api/v1/config`) and receives data deletion requests. With `NOTIFY_ADMINS=email`, all configured addresses receive admin notifications; the default is not used for those |
| backup                         | BACKUP_PATH                    | `./var/backup`          | backups location                                         |
| max-back                       | MAX_BACKUP_FILES               | `10`                    | max backup files to keep                                 |
| cache.type                     | CACHE_TYPE                     | `mem`                   | type of cache, `redis_pub_sub` or `mem` or `none`        |
| cache.redis_addr               | CACHE_REDIS_ADDR               | `127.0.0.1:6379`        | address of Redis PubSub instance, turn `redis_pub_sub` cache on for distributed cache |
| cache.max.items                | CACHE_MAX_ITEMS                | `1000`                  | max number of cached items, `0` - unlimited              |
| cache.max.value                | CACHE_MAX_VALUE                | `0`                     | max size of the cached value, `0` - unlimited            |
| cache.max.size                 | CACHE_MAX_SIZE                 | `50000000`              | max size of all cached values, `0` - unlimited           |
| avatar.type                    | AVATAR_TYPE                    | `fs`                    | type of avatar storage, `fs`, `bolt`, or `uri`           |
| avatar.fs.path                 | AVATAR_FS_PATH                 | `./var/avatars`         | avatars location for `fs` store                          |
| avatar.bolt.file               | AVATAR_BOLT_FILE               | `./var/avatars.db`      | avatars `bolt` file location                             |
| avatar.uri                     | AVATAR_URI                     | `./var/avatars`         | avatars store URI                                        |
| avatar.rsz-lmt                 | AVATAR_RESIZE                  | `0` (disabled)          | max avatar width and height in pixels; larger avatars are scaled down on save, keeping the aspect ratio |
| avatar.identicon-size          | AVATAR_IDENTICON_SIZE          | `48`                    | width and height in pixels of the avatar generated for users without a picture; 48 keeps the blocks of the 24px comment avatar sharp at device pixel ratios 1 and 2, and a smaller `avatar.rsz-lmt` scales it down. A value below 5 or above 4096 gives 300 |
| image.type                     | IMAGE_TYPE                     | `fs`                    | type of image storage, `fs`, `bolt` or `rpc`             |
| image.fs.path                  | IMAGE_FS_PATH                  | `./var/pictures`        | permanent location of images                             |
| image.fs.staging               | IMAGE_FS_STAGING               | `./var/pictures.staging` | staging location of images                               |
| image.fs.partitions            | IMAGE_FS_PARTITIONS            | `100`                   | number of image partitions                               |
| image.bolt.file                | IMAGE_BOLT_FILE                | `./var/pictures.db`     | images bolt file location                                |
| image.rpc.api                  | IMAGE_RPC_API                  |                         | rpc extension api url                                    |
| image.rpc.timeout              | IMAGE_RPC_TIMEOUT              | `5s`                    | http timeout |
| image.rpc.auth_user            | IMAGE_RPC_AUTH_USER            |                         | basic auth user name                                     |
| image.rpc.auth_passwd          | IMAGE_RPC_AUTH_PASSWD          |                         | basic auth user password                                 |
| image.max-size                 | IMAGE_MAX_SIZE                 | `5000000`               | max size of image file                                   |
| image.resize-width             | IMAGE_RESIZE_WIDTH             | `2400`                  | width of a resized image                                 |
| image.resize-height            | IMAGE_RESIZE_HEIGHT            | `900`                   | height of a resized image                                |
| auth.ttl.jwt                   | AUTH_TTL_JWT                   | `5m`                    | JWT TTL                                                  |
| auth.ttl.cookie                | AUTH_TTL_COOKIE                | `200h`                  | TTL of the server-set auth cookie |
| auth.send-jwt-header           | AUTH_SEND_JWT_HEADER           | `false`                 | also send JWT as a header and, for a widget embedded on another host, set the auth cookies a second time as `Partitioned`, so its session survives third-party cookie blocking; the server-set cookies are still sent. [See security considerations](#security-considerations-for-authsend-jwt-header). |
| auth.same-site                 | AUTH_SAME_SITE                 | `default`               | SameSite attribute for the server-set auth cookies (`default`, `none`, `lax` or `strict`). `default` emits no attribute at all and leaves the choice to the browser, which is not the same as `lax` |
| auth.apple.cid                 | AUTH_APPLE_CID                 |                         | Apple client ID (App ID or Services ID)                  |
| auth.apple.tid                 | AUTH_APPLE_TID                 |                         | Apple Team ID                                            |
| auth.apple.kid                 | AUTH_APPLE_KID                 |                         | Apple Private key ID                                     |
| auth.apple.private-key-filepath | AUTH_APPLE_PRIVATE_KEY_FILEPATH | `/srv/var/apple.p8`       | Apple Private key file location                          |
| auth.google.cid                | AUTH_GOOGLE_CID                |                         | Google OAuth client ID                                   |
| auth.google.csec               | AUTH_GOOGLE_CSEC               |                         | Google OAuth client secret                               |
| auth.facebook.cid              | AUTH_FACEBOOK_CID              |                         | Facebook OAuth client ID                                 |
| auth.facebook.csec             | AUTH_FACEBOOK_CSEC             |                         | Facebook OAuth client secret                             |
| auth.microsoft.cid             | AUTH_MICROSOFT_CID             |                         | Microsoft OAuth client ID                                |
| auth.microsoft.csec            | AUTH_MICROSOFT_CSEC            |                         | Microsoft OAuth client secret                            |
| auth.microsoft.tenant          | AUTH_MICROSOFT_TENANT          | `common`                | Azure AD tenant ID, domain, or "common"                  |
| auth.github.cid                | AUTH_GITHUB_CID                |                         | GitHub OAuth client ID                                   |
| auth.github.csec               | AUTH_GITHUB_CSEC               |                         | GitHub OAuth client secret                               |
| auth.twitter.cid               | AUTH_TWITTER_CID               |                         | X (Twitter) Consumer Key                                 |
| auth.twitter.csec              | AUTH_TWITTER_CSEC              |                         | X (Twitter) Consumer Key Secret                          |
| auth.patreon.cid               | AUTH_PATREON_CID               |                         | Patreon OAuth Client ID                                  |
| auth.patreon.csec              | AUTH_PATREON_CSEC              |                         | Patreon OAuth Client Secret                              |
| auth.discord.cid               | AUTH_DISCORD_CID               |                         | Discord OAuth Client ID                                  |
| auth.discord.csec              | AUTH_DISCORD_CSEC              |                         | Discord OAuth Client Secret                              |
| auth.custom.name               | AUTH_CUSTOM_NAME               |                         | custom OAuth provider name (used in `/auth/<name>/...`) |
| auth.custom.cid                | AUTH_CUSTOM_CID                |                         | custom OAuth client ID                                   |
| auth.custom.csec               | AUTH_CUSTOM_CSEC               |                         | custom OAuth client secret                               |
| auth.custom.auth-url           | AUTH_CUSTOM_AUTH_URL           |                         | custom OAuth authorization endpoint                      |
| auth.custom.token-url          | AUTH_CUSTOM_TOKEN_URL          |                         | custom OAuth token endpoint                              |
| auth.custom.info-url           | AUTH_CUSTOM_INFO_URL           |                         | custom OAuth user info endpoint                          |
| auth.custom.scopes             | AUTH_CUSTOM_SCOPES             | none                    | custom OAuth scopes, comma-separated                     |
| auth.custom.id-field           | AUTH_CUSTOM_ID_FIELD           | `sub`                   | user info field used as unique id; without a value in it the email identifies the user, and without both the sign-in is refused |
| auth.custom.name-field         | AUTH_CUSTOM_NAME_FIELD         | `name`                  | user info field used as display name                     |
| auth.custom.picture-field      | AUTH_CUSTOM_PICTURE_FIELD      | `picture`               | user info field used as avatar URL                       |
| auth.custom.email-field        | AUTH_CUSTOM_EMAIL_FIELD        | `email`                 | user info field holding the email, used only as a fallback id |
| auth.telegram                  | AUTH_TELEGRAM                  | `false`                 | Enable Telegram auth (telegram.token must be present)    |
| auth.yandex.cid                | AUTH_YANDEX_CID                |                         | Yandex OAuth client ID                                   |
| auth.yandex.csec               | AUTH_YANDEX_CSEC               |                         | Yandex OAuth client secret                               |
| auth.dev                       | AUTH_DEV                       | `false`                 | local OAuth2 server, development mode only               |
| auth.anon                      | AUTH_ANON                      | `false`                 | enable anonymous login                                   |
| auth.email.enable              | AUTH_EMAIL_ENABLE              | `false`                 | enable auth via email                                    |
| auth.email.from                | AUTH_EMAIL_FROM                |                         | email from (e.g. `john.doe@example.com` or `"John Doe"<john.doe@example.com>`) |
| auth.email.subj                | AUTH_EMAIL_SUBJ                | `remark42 confirmation` | email subject                                            |
| auth.email.content-type        | AUTH_EMAIL_CONTENT_TYPE        | `text/html`             | email content type                                       |
| notify.users                   | NOTIFY_USERS                   | none                    | type of user notifications (`telegram`, `email`), _multi_ |
| notify.admins                  | NOTIFY_ADMINS                  | none                    | type of admin notifications (`telegram`, `slack`, `webhook` and/or `email`), _multi_ |
| notify.queue                   | NOTIFY_QUEUE                   | `100`                   | size of notification queue                               |
| notify.telegram.chan           | NOTIFY_TELEGRAM_CHAN           |                         | the ID of telegram channel for admin notifications       |
| notify.slack.token             | NOTIFY_SLACK_TOKEN             |                         | Slack token                                              |
| notify.slack.chan              | NOTIFY_SLACK_CHAN              | `general`               | Slack channel for admin notifications                    |
| notify.webhook.url             | NOTIFY_WEBHOOK_URL             |                         | Webhook notification URL for admin notifications         |
| notify.webhook.template        | NOTIFY_WEBHOOK_TEMPLATE        | `{"text": {{.Text \| escapeJSONString}}}` | Webhook payload template (Go text/template) |
| notify.webhook.headers         | NOTIFY_WEBHOOK_HEADERS         |                         | HTTP headers in format `Header1:Value1,Header2:Value2`; the flag takes one `Header:Value` per use, _multi_ |
| notify.webhook.timeout         | NOTIFY_WEBHOOK_TIMEOUT         | `5s`                    | Webhook connection timeout                               |
| notify.email.from_address      | NOTIFY_EMAIL_FROM              |                         | from email address (e.g. `john.doe@example.com` or `"John Doe"<john.doe@example.com>`) |
| notify.email.verification_subj | NOTIFY_EMAIL_VERIFICATION_SUBJ | `Email verification`    | verification message subject                             |
| telegram.token                 | TELEGRAM_TOKEN                 |                         | Telegram token (used for auth and Telegram notifications) |
| telegram.timeout               | TELEGRAM_TIMEOUT               | `5s`                    | Telegram connection timeout                              |
| telegram.api-url               | TELEGRAM_API_URL               |                         | Telegram Bot API origin/base path for auth and notifications, without `/bot`; the bot token is appended to request paths, and the public API is used when unset |
| smtp.host                      | SMTP_HOST                      |                         | SMTP host                                                |
| smtp.port                      | SMTP_PORT                      | `25`                    | SMTP port                                                |
| smtp.helo_host                 | SMTP_HELO_HOST                 |                         | SMTP HELO/EHLO hostname                                  |
| smtp.username                  | SMTP_USERNAME                  |                         | SMTP user name                                           |
| smtp.password                  | SMTP_PASSWORD                  |                         | SMTP password                                            |
| smtp.login_auth                | SMTP_LOGIN_AUTH                | `false`                 | enable LOGIN auth instead of PLAIN                       |
| smtp.tls                       | SMTP_TLS                       | `false`                 | enable TLS for SMTP                                      |
| smtp.starttls                  | SMTP_STARTTLS                  | `false`                 | enable StartTLS for SMTP                                 |
| smtp.insecure_skip_verify      | SMTP_INSECURE_SKIP_VERIFY      | `false`                 | skip certificate verification for SMTP                   |
| smtp.timeout                   | SMTP_TIMEOUT                   | `10s`                   | SMTP TCP connection timeout                              |
| ssl.type                       | SSL_TYPE                       | none                    | `none`-HTTP, `static`-HTTPS, `auto`-HTTPS + le           |
| ssl.port                       | SSL_PORT                       | `8443`                  | port for HTTPS server                                    |
| ssl.cert                       | SSL_CERT                       |                         | path to the cert.pem file                                |
| ssl.key                        | SSL_KEY                        |                         | path to the key.pem file                                 |
| ssl.acme-location              | SSL_ACME_LOCATION              | `./var/acme`            | dir where obtained le-certs will be stored               |
| ssl.acme-email                 | SSL_ACME_EMAIL                 |                         | admin email for receiving notifications from LE; when unset, the first `admin.shared.email` with `admin.type=shared`, otherwise `admin@` plus the host name of `REMARK_URL` |
| max-comment                    | MAX_COMMENT_SIZE               | `2048`                  | comment's size limit                                     |
| min-comment                    | MIN_COMMENT_SIZE               | `0`                     | comment's minimal size limit, `0` - unlimited            |
| max-votes                      | MAX_VOTES                      | `-1`                    | votes limit per comment, `-1` - unlimited, `0` - voting disabled |
| votes-ip                       | VOTES_IP                       | `false`                 | restrict votes from the same IP; an IPv6 address counts as its /64 network |
| anon-vote                      | ANON_VOTE                      | `false`                 | allow voting for anonymous users, require VOTES_IP to be enabled as well |
| votes-ip-time                  | VOTES_IP_TIME                  | `5m`                    | same IP vote restriction time, `0s` - unlimited. With `votes-ip` on, a new vote on a comment is refused while that IP has a vote record inside this time, whatever the direction; a user can still take their own vote back. New IP hash records include the voter's user ID. Records older than this are removed on the comment's next accepted vote, also with `votes-ip` off |
| low-score                      | LOW_SCORE                      | `-5`                    | low score threshold                                      |
| critical-score                 | CRITICAL_SCORE                 | `-10`                   | critical score threshold                                 |
| positive-score                 | POSITIVE_SCORE                 | `false`                 | restricts comment's score to be only positive            |
| restricted-words               | RESTRICTED_WORDS               |                         | words banned in comments (can use `*`), matched against whole words, case-insensitively, up to 64 characters each, _multi_ |
| restricted-words-file          | RESTRICTED_WORDS_FILE          |                         | file with words banned in comments, one per line (can use `*`, lines starting with `#` are skipped), added to `restricted-words` and re-read when changed; a file that cannot be read at start stops the server |
| restricted-names               | RESTRICTED_NAMES               |                         | names that anonymous users and non-admin email users cannot take, compared with the whole name case-insensitively; a user with such a name is blocked. Names from other sign-in methods are not checked, _multi_ |
| edit-time                      | EDIT_TIME                      | `5m`                    | edit window; `0` hides editing in the widget and disables staged image cleanup, but the API does not limit edit time |
| admin-edit                     | ADMIN_EDIT                     | `false`                 | unlimited edit for admins                                |
| read-age                       | READONLY_AGE                   | `0`                     | read-only age of comments, days                          |
| image-proxy.http2https         | IMAGE_PROXY_HTTP2HTTPS         | `false`                 | enable HTTP->HTTPS proxy for images; has no effect when `REMARK_URL` is http |
| image-proxy.cache-external     | IMAGE_PROXY_CACHE_EXTERNAL     | `false`                 | proxy external images, store the ones posted comments show |
| emoji                          | EMOJI                          | `false`                 | enable emoji support                                     |
| simple-view                    | SIMPLE_VIEW                    | `false`                 | simplified comment form, without the formatting toolbar and preview |
| proxy-cors                     | PROXY_CORS                     | `false`                 | disable internal CORS and delegate it to proxy; the proxy has to cover `/web/` as well as `/api/`, since a host page loads the widget scripts and message catalogues from there |
| allowed-hosts                  | ALLOWED_HOSTS                  | enable all              | hosts allowed to embed comments, via CSP `frame-ancestors`, any host when empty, _multi_. Plain host entries, with the host of `REMARK_URL`, also limit the address a sign-in returns to; CSP keywords and entries with `*` do not count there. Page titles are fetched only from addresses on the listed domains and their subdomains and on the domain formed by the last two labels of the `REMARK_URL` host name and its subdomains; every redirect target is checked against the same domains |
| address                        | REMARK_ADDRESS                 | all interfaces          | web server listening address                             |
| port                           | REMARK_PORT                    | `8080`                  | web server port                                          |
| web-root                       | REMARK_WEB_ROOT                | `./web`                 | web server root, supplies the frontend when it exists    |
| update-limit                   | UPDATE_LIMIT                   | `0.5`                   | updates/sec limit                                        |
| trusted-proxy                  | TRUSTED_PROXY                  | none (trust any)        | reverse-proxy networks (CIDR/IP, comma-separated) trusted to set the client IP; see [Trusted proxies and client IP](#trusted-proxies-and-client-ip) |
| subscribers-only               | SUBSCRIBERS_ONLY               | `false`                 | enable commenting only for Patreon subscribers           |
| disable-signature              | DISABLE_SIGNATURE              | `false`                 | disable server signature in headers                      |
| disable-fancy-text-formatting  | DISABLE_FANCY_TEXT_FORMATTING  | `false`                 | disable fancy comments text formatting (replacement of quotes, dashes, fractions, etc) |
| admin-passwd                   | ADMIN_PASSWD                   | none (disabled)         | password for `admin` basic auth                          |
| dbg                            | DEBUG                          | `false`                 | debug mode                                               |

- command-line parameters are long-form `--<key>=value`, i.e., `--url=https://demo.remark42.com`
- _multi_ parameters separated by `,` in the environment or repeated with command-line keys, like `--site=s1 --site=s2 ...`
- _required_ parameters have to be presented in the environment or provided in the command-line

## Custom OAuth2 integration

Custom OAuth2 integration currently supports only one custom provider at a time, and `AUTH_CUSTOM_NAME` must match `^[a-z0-9][a-z0-9_-]*$`.

## Security Considerations for auth.send-jwt-header

When `auth.send-jwt-header=true` is enabled:

- **Security Impact**: the JWT is sent to the widget in a response header, where JavaScript can read it, and the widget keeps it in memory for as long as the page stays open
- **Vulnerability**: This increases vulnerability to XSS attacks compared to a session kept only in server-set HttpOnly cookies
- **Implementation Mitigations**:
  - The widget writes the token to no cookie and to no storage. The cookies that carry the session are the server's own, and the `JWT` cookie is `HttpOnly`
  - For a widget served over https and embedded on a page of another origin, a subdomain of the same site included, the server sets those cookies a second time as `SameSite=None; Secure; Partitioned`. `Partitioned` keys the cookie to the embedding top-level site, so a different site gets a separate cookie and cannot reach this one. Pages and subdomains under that same site do share it, since the partition key is the site rather than the page
  - Double Submit Cookie pattern with XSRF token matching the JWT ID

This configuration should only be used when:
1. You need cross-domain authentication support
2. You understand and accept the increased XSS risk
3. You have implemented strong XSS protections on your site

## Trusted proxies and client IP

Remark42 uses the client IP for five things: per-IP rate limiting, vote de-duplication when `--votes-ip` is enabled, the IP hash stored with every comment and every vote, the ID of anonymous users, and the IP hash in the request log. When Remark42 runs behind a reverse proxy (nginx, Reproxy, Traefik, Cloudflare, an ALB, a k8s ingress, …) the TCP connection it sees comes from the **proxy**, not the visitor, so the proxy forwards the real client IP in a header and Remark42 reads it to recover the real IP.

> ⚠️ **Security note.** Those headers can be set by any client. If Remark42 trusts them from everyone, a caller can send `X-Real-IP: <anything>` and rotate its apparent IP to **bypass rate limiting and vote de-duplication**. Use `--trusted-proxy` so Remark42 reads the forwarding headers **only** when the request actually arrives from your proxy.

`--trusted-proxy` / `TRUSTED_PROXY` takes a comma-separated list of networks (CIDR) or bare IPs. When set, the forwarding headers are honored only if the **direct peer** — the machine that opened the TCP connection to Remark42 — falls inside one of them; for a request from any other peer those headers are dropped and the real socket IP is used. When it is **not** set, the headers are trusted from any client (priority: `X-Real-IP`, then `CF-Connecting-IP`, then the leftmost public `X-Forwarded-For` entry): this preserves the historical behavior so existing deployments keep working, but leaves the bypass above open, and Remark42 prints a warning at startup. **If Remark42 is reachable from the internet, set this.**

From a trusted proxy, Remark42 takes the client IP from the first of these that applies:

1. `X-Real-IP`, when it holds an IP address. It is used as sent, whether the address is public, private or inside `--trusted-proxy`.
2. `X-Forwarded-For`, read from right to left. An entry inside `--trusted-proxy` is another proxy and is skipped; the first entry outside it is the client. If every entry is inside `--trusted-proxy`, the leftmost one is used. An empty or malformed entry met on the way ends the reading, and the address of the proxy that connected to Remark42 is used.
3. The address of the proxy that connected to Remark42.

`CF-Connecting-IP` is not read from a trusted proxy.

Two conditions must **both** hold to be safe:

**1. Trust the right peer.** Point `--trusted-proxy` at the network your proxy connects _from_, and nothing wider.

**2. Your proxy must set the IP itself.** Trusting a proxy is not enough if the proxy relays what the client sent. Because Remark42 reads `X-Real-IP` first, the proxy must either **set** `X-Real-IP` to the real connecting client and overwrite any client value, or **remove** an `X-Real-IP` the client sent and append the real connecting client to `X-Forwarded-For`. A proxy that passes on the client's `X-Real-IP`, or that derives `X-Real-IP` from a client-controlled `X-Forwarded-For`, leaves the visitor in control of the reported IP **even through a trusted proxy**. A proxy that only _appends_ to `X-Forwarded-For` (Traefik, cloud ALBs, k8s ingress, HAProxy) is safe only when it does not pass on the client's `X-Real-IP`, so check that for yours:

- **nginx** — safe with the [manual's](../../manuals/nginx/) `proxy_set_header X-Real-IP $remote_addr;` (overwrites any client value).
- **Reproxy** — by default sets `X-Real-IP` to the address of the connection and overwrites any client value. Behind another proxy, set Reproxy's `TRUSTED_PROXIES` to that proxy's address and have the proxy append the client address to `X-Forwarded-For`, or every visitor arrives from the proxy's address.
- **Cloudflare** — connecting to Remark42 directly, it removes `X-Real-IP` and appends the visitor's address to `X-Forwarded-For`, and that last entry is what Remark42 takes; `CF-Connecting-IP` is not read. Put [Cloudflare's IP ranges](https://www.cloudflare.com/ips/) in `--trusted-proxy`. With your own proxy between Cloudflare and Remark42, see "More than one proxy" below.

**Recommended `--trusted-proxy` per deployment:**

| Deployment | Direct peer Remark42 sees | `--trusted-proxy` |
| --- | --- | --- |
| nginx in the **same Docker Compose** (the [nginx manual](../../manuals/nginx/)) | the proxy container's Docker IP | the Docker network, e.g. `172.16.0.0/12` (covers Docker's default bridge pool) — or pin your compose network's subnet |
| reverse proxy on the **host** / another machine | the proxy's host IP | that proxy's IP or CIDR, e.g. `10.0.0.5` |
| **Directly exposed**, no proxy | the real client IP already | `--trusted-proxy=127.0.0.1/32` — a value no client matches, so forwarding headers are always ignored and the real connecting IP is used |

Examples:

```
# docker-compose: nginx on the same network
TRUSTED_PROXY=172.16.0.0/12

# a single upstream proxy
--trusted-proxy=10.0.0.5

# multiple ranges (v4 and v6)
TRUSTED_PROXY=10.0.0.0/8,fd00::/8
```

Caveats:

- **IPv6.** CIDRs are matched by family — `172.16.0.0/12` (or `0.0.0.0/0`) never matches an IPv6 peer. If your proxy reaches Remark42 over IPv6 (dual-stack / IPv6-enabled Docker), add the IPv6 network too, or every visitor collapses onto the proxy's IP and gets over-throttled.
- **Don't publish Remark42's own port** when trusting a Docker range. If Remark42's port is exposed to the host, external traffic is SNAT'd to the Docker gateway (inside `172.16.0.0/12`) and appears trusted — re-opening the bypass. Publish only the proxy.
- `0.0.0.0/0` trusts everyone and re-opens the bypass; too narrow a range over-throttles real visitors. The request log holds only a hash of the client address, so look up the proxy's network on the proxy side, for example with `docker network inspect <network>` for Docker Compose.
- **More than one proxy.** Unless the last proxy is set up to restore the visitor's address, its `X-Real-IP` holds the address it accepted the connection from. Behind a CDN, a load balancer or a tunnel such as `cloudflared` that is the front hop's address, often a single private one, and every visitor arriving through the same front hop shares one rate limit, the per-IP vote restriction of `--votes-ip` and, under the same name, one anonymous user ID. Either make the last proxy restore the visitor's address (nginx: `set_real_ip_from` and `real_ip_header`), or have it send no `X-Real-IP` and append to `X-Forwarded-For`, and add the addresses of the proxies in front of it to `--trusted-proxy`.
- **Trusted networks and `X-Forwarded-For`.** When the client IP comes from `X-Forwarded-For`, every entry inside `--trusted-proxy` is taken for a proxy that reported its own client. If a visitor's connection reaches your proxy from such an address, the entry before it is used, and that entry the visitor can write. The Docker gateway is such an address when Docker itself relays the connection, as it does for IPv6 visitors while the Docker network has no IPv6. A proxy that sets `X-Real-IP` is not affected; for one that does not, put the proxy's own address in `--trusted-proxy` instead of the whole Docker network.
- **Private client addresses.** From a trusted proxy, a private, shared (`100.64.0.0/10`), link-local or loopback address is used like any other, so visitors on an intranet or behind a VPN are told apart. Without `--trusted-proxy` such an address is skipped and the next header is tried.

## Deprecated parameters

The following list of command-line options is deprecated and might be removed in the next major release after the version they were deprecated. After the Remark42 version update, please check the startup log once for deprecation warning messages to avoid trouble with unrecognized command-line options in the future.

<details>
<summary>Deprecated options</summary>

| Command line       | Replacement  | Environment         | Replacement   | Default | Description    | Deprecation version |
| ------------------ | ------------ | ------------------- | ------------- | ------- | -------------- | ------------------- |
| auth.email.template| none         | AUTH_EMAIL_TEMPLATE | none          | `email_confirmation_login.html.tmpl` | custom email message template file | 1.5.0 |
| auth.email.host    | smtp.host    | AUTH_EMAIL_HOST     | SMTP_HOST     |         | smtp host      | 1.5.0               |
| auth.email.port    | smtp.port    | AUTH_EMAIL_PORT     | SMTP_PORT     |         | smtp port      | 1.5.0               |
| auth.email.user    | smtp.username | AUTH_EMAIL_USER    | SMTP_USERNAME |         | smtp user name | 1.5.0               |
| auth.email.passwd  | smtp.password | AUTH_EMAIL_PASSWD  | SMTP_PASSWORD |         | smtp password  | 1.5.0               |
| auth.email.tls     | smtp.tls     | AUTH_EMAIL_TLS      | SMTP_TLS      | `false` | enable TLS     | 1.5.0               |
| auth.email.timeout | smtp.timeout | AUTH_EMAIL_TIMEOUT  | SMTP_TIMEOUT  | `10s`   | smtp timeout   | 1.5.0               |
| img-proxy          | image-proxy.http2https | IMG_PROXY | IMAGE_PROXY_HTTP2HTTPS | `false` | enable HTTP->HTTPS proxy for images | 1.5.0 |
| notify.type        | notify.admins, notify.users | NOTIFY_TYPE | NOTIFY_ADMINS, NOTIFY_USERS | | old notification types: `email` becomes a user notification, `telegram` and `slack` admin ones; ignored when `notify.users` or `notify.admins` is set | 1.9.0               |
| notify.email.notify_admin | notify.admins=email | NOTIFY_EMAIL_ADMIN | NOTIFY_ADMINS=email | |     | 1.9.0               |
| notify.telegram.token | telegram.token | NOTIFY_TELEGRAM_TOKEN | TELEGRAM_TOKEN | | Telegram token | 1.9.0               |
| notify.telegram.timeout | telegram.timeout | NOTIFY_TELEGRAM_TIMEOUT | TELEGRAM_TIMEOUT | | Telegram timeout | 1.9.0     |
| notify.telegram.api | telegram.api-url | NOTIFY_TELEGRAM_API | TELEGRAM_API_URL | `https://api.telegram.org/bot` | Telegram API prefix; used as `telegram.api-url` only when it ends in `/bot` and `telegram.api-url` is not set | 1.9.0 |

</details>

## Admin users

Admins/moderators should be defined in `docker-compose.yml` as a list of user IDs or passed in the command line.

```yaml
environment:
  - ADMIN_SHARED_ID=github_ef0f706a79cc24b17bbbb374cd234a691a034128,github_dae9983158e9e5e127ef2b87a411ef13c891e9e5
```

To get a user ID, sign in and click your name or the name of any other user you want to promote to admin. The profile panel that opens shows the full user ID under the name.

## Docker image

Two parameters allow customizing the Docker container on the system level:

- `APP_UID` - sets UID to run Remark42 application in container (default=1001)
- `TIME_ZONE` - sets time zone of Remark42 container, would be used only on the backend as comments shown with user's timezone in the web interface (default=America/Chicago)

_See [umputun/baseimage](https://github.com/umputun/baseimage) for more details._
