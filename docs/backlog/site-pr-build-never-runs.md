---
worth: yes
where: .github/workflows/ci-site.yml:24
added: 2026-08-11
---
# site/** pull requests get no build validation

`ci-site.yml` declares a `pull_request` trigger on `site/**` (lines 12-15), but the only job that
installs and builds is gated:

```yaml
if: github.ref == 'refs/heads/master' || startsWith(github.ref, 'refs/tags/')
```

On a pull request `github.ref` is `refs/pull/N/merge`, so `build` skips, and `merge` (`needs: build`)
and `deploy` skip with it. The trigger is dead: it produces a run in which every job is skipped, which
renders in the checks list the same way a pass does. No required status checks are configured on
master either, so nothing else catches it.

Consequence: a `site/yarn.lock` that fails `yarn --frozen-lockfile` first fails on the master run
after merge, which is the same run that rebuilds and deploys remark42.com. `site/Dockerfile:5-6` is
the only place the lockfile is exercised, and it runs post-merge.

Fix: add a PR-only job mirroring `.github/workflows/ci-build.yml`'s existing pattern, using
`docker/build-push-action` with `context: ./site`, `load: true`, no `outputs:`, and no
`docker/login-action`.

Constraint any fix must preserve: the current gate exists because `build` pushes to ghcr using
`secrets.PKG_TOKEN`, which must not run for pull requests, since fork PRs receive no secrets.
Relaxing the `if` on its own is not sufficient and would break fork PRs.

Surfaced while reviewing PR #2141 (a dependabot js-yaml lockfile bump), where all three site jobs
reported as skipped and nothing verified the lockfile before merge. `site/**` PRs are mostly
generated lockfile bumps, which is exactly the class a frozen-lockfile install catches and a human
reviewer cannot.
