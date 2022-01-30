---
title: Frontend Development Guidelines
---

### Frontend development

#### Development tools installation

You must have at least 2GB RAM or swap enabled for building.

- install [Node.js 12.22](https://nodejs.org/en/) or higher
- install [NPM 6.14.16](https://www.npmjs.com/package/npm)
- run `npm install` inside `./frontend`

Running `npm install` will set up pre-commit hooks into your git repository. They are used to reformat your frontend code using `prettier` and lint with `eslint` and `stylelint` before every commit.

**Important**: use `127.0.0.1` and not `localhost` to access the server, as otherwise, CORS will prevent your browser from authentication to work correctly.

#### Development against demo.remark42.com

This variant of running Remark42 frontend code is preferred when you make a translation or visual adjustments that are easy to see without extensive testing.

For local development mode with Hot Reloading, use `npm start`. In this case, `webpack` will serve files using `webpack-dev-server` on `127.0.0.1:9000`. By visiting <http://127.0.0.1:9000/web/>, you will get a page with the main comments' widget communicating with a demo server backend running on `https://demo.remark42.com`. But you will not be able to log in with any OAuth providers due to security reasons.

You can attach the frontend to the locally running backend by providing the `REMARK_URL` environment variable.

```shell
npx cross-env REMARK_URL=http://127.0.0.1:8080 npm start
```

**Note:** If you want to redefine env variables such as `PORT` on your local instance, you can add the `.env` file to the `./frontend` folder and rewrite variables as you wish. For such functional, we use `dotenv`.

#### Development against the local backend

This variant of running Remark42 frontend code is preferred when you need extensive testing of your code changes, as you'll have your backend and configure it as you want, for example, enable any auth and notifications method you need to test. You can use that set up to develop and test both frontend and backend.

To bring the backend up, run:

```shell
cp compose-dev-frontend.yml compose-private-frontend.yml
# now, edit / debug `compose-private.yml` to your heart's content

# build and run
make rundev
# this is an equivalent of these two commands:
# docker-compose -f compose-private.yml build
# docker-compose -f compose-private.yml up
```

Then in the new terminal tab or window, run the following to start the frontend with Hot Reloading:

```shell
cd frontend
npm run dev
```

Developer build running by `webpack-dev-server` supports devtools for [React](https://reactjs.org/blog/2019/08/15/new-react-devtools.html#how-do-i-get-the-new-devtools) and [Redux](https://github.com/reduxjs/redux-devtools).

It starts Remark42 backend on `127.0.0.1:8080` and adds local OAuth2 provider "Dev". To access the frontend running by Node, go to <http://127.0.0.1:9000/web/>. By default, you would be logged in as `dev_user`, defined as admin. You can tweak any of the [supported parameters](https://remark42.com/docs/configuration/parameters/) in corresponded yml file.

Frontend Docker Compose config (`compose-dev-frontend.yml`) by default skips running backend related tests and sets `NODE_ENV=development` for frontend build.

**Important**: Before submitting your changes as a Pull Request, re-run the backend using the `make rundev` command and test your changes against <http://127.0.0.1:8080/web/>, frontend, built statically (unlike frontend on port 9000, which runs dynamically). That is how Remark42 authors will test your changes once you submit them.

#### Static build

Remark42 frontend can be built statically, and that's how the production version works: frontend is built and then resulting files embedded into the backend, which serves them as-is. Node is not running when a user starts remark42, only the backend written in Go programming language, which also serves pre-built frontend HTML and JS and CSS files.

Run `npm run build` inside `./frontend`, and result files will be saved in `./frontend/public`.

### Code Style

- The project uses TypeScript to analyze code statically
- The project uses `eslint` and `stylelint` to check the frontend code. You can manually run via `npm run lint`
- Git Hooks (via husky) installed automatically on `npm install`. They check and try to fix code style if possible, otherwise commit will be rejected
- If you want IDE integration, you need `eslint` and `stylelint` plugin to be installed

### CSS Styles

- Now we are migrating to CSS Modules, which is a recommended way of stylization. A file with styles should be named like `component.module.css`
- Old component styles use BEM notation (at least it should): `block__element_modifier`. Also, there are `mix` classes: `block_modifier`
- The new way to name CSS selectors is camel-case like `blockElemenModifier` and use `classnames` to combine it
- Component base style resides in the component's root directory with a name of component converted to kebab-case. For example, `ListComments` style is located in `./app/components/list-comments/list-component.tsx`
- Any other files should also be named in kebab-case. For example, `./app/utils/get-param.ts`

### Imports

- Imports for TypeScript, JavaScript files should be without extension: `./index`, not `./index.ts`
- If the file resides in the same directory or subdirectory, the import should be relative: `./types/something`
- Otherwise, it should be imported by absolute path relative to `src` folder like `common/store` which mapped to `./app/common/store.ts` in webpack, tsconfig, and Jest

### Testing

- Project uses `jest` as test harness
- Jest checks files that match regex `\.(test|spec)\.ts(x?)$`, i.e., `comment.test.tsx`, `comment.spec.ts`
- Tests are running on push attempt
- Example tests can be found in `./app/store/user/reducers.test.ts`, `./app/components/auth-panel/auth-panel.test.tsx`

### How to add a new locale

Please see [this documentation](https://remark42.com/docs/contributing/translations/).

### Notes

Frontend part being bundled on docker env gets placed on `/src/web` and is available via `http://{host}/web`. For example, `embed.js` entry point will be available at `http://{host}/web/embed.js`
