---
title: Backend developer guide
---

#  Backend developer guide

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
