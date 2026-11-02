import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { collapsedThreads } from './reducers.ts';
import { THREAD_RESTORE_COLLAPSE, THREAD_SET_COLLAPSE } from './types.ts';

/**
 * Which threads are collapsed is the one piece of reading position remark42 keeps, and this reducer
 * is where the stored list and the reader's clicks meet. The jest suite of the same name asserts
 * that the action creator dispatched, which reaches neither of the two cases below.
 */
describe('collapsedThreads', () => {
  it('starts with nothing collapsed', () => {
    assert.deepEqual(collapsedThreads(undefined, { type: THREAD_SET_COLLAPSE, id: 'a', collapsed: true }), {
      a: true,
    });
  });

  it('collapses a thread without disturbing the others', () => {
    const state = { a: true, b: false };

    assert.deepEqual(collapsedThreads(state, { type: THREAD_SET_COLLAPSE, id: 'c', collapsed: true }), {
      a: true,
      b: false,
      c: true,
    });
  });

  it('expands a thread by storing false, not by forgetting it', () => {
    const state = { a: true };
    const next = collapsedThreads(state, { type: THREAD_SET_COLLAPSE, id: 'a', collapsed: false });

    assert.deepEqual(next, { a: false });
    assert.equal('a' in next, true);
  });

  it('leaves the previous state untouched', () => {
    const state = { a: true };

    collapsedThreads(state, { type: THREAD_SET_COLLAPSE, id: 'b', collapsed: true });
    assert.deepEqual(state, { a: true });
  });

  // restore rebuilds the map from the stored ids instead of merging into it, so anything collapsed
  // before the restore lands is dropped -- which is what makes restoring twice idempotent
  it('rebuilds the whole map from the restored ids', () => {
    const next = collapsedThreads({ a: true }, { type: THREAD_RESTORE_COLLAPSE, ids: ['b', 'c'] });

    assert.deepEqual(next, { b: true, c: true });
  });

  it('restores nothing collapsed from an empty list', () => {
    assert.deepEqual(collapsedThreads({ a: true }, { type: THREAD_RESTORE_COLLAPSE, ids: [] }), {});
  });

  it('ignores an action it does not handle', () => {
    const state = { a: true };

    assert.equal(collapsedThreads(state, { type: 'SOMETHING/ELSE' } as never), state);
  });
});
