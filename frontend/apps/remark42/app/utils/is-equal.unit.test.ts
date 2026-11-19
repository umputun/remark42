import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { isEqual } from './is-equal.ts';

describe('isEqual', () => {
  it('compares primitives by value', () => {
    const equal: [unknown, unknown][] = [
      [1, 1],
      ['a', 'a'],
      [true, true],
      [null, null],
      [undefined, undefined],
      [NaN, NaN],
      [0, -0],
    ];
    for (const [a, b] of equal) {
      assert.equal(isEqual(a, b), true, `${String(a)} and ${String(b)}`);
    }

    const different: [unknown, unknown][] = [
      [1, 2],
      [1, '1'],
      [0, false],
      [null, undefined],
      ['', 0],
    ];
    for (const [a, b] of different) {
      assert.equal(isEqual(a, b), false, `${String(a)} and ${String(b)}`);
    }
  });

  it('calls two structures equal when every nested value is', () => {
    const post = { url: 'https://example.com/post', count: 3, read_only: false, first_time: '', last_time: '' };

    assert.equal(isEqual(post, { ...post }), true);
    assert.equal(isEqual(['a', 'b'], ['a', 'b']), true);
    assert.equal(isEqual({ ids: ['a', { nested: [1, 2] }] }, { ids: ['a', { nested: [1, 2] }] }), true);
    assert.equal(isEqual({}, {}), true);
    assert.equal(isEqual([], []), true);
  });

  it('sees a difference at any depth', () => {
    assert.equal(isEqual({ count: 3 }, { count: 4 }), false);
    assert.equal(isEqual({ ids: ['a', { nested: [1, 2] }] }, { ids: ['a', { nested: [1, 3] }] }), false);
    assert.equal(isEqual(['a', 'b'], ['b', 'a']), false, 'order of an array matters');
    assert.equal(isEqual({ a: 1, b: 2 }, { b: 2, a: 1 }), true, 'order of keys does not');
  });

  // equal key counts with different names, which a comparison walking only the first value's
  // keys and reading a missing one as undefined would let through
  it('sees a key that only one side has', () => {
    assert.equal(isEqual({ a: 1 }, { a: 1, b: undefined }), false);
    assert.equal(isEqual({ a: undefined }, { b: undefined }), false);
    assert.equal(isEqual(['a'], ['a', 'b']), false);
    assert.equal(isEqual(['a', 'b'], ['a']), false);
  });

  it('never calls an array and an object equal', () => {
    assert.equal(isEqual([], {}), false);
    assert.equal(isEqual(['a'], { 0: 'a' }), false);
    assert.equal(isEqual({ 0: 'a' }, ['a']), false);
    assert.equal(isEqual({}, null), false);
    assert.equal(isEqual(null, {}), false);
  });

  // neither date has own keys, so a comparison by keys alone would call them equal and the
  // reducer would keep the old one
  it('calls a value that is not JSON equal only to itself', () => {
    const date = new Date(0);

    assert.equal(isEqual(date, date), true);
    assert.equal(isEqual(new Date(0), new Date(1)), false);
    assert.equal(isEqual(new Date(0), new Date(0)), false);
    assert.equal(isEqual(new Map(), new Map()), false);
    assert.equal(isEqual(Object.create(null), Object.create(null)), false);
    assert.equal(isEqual({}, new Date(0)), false);
  });
});
