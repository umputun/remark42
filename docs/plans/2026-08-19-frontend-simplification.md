# Frontend simplification: shipping the widget as static HTML, CSS and JavaScript

The goal is a widget that ships without npm: no installed dependency tree, no bundler, no node in
the build, and static HTML, CSS and JavaScript as the output. This document plans that and nothing
else.

Two subjects that earlier revisions carried have been taken out. The server-rendered direction of
#825 is a different proposal and is not costed here; removing it from this document does not decide
against it. Multi-domain hosting is an authentication question that constrains this work at exactly
one point, noted in its place below, and is tracked separately so that a backend decision does not
block a build decision.

## Where this stands

The direction is settled. What was open was whether the verification could survive the move, and
that is now half answered.

The test suite has split in two. Pure logic, meaning anything that is a function over strings,
objects and dates, runs under `node --test` directly from source with nothing installed, and its CI
job installs nothing so that a test reaching for a dependency fails. What remains on jest is the
half that renders components into jsdom.

That retires the earlier framing, which held that dropping npm drops the whole unit suite and that
the only route was to close every gap in the e2e suite first. The question is now specifically about
DOM-rendering cases, and each has three possible answers: move the subject's logic out and test it
under node, cover the behaviour in the e2e suite, or accept the loss.

It also settles, by precedent, a question earlier revisions listed as open. Dependency-free node
scripts may stay, so the translation tooling written against node built-ins does not have to be
ported to Go to satisfy the goal.

### The extraction pattern

Value imports in the node layer write the extension in full and use a relative path, never the
`common/…` alias, because node resolves neither. Type-only imports erase and keep the ordinary
style. This is recorded in both `CLAUDE.md` files; it is repeated here because it decides whether a
given jest case can move at all. A case whose subject imports through the alias cannot move until
the subject does, which is why the split extracted the pure half of each module first.

The pattern is the reusable part. Pull the logic out of the module that touches the DOM, test that
half under node, and leave the thin DOM half to jest or to the e2e suite. It found several defects
on its first pass, which is the argument for continuing it.

## Standing requirement: the language must be selectable per page

The locale is read once from `remark_config.locale` and resolved to one catalogue. There is no
runtime switcher and no locale entry in the postMessage contract, so the language cannot change
during an instance's life, and a site with several language versions embeds each page with its own
locale.

Must survive any build change:

- `remark_config.locale` keeps selecting the language per instance
- every supported language keeps working. Adding one is not catalogue-only today: the language goes
  into a registry file, and generation then rewrites the loader from it. Preserve that
  registry-plus-generation shape or replace it deliberately
- the translation completeness check survives in some form
- date and time rendering keeps resolving in the viewer's timezone. The locale is not the viewer's,
  since it is configured and the server knows it too. Only the timezone is genuinely client-side, so
  the carve-out is timezone-local rendering and not a whole date-formatting layer

## Sequence: parity first, changes second, CSS excepted

The build change must reproduce current behaviour exactly, with no feature change, no markup change
and no visible difference, so that any regression it causes is attributable to the build and to
nothing else. Interleaving a rendering bug with a deliberate redesign in one diff is expensive to
untangle, and the widget's most stateful code is the composer and authentication, which is where it
is most expensive.

CSS is the exception and has to be correct on the first pass. Reproducing the hashed class names
exactly would freeze an accidental surface as though it had been designed, and renaming later breaks
every override written against the first set. Public names ship once, and #5 has been open since
2018 asking for stable ones.

One piece of work in flight collides with this: #2128 rewrites the CSS layer on the assumption that
hashed names stay. Settling the class-name direction first is what stops that work being redone.

## What replaces each npm gate

| gate | replacement | state |
|---|---|---|
| dependency-free unit tests | none needed | in place |
| jest | extract logic into the node layer, cover the rest in the e2e suite, accept the remainder | in progress |
| type checking | undecided | open |
| lint and style lint | undecided, and an accepted loss is a real option | open |
| size budgets | a Go test over the embedded filesystem | designed |
| translation completeness | only message extraction is bound to npm; everything downstream of it is already node built-ins | narrowed |

Type checking is the hard one, and the only gate with no route. Dropping the bundler without
answering it means either shipping untyped source or finding a checker that is not npm, and there is
no third option that keeps the current guarantee.

The translation failure mode is worth carrying forward because it is silent. Extraction matches the
identifiers the widget calls, not an import source, so moving strings into Go templates would
extract nothing with a zero exit code, after which generation deletes them from every catalogue.

## What the bundler still does

The list any no-bundler proposal has to answer for.

- transpiles TypeScript and JSX
- CSS modules, with nesting and with one `composes` that crosses a file boundary. That one has no
  plain-CSS equivalent and has to be flattened instead of translated
- code-splits the locale catalogues and the emoji data
- runs an autoprefixer pass against a browserslist query. Without a transpiler that query becomes the
  literal browser floor and source syntax ships as written
- builds the HTML entry pages, including the fallback page a reader is offered when the widget
  detects a third-party context with storage unavailable. This is the one point where multi-domain
  hosting constrains this work: that page went unbuilt from the January 2021 rewrite until #2197
  restored it, so it returned 404 for four years in exactly the configuration that needs it, and
  nothing in the e2e suite pins it. Any replacement build has to keep emitting it
- derives asset URLs. Since #2197 the public path is computed at runtime from the URL the bundle was
  loaded from, so a sub-path install resolves its own assets. A build with no bundler inherits that
  requirement instead of the fix, and #2224 is what it costs when it goes wrong
- strips test-only attributes through a local plugin, which nothing else replicates
- minifies
- substitutes build-time constants. The instance URL was the one with teeth, and #2198 moved that
  substitution to serve time in the Go file server, so a build with no bundler inherits a working
  mechanism and not a broken one

## What is already out

Worth stating because the widget build is now the last npm consumer in the repository.

- the documentation site is a Go static generator since #2179 and has no manifest at all
- the e2e suite has been a Go module since #2180. Its browser driver fetches a node runtime at test
  time, which is a fetched runtime and not a tracked dependency, so it survives whatever happens here
- the published API package and the shared frontend packages went with #2172
- static assets needing no processing are embedded in the Go binary since #2181, which is the working
  proof that the no-build serving path carries real files
- since #2197 there is one manifest and one lockfile. The release pipeline still runs pnpm to build
  the widget, and that step goes when the widget build does

## Still open

Each of these can move the answer.

- **A definition of done.** Which files and jobs actually go: the node stages in the image build, the
  remaining npm CI jobs, the pnpm steps in the release pipeline, and the git hooks. For each, the
  replacement or the accepted loss
- **Type checking**, the one gate with no route yet
- **A runtime-dependency inventory.** For every package the widget actually ships with: whether a
  usable ESM build exists, the licence, and whether it is vendored or dropped. Vendoring the custom
  elements understates them, because they mutate the textarea selection, emit their own events and
  lazily load data, so rendering the tags is not the same as the behaviour working
- **A build-output acceptance matrix.** The generated pages, the catalogue and emoji payloads, the
  copied images, the stripped attributes and the size budgets are today the emergent result of
  separate bundler rules and plugins. Without a statement of what the output must contain, a
  replacement cannot be checked against anything
- **The integrator contract.** The documented `/web` URLs, the component loader, and the legacy `.js`
  names, which must stay module-free. `/web` is an overlay: a `--web-root` file wins and only a
  missing name falls through to the embedded assets, so moving a page between the two sources
  silently changes whether an operator can override it
- **The development loop.** The dev server goes with npm, and `make rundev` with `--web-root`
  replaces it
- **Message-channel hardening**, available and not done. Both ends post with `'*'` and check the
  sender instead of the target. The parent builds the iframe URL itself, so it knows the widget
  origin and could pass it as `targetOrigin` and compare the origin alongside the source

## Rejected

- **A full vanilla rewrite**: the render layer becomes an unowned in-house framework
- **lit, alpine, solid, petite-vue**: none removes more tooling than this path, and each trades a
  known small library for a less-known one
- **preact with htm over native ESM**: dropping the JSX transpile drops type-checked markup, since
  the JSX pragma is what types templates today and tagged literals are opaque to the type checker.
  Unbundling every source file and locale chunk into individually fetched modules also multiplies a
  cold load's request count against the rate limit the file server applies, and the cache headers
  revalidate each one
- **Packaging the widget for npm (#1715)**: it must run in an iframe on the remark42 origin for OAuth
  popups, so importing it into a host bundle breaks authentication. A types-only package is a
  separate question
