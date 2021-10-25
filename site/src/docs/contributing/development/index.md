---
title: Development
---

You can use a fully functional local version to develop and test both frontend and backend. It requires at least 2GB RAM or swap enabled.

To bring it up run:

```shell
# if you mainly work on backend
cp compose-dev-backend.yml compose-private.yml
# if you mainly work on frontend
cp compose-dev-frontend.yml compose-private.yml
# now, edit / debug `compose-private.yml` to your heart's content

# build and run
docker-compose -f compose-private.yml build
docker-compose -f compose-private.yml up
```

It starts Remark42 on `127.0.0.1:8080` and adds local OAuth2 provider "Dev". To access the UI demo page go to `127.0.0.1:8080/web`. By default, you would be logged in as `dev_user` which is defined as admin. You can tweak any of [supported parameters](#parameters) in corresponded yml file.

Backend Docker Compose config by default skips running frontend related tests. Frontend Docker Compose config by default skips running backend related tests and sets `NODE_ENV=development` for frontend build.

### Backend development

To run backend locally (development mode, without Docker) you have to have the latest stable `go` toolchain [installed](https://golang.org/doc/install).

To run backend - `cd backend; go run app/main.go server --dbg --secret=12345 --url=http://127.0.0.1:8080 --admin-passwd=password --site=remark`. It stars backend service with embedded bolt store on port `8080` with basic auth, allowing to authenticate and run requests directly, like this:

`HTTP http://admin:password@127.0.0.1:8080/api/v1/find?site=remark&sort=-active&format=tree&url=http://127.0.0.1:8080`

### Frontend development

Frontend development guide can be found [here](https://remark42.com/docs/contributing/development/frontend/).

#### Build

You should have at least 2GB RAM or swap enabled for building.

- install [Node.js 12.11](https://nodejs.org/en/) or higher
- install [NPM 6.13.4](https://www.npmjs.com/package/npm)
- run `npm install` inside `./frontend`
- run `npm run build` there
- result files will be saved in `./frontend/public`

**Note:** Running `npm install` will set up pre-commit hooks into your git repository. It used to reformat your frontend code using `prettier` and lint with `eslint` and `stylelint` before every commit.

#### Development server

For local development mode with Hot Reloading use `npm start` instead of `npm run build`. In this case, `webpack` will serve files using `webpack-dev-server` on `localhost:9000`. By visiting `127.0.0.1:9000/web` you will get a page with the main comments widget communicating with a demo server backend running on `https://demo.remark42.com`. But you will not be able to log in with any OAuth providers due to security reasons.

You can attach to the locally running backend by providing `REMARK_URL` environment variable.

```shell
npx cross-env REMARK_URL=http://127.0.0.1:8080 npm start
```

**Note:** If you want to redefine env variables such as `PORT` on your local instance you can add `.env` file to `./frontend` folder and rewrite variables as you wish. For such functional, we use `dotenv`.

The best way to start a local developer environment:

```shell
cp compose-dev-frontend.yml compose-private-frontend.yml
docker-compose -f compose-private-frontend.yml up --build
cd frontend
npm run dev
```

Developer build running by `webpack-dev-server` supports devtools for [React](https://github.com/facebook/react-devtools) and
[Redux](https://github.com/zalmoxisus/redux-devtools-extension).
