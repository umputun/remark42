---
worth: yes
where: frontend/apps/remark42/app/common/fetcher.ts
added: 2026-10-17
---
# the backend's error codes never reach their messages

The catalogue carries a message for every backend error code, `errors.1` to `errors.20` ("You cannot
vote for your own comment.", "Comments on this page are read-only." and so on), but a reader never
sees any of them.

The backend sends those errors as status 400 with a JSON body such as `{"code": 13, ...}`. The fetcher
turns a status into a `RequestError` only when `errorMessages` has an entry for that status (401, 403,
409, 429); for a 400 it parses the body and throws the plain object. `extractErrorMessageFromResponse`
looks a code up only on a `RequestError`, so every one of these errors reads as `errors.0`,
"Something went wrong. Please try again a bit later." `errorUtils.unit.test.ts` pins this down in
"does not read the code off a plain error object".

Fix: have the fetcher wrap a JSON error body that carries a numeric `code` in a `RequestError` with
that code, and change that test to expect the code's own message.
