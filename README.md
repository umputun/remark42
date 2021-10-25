# Remark42 [![Build Status](https://github.com/umputun/remark42/workflows/build/badge.svg)](https://github.com/umputun/remark42/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/umputun/remark42)](https://goreportcard.com/report/github.com/umputun/remark42) [![Coverage Status](https://coveralls.io/repos/github/umputun/remark42/badge.svg?branch=master)](https://coveralls.io/github/umputun/remark42?branch=master) [![codecov](https://codecov.io/gh/umputun/remark42/branch/master/graph/badge.svg)](https://app.codecov.io/gh/umputun/remark42)

Remark42 is a self-hosted, lightweight and simple (yet functional) comment engine, which doesn't spy on users. It can be embedded into blogs, articles, or any other place where readers add comments.

* Social login via Google, Twitter, Facebook, Microsoft, GitHub, Yandex, Patreon and Telegram
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
* Telegram, Slack, Webhook and email notifications for Admins (get notified for each new comment)
* Email and Telegram notifications for users (get notified when someone responds to your comment)
* Export data to JSON with automatic backups
* No external databases, everything embedded in a single data file
* Fully dockerized and can be deployed in a single command
* Self-contained executable can be deployed directly to Linux, Windows and macOS
* Clean, lightweight and customizable UI with white and dark themes
* Multi-site mode from a single instance
* Integration with automatic SSL (direct and via [nginx-le](https://github.com/nginx-le/nginx-le))
* [Privacy focused](#privacy)

[Demo site](https://remark42.com/demo/) available with all authentication methods, including email auth and anonymous access.

<details><summary>Screenshots</summary>

Comments example:
![](screenshots/comments.png)

For admin screenshots see [Admin UI wiki](https://github.com/umputun/remark42/wiki/Admin-UI)
</details>

All remark42 documentation is available [by the link](https://remark42.com/docs/).

#

  - [Install](#install)
      - [Quick installation test](#quick-installation-test)
      - [Backup format](#backup-format)
      - [Admin users](#admin-users)
      - [Docker parameters](#docker-parameters)
  - [Build from the source](#build-from-the-source)
  - [Privacy](#privacy)

## Install

#### Quick installation test

To verify if Remark42 has been properly installed, check a demo page at `${REMARK_URL}/web` URL. Make sure to include `remark` site ID to `${SITE}` list.

#### Backup format

The backup file is a text file with all exported comments separated by EOL. Each backup record is a valid JSON with all key/value unmarshaled from `Comment` struct (see below).

#### Admin users

Admins/moderators should be defined in `docker-compose.yml` as a list of user IDs or passed in the command line.

```yaml
environment:
  - ADMIN_SHARED_ID=github_ef0f706a79cc24b17bbbb374cd234a691a034128,github_dae9983158e9e5e127ef2b87a411ef13c891e9e5
```

To get a user ID just log in and click on your username or any other user you want to promote to admins. It will expand login info and show the full user ID.

#### Docker parameters

Two parameters allow customizing Docker container on the system level:

* `APP_UID` - sets UID to run Remark42 application in container (default=1001)
* `TIME_ZONE` - sets time zone of Remark42 container (default=America/Chicago)

_see [umputun/baseimage](https://github.com/umputun/baseimage) for more details_

Example of `docker-compose.yml`:

```yaml
version: '2'

services:
  remark42:
    image: umputun/remark42:latest
    restart: always
    container_name: "remark42"
    environment:
      - APP_UID=2000                          # runs Remark42 app with non-default UID
      - TIME_ZONE=GTC                         # sets container time to UTC
      - REMARK_URL=https://demo.remark42.com  # URL pointing to your Remark42 server
      - SITE=YOUR_SITE_ID                     # site ID, same as used for `site_id`, see "Setup on your website"
      - SECRET=abcd-123456-xyz-$%^&           # secret key
      - AUTH_GITHUB_CID=12345667890           # OAuth2 client ID
      - AUTH_GITHUB_CSEC=abcdefg12345678      # OAuth2 client secret
    volumes:
      - ./var:/srv/var                        # persistent volume to store all Remark42 data
```

## Build from the source

* to build Docker container - `make docker`. This command will produce container `umputun/remark42`
* to build a single binary for direct execution - `make OS=<linux|windows|darwin> ARCH=<amd64|386>`. This step will produce executable
  `remark42` file with everything embedded

## Privacy

* Remark42 is trying to be very sensitive to any private or semi-private information.
* Authentication requesting the minimal possible scope from authentication providers. All extra information returned by them is immediately dropped and not stored in any form.
* Generally, Remark42 keeps user ID, username and avatar link only. None of these fields exposed directly - ID and name hashed, avatar proxied.
* There is no tracking of any sort.
* Login mechanic uses JWT stored in a cookie (HttpOnly, secured). The second cookie (XSRF_TOKEN) is a random ID preventing CSRF.
* There is no cross-site login, i.e. user's behavior can't be analyzed across independent sites running Remark42.
* There are no third-party analytic services involved.
* User can request all information Remark42 knows about and export to gz file.
* Supported complete cleanup of all information related to user's activity.
* Cookie lifespan can be restricted to session-only.
* All potentially sensitive data stored by Remark42 hashed and encrypted.

## Related projects

* [A Helm chart for Remark42 on Kubernetes](https://github.com/groundhog2k/helm-charts/tree/master/charts/remark42)
* [django-remark42](https://github.com/andrewp-as-is/django-remark42.py)
