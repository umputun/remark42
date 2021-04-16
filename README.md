# remark42 [![Build Status](https://github.com/umputun/remark42/workflows/build/badge.svg)](https://github.com/umputun/remark42/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/umputun/remark42)](https://goreportcard.com/report/github.com/umputun/remark42) [![Coverage Status](https://coveralls.io/repos/github/umputun/remark42/badge.svg?branch=master)](https://coveralls.io/github/umputun/remark42?branch=master) [![codecov](https://codecov.io/gh/umputun/remark42/branch/master/graph/badge.svg)](https://codecov.io/gh/umputun/remark42)




Remark42 is a self-hosted, lightweight, and simple (yet functional) comment engine, which doesn't spy on users. It can be embedded into blogs, articles or any other place where readers add comments.

* Social login via Google, Twitter, Facebook, Microsoft, GitHub and Yandex
* Login via email
* Optional anonymous access
* Multi-level nested comments with both tree and plain presentations
* Import from Disqus and WordPress
* Markdown support with friendly formatter toolbar
* Moderator can remove comments and block users
* Voting, pinning and verification system
* Sortable comments
* Images upload with drag-and-drop
* Extractor for recent comments, cross-post
* RSS for all comments and each post
* Telegram and email notifications
* Export data to json with automatic backups
* No external databases, everything embedded in a single data file
* Fully dockerized and can be deployed in a single command
* Self-contained executable can be deployed directly to Linux, Windows and MacOS
* Clean, lightweight and customizable UI with white and dark themes
* Multi-site mode from a single instance
* Integration with automatic ssl (direct and via [nginx-le](https://github.com/umputun/nginx-le))
* [Privacy focused](#privacy)

[Demo site](https://remark42.com/demo/) available with all authentication methods, including email auth and anonymous access.

<details><summary>Screenshots</summary>

Comments example:
![](https://github.com/umputun/remark42/blob/master/screenshots/comments.png)

For admin screenshots see [Admin UI wiki](https://github.com/umputun/remark42/wiki/Admin-UI)
</details>


#

  - [Install](#install)
    - [Backend](#backend)
      - [With Docker](#with-docker)
      - [Without Docker](#without-docker)
      - [Parameters](#parameters)
        - [Required parameters](#required-parameters)
      - [Quick installation test](#quick-installation-test)
      - [Register oauth2 providers](#register-oauth2-providers)
        - [Google Auth Provider](#google-auth-provider)
        - [GitHub Auth Provider](#github-auth-provider)
        - [Facebook Auth Provider](#facebook-auth-provider)
        - [Twitter Auth Provider](#twitter-auth-provider)
        - [Yandex Auth Provider](#yandex-auth-provider)
      - [Initial import from Disqus](#initial-import-from-disqus)
      - [Initial import from WordPress](#initial-import-from-wordpress)
      - [Backup and restore](#backup-and-restore)
        - [Automatic backups](#automatic-backups)
        - [Manual backup](#manual-backup)
        - [Restore from backup](#restore-from-backup)
        - [Backup format](#backup-format)
      - [Admin users](#admin-users)
    - [Setup on your website](#setup-on-your-website)
      - [Comments](#comments)
      - [Last comments](#last-comments)
      - [Counter](#counter)
  - [Build from the source](#build-from-the-source)
  - [Development](#development)
    - [Backend development](#backend-development)
    - [Frontend development](#frontend-development)
      - [Build](#build)
      - [Devserver](#devserver)
  - [API](#api)
    - [Authorization](#authorization)
    - [Commenting](#commenting)
    - [RSS feeds](#rss-feeds)
    - [Admin](#admin)
  - [Privacy](#privacy)
  - [Technical details](#technical-details)


## Install

### Backend

#### With Docker

_this is the recommended way to run remark42_

* copy provided `docker-compose.yml` and customize for your needs
* make sure you **don't keep** `ADMIN_PASSWD=something...` for any non-development deployments
* pull prepared images from the DockerHub and start - `docker-compose pull && docker-compose up -d`
* alternatively compile from the sources - `docker-compose build && docker-compose up -d`

#### Without Docker

* download archive for [stable release](https://github.com/umputun/remark42/releases) or [development version](https://remark42.com/downloads)
* unpack with `gunzip` (Linux, macOS) or with `zip` (Windows)
* run as `remark42.{os}-{arch} server {parameters...}`, i.e. `remark42.linux-amd64 server --secret=12345 --url=http://127.0.0.1:8080`
* alternatively compile from the sources - `make OS=[linux|darwin|windows] ARCH=[amd64,386,arm64,arm]`

#### Parameters

| Command line            | Environment             | Default                  | Description                                     |
| ----------------------- | ----------------------- | ------------------------ | ----------------------------------------------- |
| url                     | REMARK_URL              |                          | url to remark42 server, _required_              |
| secret                  | SECRET                  |                          | secret key, _required_                          |
| site                    | SITE                    | `remark`                 | site name(s), _multi_                           |
| store.type              | STORE_TYPE              | `bolt`                   | type of storage, `bolt` or `rpc`                |
| store.bolt.path         | STORE_BOLT_PATH         | `./var`                  | path to data directory                          |
| store.bolt.timeout      | STORE_BOLT_TIMEOUT      | `30s`                    | boltdb access timeout                           |
| admin.shared.id         | ADMIN_SHARED_ID         |                          | admin ids (list of user ids), _multi_           |
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
| avatar.bolt.file        | AVATAR_BOLT_FILE        | `./var/avatars.db`       | file name for  `bolt` store                     |
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
| auth.same-site          | AUTH_SAME_SITE          | `default`                | set same site policy for cookies (`default`, `none`, `lax` or `strict`)|
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
| auth.yandex.cid         | AUTH_YANDEX_CID         |                          | Yandex OAuth client ID                          |
| auth.yandex.csec        | AUTH_YANDEX_CSEC        |                          | Yandex OAuth client secret                      |
| auth.dev                | AUTH_DEV                | `false`                  | local oauth2 server, development mode only      |
| auth.anon               | AUTH_ANON               | `false`                  | enable anonymous login                          |
| auth.email.enable       | AUTH_EMAIL_ENABLE       | `false`                  | enable auth via email                           |
| auth.email.from         | AUTH_EMAIL_FROM         |                          | email from                                      |
| auth.email.subj         | AUTH_EMAIL_SUBJ         | `remark42 confirmation`  | email subject                                   |
| auth.email.content-type | AUTH_EMAIL_CONTENT_TYPE | `text/html`              | email content type                              |
| auth.email.template     | AUTH_EMAIL_TEMPLATE     | none (predefined)        | custom email message template file              |
| notify.type             | NOTIFY_TYPE             | none                     | type of notification (telegram and/or email)    |
| notify.queue            | NOTIFY_QUEUE            | `100`                    | size of notification queue                      |
| notify.telegram.token   | NOTIFY_TELEGRAM_TOKEN   |                          | telegram token                                  |
| notify.telegram.chan    | NOTIFY_TELEGRAM_CHAN    |                          | telegram channel                                |
| notify.telegram.timeout | NOTIFY_TELEGRAM_TIMEOUT | `5s`                     | telegram timeout                                |
| notify.email.fromAddress | NOTIFY_EMAIL_FROM      |                          | from email address                              |
| notify.email.verification_subj | NOTIFY_EMAIL_VERIFICATION_SUBJ | `Email verification` | verification message subject          |
| notify.email.notify_admin | NOTIFY_EMAIL_ADMIN    | `false`                  | notify admin on new comments via ADMIN_SHARED_EMAIL |
| smtp.host               | SMTP_HOST               |                          | SMTP host                                       |
| smtp.port               | SMTP_PORT               |                          | SMTP port                                       |
| smtp.username           | SMTP_USERNAME           |                          | SMTP user name                                  |
| smtp.password           | SMTP_PASSWORD           |                          | SMTP password                                   |
| smtp.tls                | SMTP_TLS                |                          | enable TLS for SMTP                             |
| smtp.timeout            | SMTP_TIMEOUT            | `10s`                    | SMTP TCP connection timeout                     |
| ssl.type                | SSL_TYPE                | none                     | `none`-http, `static`-https, `auto`-https + le  |
| ssl.port                | SSL_PORT                | `8443`                   | port for https server                           |
| ssl.cert                | SSL_CERT                |                          | path to cert.pem file                           |
| ssl.key                 | SSL_KEY                 |                          | path to key.pem file                            |
| ssl.acme-location       | SSL_ACME_LOCATION       | `./var/acme`             | dir where obtained le-certs will be stored      |
| ssl.acme-email          | SSL_ACME_EMAIL          |                          | admin email for receiving notifications from LE |
| max-comment             | MAX_COMMENT_SIZE        | `2048`                   | comment's size limit                            |
| max-votes               | MAX_VOTES               | `-1`                     | votes limit per comment, `-1` - unlimited       |
| votes-ip                | VOTES_IP                | `false`                  | restrict votes from the same ip                 |
| anon-vote               | ANON_VOTE               | `false`                  | allow voting for anonymous users, require VOTES_IP to be enabled as well |
| votes-ip-time           | VOTES_IP_TIME           | `5m`                     | same ip vote restriction time, `0s` - unlimited |
| low-score               | LOW_SCORE               | `-5`                     | low score threshold                             |
| critical-score          | CRITICAL_SCORE          | `-10`                    | critical score threshold                        |
| positive-score          | POSITIVE_SCORE          | `false`                  | restricts comment's score to be only positive   |
| restricted-words        | RESTRICTED_WORDS        |                          | words banned in comments (can use `*`), _multi_ |
| restricted-names        | RESTRICTED_NAMES        |                          | names prohibited to use by the user, _multi_    |
| edit-time               | EDIT_TIME               | `5m`                     | edit window                                     |
| read-age                | READONLY_AGE            |                          | read-only age of comments, days                 |
| image-proxy.http2https  |  IMAGE_PROXY_HTTP2HTTPS | `false`                  | enable http->https proxy for images             |
| image-proxy.cache-external | IMAGE_PROXY_CACHE_EXTERNAL | `false`            | enable caching external images to current image storage |
| emoji                   | EMOJI                   | `false`                  | enable emoji support                            |
| simple-view             | SIMPLE_VIEW             | `false`                  | minimized UI with basic info only               |
| proxy-cors              | PROXY_CORS              | `false`                  | disable internal CORS and delegate it to proxy  |
| allowed-hosts           | ALLOWED_HOSTS           |  enable all              | limit hosts/sources allowed to embed comments   |
| port                    | REMARK_PORT             | `8080`                   | web server port                                 |
| web-root                | REMARK_WEB_ROOT         | `./web`                  | web server root directory                       |
| update-limit            | UPDATE_LIMIT            | `0.5`                    | updates/sec limit                               |
| admin-passwd            | ADMIN_PASSWD            | none (disabled)          | password for `admin` basic auth                 |
| dbg                     | DEBUG                   | `false`                  | debug mode                                      |

* command line parameters are long form `--<key>=value`, i.e. `--site=https://demo.remark42.com`
* _multi_ parameters separated by `,` in the environment or repeated with command line key, like `--site=s1 --site=s2 ...`
* _required_ parameters have to be presented in the environment or provided in command line

##### Deprecated

Following list of command-line options is deprecated and will be removed in 2 minor releases or 1 major release (whichever is closer)
from the version in which they were deprecated. After remark42 version update, please check startup log once for deprecation warnings to avoid
trouble with unrecognized command-line options in the future.

<details>
<summary>deprecated options</summary>

| Command line       | Replacement   | Environment        | Replacement   | Default | Description    | Deprecation version |
| ------------------ | ------------- | ------------------ | ------------- | ------- | -------------- | ------------------- |
| auth.email.host    | smtp.host     | AUTH_EMAIL_HOST    | SMTP_HOST     |         | smtp host      | 1.5.0               |
| auth.email.port    | smtp.port     | AUTH_EMAIL_PORT    | SMTP_PORT     |         | smtp port      | 1.5.0               |
| auth.email.user    | smtp.username | AUTH_EMAIL_USER    | SMTP_USERNAME |         | smtp user name | 1.5.0               |
| auth.email.passwd  | smtp.password | AUTH_EMAIL_PASSWD  | SMTP_PASSWORD |         | smtp password  | 1.5.0               |
| auth.email.tls     | smtp.tls      | AUTH_EMAIL_TLS     | SMTP_TLS      | `false` | enable TLS     | 1.5.0               |
| auth.email.timeout | smtp.timeout  | AUTH_EMAIL_TIMEOUT | SMTP_TIMEOUT  | `10s`   | smtp timeout   | 1.5.0               |
| img-proxy          | image-proxy.http2https | IMG_PROXY | IMAGE_PROXY_HTTP2HTTPS | `false` | enable http->https proxy for images | 1.5.0 |
</details>

##### Required parameters

Most of the parameters have sane defaults and don't require customization. There are only a few parameters user has to define:

1. `SECRET` - secret key, can be any long and hard-to-guess string.
2. `REMARK_URL` - url pointing to your remark42 server, i.e. `https://demo.remark42.com`
3. At least one pair of `AUTH_<PROVIDER>_CID` and `AUTH_<PROVIDER>_CSEC` defining oauth2 provider(s)

The minimal `docker-compose.yml` has to include all required parameters:

```yaml
version: '2'

services:
    remark42:
        image: umputun/remark42:latest
        restart: always
        container_name: "remark42"
        environment:
            - REMARK_URL=https://demo.remark42.com  # url pointing to your remark42 server
            - SITE=YOUR_SITE_ID                     # site ID, same as used for `site_id`, see "Setup on your website"
            - SECRET=abcd-123456-xyz-$%^&           # secret key
            - AUTH_GITHUB_CID=12345667890           # oauth2 client ID
            - AUTH_GITHUB_CSEC=abcdefg12345678      # oauth2 client secret
        volumes:
            - ./var:/srv/var                        # persistent volume to store all remark42 data
```

#### Quick installation test

To verify if remark has been properly installed, check a demo page at `${REMARK_URL}/web` URL. Make sure to include `remark` site id to `${SITE}` list.

#### Register oauth2 providers

Authentication handled by external providers. You should setup oauth2 for all (or some) of them to allow users to make comments. It is not mandatory to have all of them, but at least one should be correctly configured.

##### Google Auth Provider

1.  Create a new project: https://console.developers.google.com/project
1.  Choose the new project from the top right project dropdown (only if another project is selected)
1.  In the project Dashboard center pane, choose **"API Manager"**
1.  In the left Nav pane, choose **"Credentials"**
1.  In the center pane, choose **"OAuth consent screen"** tab. Fill in **"Product name shown to users"** and hit save.
1.  In the center pane, choose **"Credentials"** tab.
    * Open the **"New credentials"** drop down
    * Choose **"OAuth client ID"**
    * Choose **"Web application"**
    * Application name is freeform, choose something appropriate
    * Authorized origins is your domain ex: `https://remark42.mysite.com`
    * Authorized redirect URIs is the location of oauth2/callback constructed as domain + `/auth/google/callback`, ex: `https://remark42.mysite.com/auth/google/callback`
    * Choose **"Create"**
1.  Take note of the **Client ID** and **Client Secret**

_instructions for google oauth2 setup borrowed from [oauth2_proxy](https://github.com/bitly/oauth2_proxy)_

##### GitHub Auth Provider

1.  Create a new **"OAuth App"**: https://github.com/settings/developers
1.  Fill **"Application Name"** and **"Homepage URL"** for your site
1.  Under **"Authorization callback URL"** enter the correct url constructed as domain + `/auth/github/callback`. ie `https://remark42.mysite.com/auth/github/callback`
1.  Take note of the **Client ID** and **Client Secret**

##### Facebook Auth Provider

1.  From https://developers.facebook.com select **"My Apps"** / **"Add a new App"**
1.  Set **"Display Name"** and **"Contact email"**
1.  Choose **"Facebook Login"** and then **"Web"**
1.  Set "Site URL" to your domain, ex: `https://remark42.mysite.com`
1.  Under **"Facebook login"** / **"Settings"** fill "Valid OAuth redirect URIs" with your callback url constructed as domain + `/auth/facebook/callback`
1.  Select **"App Review"** and turn public flag on. This step may ask you to provide a link to your privacy policy.

#### Microsoft Auth Provider

1.  Register a new application [using the Azure portal](https://docs.microsoft.com/en-us/graph/auth-register-app-v2).
2.  Under **"Authentication/Platform configurations/Web"** enter the correct url constructed as domain + `/auth/microsoft/callback`. i.e. `https://example.mysite.com/auth/microsoft/callback`
3.  In "Overview" take note of the **Application (client) ID**
4.  Choose the new project from the top right project dropdown (only if another project is selected)
5.  Select "Certificates & secrets" and click on "+ New Client Secret".

##### Twitter Auth Provider

1.	Create a new twitter application https://developer.twitter.com/en/apps
1.	Fill **App name**, **Description** and **URL** of your site
1.	In the field **Callback URLs** enter the correct url of your callback handler e.g.  domain + `/auth/twitter/callback`
1.	Under **Key and tokens** take note of the **Consumer API Key** and **Consumer API Secret key**. Those will be used as `AUTH_TWITTER_CID` and
 `AUTH_TWITTER_CSEC`

##### Yandex Auth Provider

1.  Create a new **"OAuth App"**: https://oauth.yandex.com/client/new
1.  Fill **"App name"** for your site
1.  Under **Platforms** select **"Web services"** and enter **"Callback URI #1"** constructed as domain + `/auth/yandex/callback`. ie `https://remark42.mysite.com/auth/yandex/callback`
1.  Select **Permissions**. You need following permissions only from the **"Yandex.Passport API"** section:
    * Access to user avatar
    * Access to username, first name and surname, gender
1.  Fill out the rest of fields if needed
1.  Take note of the **ID** and **Password**

For more details refer to [Yandex OAuth](https://tech.yandex.com/oauth/doc/dg/concepts/about-docpage/) and [Yandex.Passport](https://tech.yandex.com/passport/doc/dg/index-docpage/) API documentation.

##### Anonymous Auth Provider

Optionally, anonymous access can be turned on. In this case an extra `anonymous` provider will allow logins without any social login with any name satisfying 2 conditions:

- name should be at least 3 characters long
- name has to start from the letter and contains letters, numbers, underscores and spaces only.

### Importing comments

Remark supports importing comments from Disqus, WordPress or native backup format.
All imported comments has `Imported` field set to `true`.

#### Initial import from Disqus

1.  Disqus provides an export of all comments on your site in a g-zipped file. This is found in your Moderation panel at Disqus Admin > Setup > Export. The export will be sent into a queue and then emailed to the address associated with your account once it's ready. Direct link to export will be something like `https://<siteud>.disqus.com/admin/discussions/export/`. See [importing-exporting](https://help.disqus.com/customer/portal/articles/1104797-importing-exporting) for more details.
2.  Move this file to your remark42 host within `./var` and unzip, i.e. `gunzip <disqus-export-name>.xml.gz`.
3.  Run import command - `docker exec -it remark42 import -p disqus -f /srv/var/{disqus-export-name}.xml -s {your site id}`

#### Initial import from WordPress

1. Install WordPress [plugin](https://wordpress.org/plugins/wp-exporter/) to export comments and follow it instructions. The plugin should produce a xml-based file with site content including comments.
2. Move this file to your remark42 host within `./var`
3. Run import command - `docker exec -it remark42 import -p wordpress -f {wordpress-export-name}.xml -s {your site id}`

#### Backup and restore

##### Automatic backups
Remark42 by default makes daily backup files under `${BACKUP_PATH}` (default `./var/backup`). Backups kept up to `${MAX_BACKUP_FILES}` (default 10). Each backup file contains exported and gzipped content, i.e., all comments. At any point, the user can restore such backup and revert all comments to the desirable state. Note: restore procedure cleans the current data store and replaces all comments with comments from the backup file.

For safety and security reasons restore functionality not exposed outside of your server by default. The recommended way to restore from the backup is to use provided `scripts/restore-backup.sh`. It can run inside the container:

`docker exec -it remark42 restore -f {backup-filename.gz} -s {your site id}`

##### Manual backup

In addition to automatic backups user can make a backup manually. This command makes `userbackup-{site id}-{timestamp}.gz` by default.

`docker exec -it remark42 backup -s {your site id}`

##### Restore from backup

Restore will clean all comments first and then will processed with complete import from a given file.

`docker exec -it remark42 restore -f {backup file name} -s {your site id}`

##### Backup format

Backup file is a text file with all exported comments separated by EOL. Each backup record is a valid json with all key/value
unmarshaled from `Comment` struct (see below).

#### Admin users

Admins/moderators should be defined in `docker-compose.yml` as a list of user IDs or passed in the command line.

```
    environment:
        - ADMIN_SHARED_ID=github_ef0f706a79cc24b17bbbb374cd234a691a034128,github_dae9983158e9e5e127ef2b87a411ef13c891e9e5
```

To get user id just login and click on your username or any other user you want to promote to admins.
It will expand login info and show full user ID.

#### Docker parameters

Two parameters allow customizing Docker container on the system level:

- `APP_UID` - sets UID to run remark42 application in container (default=1001)
- `TIME_ZONE` - sets time zone of remark42 container (default=America/Chicago)

_see [umputun/baseimage](https://github.com/umputun/baseimage) for more details_

example of `docker-compose.yml`:

```yaml
version: '2'

services:
    remark42:
        image: umputun/remark42:latest
        restart: always
        container_name: "remark42"
        environment:
            - APP_UID=2000                          # runs remark42 app with non-default UID
            - TIME_ZONE=GTC                         # sets container time to UTC

            - REMARK_URL=https://demo.remark42.com  # url pointing to your remark42 server
            - SITE=YOUR_SITE_ID                     # site ID, same as used for `site_id`, see "Setup on your website"
            - SECRET=abcd-123456-xyz-$%^&           # secret key
            - AUTH_GITHUB_CID=12345667890           # oauth2 client ID
            - AUTH_GITHUB_CSEC=abcdefg12345678      # oauth2 client secret
        volumes:
            - ./var:/srv/var                        # persistent volume to store all remark42 data
```


### Setup on your website

#### Comments

It's a main widget which renders list of comments.

Add this snippet to the bottom of web page:

```html
<script>
  var remark_config = {
    host: "REMARK_URL", // hostname of remark server, same as REMARK_URL in backend config, e.g. "https://demo.remark42.com"
    site_id: 'YOUR_SITE_ID',
    components: ['embed'], // optional param; which components to load. default to ["embed"]
                           // to load all components define components as ['embed', 'last-comments', 'counter']
                           // available component are:
                           //     - 'embed': basic comments widget
                           //     - 'last-comments': last comments widget, see `Last Comments` section below
                           //     - 'counter': counter widget, see `Counter` section below
    url: 'PAGE_URL', // optional param; if it isn't defined
                     // `window.location.origin + window.location.pathname` will be used,
                     //
                     // Note that if you use query parameters as significant part of url
                     // (the one that actually changes content on page)
                     // you will have to configure url manually to keep query params, as
                     // `window.location.origin + window.location.pathname` doesn't contain query params and
                     // hash. For example default url for `https://example/com/example-post?id=1#hash`
                     // would be `https://example/com/example-post`.
                     //
                     // The problem with query params is that they often contain useless params added by
                     // various trackers (utm params) and doesn't have defined order, so Remark treats differently
                     // all this examples:
                     // https://example.com/?postid=1&date=2007-02-11
                     // https://example.com/?date=2007-02-11&postid=1
                     // https://example.com/?date=2007-02-11&postid=1&utm_source=google
                     //
                     // If you deal with query parameters make sure you pass only significant part of it
                     // in well defined order
    max_shown_comments: 10, // optional param; if it isn't defined default value (15) will be used
    theme: 'dark', // optional param; if it isn't defined default value ('light') will be used
    page_title: 'Moving to Remark42', // optional param; if it isn't defined `document.title` will be used
    locale: 'en', // set up locale and language, if it isn't defined default value ('en') will be used
    show_email_subscription: false // optional param; by default it is `true` and you can see email subscription feature
                                   // in interface when enable it from backend side
                                   // if you set this param in `false` you will get notifications email notifications as admin
                                   // but your users won't have interface for subscription
  };
</script>
<script>!function(e,n){for(var o=0;o<e.length;o++){var r=n.createElement("script"),c=".js",d=n.head||n.body;"noModule"in r?(r.type="module",c=".mjs"):r.async=!0,r.defer=!0,r.src=remark_config.host+"/web/"+e[o]+c,d.appendChild(r)}}(remark_config.components||["embed"],document);</script>
```

And then add this node in the place where you want to see Remark42 widget:

```html
<div id="remark42"></div>
```

After that widget will be rendered inside this node.

If you want to set this up on a Single Page App, see [appropriate doc page](https://remark42.com/docs/latest/spa/).

##### Themes

Right now Remark has two themes: light and dark.
You can pick one using configuration object,
but there is also a possibility to switch between themes in runtime.
For this purpose Remark adds to `window` object named `REMARK42`,
which contains function `changeTheme`.
Just call this function and pass a name of the theme that you want to turn on:

```js
window.REMARK42.changeTheme('light');
```

##### Locales

Right now Remark is translated to en, ru (partially), de, and fi languages.
You can pick one using [configuration object](#setup-on-your-website).

Do you want translate remark42 to other locale? Please see [this documentation](https://github.com/umputun/remark42/blob/master/docs/translation.md) for details.

#### Last comments

It's a widget which renders list of last comments from your site.

Add this snippet to the bottom of web page, or adjust already present `remark_config` to have `last-comments` in `components` list:

```html
<script>
  var remark_config = {
    host: "REMARK_URL", // hostname of remark server, same as REMARK_URL in backend config, e.g. "https://demo.remark42.com"
    site_id: 'YOUR_SITE_ID',
    components: ['last-comments']
  };
</script>
```

And then add this node in the place where you want to see last comments widget:

```html
<div class="remark42__last-comments" data-max="50"></div>
```

`data-max` sets the max amount of comments (default: `15`).

#### Counter

It's a widget which renders a number of comments for the specified page.

Add this snippet to the bottom of web page, or adjust already present `remark_config` to have `counter` in `components` list:

```html
<script>
  var remark_config = {
    host: "REMARK_URL", // hostname of remark server, same as REMARK_URL in backend config, e.g. "https://demo.remark42.com"
    site_id: 'YOUR_SITE_ID',
    components: ['counter']
  };
</script>
```

And then add a node like this in the place where you want to see a number of comments:

```html
<span class="remark42__counter" data-url="https://domain.com/path/to/article/"></span>
```

You can use as many nodes like this as you need to.
The script will found all them by the class `remark__counter`,
and it will use `data-url` attribute to define the page with comments.

Also script can use `url` property from `remark_config` object, or `window.location.origin + window.location.pathname` if nothing else is defined.

## Build from the source

- to build Docker container - `make docker`. This command will produce container `umputun/remark42`.
- to build a single binary for direct execution - `make OS=<linux|windows|darwin> ARCH=<amd64|386>`. This step will produce executable
 `remark42` file with everything embedded.

## Development

You can use fully functional local version to develop and test both frontend & backend. It requires at least 2GB RAM or swap enabled

To bring it up run:

```bash
# if you mainly work on backend
cp compose-dev-backend.yml compose-private.yml
# if you mainly work on frontend
cp compose-dev-frontend.yml compose-private.yml
# now, edit / debug `compose-private.yml` to your heart's content.

# build and run
docker-compose -f compose-private.yml build
docker-compose -f compose-private.yml up
```

It starts Remark42 on `127.0.0.1:8080` and adds local OAuth2 provider “Dev”.
To access UI demo page go to `127.0.0.1:8080/web`.
By default, you would be logged in as `dev_user` which defined as admin.
You can tweak any of [supported parameters](#Parameters) in corresponded yml file.

Backend Docker Compose config by default skips running frontend related tests.
Frontend Docker Compose config by default skips running backend related tests and sets `NODE_ENV=development` for frontend build.

### Backend development

In order to run backend locally (development mode, without Docker) you have to have the latest stable `go` toolchain [installed](https://golang.org/doc/install).

To run backend - `cd backend; go run app/main.go server --dbg --secret=12345 --url=http://127.0.0.1:8080 --admin-passwd=password --site=remark`
It stars backend service with embedded bolt store on port `8080` with basic auth, allowing to authenticate and run requests directly, like this:
`HTTP http://admin:password@127.0.0.1:8080/api/v1/find?site=remark&sort=-active&format=tree&url=http://127.0.0.1:8080`

### Frontend development

#### Developer guide

Frontend guide can be found here: [./frontend/README.md](./frontend/README.md)

#### Build
You should have at least 2GB RAM or swap enabled for building

* install [Node.js 12.11](https://nodejs.org/en/) or higher;
* install [NPM 6.13.4](https://www.npmjs.com/package/npm);
* run `npm install` inside `./frontend`;
* run `npm run build` there;
* result files will be saved in `./frontend/public`.

**Note** Running `npm install` will set up precommit hooks into your git repository.
It used to reformat your frontend code using `prettier` and lint with `eslint` and `stylelint` before every commit.

#### Devserver

For local development mode with Hot Reloading use `npm start` instead of `npm run build`.
In this case `webpack` will serve files using `webpack-dev-server` on `localhost:9000`.
By visiting `127.0.0.1:9000/web` you will get a page with main comments widget
communicating with demo server backend running on `https://demo.remark42.com`.
But you will not be able to login with any oauth providers due to security reasons.

You can attach to locally running backend by providing `REMARK_URL` environment variable.
```sh
npx cross-env REMARK_URL=http://127.0.0.1:8080 npm start
```

**Note** If you want to redefine env variables such as `PORT` on your local instance you can add `.env` file
to `./frontend` folder and rewrite variables as you wish. For such functional we use `dotenv`

The best way for start local developer environment:
```sh
cp compose-dev-frontend.yml compose-private-frontend.yml
docker-compose -f compose-private-frontend.yml up --build
cd frontend
npm run dev
```

Developer build running by `webpack-dev-server` supports devtools for [React](https://github.com/facebook/react-devtools) and
[Redux](https://github.com/zalmoxisus/redux-devtools-extension).


## API

### Authorization

* `GET /auth/{provider}/login?from=http://url&site=site_id&session=1` - perform "social" login with one of supported providers and redirect to `url`. Presence of `session` (any non-zero value) change the default cookie expiration and makes them session-only.
* `GET /auth/logout` - logout

```go
type User struct {
    Name    string `json:"name"`
    ID      string `json:"id"`
    Picture string `json:"picture"`
    Admin   bool   `json:"admin"`
    Blocked bool   `json:"block"`
    Verified bool  `json:"verified"`
}
```

_currently supported providers are `google`, `facebook`, `github` and `yandex`_

### Commenting

* `POST /api/v1/comment` - add a comment. _auth required_

```go
type Comment struct {
    ID        string          `json:"id"`      // comment ID, read only
    ParentID  string          `json:"pid"`     // parent ID
    Text      string          `json:"text"`    // comment text, after md processing
    Orig      string          `json:"orig"`    // original comment text
    User      User            `json:"user"`    // user info, read only
    Locator   Locator         `json:"locator"` // post locator
    Score     int             `json:"score"`   // comment score, read only
    Vote      int             `json:"vote"`    // vote for the current user, -1/1/0.
    Controversy float64       `json:"controversy,omitempty"` // comment controversy, read only
    Timestamp time.Time       `json:"time"`    // time stamp, read only
    Edit      *Edit           `json:"edit,omitempty" bson:"edit,omitempty"` // pointer to have empty default in json response
    Pin       bool            `json:"pin"`     // pinned status, read only
    Delete    bool            `json:"delete"`  // delete status, read only
    PostTitle string          `json:"title"`   // post title
}

type Locator struct {
    SiteID string `json:"site"`     // site id
    URL    string `json:"url"`      // post url
}

type Edit struct {
  Timestamp time.Time `json:"time" bson:"time"`
  Summary   string    `json:"summary"`
}
```

* `POST /api/v1/preview` - preview comment in html. Body is `Comment` to render

* `GET /api/v1/find?site=site-id&url=post-url&sort=fld&format=tree|plain` - find all comments for given post

This is the primary call used by UI to show comments for given post. It can return comments in two formats - `plain` and `tree`.
In plain format result will be sorted list of `Comment`. In tree format this is going to be tree-like object with this structure:

```go
type Tree struct {
    Nodes []Node `json:"comments"`
    Info  store.PostInfo `json:"info,omitempty"`
}

type Node struct {
    Comment store.Comment `json:"comment"`
    Replies []Node        `json:"replies,omitempty"`
}
```

Sort can be `time`, `active` or `score`. Supported sort order with prefix -/+, i.e. `-time`. For `tree` mode sort will be applied to top-level comments only and all replies always sorted by time.

* `PUT /api/v1/comment/{id}?site=site-id&url=post-url` - edit comment, allowed once in `EDIT_TIME` minutes since creation.  Body is `EditRequest` json

```go
   type EditRequest struct {
     Text    string `json:"text"`    // updated text
     Summary string `json:"summary"` // optional, summary of the edit
     Delete  bool   `json:"delete"`  // delete flag
   }{}
```

* `GET /api/v1/last/{max}?site=site-id&since=ts-msec` - get up to `{max}` last comments, `since` (epoch time, milliseconds) is optional
* `GET /api/v1/id/{id}?site=site-id` - get comment by `comment id`
* `GET /api/v1/comments?site=site-id&user=id&limit=N` - get comment by `user id`, returns `response` object
  ```go
  type response struct {
      Comments []store.Comment  `json:"comments"`
      Count    int              `json:"count"`
  }{}
  ```
* `GET /api/v1/count?site=site-id&url=post-url` - get comment's count for `{url}`
* `POST /api/v1/count?site=siteID` - get number of comments for posts from post body (list of post IDs)
* `GET /api/v1/list?site=site-id&limit=5&skip=2` - list commented posts, returns array or `PostInfo`, limit=0 will return all posts
  ```go
  type PostInfo struct {
      URL   string      `json:"url"`
      Count int         `json:"count"`
      ReadOnly bool     `json:"read_only,omitempty"`
      FirstTS time.Time `json:"first_time,omitempty"`
      LastTS  time.Time `json:"last_time,omitempty"`
  }
  ```
* `GET /api/v1/user` - get user info, _auth required_
* `PUT /api/v1/vote/{id}?site=site-id&url=post-url&vote=1` - vote for comment. `vote`=1 will increase score, -1 decrease. _auth required_
* `GET /api/v1/userdata?site=site-id` - export all user data to gz stream  _auth required_
* `POST /api/v1/deleteme?site=site-id` - request deletion of user data. _auth required_
* `GET /api/v1/config?site=site-id` - returns configuration (parameters) for given site

  ```go
  type Config struct {
        Version        string   `json:"version"`
        EditDuration   int      `json:"edit_duration"`
        MaxCommentSize int      `json:"max_comment_size"`
        Admins         []string `json:"admins"`
        AdminEmail     string   `json:"admin_email"`
        Auth           []string `json:"auth_providers"`
        LowScore       int      `json:"low_score"`
        CriticalScore  int      `json:"critical_score"`
        PositiveScore  bool     `json:"positive_score"`
        ReadOnlyAge    int      `json:"readonly_age"`
        MaxImageSize   int      `json:"max_image_size"`
        EmojiEnabled   bool     `json:"emoji_enabled"`
  }
  ```

* `GET /api/v1/info?site=site-idd&url=post-url` - returns `PostInfo` for site and url

### Streaming API

Streaming API provide server-sent events for post updates as well as site update

* `GET /api/v1/stream/info?site=site-idd&url=post-url&since=unix_ts_msec` - returns stream (`event: info`) with `PostInfo` records for the site and url. `since` is optional
* `GET /api/v1/stream/last?site=site-id&since=unix_ts_msec` - returns updates stream (`event: last`) with comments for the site, `since` is optional

<details><summary>response example</summary>

```
data: {"url":"https://radio-t.com/blah1","count":2,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.142872-05:00"}

event: info
data: {"url":"https://radio-t.com/blah1","count":3,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.157709-05:00"}

event: info
data: {"url":"https://radio-t.com/blah1","count":4,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.172991-05:00"}

event: info
data: {"url":"https://radio-t.com/blah1","count":5,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.188429-05:00"}

event: info
data: {"url":"https://radio-t.com/blah1","count":6,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.204742-05:00"}

event: info
data: {"url":"https://radio-t.com/blah1","count":7,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.220692-05:00"}

event: info
data: {"url":"https://radio-t.com/blah1","count":8,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.23817-05:00"}

event: info
data: {"url":"https://radio-t.com/blah1","count":9,"first_time":"2019-06-18T12:53:48.125686-05:00","last_time":"2019-06-18T12:53:48.254669-05:00"}
```

</details>

### RSS feeds

* `GET /api/v1/rss/post?site=site-id&url=post-url` - rss feed for a post
* `GET /api/v1/rss/site?site=site-id` - rss feed for given site
* `GET /api/v1/rss/reply?site=site-id&user=user-id` - rss feed for replies to user's comments

### Images management

* `GET /api/v1/picture/{user}/{id}` - load stored image
* `POST /api/v1/picture` - upload and store image, uses post form with `FormFile("file")`. returns `{"id": user/imgid}` _auth required_

_returned id should be appended to load image url on caller side_

### Email subscription

* `GET /api/v1/email?site=site-id` - get user's email, _auth required_
* `POST /api/v1/email/subscribe?site=site-id&address=user@example.org` -  makes confirmation token and sends it to user over email, _auth required_

  Trying to subscribe same email second time will return response code `409 Conflict` and explaining error message.
* `POST /api/v1/email/confirm?site=site-id&tkn=token` - uses provided token parameter to set email for the user, _auth required_

  Setting email subscribe user for all first-level replies to his messages.
* `DELETE /api/v1/email?site=siteID` - removes user's email, _auth required_

### Admin

* `DELETE /api/v1/admin/comment/{id}?site=site-id&url=post-url` - delete comment by `id`.
* `PUT /api/v1/admin/user/{userid}?site=site-id&block=1&ttl=7d` - block or unblock user with optional ttl (default=permanent)
* `GET api/v1/admin/blocked&site=site-id` - list of blocked user ids
  ```go
  type BlockedUser struct {
      ID        string    `json:"id"`
      Name      string    `json:"name"`
      Until     time.Time `json:"time"`
  }
  ```
* `GET /api/v1/admin/export?site=site-id&mode=[stream|file]` - export all comments to json stream or gz file.
* `POST /api/v1/admin/import?site=site-id` - import comments from the backup, uses post body.
* `POST /api/v1/admin/import/form?site=site-id` - import comments from the backup, user post form.
* `POST /api/v1/admin/remap?site=site-id` - remap comments to different URLs. Expect list of "from-url new-url" pairs separated by \n.
From-url and new-url parts separated by space. If urls end with asterisk (*) it means matching by prefix. Remap procedure based on
export/import chain so make backup first.
    ```
    http://oldsite.com* https://newsite.com*
    http://oldsite.com/from-old-page/1 https://newsite.com/to-new-page/1
    ```
* `GET /api/v1/admin/wait?site=site-id` - wait for completion for any async migration ops (import or remap).
* `PUT /api/v1/admin/pin/{id}?site=site-id&url=post-url&pin=1` - pin or unpin comment.
* `GET /api/v1/admin/user/{userid}?site=site-id` - get user's info.
* `DELETE /api/v1/admin/user/{userid}?site=site-id` - delete all user's comments.
* `PUT /api/v1/admin/readonly?site=site-id&url=post-url&ro=1` - set read-only status
* `PUT /api/v1/admin/verify/{userid}?site=site-id&verified=1` - set verified status
* `GET /api/v1/admin/deleteme?token=token` - process deleteme user's request

_all admin calls require auth and admin privilege_

## Privacy

* Remark42 is trying to be very sensitive to any private or semi-private information.
* Authentication requesting the minimal possible scope from authentication providers. All extra information returned by them dropped immediately and not stored in any form.
* Generally, remark42 keeps user id, username and avatar link only. None of these fields exposed directly - id and name hashed, avatar proxied.
* There is no tracking of any sort.
* Login mechanic uses JWT stored in a cookie (httpOnly, secured). The second cookie (XSRF_TOKEN) is a random id preventing CSRF.
* There is no cross-site login, i.e., user's behavior can't be analyzed across independent sites running remark42.
* There are no third-party analytic services involved.
* User can request all information remark42 knows about and export to gz file.
* Supported complete cleanup of all information related to user's activity.
* Cookie lifespan can be restricted to session-only.
* All potentially sensitive data stored by remark42 hashed and encrypted.

## Technical details

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
