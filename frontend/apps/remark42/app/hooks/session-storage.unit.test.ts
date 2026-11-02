import { test } from 'node:test';
import assert from 'node:assert/strict';

import { readStored, resolveUpdate, writeStored, type SessionStore } from './session-storage.ts';

/**
 * The decoder behind useSessionStorage. Every case here is a shape a browser cannot be asked to
 * produce on purpose: storage holding something that is not JSON, holding a literal null, or
 * holding nothing at all. Each decides whether a caller gets its initial value or something it
 * cannot use, and the widget reads this on mount, so getting it wrong shows up as a panel that
 * renders empty instead of as an error.
 */

/** A plain object is enough: the module takes the store instead of reaching for the global. */
function store(initial: Record<string, string> = {}): SessionStore & { written: Record<string, string> } {
  const data = { ...initial };

  return {
    written: data,
    getItem: (key: string) => (key in data ? data[key] : null),
    setItem: (key: string, value: string) => {
      data[key] = value;
    },
  };
}

test('returns the stored value', () => {
  assert.deepEqual(readStored(store({ k: '{"a":1}' }), 'k', { a: 0 }), { a: 1 });
});

test('returns the initial value when the key was never written', () => {
  assert.deepEqual(readStored(store(), 'k', { a: 0 }), { a: 0 });
});

test('returns the initial value when the stored value is not json', () => {
  // the case that matters: a half-written or hand-edited entry must not propagate as a value
  assert.deepEqual(readStored(store({ k: 'not json {' }), 'k', { a: 0 }), { a: 0 });
});

test('returns a stored falsy value instead of the initial one', () => {
  // false and 0 are values a caller meant to store, and the read must not treat them as absence
  assert.equal(readStored(store({ k: 'false' }), 'k', true), false);
  assert.equal(readStored(store({ k: '0' }), 'k', 7), 0);
});

test('returns a stored null instead of the initial value', () => {
  // null parses, so it is a value. Only a missing key means "nothing to restore"
  assert.equal(readStored(store({ k: 'null' }), 'k', 'initial'), null);
});

test('returns undefined when there is no initial value and nothing stored', () => {
  assert.equal(readStored(store(), 'k'), undefined);
});

test('writes in the form the read expects to find', () => {
  const s = store();

  writeStored(s, 'k', { a: 1 });

  assert.equal(s.written.k, '{"a":1}');
  assert.deepEqual(readStored(s, 'k'), { a: 1 });
});

test('resolves an update given as a value', () => {
  assert.equal(resolveUpdate(2, 1), 2);
});

test('resolves an update given as a function of the current value', () => {
  assert.equal(
    resolveUpdate((current: number) => current + 1, 1),
    2
  );
});
