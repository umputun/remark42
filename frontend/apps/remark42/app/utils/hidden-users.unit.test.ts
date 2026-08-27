import assert from 'node:assert/strict';
import { afterEach, describe, it } from 'node:test';

import type { User } from '../common/types.ts';

import { parseHiddenUsers } from './hidden-users.ts';

function captureErrors() {
  const real = console.error;
  const seen: unknown[][] = [];

  console.error = (...args: unknown[]) => {
    seen.push(args);
  };

  return { seen, restore: () => (console.error = real) };
}

let capture: ReturnType<typeof captureErrors> | undefined;

afterEach(() => {
  capture?.restore();
  capture = undefined;
});

/**
 * The record is keyed by user id and the caller indexes it directly, so anything of another shape
 * reaching it would be spread over the comment list. The key is readable and writable by anything
 * else on the page, which is why the shape is checked instead of trusted.
 */
describe('parseHiddenUsers', () => {
  it('returns the users that were stored', () => {
    const stored = { github_1: { id: 'github_1', name: 'someone' } as User };

    assert.deepEqual(parseHiddenUsers(JSON.stringify(stored)), stored);
  });

  it('returns nothing hidden when the key was never written', () => {
    assert.deepEqual(parseHiddenUsers(null), {});
  });

  it('returns nothing hidden for an empty record', () => {
    assert.deepEqual(parseHiddenUsers('{}'), {});
  });

  // an array is an object as far as typeof is concerned, and indexing one by user id would yield
  // undefined for every reader, so it is rejected instead of passed on
  it('rejects a list, which would index as nothing hidden for anyone', () => {
    assert.deepEqual(parseHiddenUsers('[]'), {});
    assert.deepEqual(parseHiddenUsers('["github_1"]'), {});
  });

  it('rejects a stored null, which typeof also calls an object', () => {
    assert.deepEqual(parseHiddenUsers('null'), {});
  });

  it('rejects the scalars JSON allows', () => {
    assert.deepEqual(parseHiddenUsers('"string"'), {});
    assert.deepEqual(parseHiddenUsers('42'), {});
    assert.deepEqual(parseHiddenUsers('true'), {});
  });

  it('returns nothing hidden and reports when the value is not JSON', () => {
    capture = captureErrors();

    assert.deepEqual(parseHiddenUsers('"{:"""'), {});
    assert.equal(capture.seen.length, 1);
  });

  it('reads an empty string as nothing stored instead of as broken JSON', () => {
    capture = captureErrors();

    assert.deepEqual(parseHiddenUsers(''), {});
    assert.deepEqual(capture.seen, []);
  });
});
