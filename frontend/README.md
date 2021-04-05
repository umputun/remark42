# Frontend guide
### Code Style

- project uses typescript to statically analyze code
- project uses `eslint` and `stylelint` to check frontend code. You can manually run via `npm run lint`.
- git hooks (via husky) installed automatically on `npm install` and check and try to fix code style if possible, otherwise commit will be rejected
- if you want IDE integration, you need `eslint` and `stylelint` plugin to be installed.

### CSS Styles

- now we are migrating to css-modules and this is recomended way to stylization. A file with styles should be named like `component.module.css`
- old component styles use BEM notation (at least it should): `block__element_modifier`. Also there are `mix` classes: `block_modifier`.
- new way to naming CSS selectors is camel-case like `blockElemenModifier` and use `classnames` to combine it
- component base style resides in the component's root directory with name of component converted to kebab-case. For example `ListComments` style is located in `./app/components/list-comments/list-component.tsx`
- any other files should be named also in kebab-case. For example `./app/utils/get-param.ts`

### Imports

- imports for typescript, javascript files should be without extension: `./index`, not `./index.ts`
- if file resides in the same directory or in subdirectory import should be relative: `./types/something`
- otherwise it should be imported by absolute path relative to `src` folder like `common/store` which mapped to `./app/common/store.ts` in webpack, tsconfig and jest

### Testing

- project uses `jest` as test harness.
- jest check files that match regex `\.(test|spec)\.ts(x?)$`, i.e `comment.test.tsx`, `comment.spec.ts`
- tests are running on push attempt
- example tests can be found in `./app/store/user/reducers.test.ts`, `./app/components/auth-panel/auth-panel.test.tsx`

### How to add new locale

Please see [this documentation](https://github.com/umputun/remark42/blob/master/docs/latest/translation.md).

### Notes

- Frontend part being bundled on docker env gets placed on `/src/web` and is available via `http://{host}/web`. for example `embed.js` entry point will be available at `http://{host}/web/embed.js`
