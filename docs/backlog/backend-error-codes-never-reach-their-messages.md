---
worth: yes
where: frontend/apps/remark42/app/common/fetcher.ts
added: 2026-10-17
---
# the backend's error codes never reach their messages

The catalogue carries a message for every backend error code, `errors.1` to `errors.20` ("You cannot
vote for your own comment.", "Comments on this page are read-only." and so on), but a reader never
sees any of them. The fetcher loses the code on two paths:

- A status with an entry in `errorMessages` (401, 403, 409, 429, 500) becomes a `RequestError`
  carrying that status, and the body is never read. The backend sends a blocked user (`ErrUserBlocked`), a
  read-only post (`ErrReadOnly`) and a refused data-removal token (`ErrNoAccess`) as 403 with the code in
  the body (`rest_private.go`, `admin.go`), so all three read as `errors.forbidden`.
- Any other status, in practice 400, has its JSON body parsed and thrown as a plain object.
  `extractErrorMessageFromResponse` looks a code up only on a `RequestError`, so these read as
  `errors.0`, "Something went wrong. Please try again a bit later." `errorUtils.unit.test.ts` pins
  this down in "does not read the code off a plain error object".

Fix: have the fetcher read a JSON error body first and, when it carries a non-zero numeric `code`
with a catalogue entry, throw a `RequestError` with that code, falling back to the HTTP status
otherwise. Code 0 is `ErrInternal`, which the backend uses for many unrelated failures, so letting it
win would replace today's specific 403, 409 and 500 messages with the generic one. Change that unit
test to expect the code's own message.
