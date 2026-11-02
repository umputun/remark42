import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import type { User } from '../common/types.ts';

import { isUserAnonymous } from './isUserAnonymous.ts';

/**
 * Anonymous users are recognized by the `anonymous_` prefix their id carries, and several controls
 * are withheld from them. A signed-out reader has no user at all, which counts as anonymous too.
 */
describe('isUserAnonymous', () => {
  it('treats a missing user as anonymous', () => {
    assert.equal(isUserAnonymous(null), true);
  });

  it('recognizes an anonymous id by its prefix', () => {
    assert.equal(isUserAnonymous({ id: 'anonymous_1' } as User), true);
  });

  it('does not treat another provider as anonymous', () => {
    assert.equal(isUserAnonymous({ id: 'email_1' } as User), false);
  });

  it('matches the prefix only at the start of the id', () => {
    assert.equal(isUserAnonymous({ id: 'github_anonymous_1' } as User), false);
  });
});
