# Frontend gotchas

Non-obvious constraints in the frontend toolchain and widget. Read before bumping dependencies or the node/pnpm versions, and before changing how the widget renders or how translations are extracted.

## The node/pnpm version is pinned in many places, not one

CI staying green does **not** mean every pin is consistent — `.nvmrc` in particular is never read by CI, so it can silently drift. After changing the node or pnpm version, grep the whole repo and update every one of these, not just the ones CI exercises:

- `Dockerfile` (production image) — `FROM node:X-alpine` and `npm i -g pnpm@X.Y.Z`
- `frontend/Dockerfile.e2e` — `FROM mcr.microsoft.com/playwright:vX.Y.Z-noble` **and** `corepack prepare pnpm@X.Y.Z`
- `site/Dockerfile`, `site/Dockerfile.dev` — `FROM node:X-alpine` (site uses yarn, not pnpm)
- `frontend/.nvmrc`, `site/.nvmrc` — not read by CI at all; only matters to a human running `nvm use` locally. This is the one that drifted unnoticed: it sat at `16` through the whole node-20 migration because nothing red ever pointed at it.
- Every `package.json`'s `packageManager` field (`frontend/package.json`, `frontend/apps/remark42/package.json` — `frontend/e2e/package.json` has none) and `frontend/apps/remark42/package.json`'s `engines` block
- `pnpm/action-setup@vN` blocks in `.github/workflows/ci-frontend.yml` (5) and `release.yml` (2) — pin `version:` to the **exact** patch (e.g. `10.10.0`), matching `packageManager`, not just the major. A floating major here is silent in CI (it just resolves to whatever the latest patch is at run time) but breaks the "Dockerfile and CI use the same pnpm" guarantee.
- `node:` matrices in `.github/workflows/ci-frontend.yml` (every entry, not just the first) and the `node-version:` values in `release.yml`
- `site/package.json`'s `engines.node` and `engines.yarn` (site uses yarn, so its `packageManager` moves independently)
- `frontend/e2e/package.json`'s `@playwright/test`/`playwright` versions must match `frontend/Dockerfile.e2e`'s base image tag exactly, or the e2e container's bundled browser revision mismatches what the npm package expects.

When bumping pnpm/node, also re-check `frontend/apps/remark42/package.json`'s `engines` field — it's separate from `packageManager` and won't update itself.

`engines.node` states the major we support, currently `>=20`, and the docs say the same. Individual dev dependencies can be stricter within that major (`undici` wants `>=20.18.1`), which any current Node 20 satisfies; do not chase those patch floors into `engines` or the docs, or every lockfile refresh becomes a documentation change.

## pnpm 10's stricter `node-linker` layout needs explicit pins

One dep is pinned specifically because of pnpm 10's hoisting changes, not because of the dep itself:
- `@types/minimatch` 5.1.2 — 6.x is an empty stub that the hoisted layout picks up instead of the real types

If a dependency bump mysteriously breaks types or module resolution only after a pnpm major bump, suspect the layout change before suspecting the dependency.

## node 20's native fetch requires absolute URLs in tests

Unlike the polyfilled fetch in node 16 and 18, it rejects relative request URLs, and the failure is
silent: requests simply never match. Any test harness that mocks fetch needs absolute base URLs and
a jsdom base URL set.

## JSX runs on the automatic runtime, in three places that must agree

`preact` 10.29 types a component's return as `ComponentChildren`, which only satisfies a JSX check on
TypeScript 5.1+ via `JSX.ElementType`, and it scopes the `JSX` namespace to `preact/jsx-runtime` rather
than declaring it globally. So the type layer has to use the automatic runtime:

- `tsconfig.json`: `jsx: react-jsx` with `jsxImportSource: preact`
- `.babelrc.js`: `@babel/preset-react` with `runtime: 'automatic'`, `importSource: 'preact'`
- `jest.config.ts`: `@swc/jest` with `transform.react.runtime: 'automatic'`, `importSource: 'preact'`

`ts-loader` in `webpack.config.js` overrides `jsx` back to `preserve`. That is deliberate: JSX has to
survive as JSX until babel runs, or `babel-plugin-jsx-remove-data-test-id` has nothing to strip and
`data-testid` attributes ship to production. Verify with `grep -c data-testid public/*.mjs` after a
production build; it must be 0.

Keep all three in step. If babel alone were left on the classic `pragma: 'h'` transform, a new `.tsx`
without `import { h }` would type-check and lint clean, then throw at runtime, because
`eslint-config-preact` sets `react/react-in-jsx-scope` to 0 and the local config turns `no-undef` off.

## Held-back majors

These were deliberately not bumped because each is a config-migration or bundle-changing major, not a drop-in update — don't bump them opportunistically inside an unrelated dependency PR:

- `eslint` 8 (9/10 need flat-config migration), `stylelint` 14 (16 has breaking rule changes), `babel` 7, `jest` 28 (30 needs config changes)
- `redux` 4, `tailwindcss` 3.4 (v4 is a full config rewrite), `@11ty/eleventy` 2 (v3 is an ESM migration) in `site/`

## `html-minifier` is abandoned — use `html-minifier-terser`

`site/.eleventy.js` uses `html-minifier-terser` (a maintained fork), not `html-minifier` (unpatched ReDoS advisory, no fix ever released). The eleventy transform had to become `async` for this fork's API.

## Verifying a build didn't regress

There's no automated build-output diff in CI. Before merging a dependency PR that touches the bundler/build tooling, manually diff the build output against a clean `master` checkout:
- `apps/remark42`: expect webpack module-id numbers and css-module class tokens (e.g. `.F_A` → `.L_A`) to differ — that's normal churn from a webpack/css-loader bump. HTML, CSS values, and translation content should be byte-identical.
- `site`: expect the `?v=<timestamp>` cache-bust query string to differ on every HTML file — that's expected. Anything else differing is a real regression.

## Where the alerts actually were

When clearing Dependabot/audit alerts, check whether the flagged package is actually reachable from production code or only from the dev/test toolchain — `pnpm audit`/`yarn audit` don't distinguish. Several alerts here were in build-time-only tooling (webpack-dev-server, laravel-mix-equivalent dev deps) with no patched release available; those are lower-risk than a runtime dependency with the same severity label.

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
