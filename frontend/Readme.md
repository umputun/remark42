# Frontend guide

### Code Style

- project uses typescript to statically analyze code
- project uses `eslint` to check frontend code. You can manually run via `npm run lint`.
- git hooks (via husky) installed automatically on `npm install` and check and try to fix code style if possible, otherwise commit will be rejected
- if you want IDE integration, you need `eslint` plugin to be installed.

### CSS Styles

- although styles have `scss` extension, it is actually pack of post-css plugins, so syntax differs, for example in `calc` function.
- component styles use BEM notation (at least it should): `block__element_modifier`. Also there are `mix` classes: `block_modifier`.
- component base style resides in the component's root directory with name of component converted to kebab-case. For example `ListComments` style is located in `./app/components/list-comments/list-comments/scss`
- component's element style resides in its own subdirectory, with name consisting of full elements selector, for example `ListComments` `item` element is placed in `__item` directory under name `./list-comments__item.scss`
- each style should be `require`d in `index.ts` of component's root directory

### Imports

- imports for typescript, javascript files should be without extension: `./index`, not `./index.ts`
- if file resides in same directory or in subdirectory import should be relative: `./types/something`
- otherwise it should start from `@app` namespace: `@app/common/store` which mapped to `/app/common/store.ts` in webpack, tsconfig and jest

### Testing

- project uses `jest` as test harness.
- jest check files that match regex `\.test\.(j|t)s(x?)$`, i.e `comment.test.tsx`, `comment.test.js`
- tests are running on push attempt
- example tests can be found in `./app/store/user/reducers.test.ts`, `./app/components/auth-panel/auth-panel.test.tsx`

### how to add new locale.

- add new item to `./tasks/supportedLocales.json`
- run `npm run generate-langs`
- commit all changed files
- translate all string in new generated dictionary `./app/locale/<new-locale>.json`

### Notes

- Frontend part being bundled on docker env gets placed on `/src/web` and is available via `http://{host}/web`. for example `embed.js` entry point will be available at `http://{host}/web/embed.js`
