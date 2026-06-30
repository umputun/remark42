# Frontend dependency updates

Non-obvious things that bit us during the pnpm 8→10 / node 16→20 migration (PR #2091, fixing #2085). Read before bumping dependencies or the node/pnpm toolchain again.

## The node/pnpm version is pinned in ten places, not one

CI staying green does **not** mean every pin is consistent — `.nvmrc` in particular is never read by CI, so it can silently drift. After changing the node or pnpm version, grep the whole repo and update every one of these, not just the ones CI exercises:

- `Dockerfile` (production image) — `FROM node:X-alpine` and `npm i -g pnpm@X.Y.Z`
- `frontend/Dockerfile.e2e` — `FROM mcr.microsoft.com/playwright:vX.Y.Z-noble` **and** `corepack prepare pnpm@X.Y.Z`
- `site/Dockerfile`, `site/Dockerfile.dev` — `FROM node:X-alpine` (site uses yarn, not pnpm)
- `frontend/.nvmrc`, `site/.nvmrc` — not read by CI at all; only matters to a human running `nvm use` locally. This is the one that drifted unnoticed: it sat at `16` through the whole node-20 migration because nothing red ever pointed at it.
- Every `package.json`'s `packageManager` field (`frontend/package.json`, `frontend/apps/remark42/package.json`, `frontend/packages/api/package.json` — `frontend/e2e/package.json` has none) and `frontend/apps/remark42/package.json`'s `engines` block
- `pnpm/action-setup@vN` blocks in `.github/workflows/ci-frontend.yml` (5), `ci-frontend-api.yml` (3), `release.yml` (2) — pin `version:` to the **exact** patch (e.g. `10.10.0`), matching `packageManager`, not just the major. A floating major here is silent in CI (it just resolves to whatever the latest patch is at run time) but breaks the "Dockerfile and CI use the same pnpm" guarantee.
- `frontend/e2e/package.json`'s `@playwright/test`/`playwright` versions must match `frontend/Dockerfile.e2e`'s base image tag exactly, or the e2e container's bundled browser revision mismatches what the npm package expects.

When bumping pnpm/node, also re-check `frontend/apps/remark42/package.json`'s `engines` field — it's separate from `packageManager` and won't update itself.

## pnpm 10's stricter `node-linker` layout needs explicit pins

A few deps needed pinning specifically because of pnpm 10's hoisting changes, not because of the deps themselves:
- `preact` pinned via `pnpm.overrides` (10.6.2) — newer breaks the build under the stricter layout
- `react-intl` 6.0.5 and `@testing-library/preact` 3.2.2 — newer versions' type declarations break the preact-compat alias setup
- `@types/minimatch` 5.1.2 — 6.x is an empty stub that the hoisted layout picks up instead of the real types
- `cheerio` 1.0.0-rc.12 — 1.2 is ESM-only and breaks under jest 28

If a dependency bump mysteriously breaks types or module resolution only after a pnpm major bump, suspect the layout change before suspecting the dependency.

## msw 1→2 needed a real migration, not just a version bump

`frontend/packages/api` test mocks (`tests/test-utils.ts`) moved from msw's `rest` API to `http`/`HttpResponse`. Also: test base URLs had to become absolute, and a jsdom base URL had to be set, because node 20's native `fetch` (unlike node 16/18's polyfilled fetch) requires an absolute URL — relative request URLs in tests started failing silently otherwise.

## Held-back majors

These were deliberately not bumped because each is a config-migration or bundle-changing major, not a drop-in update — don't bump them opportunistically inside an unrelated dependency PR:

- `eslint` 8 (9/10 need flat-config migration), `stylelint` 14 (16 has breaking rule changes), `babel` 7, `jest` 28 (30 needs config changes)
- `typescript` 4.7 in `apps/remark42` specifically (the api package is on a newer TS — they're intentionally decoupled)
- `react`/`react-dom` (the app uses `preact` aliased as `react`/`react-dom` via `preact/compat` — don't "fix" this by installing real react)
- `redux`/`react-redux`, `tailwindcss` 3.4 (v4 is a full config rewrite), `@11ty/eleventy` 2 (v3 is an ESM migration) in `site/`

## `html-minifier` is abandoned — use `html-minifier-terser`

`site/.eleventy.js` uses `html-minifier-terser` (a maintained fork), not `html-minifier` (unpatched ReDoS advisory, no fix ever released). The eleventy transform had to become `async` for this fork's API.

## Verifying a build didn't regress

There's no automated build-output diff in CI. Before merging a dependency PR that touches the bundler/build tooling, manually diff the build output against a clean `master` checkout:
- `apps/remark42`: expect webpack module-id numbers and css-module class tokens (e.g. `.F_A` → `.L_A`) to differ — that's normal churn from a webpack/css-loader bump. HTML, CSS values, and translation content should be byte-identical.
- `site`: expect the `?v=<timestamp>` cache-bust query string to differ on every HTML file — that's expected. Anything else differing is a real regression.

## Where the alerts actually were

When clearing Dependabot/audit alerts, check whether the flagged package is actually reachable from production code or only from the dev/test toolchain — `pnpm audit`/`yarn audit` don't distinguish. Several alerts here were in build-time-only tooling (webpack-dev-server, laravel-mix-equivalent dev deps) with no patched release available; those are lower-risk than a runtime dependency with the same severity label.
