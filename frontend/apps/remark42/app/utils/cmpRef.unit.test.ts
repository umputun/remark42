import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { cmpRef } from './cmpRef.ts';

describe('cmpRef', () => {
  // the reducers rely on identity: a state object that comes back as the same reference is what
  // tells the store nothing changed
  it('returns the first reference when both hold the same values', () => {
    const state = ['a', 'b'];
    const next = ['a', 'b'];

    assert.equal(cmpRef(state, next), state);
    assert.notEqual(cmpRef(state, next), next);
  });

  it('returns the second reference when the values differ', () => {
    const state = { url: 'https://example.com/post', count: 3 };
    const next = { url: 'https://example.com/post', count: 4 };

    assert.equal(cmpRef(state, next), next);
  });

  it('returns the same reference when given it twice', () => {
    const state = { count: 3 };

    assert.equal(cmpRef(state, state), state);
  });
});
