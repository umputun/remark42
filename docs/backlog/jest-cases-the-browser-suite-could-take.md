---
worth: later
where: frontend/apps/remark42/app, e2e/
added: 2026-08-30
---
# What keeps jest cases out of the browser suite, and what would let them go

Removing a jest case is only safe when a named e2e case asserts the same behaviour, assertion for
assertion. Working through the suites on that rule, the cases that stay divide into a few recurring
causes, and most of them are a gap in the browser suite rather than something it cannot reach.

This was the list to work from. The items marked **done** below have their browser case and their
jest cases have gone with them; what is left is what is still open, and the reasons it is open are
worth more than the list itself.

## Selectors the browser cannot reach

`tasks/babel-plugin-remove-test-id.js` removes `data-testid` from anything but a test build, so a
browser case cannot select by it. Only one of the elements these jest cases assert through is
actually stuck behind that:

- **done.** `comments-counter` in `profile.spec.tsx` is
  `<div className={styles.container} data-testid="comments-counter">`. Its only class is hashed by
  the CSS modules, so nothing outside the bundle can name it. Giving it a stable class, the way
  `.auth-button`, `.auth-submit`, `.comment-actions` and `.sort-picker` are kept outside the
  modules for this reason, is what makes the counter assertable in a browser

Two others are already reachable and simply have no browser case written yet, which is a smaller
job than it looked:

- `spinner` renders `clsx('spinner', styles.root, …)` and is still open. It is the *between-pages*
  indicator at `profile.tsx:162-165`, shown in place of the Load more button once a page is being
  fetched, so it is reachable only when `comments` is not null. The profile's initial load and its
  failure are a different element, the `Preloader` at `profile.tsx:221`, and that one is covered by
  `TestProfile_LoadingAndFailureStates`; the two are easy to conflate and are not the same thing
- **done for the telegram panel.** `preloader` renders `clsx('preloader', className)`, asserted
  by `TestTelegramSub_ThePanelSaysItIsWorking`
- **done.** `comment-actions-additional` carries a stable class, and the order is asserted by
  `TestComment_AdminActionsKeepTheirOrder`

## Transient states nothing waits on

- **done.** buttons disabled while a vote request is in flight, `TestVote_BothButtonsAreDisabledWhileTheVoteIsInFlight`
- **done.** the loading indicator in the telegram subscription panel
- **done.** the profile's loading and error states
- the preloader that must not reappear after a load-more click

Each needs a browser case that holds the request open, which the suite already knows how to do:
`TestVote_FailureShowsAnErrorAndRestoresTheScore` blocks a route and asserts the optimistic state
before releasing it.

## Absence with no positive control

The browser suite asserts what appears far more readily than what does not. Cases kept for this:

- **done.** the Reply action gone in a read-only thread, now asserted in
  `TestComment_ReadOnlyThreadTakesTheFormAway`, which posts a comment first so the assertion has
  something to be about
- **done.** Hide absent on a reader's own comment, Delete absent on another reader's,
  `TestComment_ActionsDependOnWhoseCommentItIs`
- **done.** the verification icon absent before an admin verifies
- the auth dropdown starting closed

## Configurations no instance runs

- **done.** `email_notifications` and `telegram_notifications` off, each instance being the
  other's negative case
- upvote-only voting, and voting hidden altogether
- the controversy tooltip, which needs a comment with controversy in it

## Values inside an element, where the browser case only waits for the element

- **done.** the edit countdown's value, `TestComment_EditCountdownCountsDown`
- the telegram link's full `https://t.me/<bot>/?start=<token>`, where the browser helper parses out
  the `start` parameter and never looks at the host or the bot name

## Error branches that are not the one the browser drives

- the generic fallback message for an unrecognised code. The browser suite fulfils a 409 and
  asserts the catalogued string, which never enters that branch
- **done.** a failed check or unsubscribe cleared by a later success, driven separately because
  the two handlers clear their own error

## What is not worth moving

Call counts and call arguments (`api.telegramSubscribe` called once, `getUserComments` called with
a page size) assert how a client method was used, not what a reader gets. The browser suite asserts
the request and the response instead, which is the better test of the same thing, so these stay in
jest or go away on their own when the code changes.


## What the static build would settle on its own

Thirty-four of the cases still in jest exist because of the build rather than because of the
behaviour: eighteen assert a hashed css-module class and sixteen select by a `data-testid` the
production bundle strips. With static css there is no hash to pin and the class in the source is
the class in the browser, so those assertions have nothing left to say and the browser can select
what ships. Covering them now would mean writing browser cases whose purpose disappears with the
build, which is why they are left here rather than done.
