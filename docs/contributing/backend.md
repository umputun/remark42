---
title: Backend Development Guidelines
---

You can use a fully functional local version to develop and test both frontend and backend. It requires at least 2GB RAM or swap enabled.

To bring it up, run:

```shell
cp compose-dev-backend.yml compose-private.yml
# now, edit / debug `compose-private.yml` to your heart's content

# build and run
docker-compose -f compose-private.yml up --build
```

It starts Remark42 on `127.0.0.1:8080` and adds local OAuth2 provider "Dev". To access the UI demo page go to <http://127.0.0.1:8080/web/>. By default, you would be logged in as `dev_user`, defined as admin. You can tweak any of the [supported parameters](https://remark42.com/docs/configuration/parameters/) in corresponded yml file.

**Important**: use `127.0.0.1` and not `localhost` to access the server, as otherwise, CORS will prevent your browser from authentication to work correctly.

Backend Docker Compose config (`compose-dev-backend.yml`) by default skips running frontend related tests. Frontend Docker Compose config (`compose-dev-frontend.yml`) by default skips running backend related tests and sets `NODE_ENV=development` for frontend build.

### Backend development

#### With Docker

Run tests in your IDE, and re-run `make rundev` each time you want to see how your code changes behave to test them at <http://127.0.0.1:8080/web/>.

#### Without Docker

You have to [install](https://golang.org/doc/install) the latest stable `go` toolchain to run the backend locally.

In order to have working Remark42 installation you need once to copy frontend static files to `./backend/web` directory from `master` docker image, as it is expected to be where application compiles:

```shell
# frontend files
docker pull umputun/remark42:master
docker create -ti --name remark42files umputun/remark42:master sh
docker cp remark42files:/srv/web/ ./backend/app/cmd/
docker rm -f remark42files
# fix frontend files to point to the right URL
## Mac version
find -E ./backend/app/cmd/web -regex '.*\.(html|js|mjs)$' -print -exec sed -i '' "s|{% REMARK_URL %}|http://127.0.0.1:8080|g" {} \;
## Linux version
find ./backend/app/cmd/web -regex '.*\.\(html\|js\|mjs\)$' -print -exec sed -i "s|{% REMARK_URL %}|http://127.0.0.1:8080|g" {} \;
```

To run backend - `cd backend; go run app/main.go server --dbg --secret=12345 --url=http://127.0.0.1:8080 --admin-passwd=password --site=remark`. It stars backend service with embedded bolt store on port `8080` with basic auth, allowing to authenticate and run requests directly, like this:

`HTTP http://admin:password@127.0.0.1:8080/api/v1/find?site=remark&sort=-active&format=tree&url=http://127.0.0.1:8080`

## Technical Details

Data stored in [boltdb](https://github.com/etcd-io/bbolt) (embedded key/value database) files under `STORE_BOLT_PATH`. Each site is stored in a separate boltdb file.

To migrate/move Remark42 to another host, boltdb files and avatars directory `AVATAR_FS_PATH` should be transferred. Optionally, boltdb can be used to store avatars as well.

The automatic backup process runs every 24h and exports all content in JSON-like format to `backup-remark-YYYYMMDD.gz`.

Authentication implemented with [go-pkgz/auth](https://github.com/go-pkgz/auth) stored in a cookie. It uses HttpOnly, secure cookies.

All heavy REST calls cached internally in LRU cache limited by `CACHE_MAX_ITEMS` and `CACHE_MAX_SIZE` with [go-pkgz/rest](https://github.com/go-pkgz/rest).

User's activity throttled globally (up to 1000 simultaneous requests) and limited locally (per user, usually up to 10 req/sec).

Request timeout set to 60sec.

Admin authentication (`--admin-password` set) allows to hit Remark42 API without social login and admin privileges. Adds basic-auth for username: `admin`, password: `${ADMIN_PASSWD}`. Enable it only for the initial comment import or for manual backups. Do not leave server running with admin password set if you don't have intention to keep creating backups manually!

User can vote for the comment multiple times but only to change the vote. Double voting is not allowed.

User can edit comments in 5 mins (configurable) window after creation.

User ID hashed and prefixed by OAuth provider name to avoid collisions and potential abuse.

All avatars resized and cached locally to prevent rate limiters from OAuth providers, part of [go-pkgz/auth](https://github.com/go-pkgz/auth) functionality.

Images served over HTTP can be proxied to HTTPS (`IMAGE_PROXY_HTTP2HTTPS=true`) to prevent mixed HTTP/HTTPS.

All images can be proxied and saved locally (`IMAGE_PROXY_CACHE_EXTERNAL=true`) instead of serving from the original location. Beware, images that are posted with this parameter enabled will be served from proxy even after it is disabled.

Docker build uses [publicly available](https://github.com/umputun/baseimage) base images.
