# Frontend gotchas

Non-obvious constraints in the frontend toolchain and widget. Read before bumping dependencies or the node/pnpm versions, and before changing how the widget renders or how translations are extracted.

## The node/pnpm version is pinned in many places, not one

CI staying green does **not** mean every pin is consistent: CI compares only `.nvmrc` and the installation page against `package.json` (`scripts/check-documented-versions.sh`, run by `ci-docs-versions.yml`), and the other places below can silently drift. After changing the node or pnpm version, grep the whole repo and update every one of these, not just the ones CI exercises:

- `Dockerfile` (production image) — `FROM node:X-alpine` and `npm i -g pnpm@X.Y.Z`
- `frontend/.nvmrc` — what a human running `nvm use` gets. It sat at `16` through the whole node-20 migration because nothing red pointed at it; `check-documented-versions.sh` now fails when it differs from `engines.node`.
- `frontend/apps/remark42/package.json`, both its `packageManager` field and its `engines` block
- every `pnpm/action-setup@vN` block in `.github/workflows/ci-frontend.yml` and `release.yml` — pin `version:` to the **exact** patch (e.g. `10.10.0`), matching `packageManager`, not just the major. A floating major here is silent in CI (it just resolves to whatever the latest patch is at run time) but breaks the "Dockerfile and CI use the same pnpm" guarantee.
- `node:` matrices in `.github/workflows/ci-frontend.yml` (every entry, not just the first) and the `node-version:` values in `release.yml`
- `.oss-scanner/Dockerfile` — `FROM node:X-bookworm` and `npm i -g pnpm@X.Y.Z`
- the versions stated in `site/content/docs/getting-started/installation/index.md` (checked by CI), `site/content/docs/contributing/frontend/index.md` and the release procedure in the root `AGENTS.md`
- the dependency-free commands live in `package.json` as `test:unit`, `translation:extract` and `translation:check`, and are **repeated inline** in the `unit` and `translations-check` jobs of `ci-frontend.yml`, which carry no pnpm by design. Change a script and that job keeps running the old one, green

When bumping pnpm/node, also re-check `frontend/apps/remark42/package.json`'s `engines` field — it's separate from `packageManager` and won't update itself.

`engines.node` states the major we support, currently `>=24`, which is the active LTS; 22 has dropped to maintenance. `@babel/core` 8 wants `^22.18 || >=24.11`, so 22 was the floor rather than the target. Individual dev dependencies can be stricter within that major (`@babel/core` 8 needs 24.11 or later); do not chase those patch floors into `engines` or the docs, or every lockfile refresh becomes a documentation change.

## One manifest, at `frontend/apps/remark42`

`frontend/` holds no `package.json`, no lockfile and no `pnpm-workspace.yaml`. Everything pnpm reads
lives in `frontend/apps/remark42`: the dependencies, `packageManager`, `engines` and the
`pnpm.overrides` block. Install and run from there, not from `frontend/`.

The directory nesting is kept because every path in the repository points at it, from the Dockerfile
and the workflows to the published contributing docs. `frontend/` still carries `.nvmrc` and this
file, neither of which pnpm reads.

## pnpm's `overrides` pin vulnerable transitive versions

Most entries in `pnpm.overrides` raise a transitive dependency past a security fix. An entry for a package nothing installs any more does nothing, so drop it in the change that removes the last package pulling it in; the lockfile's `packages:` section shows what is installed.

If a dependency bump breaks types or module resolution only after a pnpm major bump, suspect pnpm's hoisting layout before the dependency: pnpm 10 changed it.

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
sets `BABEL_ENV` itself and `.babelrc.js` reads it for two things: the compile targets, and
keeping the e2e coverage instrumentation out of the suites. The `data-testid` attributes the
suites query by ship in the bundle as written; nothing strips them.

`tsconfig.json` types but never emits, so its `target` has no effect on output; browserslist in
`.babelrc.js` decides that. The build does not type check: `pnpm type-check` does, as its own CI
job, in the Dockerfile's test step and in `release.yml`.

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

- `eslint` 10 (`eslint-config-preact` peers on ^9), `typescript` 7, `redux` 5 with `redux-thunk` 3, and `postcss-preset-env` 11 with `cssnano` 8, which change generated CSS

## `/web` has a second source

`privacy.html`, `markdown-help.html`, `400x400.jpeg`, the ghost avatar `ghost.svg`, the sign-in
provider icons, the emoji table `emoji.json`, the message catalogues `<locale>.json`, the widget's own document `iframe.html` and the pages
that host the widget (`index.html`, `counter.html`, `last-comments.html`, `deleteme.html`,
`comments.html`) live in `backend/app/webassets/assets` and are embedded in the Go binary. They are
plain files: the bundler builds no HTML. `iframe.html` links `remark.css` itself, so the dev server
extracts CSS to files as the production build does. The three pages that name a site spell it
`site_id:"remark"`, exactly, because the server replaces that text with the first configured site,
and `comments.html` carries `{% REMARK_URL %}`, which the server fills in too. `counter.mjs`, the
comment counter for host pages, is a script written by hand in the same directory: no type checking,
no lint, no transpiling. It has to stay a classic script with no module syntax, since the server
answers `counter.js` with its bytes, and `TestFS_CounterIsAClassicScript` checks that. The widget
addresses the images by name through `webAssetUrl` in `app/common/web-asset.ts`; the bundle resolves
no image files. The comment form fetches `emoji.json` relative to its own document on the first `:`
typed (`emoji-data.ts`), and `emoji-search.ts` matches names against it. The message catalogues are
fetched the same way: `app/utils/loadLocale.ts` asks for `/web/<locale>.json`, only for a locale
listed in `tasks/supportedLocales.json`, and never for English. `en.json` is not read by the widget:
the English text is the `defaultMessage` in the code, `translation:generate` copies it into `en.json`,
and the file exists as the reference translators work from and the extraction checks compare against.
Change English text in the code and regenerate; an edit made only in `en.json` changes nothing on
screen, and `extract-messages.unit.test.ts` fails until the two agree again. last-comments runs on the host page,
so there the fetch is cross-origin and depends on the server's CORS headers, or the proxy's with
`--proxy-cors`. Tests import a catalogue as `locales/<locale>.json`, which `tsconfig.json` and
`jest.config.mjs` map to that directory. The frontend build wins
for any name present in both, which is what
lets an operator override one by dropping a file next to the frontend files in `--web-root`. That
directory replaces the embedded frontend outright when it exists, so an override belongs in a
populated one. These files sit outside this toolchain: prettier, stylelint and `pnpm lint` do not
see them, and they are served unminified. `devServer.static` lists the build output first and that
directory second, matching the backend's order, so links to them resolve on the dev port too.

## eslint config resolves from the process cwd

`eslint.config.mjs` lives in `apps/remark42`, and eslint loads the config next to the directory it
is *run from* rather than the one nearest the file being linted. Anything that invokes eslint has
to have `apps/remark42` as its working directory: it is where every script runs, and an IDE
integration needs
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

## i18n is a hand-written binding, and extraction reads `id` and `defaultMessage` pairs

`app/common/intl.tsx` provides `IntlProvider`, `useIntl`, `createIntl`, `defineMessages`,
`FormattedMessage` and `IntlShape`. The formatting itself lives in `app/common/intl-message.ts`,
whose only preact reference is a type-only import that erases at build time, so it pulls in nothing at runtime, and which is where every rule below is implemented; `intl.tsx` is the
binding over it and re-exports `createIntl` and `defineMessages` unchanged, so no call site can tell
the two apart. `translation:extract` is `tasks/extract-messages.ts`, a node script with no
dependencies. It reads every `id` written just before its `defaultMessage`, both as plain strings,
in an object literal or as JSX attributes, whatever function or component they are passed to, so
renaming a helper loses nothing. It has no parser: it tells a key from a ternary, an assignment or a
destructuring default by prettier's spacing, which `pnpm lint` enforces, and it reads comments like
code, so a commented-out message keeps its key. A text given by a variable is not seen at all.

It fails instead of writing in these cases, because `tasks/generateDictionary.js` calls
`removeAbandonedKeys` and `translation:generate` would otherwise delete the lost keys from every
catalogue in `backend/app/webassets/assets/`:

- a `defaultMessage` given as a string that it cannot pair with a plain-string `id` (an `id` after
  its text or held in a variable), or a string that goes on into a concatenation or a call
- a template literal with an expression, or a JSX attribute with a backslash, an entity or a line
  break
- one `id` is defined twice with different text
- a key in `en.json` is no longer in the source. When the message was removed on
  purpose, `pnpm translation:extract --allow-removal` writes the file and `translation:generate`
  then drops the key from the catalogues

`tasks/extract-messages.unit.test.ts` covers each case, and checks that the whole widget source
yields exactly the keys and texts of `en.json`.

The binding implements `{name}` interpolation and paired `<tag>text</tag>` rich text, and
nothing else of ICU: no plural, select or selectordinal, no typed arguments such as
`{n, number}`, and no apostrophe quoting. A plural or select form has braces the placeholder
rule rejects, so it falls back to English wherever values are passed and shows its raw text
where they are not; `''` simply stays as two apostrophes.

`translation:check` validates every translated value's markup and placeholders against the
English string it translates, mirroring the binding's rule, and the catalogue sweep in
`app/common/intl.test.tsx` additionally renders the two messages that carry a link.
`app/common/intl-message.unit.test.ts` covers the formatter's own edge cases -- broken markup, an
unclosed tag, a regex metacharacter in a tag name -- which are the ones no shipped catalogue can
produce and so no sweep can reach. What
neither catches is unsupported ICU syntax, since a plural form is well-formed text as far as
both are concerned. A message the binding cannot resolve falls back to the English source
rather than reaching the page, so without these checks a broken translation is invisible in
the interface.

## `*.unit.test.ts` runs with no toolchain, and that is what it is for

Two test layers, split by what they need to run. Jest owns anything with a DOM in it; `*.unit.test.ts`
files, in `app/` and in `tasks/`, import nothing but the module under test, its fixtures and node's
own modules, and run under plain node straight from source:

```sh
pnpm test:unit
```

The `unit` job in `ci-frontend.yml` runs them with **no `pnpm install` step**, deliberately.
`release.yml` runs the same script after an install, as a second gate. The layer's value is that a
function over plain values can be tested without a bundler, a jsdom or a `node_modules`, and the
absent install is the only thing that keeps that true: add an import that needs resolving and the
job goes red instead of quietly acquiring the dependency. `jest.config.mjs` excludes the suffix, so
a file is in exactly one layer.

`node --test` exits 0 on a glob that matches nothing, so it cannot tell a renamed suffix from a
passing run. The CI step counts the files first and fails when none match. Two import rules follow
from node's ESM resolver, and they contradict the repository's usual style, so they are local to
this layer:

- **write the extension in full**, `./types.ts`. This is why `app/store/thread/reducers.ts` and
  `app/store/user/reducers.ts` import `./types.ts` while their neighbours do not, and why
  `app/common/json-store.ts` and `app/utils/errorUtils.ts` carry it too. `allowImportingTsExtensions` in `tsconfig.json` is what
  stops `tsc` rejecting it, and webpack and jest resolve the explicit path unchanged
- **import by relative path**, never through the `common/…` alias, which only webpack, jest and
  `tsc` know about. A type-only import of an alias is fine, since it erases before either runtime
  sees the file

A module that mixes computation with browser state gets split instead of skipped:
`app/common/intl-message.ts` and `app/hooks/session-storage.ts` are the pure halves of `intl.tsx`
and `useSessionState.ts`, and `session-storage.ts` takes the store as an argument instead of reading
the `sessionStorage` global, which is what lets a test hand it a plain object.
