---
worth: later
where: frontend/apps/remark42/app, e2e/
added: 2026-08-30
---
# What keeps jest cases out of the browser suite, and what would let them go

Removing a jest case is only safe when a named e2e case asserts the same behaviour, assertion for
assertion. Working through the suites on that rule, the cases that stay divide into a few recurring
causes, and most of them are a gap in the browser suite rather than something it cannot reach.

This is the list to work from. It is deliberately not acted on in the same change as the deletions:
each item is an e2e case to write, and the jest cases it releases can then go with it.

## Selectors the browser cannot reach

`tasks/babel-plugin-remove-test-id.js` removes `data-testid` from anything but a test build, so a
browser case cannot select by it. Only one of the elements these jest cases assert through is
actually stuck behind that:

- `comments-counter` in `profile.spec.tsx`, asserted in four cases, is
  `<div className={styles.container} data-testid="comments-counter">`. Its only class is hashed by
  the CSS modules, so nothing outside the bundle can name it. Giving it a stable class, the way
  `.auth-button`, `.auth-submit`, `.comment-actions` and `.sort-picker` are kept outside the
  modules for this reason, is what makes the counter assertable in a browser

Three others are already reachable and simply have no browser case written yet, which is a smaller
job than it looked:

- `spinner` renders `clsx('spinner', styles.root, …)` with `role="presentation"`
- `preloader` renders `clsx('preloader', className)` with `aria-label="Loading..."`
- `comment-actions-additional` renders `clsx('comment-actions-additional', …)`, so the order of the
  admin actions can be asserted from a browser case today

## Transient states nothing waits on

- buttons disabled while a vote request is in flight (`comment-votes.spec.tsx`, three cases)
- the loading indicator in the telegram subscription panel
- the spinner between pages of the profile list
- the preloader that must not reappear after a load-more click

Each needs a browser case that holds the request open, which the suite already knows how to do:
`TestVote_FailureShowsAnErrorAndRestoresTheScore` blocks a route and asserts the optimistic state
before releasing it.

## Absence with no positive control

The browser suite asserts what appears far more readily than what does not. Cases kept for this:

- the Reply action gone in a read-only thread. `TestComment_ReadOnlyThreadTakesTheFormAway`
  asserts the comment form is gone and says nothing about the action
- Hide absent on a reader's own comment, Delete absent on another reader's
- the verification icon absent on an unverified user
- the auth dropdown starting closed

## Configurations no instance runs

- `email_notifications` and `telegram_notifications` off. The stack covers
  `show_rss_subscription` and `show_email_subscription`, which are different settings
- upvote-only voting, and voting hidden altogether
- the controversy tooltip, which needs a comment with controversy in it

## Values inside an element, where the browser case only waits for the element

- the edit countdown's remaining seconds. `TestComment_EditWithinTheDeadline` waits for the timer
  and `TestComment_EditExpiresAfterTheDeadline` waits for it to go, so a blank timer passes both
- the telegram link's full `https://t.me/<bot>/?start=<token>`, where the browser helper parses out
  the `start` parameter and never looks at the host or the bot name

## Error branches that are not the one the browser drives

- the generic fallback message for an unrecognised code. The browser suite fulfils a 409 and
  asserts the catalogued string, which never enters that branch
- a failed check or unsubscribe being cleared by a later success, in the telegram panel

## What is not worth moving

Call counts and call arguments (`api.telegramSubscribe` called once, `getUserComments` called with
a page size) assert how a client method was used, not what a reader gets. The browser suite asserts
the request and the response instead, which is the better test of the same thing, so these stay in
jest or go away on their own when the code changes.
