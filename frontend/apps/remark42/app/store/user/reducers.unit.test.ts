import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import type { BlockedUser, User } from '../../common/types.ts';

import { bannedUsers, hiddenUsers, user } from './reducers.ts';
import {
  USER_BAN,
  USER_BANLIST_SET,
  USER_HIDE,
  USER_HIDELIST_SET,
  USER_SET,
  USER_SUBSCRIPTION_SET,
  USER_UNBAN,
  USER_UNHIDE,
} from './types.ts';

const other = { type: 'SOMETHING/ELSE' } as never;

function person(id: string): User {
  return { id, name: id } as User;
}

function blocked(id: string): BlockedUser {
  return { id, name: id, time: '' } as BlockedUser;
}

describe('user', () => {
  it('starts with nobody signed in', () => {
    assert.equal(user(undefined, other), null);
  });

  it('takes the user it is given', () => {
    const someone = person('github_1');

    assert.equal(user(null, { type: USER_SET, user: someone }), someone);
  });

  it('clears the user on sign out', () => {
    assert.equal(user(person('github_1'), { type: USER_SET, user: null }), null);
  });

  it('records the subscription without disturbing the rest of the user', () => {
    const next = user(person('github_1'), { type: USER_SUBSCRIPTION_SET, payload: true });

    assert.equal(next?.email_subscription, true);
    assert.equal(next?.id, 'github_1');
  });

  // a subscription arriving for nobody is not an empty user: writing one would sign the reader in
  it('ignores a subscription when nobody is signed in', () => {
    assert.equal(user(null, { type: USER_SUBSCRIPTION_SET, payload: true }), null);
  });

  it('keeps the same state for an action it does not handle', () => {
    const state = person('github_1');

    assert.equal(user(state, other), state);
  });
});

describe('bannedUsers', () => {
  it('starts with nobody banned', () => {
    assert.deepEqual(bannedUsers(undefined, other), []);
  });

  it('takes the list it is given', () => {
    const list = [blocked('a')];

    assert.equal(bannedUsers([blocked('b')], { type: USER_BANLIST_SET, list }), list);
  });

  it('puts a newly banned user at the front', () => {
    const next = bannedUsers([blocked('a')], { type: USER_BAN, user: blocked('b') });

    assert.deepEqual(
      next.map((u) => u.id),
      ['b', 'a']
    );
  });

  // the same reference, not merely an equal list: the panels re-render on identity
  it('does not list an already banned user twice', () => {
    const state = [blocked('a')];

    assert.equal(bannedUsers(state, { type: USER_BAN, user: blocked('a') }), state);
  });

  it('removes an unbanned user', () => {
    const next = bannedUsers([blocked('a'), blocked('b'), blocked('c')], { type: USER_UNBAN, id: 'b' });

    assert.deepEqual(
      next.map((u) => u.id),
      ['a', 'c']
    );
  });

  it('keeps the same state when unbanning someone who was not banned', () => {
    const state = [blocked('a')];

    assert.equal(bannedUsers(state, { type: USER_UNBAN, id: 'zzz' }), state);
  });

  it('leaves the previous list untouched', () => {
    const state = [blocked('a')];

    bannedUsers(state, { type: USER_BAN, user: blocked('b') });
    bannedUsers(state, { type: USER_UNBAN, id: 'a' });
    assert.deepEqual(
      state.map((u) => u.id),
      ['a']
    );
  });
});

describe('hiddenUsers', () => {
  it('starts with nobody hidden', () => {
    assert.deepEqual(hiddenUsers(undefined, other), {});
  });

  it('takes the record it is given', () => {
    const payload = { a: person('a') };

    assert.equal(hiddenUsers({}, { type: USER_HIDELIST_SET, payload }), payload);
  });

  it('hides a user under their own id', () => {
    const next = hiddenUsers({}, { type: USER_HIDE, user: person('github_1') });

    assert.deepEqual(Object.keys(next), ['github_1']);
  });

  it('keeps the others hidden', () => {
    const next = hiddenUsers({ a: person('a') }, { type: USER_HIDE, user: person('b') });

    assert.deepEqual(Object.keys(next).sort(), ['a', 'b']);
  });

  it('reveals a hidden user', () => {
    const next = hiddenUsers({ a: person('a'), b: person('b') }, { type: USER_UNHIDE, id: 'a' });

    assert.deepEqual(Object.keys(next), ['b']);
  });

  it('keeps the same state when revealing someone who was not hidden', () => {
    const state = { a: person('a') };

    assert.equal(hiddenUsers(state, { type: USER_UNHIDE, id: 'zzz' }), state);
  });

  // the ids are user ids, so they are whatever an auth provider hands over, and the record is a
  // plain object: `state[id]` would find Object.prototype's own members for a user id like
  // "toString" and take the widget down a branch that copies and deletes nothing
  it('is not fooled by a user id that names something on Object.prototype', () => {
    const state = { a: person('a') };

    assert.equal(hiddenUsers(state, { type: USER_UNHIDE, id: 'toString' }), state);
    assert.equal(hiddenUsers(state, { type: USER_UNHIDE, id: 'constructor' }), state);
  });

  it('leaves the previous record untouched', () => {
    const state = { a: person('a') };

    hiddenUsers(state, { type: USER_HIDE, user: person('b') });
    hiddenUsers(state, { type: USER_UNHIDE, id: 'a' });
    assert.deepEqual(Object.keys(state), ['a']);
  });
});
