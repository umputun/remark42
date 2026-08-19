---
worth: yes
where: frontend/apps/remark42/package.json:38
added: 2026-08-19
---
# preact cannot be bumped past 10.6.2 without code changes

`preact` is pinned exactly at `10.6.2` in two places, `frontend/apps/remark42/package.json:38` and
the `pnpm.overrides` block in `frontend/package.json`, so it never moves on a lockfile refresh. The
current 10.x release is 10.29.8, which is 93 releases ahead. It looks like routine drift and is not.

Type errors climb steadily with the version, all of them `TS2786` (cannot be used as a JSX
component) and `TS2322` (prop type not assignable), concentrated in components that take props
through the `react` alias to `@preact/compat@^17.1.1`:

| preact | `pnpm type-check` errors |
|---|---|
| 10.10.6 | 0 |
| 10.11.3 | 4 |
| 10.12.1 | 4 |
| 10.13.2 | 11 |
| 10.15.1 | 11 |
| 10.22.1 | 18 |
| 10.29.8 | 78 |

The blocker is not the type layer. 10.10.6 type-checks, lints, builds and passes translation and
size checks, and still fails 10 tests across 2 suites:

- `<Select/> should highlight select on focus` expects `select_focused` on the element after focus
  and gets `select root md select_md`
- `<Auth/> should remove spaces in the first/last position in username` blurs the input and expects
  the trimmed value, receiving the untrimmed one
- plus `should leave spaces in the middle of username`, `should send email and then verify forms`,
  `should show validation error for token` and `Telegram auth should go through the auth flow`

Both readable failures are the same shape: a state change that no longer produces a DOM update, so
this is a reconciliation or event-handling difference rather than a typing artefact. The identical
tree on 10.6.2 passes all 320 tests, so the failures are caused by the bump.

Fix: treat as a real upgrade. The `react`/`react-dom` alias to `@preact/compat@^17.1.1` is the thing
to settle first, since that package mirrors the React 17 API and its own 18.3.2 release is what
current preact expects; the component prop typings and the affected tests follow from that choice.
