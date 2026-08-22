# Frontend gotchas

Non-obvious constraints in the frontend toolchain and widget. Read before bumping dependencies or the node/pnpm versions, and before changing how the widget renders or how translations are extracted.

## The node/pnpm version is pinned in many places, not one

CI staying green does **not** mean every pin is consistent — `.nvmrc` in particular is never read by CI, so it can silently drift. After changing the node or pnpm version, grep the whole repo and update every one of these, not just the ones CI exercises:

- `Dockerfile` (production image) — `FROM node:X-alpine` and `npm i -g pnpm@X.Y.Z`
- `frontend/.nvmrc` — not read by CI at all; only matters to a human running `nvm use` locally. This is the one that drifted unnoticed: it sat at `16` through the whole node-20 migration because nothing red ever pointed at it.
- `frontend/apps/remark42/package.json`, both its `packageManager` field and its `engines` block
- `pnpm/action-setup@vN` blocks in `.github/workflows/ci-frontend.yml` (5) and `release.yml` (2) — pin `version:` to the **exact** patch (e.g. `10.10.0`), matching `packageManager`, not just the major. A floating major here is silent in CI (it just resolves to whatever the latest patch is at run time) but breaks the "Dockerfile and CI use the same pnpm" guarantee.
- `node:` matrices in `.github/workflows/ci-frontend.yml` (every entry, not just the first) and the `node-version:` values in `release.yml`

When bumping pnpm/node, also re-check `frontend/apps/remark42/package.json`'s `engines` field — it's separate from `packageManager` and won't update itself.

`engines.node` states the major we support, currently `>=24`, which is the active LTS; 22 has dropped to maintenance. `@babel/core` 8 wants `^22.18 || >=24.11` and `size-limit` 13 wants `^22.18 || ^24 || >=26`, so 22 was the floor rather than the target. Individual dev dependencies can be stricter within that major (`undici` wants `>=20.18.1`); do not chase those patch floors into `engines` or the docs, or every lockfile refresh becomes a documentation change.

## One manifest, at `frontend/apps/remark42`

`frontend/` holds no `package.json`, no lockfile and no `pnpm-workspace.yaml`. Everything pnpm reads
lives in `frontend/apps/remark42`: the dependencies, `packageManager`, `engines` and the
`pnpm.overrides` block. Install and run from there, not from `frontend/`.

The directory nesting is kept because every path in the repository points at it, from the Dockerfile
and the workflows to the published contributing docs. `frontend/` still carries `.nvmrc`, `.husky`
and this file, none of which pnpm reads.

## pnpm 10's stricter `node-linker` layout needs explicit pins

One dep is pinned specifically because of pnpm 10's hoisting changes, not because of the dep itself:
- `@types/minimatch` 5.1.2 — 6.x is an empty stub that the hoisted layout picks up instead of the real types

If a dependency bump mysteriously breaks types or module resolution only after a pnpm major bump, suspect the layout change before suspecting the dependency.

## node's native fetch requires absolute URLs in tests

It rejects relative request URLs, and the failure is
silent: requests simply never match. Any test harness that mocks fetch needs absolute base URLs and
a jsdom base URL set.

## Babel is the only thing that compiles

`.babelrc.js` drives both the bundle, through `babel-loader`, and the tests, which get it as
`babel-jest`'s `configFile` in `jest.config.mjs`. That indirection is not decoration: `.babelrc.js`
is a file-relative config and would not reach `node_modules`, so the ESM-only packages in
`transformIgnorePatterns` would arrive at jest untransformed.

Babel merges an `env` block over the root rather than replacing it, so anything the test run must
not see has to be decided before the object is built rather than put in an `env`. `jest.config.mjs`
sets `BABEL_ENV` itself and `.babelrc.js` reads it for two things: the compile targets, and the
`data-testid` stripper, which the suites query by and the bundle must not carry. Verify with
`grep -c data-testid public/*.mjs` after a production build; it must be 0. The plugin matches a
literal `JSXAttribute`, which is every use in the tree; a spread such as
`{...{'data-testid': x}}` would reach production, and that grep is what would catch it.

`tsconfig.json` types but never emits, so its `target` has no effect on output; browserslist in
`.babelrc.js` decides that. Type checking runs out of band through
`fork-ts-checker-webpack-plugin`.

## Type-only imports have to say so

Babel compiles one file at a time with no type information, so it cannot tell that an import is
types-only. Left unmarked, the module and everything it imports stay in the bundle, and a single
type import of a store-connected component is enough to pull the whole redux graph into an entry
that never uses it.

`verbatimModuleSyntax` in `tsconfig.json` and `@typescript-eslint/consistent-type-imports` are what
enforce this, and the fix has to be a separate `import type { ... }` statement. Inline
`import { type X }` does **not** work here: verbatim semantics keep the statement, so the module is
still loaded. That is also why `no-duplicate-imports` runs with `allowSeparateTypeImports`, since a
module legitimately appears twice.

## JSX runs on the automatic runtime, in two places that must agree

`preact` 10.29 types a component's return as `ComponentChildren`, which only satisfies a JSX check on
TypeScript 5.1+ via `JSX.ElementType`, and it scopes the `JSX` namespace to `preact/jsx-runtime` rather
than declaring it globally. So the type layer has to use the automatic runtime:

- `tsconfig.json`: `jsx: react-jsx` with `jsxImportSource: preact`
- `.babelrc.js`: `@babel/preset-react` with `runtime: 'automatic'`, `importSource: 'preact'`

Keep both in step. If babel were left on the classic `pragma: 'h'` transform, a new `.tsx`
without `import { h }` would type-check and lint clean, then throw at runtime, because
`eslint-config-preact` sets `react/react-in-jsx-scope` to 0 and the local config turns `no-undef` off.

## `@babel/core` is pinned to 8 for the whole dependency tree

`@jest/transform` and `istanbul-lib-instrument` depend on `@babel/core` 7 outright, and a babel 8
preset loaded into a babel 7 core fails on the first `enum` it meets. The `pnpm.overrides` entry in
`frontend/apps/remark42/package.json` is what stops that. `eslint-config-preact` is the one consumer that cannot
take it: its `@babel/eslint-parser` loads babel 7 syntax plugins that babel 8 rejects, so a second
scoped override, `eslint-config-preact>@babel/core`, holds that subtree on 7.

## Held-back majors

These were deliberately not bumped because each is a config-migration or bundle-changing major, not a drop-in update — don't bump them opportunistically inside an unrelated dependency PR:

- `eslint` 10 (`eslint-config-preact` peers on ^9), `typescript` 7, `redux` 5 with `redux-thunk` 3, `node-emoji` 2 (renames the field `search()` returns), `@formatjs/cli` 6 (the translation extraction pipeline), and `postcss-preset-env` 11 with `cssnano` 8 and `postcss-html` 2, which change generated CSS
- `redux` 4

## `/web` has a second source

`privacy.html`, `markdown-help.html` and `400x400.jpeg` live in `backend/app/webassets/assets` and
are embedded in the Go binary. The frontend build wins for any name present in both, which is what
lets an operator override one by dropping a file next to the frontend files in `--web-root`. That
directory replaces the embedded frontend outright when it exists, so an override belongs in a
populated one. These three sit outside this toolchain: prettier, stylelint and `pnpm lint` do not
see them, and they are served unminified. `devServer.static` lists the build output first and that
directory second, matching the backend's order, so links to them resolve on the dev port too.

## eslint config resolves from the process cwd

`eslint.config.mjs` lives in `apps/remark42`, and eslint loads the config next to the directory it
is *run from* rather than the one nearest the file being linted. Anything that invokes eslint has
to have `apps/remark42` as its working directory: it is where every script runs, `.husky/pre-commit`
cd's into it, and an IDE integration needs
`"eslint.workingDirectories": ["frontend/apps/remark42"]`.

Rules that only exist under a plugin's flat-config export are spread in explicitly; the block that
re-enables project rules is the last entry in the array, because `eslint-config-prettier` (pulled in
by `prettierRecommended`) switches several of them off and later entries win.

## Verifying a build didn't regress

There's no automated build-output diff in CI. Before merging a dependency PR that touches the bundler/build tooling, manually diff the build output against a clean `master` checkout:
- `apps/remark42`: expect webpack module-id numbers and css-module class tokens (e.g. `.F_A` → `.L_A`) to differ — that's normal churn from a webpack/css-loader bump. HTML, CSS values, and translation content should be byte-identical.
- `site`: asset URLs carry a content hash, so a stylesheet or script change moves the filename on every page referencing it. Anything else differing is a real regression.

## Where the alerts actually were

When clearing Dependabot/audit alerts, check whether the flagged package is actually reachable from production code or only from the dev/test toolchain — `pnpm audit` does not distinguish. Several alerts here were in build-time-only tooling (webpack-dev-server, laravel-mix-equivalent dev deps) with no patched release available; those are lower-risk than a runtime dependency with the same severity label.

## Don't import `preact/compat`

It installs hooks on preact's shared `options` that remap `onFocus`/`onBlur` to
`focusin`/`focusout` on every element and make `@testing-library/preact` rewrite `change` to
`input`, so one import changes event behaviour across the whole widget, and it adds about
3.8 kB gzipped.

`forwardRef` lives only there, so a component that needs a ref takes it as an ordinary prop;
`TextareaAutosize` is the pattern. For a component type annotation use `FunctionComponent`
from `preact`, not `React.FC`.

## i18n is a hand-written binding whose export names are fixed by the extractor

`app/common/intl.tsx` provides `IntlProvider`, `useIntl`, `createIntl`, `defineMessages`,
`FormattedMessage` and `IntlShape`. `formatjs extract` (`translation:extract`) finds messages
by recognising the identifiers `defineMessages`, `FormattedMessage` and `intl.formatMessage`
in the AST, not by import source, so those three names are fixed. Rename any of them and
extraction returns nothing, with no error and a zero exit code.

The damage lands on the next step. `translation:check` compares locale keys against extracted
keys and fails loudly, but `tasks/generateDictionary.js` calls `removeAbandonedKeys`, so
`translation:generate` after a rename deletes the now-unextracted keys from all 24 files in
`app/locales/` and writes them back, after which the check passes. Extract-then-generate is
the documented translator workflow, so the destructive order is the normal one.

After any rename or refactor touching those identifiers, verify the count rather than the exit
code:

```sh
pnpm translation:extract
node -e "console.log(Object.keys(require('./extracted-messages/messages')).length)"
```

It must match the key count in `app/locales/en.json`, not drop to 0.

The binding implements `{name}` interpolation and paired `<tag>text</tag>` rich text, and
nothing else of ICU: no plural, select or selectordinal, no typed arguments such as
`{n, number}`, and no apostrophe quoting. A plural or select form has braces the placeholder
rule rejects, so it falls back to English wherever values are passed and shows its raw text
where they are not; `''` simply stays as two apostrophes.

`translation:check` validates every translated value's markup and placeholders against the
English string it translates, mirroring the binding's rule, and the catalogue sweep in
`app/common/intl.test.tsx` additionally renders the two messages that carry a link. What
neither catches is unsupported ICU syntax, since a plural form is well-formed text as far as
both are concerned. A message the binding cannot resolve falls back to the English source
rather than reaching the page, so without these checks a broken translation is invisible in
the interface.
