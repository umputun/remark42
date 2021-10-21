---
title: Command-Line Interface parameters
---

### Required parameters

Most of the parameters have sane defaults and don't require customization. There are only a few parameters the user has to define:

1. `SECRET` - secret key, can be any long and hard-to-guess string
2. `REMARK_URL` - URL pointing to your Remark42 server, i.e. `https://demo.remark42.com`
3. At least one pair of `AUTH_<PROVIDER>_CID` and `AUTH_<PROVIDER>_CSEC` defining OAuth2 provider(s)

The minimal `docker-compose.yml` has to include all required parameters:

```yaml
version: '2'

services:
  remark42:
    image: umputun/remark42:latest
    restart: always
    container_name: "remark42"
    environment:
      - REMARK_URL=https://demo.remark42.com  # URL pointing to your Remark42 server
      - SITE=YOUR_SITE_ID                     # site ID, same as used for `site_id`, see "Setup on your website"
      - SECRET=abcd-123456-xyz-$%^&           # secret key
      - AUTH_GITHUB_CID=12345667890           # OAuth2 client ID
      - AUTH_GITHUB_CSEC=abcdefg12345678      # OAuth2 client secret
    volumes:
      - ./var:/srv/var                        # persistent volume to store all Remark42 data
```

### Complete parameters list

| Command line            | Environment             | Default                  | Description                                     |
| ----------------------- | ----------------------- | ------------------------ | ----------------------------------------------- |
| url                     | REMARK_URL              |                          | URL to Remark42 server, _required_              |
| secret                  | SECRET                  |                          | shared secret key used to sign JWT, should be a random, long, hard-to-guess string, _required_ |
| site                    | SITE                    | `remark`                 | site name(s), _multi_                           |
| store.type              | STORE_TYPE              | `bolt`                   | type of storage, `bolt` or `rpc`                |
| store.bolt.path         | STORE_BOLT_PATH         | `./var`                  | path to data directory                          |
| store.bolt.timeout      | STORE_BOLT_TIMEOUT      | `30s`                    | boltdb access timeout                           |
| admin.shared.id         | ADMIN_SHARED_ID         |                          | admin IDs (list of user IDs), _multi_           |
| admin.shared.email      | ADMIN_SHARED_EMAIL      | `admin@${REMARK_URL}`    | admin emails, _multi_                           |
| backup                  | BACKUP_PATH             | `./var/backup`           | backups location                                |
| max-back                | MAX_BACKUP_FILES        | `10`                     | max backup files to keep                        |
| cache.type              | CACHE_TYPE              | `mem`                    | type of cache, `redis_pub_sub` or `mem` or `none` |
| cache.redis_addr        | CACHE_REDIS_ADDR        | `127.0.0.1:6379`         | address of redis PubSub instance, turn `redis_pub_sub` cache on for distributed cache |
| cache.max.items         | CACHE_MAX_ITEMS         | `1000`                   | max number of cached items, `0` - unlimited     |
| cache.max.value         | CACHE_MAX_VALUE         | `65536`                  | max size of cached value, `0` - unlimited       |
| cache.max.size          | CACHE_MAX_SIZE          | `50000000`               | max size of all cached values, `0` - unlimited  |
| avatar.type             | AVATAR_TYPE             | `fs`                     | type of avatar storage, `fs`, `bolt`, or `uri`  |
| avatar.fs.path          | AVATAR_FS_PATH          | `./var/avatars`          | avatars location for `fs` store                 |
| avatar.bolt.file        | AVATAR_BOLT_FILE        | `./var/avatars.db`       | file name for `bolt` store                      |
| avatar.uri              | AVATAR_URI              | `./var/avatars`          | avatar store uri                                |
| avatar.rsz-lmt          | AVATAR_RSZ_LMT          | `0` (disabled)           | max image size for resizing avatars on save     |
| image.type              | IMAGE_TYPE              | `fs`                     | type of image storage, `fs`, `bolt`             |
| image.max-size          | IMAGE_MAX_SIZE          | `5000000`                | max size of image file                          |
| image.fs.path           | IMAGE_FS_PATH           | `./var/pictures`         | permanent location of images                    |
| image.fs.staging        | IMAGE_FS_STAGING        | `./var/pictures.staging` | staging location of images                      |
| image.fs.partitions     | IMAGE_FS_PARTITIONS     | `100`                    | number of image partitions                      |
| image.bolt.file         | IMAGE_BOLT_FILE         | `/var/pictures.db`       | images bolt file location                       |
| image.resize-width      | IMAGE_RESIZE_WIDTH      | `2400`                   | width of resized image                          |
| image.resize-height     | IMAGE_RESIZE_HEIGHT     | `900`                    | height of resized image                         |
| auth.ttl.jwt            | AUTH_TTL_JWT            | `5m`                     | jwt TTL                                         |
| auth.ttl.cookie         | AUTH_TTL_COOKIE         | `200h`                   | cookie TTL                                      |
| auth.send-jwt-header    | AUTH_SEND_JWT_HEADER    | `false`                  | send JWT as a header instead of cookie          |
| auth.same-site          | AUTH_SAME_SITE          | `default`                | set same site policy for cookies (`default`, `none`, `lax` or `strict`) |
| auth.google.cid         | AUTH_GOOGLE_CID         |                          | Google OAuth client ID                          |
| auth.google.csec        | AUTH_GOOGLE_CSEC        |                          | Google OAuth client secret                      |
| auth.facebook.cid       | AUTH_FACEBOOK_CID       |                          | Facebook OAuth client ID                        |
| auth.facebook.csec      | AUTH_FACEBOOK_CSEC      |                          | Facebook OAuth client secret                    |
| auth.microsoft.cid      | AUTH_MICROSOFT_CID      |                          | Microsoft OAuth client ID                       |
| auth.microsoft.csec     | AUTH_MICROSOFT_CSEC     |                          | Microsoft OAuth client secret                   |
| auth.github.cid         | AUTH_GITHUB_CID         |                          | GitHub OAuth client ID                          |
| auth.github.csec        | AUTH_GITHUB_CSEC        |                          | GitHub OAuth client secret                      |
| auth.twitter.cid        | AUTH_TWITTER_CID        |                          | Twitter Consumer API Key                        |
| auth.twitter.csec       | AUTH_TWITTER_CSEC       |                          | Twitter Consumer API Secret key                 |
| auth.patreon.cid        | AUTH_PATREON_CID        |                          | Patreon OAuth Client ID                         |
| auth.patreon.csec       | AUTH_PATREON_CSEC       |                          | Patreon OAuth Client Secret                     |
| auth.telegram           | AUTH_TELEGRAM           |                          | Enable Telegram auth (telegram.token must be present) |
| auth.yandex.cid         | AUTH_YANDEX_CID         |                          | Yandex OAuth client ID                          |
| auth.yandex.csec        | AUTH_YANDEX_CSEC        |                          | Yandex OAuth client secret                      |
| auth.dev                | AUTH_DEV                | `false`                  | local OAuth2 server, development mode only      |
| auth.anon               | AUTH_ANON               | `false`                  | enable anonymous login                          |
| auth.email.enable       | AUTH_EMAIL_ENABLE       | `false`                  | enable auth via email                           |
| auth.email.from         | AUTH_EMAIL_FROM         |                          | email from                                      |
| auth.email.subj         | AUTH_EMAIL_SUBJ         | `remark42 confirmation`  | email subject                                   |
| auth.email.content-type | AUTH_EMAIL_CONTENT_TYPE | `text/html`              | email content type                              |
| auth.email.template     | AUTH_EMAIL_TEMPLATE     | none (predefined)        | custom email message template file              |
| notify.users            | NOTIFY_USERS            | none                     | type of user notifications (Telegram, email)    |
| notify.admins           | NOTIFY_ADMINS           | none                     | type of admin notifications (Telegram, Slack, webhook and/or email) |
| notify.queue            | NOTIFY_QUEUE            | `100`                    | size of notification queue                      |
| notify.telegram.chan    | NOTIFY_TELEGRAM_CHAN    |                          | Telegram channel                                |
| notify.slack.token      | NOTIFY_SLACK_TOKEN      |                          | Slack token                                     |
| notify.slack.chan       | NOTIFY_SLACK_CHAN       | `general`                | Slack channel                                   |
| notify.webhook.url      | NOTIFY_WEBHOOK_URL      |                          | Webhook notification URL                        |
| notify.webhook.template | NOTIFY_WEBHOOK_TEMPLATE | `{"text": "{{.Text}}"}`  | Webhook payload template                        |
| notify.webhook.headers  | NOTIFY_WEBHOOK_HEADERS  |                          | HTTP header in format Header1:Value1,Header2:Value2,...|
| notify.webhook.timeout  | NOTIFY_WEBHOOK_TIMEOUT  | `5s`                     | Webhook connection timeout                      |
| notify.email.fromAddress| NOTIFY_EMAIL_FROM       |                          | from email address                              |
| notify.email.verification_subj | NOTIFY_EMAIL_VERIFICATION_SUBJ | `Email verification` | verification message subject          |
| telegram.token          | TELEGRAM_TOKEN          |                          | Telegram token (used for auth and Telegram notifications) |
| telegram.timeout        | TELEGRAM_TIMEOUT        | `5s`                     | Telegram connection timeout                     |
| smtp.host               | SMTP_HOST               |                          | SMTP host                                       |
| smtp.port               | SMTP_PORT               |                          | SMTP port                                       |
| smtp.username           | SMTP_USERNAME           |                          | SMTP user name                                  |
| smtp.password           | SMTP_PASSWORD           |                          | SMTP password                                   |
| smtp.tls                | SMTP_TLS                |                          | enable TLS for SMTP                             |
| smtp.timeout            | SMTP_TIMEOUT            | `10s`                    | SMTP TCP connection timeout                     |
| ssl.type                | SSL_TYPE                | none                     | `none`-HTTP, `static`-HTTPS, `auto`-HTTPS + le  |
| ssl.port                | SSL_PORT                | `8443`                   | port for HTTPS server                           |
| ssl.cert                | SSL_CERT                |                          | path to cert.pem file                           |
| ssl.key                 | SSL_KEY                 |                          | path to key.pem file                            |
| ssl.acme-location       | SSL_ACME_LOCATION       | `./var/acme`             | dir where obtained le-certs will be stored      |
| ssl.acme-email          | SSL_ACME_EMAIL          |                          | admin email for receiving notifications from LE |
| max-comment             | MAX_COMMENT_SIZE        | `2048`                   | comment's size limit                            |
| max-votes               | MAX_VOTES               | `-1`                     | votes limit per comment, `-1` - unlimited       |
| votes-ip                | VOTES_IP                | `false`                  | restrict votes from the same IP                 |
| anon-vote               | ANON_VOTE               | `false`                  | allow voting for anonymous users, require VOTES_IP to be enabled as well |
| votes-ip-time           | VOTES_IP_TIME           | `5m`                     | same IP vote restriction time, `0s` - unlimited |
| low-score               | LOW_SCORE               | `-5`                     | low score threshold                             |
| critical-score          | CRITICAL_SCORE          | `-10`                    | critical score threshold                        |
| positive-score          | POSITIVE_SCORE          | `false`                  | restricts comment's score to be only positive   |
| restricted-words        | RESTRICTED_WORDS        |                          | words banned in comments (can use `*`), _multi_ |
| restricted-names        | RESTRICTED_NAMES        |                          | names prohibited to use by the user, _multi_    |
| edit-time               | EDIT_TIME               | `5m`                     | edit window                                     |
| admin-edit              | ADMIN_EDIT              | `false`                  | unlimited edit for admins                       |
| read-age                | READONLY_AGE            |                          | read-only age of comments, days                 |
| image-proxy.http2https  | IMAGE_PROXY_HTTP2HTTPS  | `false`                  | enable HTTP->HTTPS proxy for images             |
| image-proxy.cache-external | IMAGE_PROXY_CACHE_EXTERNAL | `false`            | enable caching external images to current image storage |
| emoji                   | EMOJI                   | `false`                  | enable emoji support                            |
| simple-view             | SIMPLE_VIEW             | `false`                  | minimized UI with basic info only               |
| proxy-cors              | PROXY_CORS              | `false`                  | disable internal CORS and delegate it to proxy  |
| allowed-hosts           | ALLOWED_HOSTS           | enable all               | limit hosts/sources allowed to embed comments   |
| address                 | REMARK_ADDRESS          | all interfaces           | web server listening address                    |
| port                    | REMARK_PORT             | `8080`                   | web server port                                 |
| web-root                | REMARK_WEB_ROOT         | `./web`                  | web server root directory                       |
| update-limit            | UPDATE_LIMIT            | `0.5`                    | updates/sec limit                               |
| subscribers-only        | SUBSCRIBERS_ONLY        | `false`                  | enable commenting only for Patreon subscribers  |
| admin-passwd            | ADMIN_PASSWD            | none (disabled)          | password for `admin` basic auth                 |
| dbg                     | DEBUG                   | `false`                  | debug mode                                      |

- command line parameters are long-form `--<key>=value`, i.e. `--site=https://demo.remark42.com`
- _multi_ parameters separated by `,` in the environment or repeated with command line key, like `--site=s1 --site=s2 ...`
- _required_ parameters have to be presented in the environment or provided in the command line

### Deprecated parameters

The following list of command-line options is deprecated and might be removed in next major release after the version in which they were deprecated. After Remark42 version update, please check the startup log once for deprecation warning messages to avoid trouble with unrecognized command-line options in the future.

<details>
<summary>Deprecated options</summary>

| Command line       | Replacement   | Environment        | Replacement   | Default | Description    | Deprecation version |
| ------------------ | ------------- | ------------------ | ------------- | ------- | -------------- | ------------------- |
| auth.email.host    | smtp.host     | AUTH_EMAIL_HOST    | SMTP_HOST     |         | smtp host      | 1.5.0               |
| auth.email.port    | smtp.port     | AUTH_EMAIL_PORT    | SMTP_PORT     |         | smtp port      | 1.5.0               |
| auth.email.user    | smtp.username | AUTH_EMAIL_USER    | SMTP_USERNAME |         | smtp user name | 1.5.0               |
| auth.email.passwd  | smtp.password | AUTH_EMAIL_PASSWD  | SMTP_PASSWORD |         | smtp password  | 1.5.0               |
| auth.email.tls     | smtp.tls      | AUTH_EMAIL_TLS     | SMTP_TLS      | `false` | enable TLS     | 1.5.0               |
| auth.email.timeout | smtp.timeout  | AUTH_EMAIL_TIMEOUT | SMTP_TIMEOUT  | `10s`   | smtp timeout   | 1.5.0               |
| img-proxy          | image-proxy.http2https | IMG_PROXY | IMAGE_PROXY_HTTP2HTTPS | `false` | enable HTTP->HTTPS proxy for images | 1.5.0 |
| notify.type        | notify.admins, notify.users | NOTIFY_TYPE | NOTIFY_ADMINS, NOTIFY_USERS | |   | 1.9.0               |
| notify.email.notify_admin | notify.admins=email | NOTIFY_EMAIL_ADMIN | NOTIFY_ADMINS=email | |     | 1.9.0               |
| notify.telegram.token | telegram.token | NOTIFY_TELEGRAM_TOKEN | TELEGRAM_TOKEN | | Telegram token | 1.9.0               |
| notify.telegram.timeout | telegram.timeout | NOTIFY_TELEGRAM_TIMEOUT | TELEGRAM_TIMEOUT | | Telegram timeout | 1.9.0     |

</details>
