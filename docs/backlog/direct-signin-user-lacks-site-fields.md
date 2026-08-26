---
worth: maybe
where: frontend/apps/remark42/app/components/auth/auth.tsx:141
added: 2026-08-26
---
# email and telegram sign-in put a user in the store without its site-scoped fields

`GET /user?site=` is the only endpoint that fills the site-scoped parts of a user: `userInfoCtrl`
(`backend/app/rest/api/rest_private.go:236-249`) sets `Verified` and `EmailSubscription` from the store.
The `/auth/*` login handlers render `claims.User` instead, which is `token.User` from vendored
go-pkgz/auth (`backend/vendor/github.com/go-pkgz/auth/v2/token/user.go:22-34`). That struct has no
`email_subscription`, `verified`, `admin`, `block` or `paid_sub` field at all.

`auth.tsx:141` (email) and `auth.tsx:170` (telegram) dispatch that raw response through `setUser`, so the
store holds a user missing those fields until something calls `fetchUser` again. `oauthSignin` is not
affected: it resolves through `getUser()` and receives the enriched shape.

Two consequences seen so far, both cleared by a reload:

- `Dropdown` mounts its children only when opened, so `SubscribeByEmailForm` reads the missing flag on
  every open. An already-subscribed reader who signs in by email or telegram and opens the panel in the
  same page view is offered the Subscribe step. Submitting the same address returns 409, which the
  component turns into the subscribed state, so the usual path self-corrects; a reader whose subscription
  address differs from the address the form prefills does not get that 409.
- `user.admin` is undefined on the same path, so the read-only switch (`auth-panel.tsx:150`) and the
  settings block (`settings.tsx:143`) stay hidden for an admin who signed in that way.

Two candidate fixes, and choosing between them is why this is not done: route every direct sign-in
through `/user` after the auth call, or have the auth responses carry the site-scoped fields. The first
keeps one source for that data and costs a round trip.

Surfaced while investigating #2206, which reports a different symptom on the reload path and does not
reproduce.
