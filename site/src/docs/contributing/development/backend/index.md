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

In order to have working Remark42 installation you need once to copy frontend static files to `./backend/web` directory from `master` docker image, and also copy files from `./templates` to the `./backend` as they are expected to be where application starts:

```shell
# frontend files
docker pull umputun/remark42:master
docker create -ti --name remark42files umputun/remark42:master sh
docker cp remark42files:/srv/web/ ./backend/
docker rm -f remark42files
# template files
cp ./backend/templates/* ./backend
# fix frontend files to point to the right URL
## Mac version
find -E ./backend/web -regex '.*\.(html|js|mjs)$' -print -exec sed -i '' "s|{% REMARK_URL %}|http://127.0.0.1:8080|g" {} \;
## Linux version
find ./backend/web -regex '.*\.\(html\|js\|mjs\)$' -print -exec sed -i "s|{% REMARK_URL %}|http://127.0.0.1:8080|g" {} \;
```

To run backend - `cd backend; go run app/main.go server --dbg --secret=12345 --url=http://127.0.0.1:8080 --admin-passwd=password --site=remark`. It stars backend service with embedded bolt store on port `8080` with basic auth, allowing to authenticate and run requests directly, like this:

`HTTP http://admin:password@127.0.0.1:8080/api/v1/find?site=remark&sort=-active&format=tree&url=http://127.0.0.1:8080`
