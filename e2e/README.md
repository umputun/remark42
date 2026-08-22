# End-to-end tests

Drives the widget in a real browser through playwright-go against a remark42 built from this checkout.

The import path is `github.com/mxschmitt/playwright-go`, which is what the module declares even though its repository is [playwright-community/playwright-go](https://github.com/playwright-community/playwright-go). Do not rewrite it to match the repository URL: the versions that carry the matching path cannot install their driver.

## Prerequisites

- Docker with compose, which the suite shells out to
- A Go toolchain matching `e2e/go.mod`
- Network access on the first run: the Playwright driver and the browsers are downloaded into the user cache directory, `~/.cache/…` on Linux and `~/Library/Caches/…` on macOS, and that download is the slowest part of a cold run

## Running

```
make e2e
```

The suite brings the compose stack up itself when it does not find one already answering, and tears it down again afterwards. To keep the containers between runs, start them first:

```
make e2e-up
make e2e
make e2e-down
```

Run from `e2e/`; the compose path is relative to it. A single test:

```
cd e2e && go test -tags=e2e -run TestComment_ReplyNestsUnderItsParent -v ./...
```

`make e2e-ui` runs with a visible browser and leaves the stack up. The env vars behind it:

- `E2E_HEADLESS=false` shows the browser and slows it to 50ms a step
- `E2E_KEEP=1` leaves the containers running afterwards, which only matters when the suite brought them up itself
- `E2E_DEBUG=1` logs every HTTP response of status 400 or above
- `E2E_BROWSERS=chromium` narrows the engines the rendering tests use, which is the quickest way to shorten a local run

Rate-limit responses are logged whether or not `E2E_DEBUG` is set, because they surface otherwise as unexplained locator timeouts.

The build tag keeps these out of `go test ./...`; nothing runs without `-tags=e2e`.

## When something fails

A failed test writes a Playwright trace to `e2e/traces/`, which CI uploads as an artifact. Nothing else writes one, so a run carrying the artifact is a run with a failure to look at. Open one with `npx playwright show-trace e2e/traces/<name>.zip`.

CI does not retry a failing test. The suite is young enough that a failure is evidence about the suite itself, and a retry is what would hide an intermittent regression. `E2E_RUN_ID` stamps the threads a run uses with the CI run id, so a thread url in a trace names the run it came from. It does not carry the data across: the stack is disposable, and a local run against the same id gets those urls on an empty database.

Beyond that: `docker compose -f compose-e2e-test.yml logs` for the server side, and mailpit's web UI on <http://127.0.0.1:8025> for anything email.

A failing test also logs whatever the browser wrote to its console, which is where a failed request or a widget-side error shows up. Those are context only. What does fail a test on its own is an uncaught exception in the page and a rate-limit response, both of which otherwise corrupt a run silently: a widget that throws while rendering still satisfies most assertions here, and a refused `/auth/status` renders as a signed-out reader.

## The stack the suite runs against

The suite refuses a running stack that was not brought up from the sources under test. Every checkout builds the image tag the compose file names, so a stack from another worktree, or from this one before an edit, answers on these ports and passes every readiness probe while serving code nobody is looking at.

`e2e/stamp.sh` digests the content of `backend`, `frontend`, `Dockerfile` and `docker-init.sh`; compose puts it in the container's environment and the suite reads it back. It digests content and not `HEAD`, so a commit touching only the suite does not invalidate a stack. `make e2e-up` stamps the same way, so a stack started by hand is accepted. On a mismatch the failure says to run `make e2e-down`.

Before pushing, `cd e2e && go vet -tags=e2e ./...` and `golangci-lint run --build-tags=e2e --config ../backend/.golangci.yml`. CI runs both, and neither is covered by a plain `go vet ./...` because of the build tag.

## The stack

`compose-e2e-test.yml` at the repository root runs six services, each bound to the loopback interface since it holds a known secret and an admin shared id:

- **remark42** on `:8080`, with the dev oauth2 provider on `:8084`, anonymous and email sign-in
- **remark42-shortedit** on `:8081`, with `EDIT_TIME=15s` and anonymous sign-in only, since the dev oauth2 provider's port is fixed at 8084 and cannot be published twice. It exists so the expired-edit path is observable without holding a test open for the default five minutes
- **remark42-adminedit** on `:8082`, with `ADMIN_EDIT=true` and the same short window, for the unlimited window an admin is supposed to get
- **remark42-jwtheader** on `:8083`, with `AUTH_SEND_JWT_HEADER=true`, where the token arrives in a header instead of a cookie and the frontend has to keep it itself
- **remark42-noauth** on `:8085`, with no auth provider at all, which the widget has to say something about, and with `ALLOWED_HOSTS` set to its own address so it doubles as the instance that refuses to be framed elsewhere
- **remark42-anonvote** on `:8086`, with `ANON_VOTE` and the `VOTES_IP` it depends on, since the default configuration turns an anonymous vote down
- **host-site** on `:8090`, an nginx serving `e2e/hostsite/`, which is a page on an origin the widget is not served from. Every other host page here is served by remark42 itself, so without it the separate-domain setup the manuals describe is never exercised. `post.html` embeds the main instance; `restricted.html` embeds the one whose `ALLOWED_HOSTS` names only itself, which is the refusal case
- **mailpit** on `:8025`, which catches the email-auth verification message and the subscription token for the suite to read back

The main instance enables the notify module (`NOTIFY_USERS=email`). Without it `email_notifications` is false in the config, the widget never renders the subscribe control, and the whole subscribe, confirm and unsubscribe flow is unreachable from a browser.

The four remark42 instances beyond the first offer anonymous sign-in only, for the reason `remark42-shortedit` does: the dev oauth2 provider binds a port fixed at 8084 and cannot be published twice. `remark42-adminedit` gets its admin from `ADMIN_SHARED_ID`, since the anonymous provider derives the user id from the name and the id for a chosen name can be written into the compose file ahead of time.

Three settings exist for the tests and not for realism, and each is there for a reason:

- `REMARK_URL` uses a **hostname**, not `127.0.0.1`. The dev oauth2 server binds whatever host it reads out of `REMARK_URL` (`localBindAddr` in go-pkgz/auth), and a loopback bind inside a container cannot be published. The browser maps the names back with `--host-resolver-rules`.
- `UPDATE_LIMIT=100`, because the default of 0.5 updates a second rejects any test that posts twice in a row.
- The suite paces its own calls to `/auth/`, which is limited to two requests a second by a bare literal at `backend/app/rest/api/rest.go:242` and not by a setting. See `pauseForAuthLimit`.

## What this suite cannot reach

Every service here speaks http, and nothing in it holds a certificate. Any behaviour the browser gates on the page protocol is therefore invisible: a cookie the widget writes with `Secure`, anything keyed on `window.location.protocol`, and the whole third-party cookie form of `SameSite=None; Secure; Partitioned`, which is the only one browsers still accept from an embedded frame.

That is not hypothetical. `setAuthCookie` prefixed its cookies with `__Host-` on any https page, so a real deployment stored `__Host-JWT` while the backend looked for `JWT`; it survived because the prefix comes from the page protocol and every test and the dev server run on http. Fixed in #2197, under a second suite pinned to an https page, because this one cannot show it.

There is a second trap waiting for whoever gives the stack TLS and then tries to prove the third-party case. Playwright's own default `--disable-features` argument carries `ThirdPartyStoragePartitioning`, and it beats both `--test-third-party-cookie-phaseout` and `--block-third-party-cookies` passed through `Args`. A run configured that way keeps an ordinary third-party cookie exactly as it would with no flags at all, so it proves nothing while looking like it proved something. The lever is `IgnoreDefaultArgs` on the launch options: drop that default entry and re-supply `--disable-features` without that one feature. Measured on a cross-site https embed:

| | ordinary third-party cookie | `Partitioned` cookie |
|---|---|---|
| Playwright defaults | kept | kept |
| partitioning left enabled | dropped | stored, with its partition key |

So a blocking run has to assert a control before anything it reports can be believed: set an ordinary `SameSite=None` cookie from inside the widget frame and require the browser to drop it. If it survives, the run is not blocking anything.

None of that reaches the widget's own storage fallback, which the auth panel offers as `comments.html` when `IS_THIRD_PARTY && !IS_STORAGE_AVAILABLE`. `IS_STORAGE_AVAILABLE` stays true even with partitioning properly enforced, because Chromium partitions `localStorage` instead of denying it, so the probe behind that constant never throws and the condition cannot fire. That case needs WebKit, not a Chromium flag.

The practical consequence is for the cross-origin case in `crossorigin_test.go`, which asserts rendering and deliberately not signing in. Give the stack TLS and signing in there becomes testable, and the assertion that matters is **the reload**: the widget holds its token in memory for the life of a page, so a case that signs in and posts without reloading passes while persistence is entirely broken.

## Isolation

Each test gets its own comment thread from a query string on the demo page, since the demo page passes `window.location.href` as `remark_config.url` and remark42 keys comments by it. A per-run id keeps threads apart from those an earlier run left behind.

Thread URLs deliberately keep the underscores a test name carries, since collapse persistence keys off the page url and a url containing an underscore is the case worth covering.

## Browsers

The `iframe_test.go` group runs in Chromium, Firefox and WebKit. It is about rendering and not logic: the widget holds the frame hidden until its document reports itself inited, and the opaque canvas that guards against is a WebKit behaviour, so Chromium alone would not exercise it.

Those tests address the server as `127.0.0.1` and not by name, since `--host-resolver-rules` is a Chromium flag and they need no dev oauth2, which is the only reason the hostname exists.

Everything else runs in Chromium alone, for the same reason inverted: those tests sign in, sign-in needs the dev oauth2 provider, and reaching it by name from the host is Chromium-only. Running them in the other engines would mean putting the suite back inside the compose network.

## Selectors

The production bundle strips `data-testid`, so tests use what ships: the stable class hooks the widget keeps outside CSS modules (`.auth-button`, `.auth-submit`, `.comment-actions`, `.sort-picker`, `.preloader`), `title` attributes on icon-only controls, and visible text. Three shapes are worth knowing:

- `.auth` only exists while signed out, so waiting on it hangs after sign-in. `widget()` waits on the comment form, which is present either way.
- The production build hashes every css-module class name to a short opaque id, so a component's own class is not something a test can hold. `role` is: the footer is `[role="contentinfo"]` and the edit countdown is `[role="timer"]`.
- Comments render through an IntersectionObserver, so one below the fold is an empty `article` with no text in it. That makes any absence assertion written as a text filter pass whether the comment is gone or merely off screen; count articles instead, which is what `articleCount` is for.
- Collapsing a thread hides the comment text, so a locator filtered by that text stops matching the element under test. `TestThread_CollapsePersistsAcrossReload` anchors on the comment's id instead.
