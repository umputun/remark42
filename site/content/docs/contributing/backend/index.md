---
title: Backend Development Guidelines
aliases:
  - /docs/contributing/
---

You can use a fully functional local version to develop and test both frontend and backend. It requires at least 2GB RAM or swap enabled.

To bring it up, run:

```shell
cp compose-dev-backend.yml compose-private.yml
# now, edit / debug `compose-private.yml` to your heart's content

# build and run
docker compose -f compose-private.yml up --build
```

It starts Remark42 on `127.0.0.1:8080` and adds local OAuth2 provider "Dev". To access the UI demo page go to <http://127.0.0.1:8080/web/>. Sign in with the Dev provider: its form offers `dev_user`, which the compose file makes an admin. You can tweak any of the [supported parameters](https://remark42.com/docs/configuration/parameters/) in the corresponding yml file.

> 🚨 Please use `127.0.0.1` and not `localhost` to access the server; otherwise, CORS will prevent your browser from authentication to work correctly. You could alter the address for dev auth with the `REMARK_URL` environment variable.

Backend Docker Compose config (`compose-dev-backend.yml`) by default skips running frontend related tests. Frontend Docker Compose config (`compose-dev-frontend.yml`) skips backend tests and the frontend build. Run the frontend separately with `pnpm dev` as described in the [frontend development guide](https://remark42.com/docs/contributing/frontend/#run-frontend-with-backend-locally).

## Backend development

### With Docker

Run tests in your IDE, and re-run `make rundev` each time you want to see how your code changes behave to test them at <http://127.0.0.1:8080/web/>.

### Without Docker

You have to [install](https://golang.org/doc/install) the latest stable `go` toolchain to run the backend locally.

In order to have working Remark42 installation you need once to copy frontend static files to `./backend/app/cmd/web` directory from `master` docker image, as it is expected to be where application compiles:

```shell
# frontend files
docker pull ghcr.io/umputun/remark42:master
docker create -ti --name remark42files ghcr.io/umputun/remark42:master sh
docker cp remark42files:/srv/web/ ./backend/app/cmd/
docker rm -f remark42files
```

The assets under `/web` the frontend does not build (`privacy.html`, `markdown-help.html`, `400x400.jpeg`, the ghost avatar, the sign-in provider icons, the emoji table `emoji.json`, the message catalogues `<locale>.json`, the widget document `iframe.html`, the demo, counter, last-comments, deleteme and comments pages and the `counter.mjs` script) come from `backend/app/webassets/assets` and are embedded in the binary, so the copy above neither covers them nor needs to. At runtime a file of the same name under `web-root` / `REMARK_WEB_ROOT` is served in preference to the embedded one.

To run backend - `cd backend; go run app/main.go server --dbg --secret=12345 --url=http://127.0.0.1:8080 --admin-passwd=password --site=remark`. It starts backend service with embedded bolt store on port `8080` with basic auth, allowing to authenticate and run requests directly, like this:

`curl "http://admin:password@127.0.0.1:8080/api/v1/find?site=remark&sort=-active&format=tree&url=http://127.0.0.1:8080"`

## Technical Details

Data stored in [boltdb](https://github.com/etcd-io/bbolt) (embedded key/value database) files under `STORE_BOLT_PATH`. Each site is stored in a separate boltdb file.

To migrate/move Remark42 to another host, these boltdb files must be transferred. Additionally, the avatars directory `AVATAR_FS_PATH` (default `./var/avatars`) and images directory `IMAGE_FS_PATH` (default `./var/pictures`) must be transferred. As an alternative to storing avatars and images in the file system, boltdb can be used by setting `AVATAR_TYPE` or `IMAGE_TYPE` to `bolt`. For each store set to `bolt`, transfer its file instead of its directory: `AVATAR_BOLT_FILE` (default `./var/avatars.db`) for avatars, `IMAGE_BOLT_FILE` (default `./var/pictures.db`) for images. With images in the file system, also transfer the `IMAGE_FS_STAGING` directory (default `./var/pictures.staging`), which can hold pictures of recently posted or edited comments.

The automatic backup process runs at start for a site with no backup of the current day, then every 24h, and exports all content in JSON-like format to `backup-{site ID}-{YYYYMMDD}.gz`.

Authentication is implemented with [go-pkgz/auth](https://github.com/go-pkgz/auth), which keeps the session token in the `JWT` cookie. That cookie is HttpOnly. The `XSRF-TOKEN` cookie is not HttpOnly, because the widget reads it to send the `X-XSRF-TOKEN` header. Both cookies are `Secure` when `REMARK_URL` starts with `https://`.

Heavy REST calls are cached in an in-memory LRU cache from [go-pkgz/lcw](https://github.com/go-pkgz/lcw), limited by `CACHE_MAX_ITEMS`, `CACHE_MAX_VALUE` and `CACHE_MAX_SIZE`.

User activity is throttled globally to 1000 simultaneous requests. Each group of routes also has its own rate limit, counted separately for every client address and URL path. The limit is 2 requests a second under `/auth/`. Posting, editing and previewing comments, voting, subscriptions, deletion requests and image uploads are limited by `UPDATE_LIMIT`, 0.5 by default. The other API routes allow 10 requests a second, `/web/` allows 20, `robots.txt` and the email unsubscribe page allow 50, and avatars allow 100.

Request timeouts depend on the route. Requests under `/auth/` and for avatars time out after 5 seconds. The routes `UPDATE_LIMIT` covers, `/web/`, `robots.txt` and the email unsubscribe page time out after 10 seconds, and the other API routes after 30 seconds. User data export, site export, import, remap and the admin `wait` call have no timeout, because they stream large bodies or wait for a long time.

Setting `--admin-passwd` (`ADMIN_PASSWD`) adds Basic auth with the username `admin` and that password. It lets you call the Remark42 API without a social login, as an admin of every site. The server refuses such a request when a browser marks it as cross-site or sends an `Origin` that differs from the origin of `REMARK_URL`; command-line tools send neither header. Enable it only for the initial comment import or for manual backups. Do not leave the server running with admin password set if you don't have an intention to keep creating backups manually!

User can vote for the comment multiple times but only to change the vote. Double voting is not allowed.

User can edit comments in 5 mins (configurable) window after creation.

User ID hashed and prefixed by OAuth provider name to avoid collisions and potential abuse.

Avatars are fetched from the provider at sign-in and saved in the avatar store, which serves every later request, so the provider is not contacted again; this is part of [go-pkgz/auth](https://github.com/go-pkgz/auth). They are scaled down only when `AVATAR_RESIZE` is set and the picture is larger. With the default `0`, they are stored as received.

Images served over HTTP can be proxied to HTTPS (`IMAGE_PROXY_HTTP2HTTPS=true`) to prevent mixed HTTP/HTTPS.

All images can be proxied and saved locally (`IMAGE_PROXY_CACHE_EXTERNAL=true`) instead of serving from the original location. Beware, images that are posted with this parameter enabled will be served from proxy even after it is disabled.

Docker build uses [publicly available](https://github.com/umputun/baseimage) base images.
