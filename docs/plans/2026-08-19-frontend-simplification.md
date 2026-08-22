# Frontend direction: two viable paths, and what has to be true for either

Written 2026-08-19, revised 2026-08-22. Every file reference below was re-checked against master
`a82dc8d3` plus #2196, #2197 and #2198, which are treated here as landed: they change the e2e net,
the manifest layout, the fallback page, the asset path and the instance URL, and costing either
direction against the state before them would be costing a world that no longer exists.

## Overview

- The widget's measured cost was the **toolchain and the React compatibility alias**, not Preact
  itself. The alias is now gone, so what remains is the toolchain
- Two paths are genuinely viable: **incremental simplification** (keep Preact, finish collapsing the
  toolchain) or the **server-rendered rewrite** of #825. This document sets out to cost both rather
  than to recommend against one, and it half succeeds: Path A is costed, Path B is not, because two
  of its inputs are still open. What language the retained client code is written in, and whether a
  scoped XSRF exemption can make the document render authenticated, both change the figure
  materially. The "months" estimate below is therefore a placeholder rather than a finding, and
  should not be quoted as though the comparison had been made
- Server rendering is not architecturally blocked. Authenticated *fragments* work; the initial
  *document* render is anonymous in every configuration, and that is forced rather than chosen
- Whichever path is taken, two standing requirements constrain it, and three pieces of work are
  worth doing first regardless. One of those three is now done

## Standing requirements

Not tasks. Tests any proposal has to pass.

### R1: Remark42 must be hostable on a domain other than the site embedding it

This is a product requirement rather than a constraint to respect in passing, and it is the one
users report against most. The target arrangement is remark42 serving from its own name, say
`remark42.example.com`, with comments appearing on a different site entirely, say `food.com`. It is
documented in `site/content/docs/manuals/separate-domain/index.md`, so operators follow it and then
find that authentication behaves differently from the same-domain case.

**Acceptance criteria.** A reader on `food.com` can sign in with any configured provider, post,
reload the page, and still be signed in, on a browser that blocks unpartitioned third-party cookies.
Nothing short of the reload proves it: the widget holds a token in memory for the life of a page, so
a test that signs in and posts without reloading passes while the persistence is entirely broken.

**Where that stands.** The e2e suite covers the rendering half through
`TestCrossOrigin_WidgetRendersOnAnotherOrigin`, which proves the document loads on another origin
and reports itself through postMessage across the boundary, and `ALLOWED_HOSTS` refusal through
`TestCrossOrigin_DisallowedHostNeverReportsInited`. The authentication half is not covered and
cannot be until the e2e stack speaks https, because an embedded cookie needs `SameSite=None`, which
browsers accept only with `Secure`.

`ALLOWED_HOSTS` sets the CSP `frame-ancestors` and `AUTH_SAME_SITE=none` lets auth cookies be set
from any embedding domain. OAuth is what visibly fails off-domain (discussion #1139). Telegram,
email and anonymous work where third-party cookies are still permitted, and stop working where they
are not: the server's cookies carry no `Partitioned`, and the header fallback is off by default, so
nothing saves them once the browser blocks unpartitioned third-party cookies.

There is a second mechanism aimed squarely at this, `AUTH_SEND_JWT_HEADER`, which returns the token
in a response header so the client can present it without relying on an ambient cookie. #1877 was
opened by an operator who wanted exactly this for exactly this reason, and #1929 was merged as its
first half. Its persistence never worked on https: the client wrote its copy under a `__Host-`
prefix that neither the backend nor the widget's own reader ever asks for, and marked it
`SameSite=Strict`, which is never sent from a third-party frame. Both are corrected here, and the
frontend now marks its cookies `SameSite=None; Secure; Partitioned` when it detects a third-party
context. The server-set cookies still need the same treatment, and that is upstream work in
`go-pkgz/auth`.

Constraints on any frontend design:

- nothing may assume the widget and host page share an origin. Note `BASE_URL` comes from
  `remark_config.host`, read into `BASE_URL` in `app/common/constants.config.ts`, so the iframe is
  cross-site only when that host differs from the page host; both configurations exist in the wild
- treat third-party cookie loss as the direction of travel. Safari blocks third-party cookies
  outright and honours only cookies explicitly marked `Partitioned`; its CHIPS support has been
  switched on, off and on again across releases, so pin the current state before relying on a
  version number. A design depending on `SameSite=None` surviving has an expiry date
- the endpoint is CHIPS (`SameSite=None; Secure; Partitioned`) or a token not relying on ambient
  cookies. Neither is a flag flip, and the first cost is upstream rather than here:
  `go-pkgz/auth/v2@v2.2.0` cannot emit `Partitioned` at all. Both cookies are hand-built in
  `Service.Set` with only `HttpOnly`, `Path`, `Domain`, `MaxAge`, `Secure` and `SameSite`, and
  `AUTH_SAME_SITE` in `cmd/server.go` offers `default`/`none`/`lax`/`strict` with no partitioned
  axis, since `Partitioned` is a separate attribute. That is a PR to the library before anything in
  this repo changes
- the popup-to-iframe handoff cannot be done in JS. The JWT cookie is `HttpOnly: true`, set in
  `Service.Set`, so the top-level popup cannot read it to hand over, and the receiving end would not
  work either: `setAuthCookie` in `cookies.ts` writes `SameSite: 'Strict'` under a `__Host-` prefix
  with no `Partitioned`, and Strict is never sent from a third-party frame. The handoff has to be
  server-mediated, a one-time code redeemed for a `Set-Cookie … Partitioned` issued from the
  embedded context. Email and anonymous authenticate over XHR from inside the iframe, which keeps
  them working while third-party cookies are permitted, but XHR does not create a partition by
  itself, so they need `Partitioned` for the same reason OAuth does
- the failure mode is silent rather than an error, which is why #1139 reads as a hang. OAuth
  completion is detected by polling: `oauthSignin` in `auth/auth.api.ts` opens the popup and, on
  `visibilitychange` once `authWindow.closed`, calls `getUser()`, with nothing posted back from the
  popup. When the cookie is not readable from the embedded context, `getUser()` returns null and the
  flow re-polls every minute until the 5 minute deadline gives up, showing a waiting state
  throughout. Until #2197 that deadline rejected without tearing the subscription down, so the retry
  outlived it and polled for the life of the page; the reason it lasted is that the arrangement it
  breaks in is the one nobody runs locally
- the documented escape hatch is the fallback page. When the widget detects a third-party context
  with storage unavailable it offers a link to `${origin}/web/comments.html` (`auth-panel.tsx`, the
  `IS_THIRD_PARTY` and `IS_STORAGE_AVAILABLE` checks in `app/common/constants.ts`). That page went
  unbuilt from the January 2021 rewrite until #2197 restored the entry, so it 404'd for four years
  in exactly R1's configuration. Any redesign has to keep emitting it, and nothing pins that yet:
  the `documentedWebPaths` table in `e2e/webfiles_test.go` still has no entry for it, which is the
  same absence that let it break in the first place
- fixing #1139 must not get harder

### R2: The language must be selectable per page, and may stay fixed after load

24 catalogs in `app/locales/`. Locale is read once from `remark_config.locale` (the `locale` export
in `app/common/settings.ts`) and resolved to one chunk by `loadLocale`. There is **no runtime
switcher** and no locale entry in the postMessage contract, so language cannot change during an
instance's life.

That makes this a comfortable requirement: a site with three language versions embeds each page with
its own locale. Server-generated per-locale output satisfies it as well as the current client
catalog.

Must survive:

- `remark_config.locale` keeps selecting the language per instance
- all 24 languages keep working. Adding one is not catalog-only today: the language goes into
  `tasks/supportedLocales.json`, and generation then rewrites `app/utils/loadLocale.ts` from it.
  Preserve that registry-plus-generation shape or replace it deliberately
- the translation completeness check in `ci-frontend.yml` survives in some form
- `formatDate`/`formatTime` (`Comment` in `comment.tsx`, implemented over `Intl.DateTimeFormat` in
  `common/intl.tsx`) keep resolving in the viewer's **timezone**. The locale is not the viewer's:
  the provider gets the configured `locale` (`remark.tsx`, from `remark_config.locale` via the
  `locale` export in `settings.ts`), which the server knows too. Only the timezone is genuinely
  client-side, so the carve-out is timezone-local time rendering rather than a whole date-formatting
  layer

## Server rendering is not blocked, but the document render is anonymous by force

SSR is sometimes assumed to be impossible here, on the grounds that the widget document is
cross-site and the auth cookie defaults to Lax. That reasoning does not hold:

- **htmx fragments are XHR, not document navigations.** They are same-origin to the widget document,
  so the server-set auth cookie rides along exactly as it does for today's API calls. What
  `htmx:configRequest` has to attach is the `X-XSRF-TOKEN` header read from the JS-readable XSRF
  cookie, attached by `request()` in `fetcher.ts`, and on **every** authenticated request rather
  than only state-changing ones: `request()` attaches it unconditionally for get, put, post and
  delete alike. There is no JS-held JWT to attach under default configuration: `activeJwtToken` in
  `fetcher.ts` is filled only from an `X-JWT` response header, which the server sends only in
  `SEND_JWT_HEADER` mode
- **`AUTH_SEND_JWT_HEADER` defaults to false** (`backend/app/cmd/server.go`). Auth is cookie-based
  by default, so the header was never what made it work
- **The cross-domain auth limitation already exists**, is documented, and is #1139's subject. SSR
  does not introduce it and does not make it worse
- **The `/web/` cache and rate limit are not a general blocker.** They apply to the current static
  route, where `addFileServer` applies `rateLimiter` and `cacheControl`; dynamic HTML served from a
  new route avoids that pair, and `/api/v1` open routes already run through `NoCache`. The open
  routes do carry their own limiter, `rateLimiter(s.openRouteLimiter)`, so a new route escapes the
  static cache, not rate limiting in general. Price which bucket it lands in: open routes allow 10
  req/s per IP (`openRouteLimiter` in `rest.go`) against 20/s for static `/web/`, applied by
  `addFileServer`, and a fragment UI multiplies requests per interaction against the tighter one.
  Under R1, with readers behind shared egress, that is the operative ceiling
- **`AUTH_SAME_SITE=none` already exists** and would let cookies accompany iframe navigations on
  browsers that still allow third-party cookies. It is inert for the document render, though, for
  the reason below: the cookie arrives and is then discarded

The real constraint is not about cookie delivery at all.

**The first document render is anonymous in the configuration remark42 ships, and changing that is a
security decision rather than a flag flip.** The auth library rejects any cookie-borne token whose
`X-XSRF-TOKEN` header does not match the JWT's `jti`, in `Service.Get` of `go-pkgz/auth/v2@v2.2.0`.
A document navigation, which is what an iframe `src` is, cannot carry a custom header, so the widget
document arrives anonymous even with `AUTH_SAME_SITE=none`, even with third-party cookies fully
permitted, and even with a perfectly delivered partitioned cookie. `authMiddleware.Trace` swallows
the error, which is why it is silent.

What makes it the current behaviour rather than a law is `XSRFIgnoreMethods`, an option the library
exposes on `Opts` and threads into the JWT service. remark42 leaves it unset, so the default empty
list applies and the short circuit in `Service.Get` never fires for any method. Setting it would
make an authenticated document render possible.

Do not reach for it globally, though. `GET /deleteme` deletes every comment a user has written, and
is a GET deliberately, so that the link in the confirmation email works when clicked. Exempting GET
from XSRF wholesale removes that protection from a destructive endpoint. Anything built on this has
to scope the exemption to the document route alone, and that route has to be provably side-effect
free. Cost that work rather than assuming either that the door is shut or that it is open.

There is also a query-parameter path, and it is worse than the constraint: `Service.Get` accepts the
token from a query parameter, `?jwt=` rather than the library default `?token=` because remark42
overrides `JWTQuery` in `cmd/server.go`, and because `fromCookie` stays false the XSRF check is
skipped entirely. That would put a live JWT into `Referer`, into history, and into the access log if
the route is one that logs bodies. It is unreachable in any case, since the JWT cookie is `HttpOnly:
true`, set in `Service.Set`, and no JS can read it to build the URL.

So anonymous-first is what the current configuration gives, and the honest Path B position is that
making the document authenticated is a scoped piece of security work with its own cost. Until that
is costed, budget for anonymous-first: render anonymous, then hydrate the user state over XHR, which
does carry the header. Anonymous-first is also what a shared cache wants, though see the hydration
item in Path B for how much that is worth.

## Verified facts

Checked against the code at `a82dc8d3` and by independent reviewers.

- 9 runtime dependencies against 61 devDependencies, and **68** override entries, all in the single
  `frontend/apps/remark42/package.json` since #2197 removed the workspace root. Before this effort
  started it was 15 against 83 across two manifests
- `pnpm audit` reports **no known vulnerabilities** as of 2026-08-20, after the override work. The
  23 it reported before that were all build or test tooling and none shipped, so the toolchain's
  security cost is recurring maintenance rather than a standing exposure, and it is a weak argument
  for Path B
- Frontend churn 67 commits in 24 months against 119 on the backend, merges excluded on both sides
- Preact appears in 53 non-test files, 57 counting typings and stubs, and not only for `h` and JSX
  types: `Fragment` in 11, `Component` in 7, `render` in 1, `createRef` in 3, `createContext` in 2,
  and `preact/hooks` in 19. **No file imports `preact/compat` any more**, no `react` or `react-dom`
  entries remain in `package.json`, and the only `paths` entries in `tsconfig.json` point at preact
  itself. `frontend/CLAUDE.md` carries a "Don't import preact/compat" section recording why
- Composer plus auth is **~2,400 non-test lines** (`comment-form/` 1,540, `auth/` 867), the most
  stateful code in the repo, plus two custom-element packages and the polyfill they need
- The e2e suite is Go and playwright-go since #2180, and passed 60 tests while this was being
  written, up from 7. Treat the exact figure as stale on sight; it is the only coverage that
  survives a rewrite
- The unit suite is 46 files, 25 `*.test.*` plus 21 `*.spec.*`, and **426 cases**, as jest
  enumerates them. Counting only `*.test.*` understates it by twenty files, which is the trap
- `en.json` is 180 keys with no ICU plural or select forms
- 136 non-test source files under `app/` excluding typings, mocks and stubs, 8,498 lines; 152 files
  and 8,715 lines counting them
- `profile.ts` (139 lines) is only the iframe host; the view is `profile.tsx` (235) reusing
  `Comment` (638) in `view="user"` mode
- `last-comments` renders into the **host page**, not an iframe, and side-loads its own stylesheet
  (`last-comments.tsx`)
- `/find` already supports server pagination via `limit` and `offset_id` in `findCommentsCtrl`; only
  the widget ignores them
- Client-only state: collapse persisted via `remark.tsx` and `store/thread/utils.ts`; hidden users
  localStorage-only (`store/user/actions.ts`); votes optimistic via component state, with the store
  patched only on success and errors merely clearing `loadingState` (`comment-votes.tsx`)
- Emoji rendering and the image proxy already run server-side in `CommentFormatter`, wired in
  `cmd/server.go`, so neither is a Path B cost
- 8 `window.confirm` call sites gate delete, pin, verify, block and hide (six in `comment.tsx`, two
  in `settings.tsx`). The iframe carries no `sandbox` attribute, so `allow-modals` is not the issue,
  but cross-origin iframe dialogs have been targeted for removal once already (Chrome 92, rolled
  back after breakage). #2024 proposes an inline replacement. Either path inherits all 8, and R1 is
  exactly the configuration where it bites
- `CommentFormatter` in `formatter.go` runs chroma with `html.WithClasses(true)`, so highlight CSS
  lives in the bundle; `Comment.Text` is pre-sanitised while `Orig` is explicitly unsafe, as
  `store.Comment` documents
- `getLocalIdent` in `webpack.config.js` emits order-dependent ids via `incstr`, **in production
  only**; development builds use readable `[name]__[local]_[hash:5]`, since `getLocalIdent` is
  production-only. They are stable for an identical module order and churn on any add or reorder, so
  no embedder has a CSS override surface that survives an upgrade

## What the bundler still does

The concrete "what is left" list, and what any no-npm proposal has to answer for.

- transpiles TS and JSX for 136 to 152 source files, typed by `tsconfig.json`, compiled by the
  preset list in `.babelrc.js`
- CSS modules for 29 `*.module.css` files, 2,219 lines, including 177 nested `&`, 4 `composes` and
  10 `:global` occurrences across 6 files, all handled by the CSS-modules rule in
  `webpack.config.js`. One of the `composes` crosses a file boundary (`auth.module.css` composes
  `input` from `components/input/input.module.css`), which has no plain-CSS equivalent and has to be
  flattened rather than translated
- code-splits the 24 locale JSON catalogs (`app/utils/loadLocale.ts`, itself generated) and
  `node-emoji` (the lazy import in `comment-form/text-expander.tsx`)
- runs postcss-preset-env against `defaults, not IE 11, not samsung 12`
- builds six `.ejs` templates through `HtmlWebpackPlugin`, the sixth being `comments.ejs`, the R1
  fallback page above
- resolves images through `file-loader`. The public path is derived at runtime from the URL the
  bundle was loaded from since #2197; before that it was fixed to `/web/`, so a sub-path install of
  the kind `site/content/docs/manuals/subdomain/index.md` documents fetched its provider icons from
  the domain root. A build with no bundler has to derive every asset URL from `BASE_URL` itself, and
  inherits that requirement rather than the fix
- strips `data-testid` through a local babel plugin, which nothing else replicates
- minifies JS and CSS
- supplies `REMARK_NODE`, `REMARK_URL` and `NODE_ENV` through `DefinePlugin`, read by `NODE_ID` and
  `BASE_URL` in `constants.config.ts`, and by `last-comments.tsx`. `REMARK_URL` is the one that had
  teeth: it was baked in as `{% REMARK_URL %}` and substituted by the release script, which left
  every published binary serving a widget pointed at loopback. #2198 moved the substitution to serve
  time in the Go file server, so a no-bundler build inherits a working mechanism rather than a
  broken one

Five CI jobs depend on npm and each needs a replacement or an accepted loss: `translations-check`,
`type-check`, `lint`, `size-limit` and `test`, all in `ci-frontend.yml`.

### What is already out

The widget build is the last npm consumer in the repository, which is worth stating because it was
not true a week ago.

- **`site/`** has no `package.json` at all since #2179 moved it from eleventy to hugo. It is now a
  Go static generator against `hugo.toml`, `layouts/` and `content/`, and is no longer a separate
  npm decision deferred to later
- **`e2e/`** is a Go module since #2180. Note the caveat: `playwright-go` downloads a node runtime
  and the playwright npm package as its browser driver at test time (`e2e-tests.yml` caches both).
  That is a fetched runtime rather than a tracked dependency, so it survives whatever happens here
- **`@remark42/api`** and `frontend/packages/` are gone with #2172
- **Three static assets** moved into `backend/app/webassets/assets` in #2181 and are embedded in the
  Go binary, which is the working proof that the no-build serving path carries real files

Since #2197 removed the workspace root there is one manifest and one lockfile,
`frontend/apps/remark42/package.json` and its `pnpm-lock.yaml`. `release.yml` still runs pnpm to
build the widget, and that step goes when the widget build does.

## Do first, regardless of path

### Task 1: Document and type the public contract

Answers discussion #1714. It is a discussion rather than an issue, so no PR closes it; the same is
true of #1715 and #1383. Every third-party integration found in the wild uses `remark_config` plus
`window.REMARK42.createInstance()`, and none of it is documented.

The contract was written out in a reply on #1714 on 2026-08-20, and
`docs/backlog/public-widget-contract-docs-and-types.md` records the remaining work. One decision is
open and belongs here rather than in the backlog note: **how a public `.d.ts` reaches a consumer**
now that npm is the direction being left. Publishing it on the site as a copy-paste block needs no
infrastructure; a types-only package does not contradict #1715, whose objection was that shipping
widget *code* through npm breaks OAuth, but it adds a publish step and a version to keep in sync.

- [ ] Document `createInstance`, `destroy`, `changeTheme`, the `REMARK42::ready` event and every
  `remark_config` field, including the two behaviours that bite: `createInstance` throws rather than
  returning an error, and its guard clauses read the **global** config while the rest of the
  function reads the argument, so passing a config does not remove the need for a valid global. A
  second call also reuses the iframe and ignores the config passed to it. #2197 stopped that call
  stacking a second listener set, but left the reuse itself alone, so what the contract should be is
  still open: replace the instance atomically, or return the existing one and require `destroy`
  before a configuration change
- [ ] Ship a `.d.ts`, after deciding how it is delivered
- [ ] Move the REST reference out of `site/content/docs/contributing/api` into the integration docs;
      #1383 states the API is the supported path for custom frontends
- [ ] Document `__colors__` from `window.name` (`templates/iframe.ejs`), which works today and is
  undocumented
- [ ] Fix the Astro and Gatsby manuals, which declare `REMARK42: any` and `remark_config: any`

### Task 2: Extend the e2e suite

**Done.** #2180 moved the suite to Go and playwright-go and took it from 7 tests to 22; #2196 took
it to 48. Every item this task originally listed is covered: vote and its failure path
(`vote_test.go`), edit inside and outside the deadline, delete and reply (`comment_test.go`), sort
change and collapse persistence (`thread_test.go`), anonymous and email auth (`auth_test.go`), the
profile iframe and last-comments (`widgets_test.go`).

What remains uncovered is a different list, and it is the contract surface rather than the
behaviour: a cross-origin host page (every host page in the suite is served from the widget origin,
so R1 has no coverage at all), the `comments.html` fallback, `remark_config` fields (`url`,
`page_title`, hash deep links, `max_shown_comments`, the three `show_*_subscription` flags,
`__colors__`), the listener leak on a repeated `createInstance`, timezone-local date rendering, the
unknown-locale fallback, and the composer.

### Task 3: Stable class names and a documented override stylesheet

Closes #5, open since 2018. Server-rendered markup needs stable names anyway, and the current ids
churn per build so there is nothing to preserve, only something to start honouring.

- [ ] Replace the `getLocalIdent` output with stable semantic names on the public surface
- [ ] Document the override point

The surface that exists today is accidental rather than designed, but it is not small, and that
changes what Task 3 is. `auth.tsx` alone emits sixteen global names (`auth`, `auth-error`,
`auth-dropdown`, `auth-form`, `auth-form-title`, `auth-row`, `auth-tabs`, `auth-tabs-item`,
`auth-divider`, `auth-close-button`, `auth-token-textarea`, `auth-submit`, `auth-button`,
`auth-back-button`, `auth-input-username`, `auth-input-email`). Add `select`, `select_focused`,
`select_<size>`, `select-arrow` and `select-element` (`select.tsx`), `sort-picker`
(`sort-picker.tsx`), `oauth-icon` (the `OAuth` component in `oauth.tsx`), bare `.dark`/`.light` on
the root wrapper, and `comment_highlighting`, applied imperatively in `root.tsx`. Six module files
reach these through `:global()`.

So the first step is an inventory rather than a design: integrators may already be relying on any of
these, and they have to be treated as a contract to preserve rather than a blank sheet. Note also
that the hashed names are deterministic for an identical module order, so they churn on a reorder
rather than on every build; the problem is that nothing tells an integrator which of the two kinds
of name they are looking at.

Two things make this more urgent than its age suggests. Production and development emit different
names, so a developer never sees what an integrator sees. And unit tests run through
`identity-obj-proxy`, mapped in `jest.config.mjs`, so no test observes a real class name and nothing
would catch a naming regression.

`app/styles/custom-properties.css` is the one stylesheet already in the right shape for this: 80
plain custom properties, no module scoping, nothing for the bundler to rename.

**Ordering caution.** #2128 is rewriting this same CSS layer on the assumption that hashed class
names stay. Settling the direction here first avoids redoing that work.

## Path A: incremental simplification

Keep Preact. Remove what makes it expensive.

- [x] Replace `react-redux`. Done in #2175: `app/store/context.tsx` is a small binding over preact
  context, and the store's own logic was already plain `redux` plus `redux-thunk`
- [x] Replace `react-intl`. Done in #2176: `app/common/intl.tsx` is a 253-line binding over preact
  context with 25 importers. Note what it does **not** implement, because it constrains R2: `{name}`
  interpolation and paired `<tag>` rich text only, with no ICU plural, select, typed arguments or
  apostrophe quoting. All 24 catalogs stay within that today
- [x] Delete the `react`/`react-dom` aliases, `@preact/compat` and the `paths` entries in
  `tsconfig.json`. Done in #2176
- [x] Collapse the compiler pipelines to one. Done in #2178: `ts-loader`, `@swc/jest`, `@swc/core`
  and enzyme are gone, `babel-loader` is the only compiler, and `jest.config.mjs` hands the same
  `.babelrc.js` to `babel-jest`. The same PR dropped the module/nomodule dual build, so output is
  `[name].mjs` only and `.js` is a server-side alias (the alias comment in
  `backend/app/rest/api/webfiles.go`)
- [ ] Evaluate **preact + htm over native ESM**, and treat the answer as likely no. It is not the
  cheap item it looks: dropping the JSX transpile drops type-checked markup, since the `jsx:
  react-jsx` and `jsxImportSource: preact` options in `tsconfig.json` is what types templates today
  and `htm` tagged literals are opaque to `tsc`. Rewriting every `.tsx` with a type-safety
  regression is the same objection the Rejected section uses against a vanilla rewrite. Unbundling
  136 or more source files plus 24 locale chunks into individually fetched modules also multiplies a
  cold load's request count against the 20/s `/web/` bucket that `addFileServer` applies, and
  because `cacheControl` sends `max-age=3600, no-cache` with an ETag even a warm cache revalidates
  each one. It additionally collides with the legacy `.js` contract, which promises module-free
  bytes

**What actually happened**: the first four items landed within two days of this document being
written, on 21 August. Only the htm evaluation is open, which retires the original framing that this
path "spends the expensive `react-intl` card on the option that does not remove npm". That card is
already played, and it cost less than estimated.

**The revised objection**: the devDependency target did not materialise. This document predicted a
fall from 82 to roughly 20; the figure after the collapse is 59. What survives is not compilation
but **gates and machinery**: eslint, stylelint, prettier, size-limit, `@formatjs/cli` and the
webpack plugin set. That reframes both paths. The remaining question is no longer how the code is
compiled, it is what verifies it.

## Path B: server-rendered, #825

- [ ] Serve dynamic HTML from a **new route**, and not just outside `/web/`. It must also sit
  outside `/api/v1`, because `apiCSPMiddleware` replaces the CSP on everything mounted there with
  `default-src 'none'; sandbox; frame-ancestors 'none'`, from `rest/image_headers.go`, which would
  sandbox the widget document and forbid framing it at all. The global `securityHeadersMiddleware`
  is what supplies `frame-ancestors` from `ALLOWED_HOSTS`, so it is also R1's enforcement point.
  Note its `form-action 'none'`: htmx uses XHR and is unaffected, but a progressive-enhancement
  design built on real form submissions is blocked by the existing policy
- [ ] Render anonymous-first, then hydrate. Budget this as a **second full render of the tree**, not
  a patch: user identity threads through every node. `prepVotes` in `store/service/service.go` sets
  each comment's `Vote` from the requester and strips the voter map, `alterCommentCached` blanks
  `User.IP` for non-admins, and on top sit the edit window, own-comment delete, admin controls and
  hidden users. The backend already shows the shape: `/find` keys its cache on `URLKeyWithUser`
  (`URLKeyWithUser`, used by `findCommentsCtrl`), one entry per user with a separate `admin!!` key.
  An HTML cache fragments the same way, so the shared-cache benefit only pays for logged-out readers
- [ ] Attach the existing auth via `htmx:configRequest` on fragment requests
- [ ] Keep client-side: embed script, auth popups, composer, collapse and hidden-user state,
  optimistic votes, and the **three subscription flows**. Email, Telegram and RSS
  (`comment-form/__subscribe-by-email/`, `__subscribe-by-telegram/`, `__subscribe-by-rss/`) are each
  multi-step and stateful (send code, enter code, confirm; QR plus poll), backing seven private
  endpoints in `routes()` plus the `GET /qr/telegram` route
- [ ] Keep the height shim on every swap. The iframe has no intrinsic height; the parent sizes it
  from `postMessage({height})` (`updateIframeHeight` in `utils/post-message.ts`, consumed in
  `embed.ts`). The ResizeObserver that feeds it already exists (the `ResizeObserver` in `root.tsx`,
  plus the dropdown one in `auth.hooks.ts`), so this is code to carry forward rather than write
- [ ] Serve the `?selfClose` stub from the new route. OAuth builds its return URL from the widget
  document's own path (`oauth.tsx` builds `origin + pathname + '?selfClose'`) and `iframe.ejs`
  closes the popup on arrival. Move the document and the return URL moves with it, so the new route
  inherits the self-closing stub and must not cache that response
- [ ] Thread `?site=` through every fragment URL. `matchSiteID` rejects an empty `site` param on
  every private and admin request, so a fragment that omits it gets a 403 rather than a 401
- [ ] Join the public response cache's flush scopes. Comment and info responses are cached keyed by
  `Scopes(siteID, URL)` and flushed on writes (`findCommentsCtrl` and `infoCtrl`); a dynamic HTML
  route outside those scopes serves stale comments after every post
- [ ] Move i18n server-side except timezone-local `formatDate`/`formatTime`. This is a from-zero
  build: `backend/app/templates` holds five templates today, four email plus one error page, and no
  catalog machinery. Go's `text/template` has no plural support either, so the current no-ICU
  position has to be stated as a rule or implemented on both sides
- [ ] Decide what happens to the two host-DOM scripts, `last-comments` (`last-comments.tsx`) and
  `counter` (`counter.ts`), which write into the host page rather than into the iframe. Cross-origin
  server rendering is available to them through CORS fragments (the backend runs `corsMiddleware()`
  unless `ProxyCORS` is set, in `routes()`) or through an iframe, so this is a design choice, not a
  blocker

**Cost**: months to an opt-in parallel UI reads as a floor derived from the optimistic architecture,
and the optimistic architecture does not hold. Anonymous-first is forced rather than chosen, so the
fragment layer reproduces the entire authenticated tree rather than a delta; add the three
subscription flows and, if R1 is to be honoured, the upstream `Partitioned` work in `go-pkgz/auth`.
On one contributor the realistic figure is long enough that the plan's own warning applies to the
schedule and not only to the design. Deletes the entire npm toolchain. **Against it**: 2,400 lines
of the most stateful code get rewritten; a second HTML-fragment API surface becomes permanent
alongside the JSON one; and no comparable project does this (isso, utterances, waline and cusdis are
all client-rendered, giscus does SSR in JS, and comentario, the closest Go analogue, ships a
framework-free TS web component).

The net that rewrite would run against is materially better than when this was written: sixty-odd
tests rather than seven, and still growing. It is still not a substitute for the 426 unit cases,
which is a separate item below.

**The unresolved tension**: "deletes the entire npm toolchain" and "keep the composer, auth, votes,
collapse and the embed script client-side" pull against each other. That kept code is 2,400 lines of
TSX today and the composer leans on two custom-element packages and a polyfill. Without npm it
becomes hand-authored vanilla or htm modules, which is the same "unowned in-house framework" risk
the Rejected section uses to rule out a full vanilla rewrite, now applied to the most stateful code
in the repo. Path B is not costed until it says what the kept client code is written in.

## What neither path has answered yet

Missing from both costings, and each one can move the verdict.

- **A definition of done for "no npm".** Which files and jobs actually go: the node stages in
  `Dockerfile`, the five `ci-frontend.yml` jobs, the pnpm steps in `release.yml`, husky and
  lint-staged. And what replaces each gate, or which loss is accepted
- **A browser-support floor.** the browserslist query in `.babelrc.js` targets `defaults, not IE 11,
  not samsung 12` today. Without a transpiler that becomes the literal floor and source syntax ships
  as written, which matters for native CSS nesting and JSON import attributes
- **A runtime-dependency inventory.** For each of `preact`, `redux`, `redux-thunk`, `clsx`,
  `lodash-es/isEqual`, `node-emoji` and the three custom elements: is there a usable ESM build, what
  is the licence, and is it vendored into `backend/app/webassets/assets` or dropped. The two the
  plan itself proposes, `htm` for Path A and `htmx` for Path B, belong in the same inventory with
  the same columns, plus what each costs the CSP. "Vendor the custom elements" understates them:
  `markdown-toolbar` mutates textarea selections and `text-expander` emits its own events and lazily
  loads the emoji data, so the tags rendering is not the same as the behaviour working
- **A security section.** Messages are still posted with `'*'` in both directions
  (`postMessageToParent` and `postMessageToIframe` in `post-message.ts`), so both ends check the
  sender rather than the target: #2197 made the parent ignore anything that did not come from a
  frame it created, and the child side checks `isFromParent`. The child cannot do better, because
  the host page is whatever site embeds the widget and `ALLOWED_HOSTS` is what constrains that,
  server side through `frame-ancestors`. The parent can: it builds the iframe URL from `BASE_URL`,
  so it knows the widget origin and could both pass that as `targetOrigin` instead of `'*'` and
  compare `event.origin` alongside `event.source`. That hardening is available and not done. Beyond
  the channel, `dangerouslySetInnerHTML` on comment text and preview (`comment.tsx` and
  `comment-form.tsx`) relies entirely on server-side sanitisation in `store.Comment`, and any new
  route needs its CSP decided
- **The integrator compatibility contract beyond the JS API.** The documented `/web` URLs pinned by
  `documentedWebPaths` in `e2e/webfiles_test.go`, the `remark_config.components` loader, and the
  legacy `.js` names, which must stay module-free. `/web` is also an overlay rather than one
  directory: a `--web-root` file wins and only a missing name falls through to the embedded assets
  (`webFileSystem` in `webfiles.go`, mounted by `addFileServer`), so moving a page between the two
  sources silently changes whether an operator can override it
- **Translation tooling after node.** Only `@formatjs/cli` is genuinely npm-bound; the downstream
  `tasks/*.js` scripts are dependency-free node over `extracted-messages/messages.json`. Note the
  failure mode: extraction matches the identifiers `defineMessages`, `FormattedMessage` and
  `intl.formatMessage` rather than an import source, so moving strings into Go templates extracts
  zero keys with a zero exit code, after which `generateDictionary.js` deletes them from all 24
  catalogs
- **A replacement strategy for the 426 unit cases**, naming what e2e cannot reach: the store, the
  fetcher, cookie handling and the intl parser. Two whole flows sit here rather than in the e2e net:
  Telegram auth and subscription have no browser route at all, which the e2e suite's own backlog
  note records, and RSS subscription is covered only for whether its control renders. Path B keeps
  all three subscription flows, so it inherits all three gaps
- **Whether dependency-free node scripts are allowed to stay.** The goal bans the installed
  dependency tree and the build and test machinery, but `tasks/checkTranslation.js` and its siblings
  need only node built-ins. Nothing in the goal as stated decides whether they survive as scripts or
  have to be ported to Go, and the answer changes the size of the i18n item
- **A build-output acceptance matrix.** Each generated HTML page, the dynamic catalogue and emoji
  payloads, the copied images, the stripped `data-testid` attributes and the compressed size budgets
  are today the emergent result of separate webpack rules and plugins rather than one replaceable
  step. Without a statement of what the output must contain, a replacement cannot be checked against
  anything
- **Size budgets and the dev loop.** `.size-limit.js` and `webpack-dev-server` both disappear with
  npm; a Go test over the embedded FS covers the first, `make rundev` plus `--web-root` the second
- **Storage partitioning semantics.** Collapse, hidden users, sort, draft and email all live in
  widget-origin `localStorage`, so on a multi-site install "persists across reload" becomes per top
  site
- **Sequencing against #2128**, which rewrites the CSS layer on the assumption that hashed class
  names stay. It is the one piece of work in flight that Task 3 collides with, and settling the
  class-name direction first is what stops that work being redone

## If Path B is chosen

Ship it as an opt-in parallel UI (`remark_config.ui: 'v2'` on a separate endpoint), time-boxed, with
a stated date at which one of the two is deleted. With one active frontend contributor the realistic
failure is not picking the wrong stack, it is carrying two half-finished ones.

## Rejected

- **Full vanilla rewrite**: the render layer becomes an unowned in-house framework
- **lit / alpine / solid / petite-vue**: none removes more tooling than Path A, each trades a known
  3 kB library for a less-known one
- **Packaging widgets for npm (#1715)**: the widget must run in an iframe on the remark42 origin for
  OAuth popups; importing it into a host bundle breaks auth. A types-only package is a separate
  question and is not covered by that objection
- **`@remark42/api`**: removed in #2172. It had no OAuth method and no consumers.
  `frontend/packages/` is gone with it, and #2197 removed the workspace root outright, leaving a
  single manifest and lockfile under `frontend/apps/remark42`
