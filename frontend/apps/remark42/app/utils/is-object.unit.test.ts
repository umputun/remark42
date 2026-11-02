import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { isObject } from './is-object.ts';

describe('isObject', () => {
  // null is typeof 'object' and an array is too, and both would otherwise be spread into a
  // settings object as if they carried named fields
  it('rejects null and arrays, which typeof calls objects', () => {
    for (const value of [null, [], [1, 2, 3]]) {
      assert.equal(isObject(value), false, JSON.stringify(value));
    }
  });

  it('rejects primitives', () => {
    for (const value of [undefined, 0, 1, '', 'string', true, false]) {
      assert.equal(isObject(value), false, String(value));
    }
  });

  it('rejects a function', () => {
    assert.equal(
      isObject(() => undefined),
      false
    );
  });

  it('accepts plain objects, empty or not', () => {
    for (const value of [{}, { a: 1 }, { a: 1, b: 2 }]) {
      assert.equal(isObject(value), true, JSON.stringify(value));
    }
  });

  // a prototype-less object has no toString, so anything that stringifies the value to build a
  // message throws before the assertion is reached
  it('accepts objects that are not plain', () => {
    const cases: [string, unknown][] = [
      ['Error', new Error()],
      ['Date', new Date()],
      ['Map', new Map()],
      ['null-prototype', Object.create(null)],
    ];

    for (const [label, value] of cases) {
      assert.equal(isObject(value), true, label);
    }
  });
});
