---
worth: yes
where: frontend/pnpm-lock.yaml:61
added: 2026-08-11
---
# frontend pnpm overrides still admit vulnerable js-yaml

Three open Dependabot alerts against `frontend/pnpm-lock.yaml` survive because the pnpm override
floors in that file are set below the patched versions.

| override (line) | current floor | resolved version | advisory | patched in |
|---|---|---|---|---|
| `js-yaml@>=3.0.0 <4.0.0` (:61) | `>=3.14.2` | 3.15.0 (:4798, :12669) | GHSA-5p4m-2wfm-xmqj (high) | 3.15.1 |
| `js-yaml@>=4.0.0 <5.0.0` (:62) | `>=5.0.0` | 5.2.0 (:4802, :12674) | GHSA-pm4m-ph32-ghv5 (high) | 5.2.2 |
| | | | GHSA-724g-mxrg-4qvm (medium) | 5.2.1 |

GHSA-5p4m-2wfm-xmqj is the same advisory PR #2141 closed for `site/`, patched there by moving to
js-yaml 3.15.1. It remains open against the frontend lockfile, which is still on 3.15.0. Verified
against the repository's Dependabot alerts, not inferred from version numbers.

`.github/dependabot.yml` sets `open-pull-requests-limit: 0` on every npm directory, so no version
update PR will ever reach these. Security updates bypass that limit, which is why #2141 existed at
all, but the override floors pin the resolution regardless of what the bot proposes.

Fix: raise both floors and re-lock.

```yaml
js-yaml@>=3.0.0 <4.0.0: '>=3.15.1 <4.0.0'
js-yaml@>=4.0.0 <5.0.0: '>=5.2.2 <6.0.0'
```

Both js-yaml lines are build and test time only in the frontend (3.x arrives via
`@istanbuljs/load-nyc-config`), so this is alert hygiene rather than a shipped vulnerability.

Separately, and not covered here: `frontend/pnpm-lock.yaml` carries a wider queue of open alerts
(brace-expansion, undici, fast-uri, postcss, svgo, webpack-dev-server, body-parser). Those were not
audited and may or may not be reachable.
