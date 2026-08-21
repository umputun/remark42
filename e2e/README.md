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

A failed test writes a Playwright trace to `e2e/traces/`, which CI uploads as an artifact whether or not the job went green. Open one with `npx playwright show-trace e2e/traces/<name>.zip`.

CI runs the suite through `gotestsum` and gives a failing test one rerun, so a test that fails and then passes leaves the job green. That is the case worth looking at: it is named in `rerun-report.txt`, uploaded beside the traces. Only attempts that failed leave a trace, and they do not overwrite each other, so a flake leaves exactly one to open. `make e2e` locally does not rerun anything, so a test red on a laptop and green in CI is a flake with a report to read rather than a disagreement.

Beyond that: `docker compose -f compose-e2e-test.yml logs` for the server side, and mailpit's web UI on <http://127.0.0.1:8025> for anything email.

Before pushing, `cd e2e && go vet -tags=e2e ./...` and `golangci-lint run --build-tags=e2e --config ../backend/.golangci.yml`. CI runs both, and neither is covered by a plain `go vet ./...` because of the build tag.

## The stack

`compose-e2e-test.yml` at the repository root runs three services, each bound to the loopback interface since it holds a known secret and an admin shared id:

- **remark42** on `:8080`, with the dev oauth2 provider on `:8084`, anonymous and email sign-in
- **remark42-shortedit** on `:8081`, with `EDIT_TIME=15s` and anonymous sign-in only, since the dev oauth2 provider's port is fixed at 8084 and cannot be published twice. It exists so the expired-edit path is observable without holding a test open for the default five minutes
- **mailpit** on `:8025`, which catches the email-auth verification message for the suite to read back

Three settings exist for the tests rather than for realism, and each is there for a reason:

- `REMARK_URL` uses a **hostname**, not `127.0.0.1`. The dev oauth2 server binds whatever host it reads out of `REMARK_URL` (`localBindAddr` in go-pkgz/auth), and a loopback bind inside a container cannot be published. The browser maps the names back with `--host-resolver-rules`.
- `UPDATE_LIMIT=100`, because the default of 0.5 updates a second rejects any test that posts twice in a row.
- The suite paces its own calls to `/auth/`, which is limited to two requests a second by a bare literal at `backend/app/rest/api/rest.go:242` rather than by a setting. See `pauseForAuthLimit`.

## Isolation

Each test gets its own comment thread from a query string on the demo page, since the demo page passes `window.location.href` as `remark_config.url` and remark42 keys comments by it. A per-run id keeps threads apart from those an earlier run left behind.

The thread URL carries no underscores on purpose: collapse persistence stores its localStorage keys as `siteID_url_commentID` and splits them on `_`, so an underscore anywhere in the page URL makes the entry unreadable on the next load.

## Browsers

The `iframe_test.go` group runs in Chromium, Firefox and WebKit. It is about rendering rather than logic: the widget holds the frame hidden until its document reports itself inited, and the opaque canvas that guards against is a WebKit behaviour, so Chromium alone would not exercise it.

Those tests address the server as `127.0.0.1` rather than by name, since `--host-resolver-rules` is a Chromium flag and they need no dev oauth2, which is the only reason the hostname exists.

Everything else runs in Chromium alone, for the same reason inverted: those tests sign in, sign-in needs the dev oauth2 provider, and reaching it by name from the host is Chromium-only. Running them in the other engines would mean putting the suite back inside the compose network.

## Selectors

The production bundle strips `data-testid`, so tests use what ships: the stable class hooks the widget keeps outside CSS modules (`.auth-button`, `.auth-submit`, `.comment-actions`, `.sort-picker`, `.preloader`), `title` attributes on icon-only controls, and visible text. Three shapes are worth knowing:

- `.auth` only exists while signed out, so waiting on it hangs after sign-in. `widget()` waits on the comment form, which is present either way.
- Comments render through an IntersectionObserver, so one below the fold is an empty `article` with no text in it. That makes any absence assertion written as a text filter pass whether the comment is gone or merely off screen; count articles instead, which is what `articleCount` is for.
- Collapsing a thread hides the comment text, so a locator filtered by that text stops matching the element under test. `TestThread_CollapsePersistsAcrossReload` anchors on the comment's id instead.
