---
worth: yes
where:
added: 2026-08-19
---
# @remark42/api is still published on npm with no deprecation notice

The npm package is live at 13 versions, all alphas. Twelve went out on 2022-06-19 and the last on
2022-07-11, and `dist-tags.latest` points at `0.6.0-alpha.12`. Anyone who finds it installs a 2022
alpha that cannot do OAuth by construction, with nothing on the listing saying it is abandoned.

Removing it from this repository does not change the npm listing. The fix is one command:

```
npm deprecate @remark42/api "use the REST API directly, see https://remark42.com/docs/contributing/api/"
```

Only akellbl4 can run it. The registry lists him as the sole maintainer and `pavel@mineev.me`
published every version, so neither umputun nor paskal has the credentials.

Worth doing because the package's own auth client exposes only anonymous, email and telegram, so a
consumer who adopts it hits a wall that no amount of reading the source explains. The REST API is the
integration surface, which is the position umputun already gave in #1383.
