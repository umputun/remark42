import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { type JsonStore, readJson, updateJson, writeJson } from './json-store.ts';

/** A storage that holds what it is given, so a test needs no browser and no cleanup between cases. */
function store(initial: Record<string, string> = {}): JsonStore & { items: Record<string, string> } {
  const items = { ...initial };

  return {
    items,
    getItem: (key: string) => (key in items ? items[key] : null),
    setItem: (key: string, value: string) => {
      items[key] = value;
    },
  };
}

/** A storage that refuses, which is what a browser with site data switched off gives. */
function refusingStore(): JsonStore {
  return {
    getItem: () => {
      throw new Error('denied');
    },
    setItem: () => {
      throw new Error('denied');
    },
  };
}

describe('readJson', () => {
  it('reads back what was written', () => {
    assert.deepEqual(readJson(store({ k: '{"a":1}' }), 'k'), { a: 1 });
  });

  it('returns null for a key that was never written', () => {
    assert.equal(readJson(store(), 'k'), null);
  });

  it('reads a stored null as null, the same as nothing', () => {
    assert.equal(readJson(store({ k: 'null' }), 'k'), null);
  });

  it('reads the scalars and arrays JSON allows, not only objects', () => {
    assert.equal(readJson(store({ k: '1' }), 'k'), 1);
    assert.equal(readJson(store({ k: '"s"' }), 'k'), 's');
    assert.equal(readJson(store({ k: 'false' }), 'k'), false);
    assert.deepEqual(readJson(store({ k: '[]' }), 'k'), []);
  });

  // anything on the page can write this key. the caller gets nothing instead of a throw, and the
  // report is what says the value was unreadable
  it('returns null and reports when the value is not JSON', (t) => {
    const errors = t.mock.method(console, 'error', () => {});

    assert.equal(readJson(store({ k: 'asdas' }), 'k'), null);
    assert.equal(readJson(store({ k: '"{:"""' }), 'k'), null);
    assert.equal(errors.mock.callCount(), 2);
  });

  it('returns null and reports when the storage itself refuses', (t) => {
    const errors = t.mock.method(console, 'error', () => {});

    assert.equal(readJson(refusingStore(), 'k'), null);
    assert.equal(errors.mock.callCount(), 1);
  });
});

describe('writeJson', () => {
  it('stores an object as JSON', () => {
    const s = store();

    writeJson(s, 'k', { a: 1 });
    assert.equal(s.items.k, '{"a":1}');
  });

  it('stores an empty object and an empty array distinguishably', () => {
    const s = store();

    writeJson(s, 'o', {});
    writeJson(s, 'a', []);
    assert.equal(s.items.o, '{}');
    assert.equal(s.items.a, '[]');
  });

  // everything kept here is a preference, and losing one is not worth taking down the caller
  it('reports instead of throwing when the storage refuses', (t) => {
    const errors = t.mock.method(console, 'error', () => {});

    assert.doesNotThrow(() => writeJson(refusingStore(), 'k', { a: 1 }));
    assert.equal(errors.mock.callCount(), 1);
  });

  it('reports instead of throwing on a value JSON cannot represent', (t) => {
    const errors = t.mock.method(console, 'error', () => {});
    const circular: Record<string, unknown> = {};

    circular.self = circular;
    assert.doesNotThrow(() => writeJson(store(), 'k', circular));
    assert.equal(errors.mock.callCount(), 1);
  });
});

describe('updateJson', () => {
  it('writes into a key that held nothing', () => {
    const s = store();

    updateJson(s, 'k', { a: 1 });
    assert.equal(s.items.k, '{"a":1}');
  });

  it('spreads an object over what is stored', () => {
    const s = store({ k: '{"x":1}' });

    updateJson(s, 'k', { y: 2 });
    assert.deepEqual(JSON.parse(s.items.k), { x: 1, y: 2 });
  });

  it('lets the new value win on a key both carry', () => {
    const s = store({ k: '{"x":1}' });

    updateJson(s, 'k', { x: 2 });
    assert.deepEqual(JSON.parse(s.items.k), { x: 2 });
  });

  it('appends to a stored array instead of replacing it', () => {
    const s = store({ k: '[1,2,3]' });

    updateJson(s, 'k', [4, 5]);
    assert.deepEqual(JSON.parse(s.items.k), [1, 2, 3, 4, 5]);
  });

  it('applies a function to what is stored', () => {
    const s = store({ k: '[3,4]' });

    updateJson<unknown[]>(s, 'k', (data) => [1, 2, ...(data ?? [])]);
    assert.deepEqual(JSON.parse(s.items.k), [1, 2, 3, 4]);
  });

  it('hands a function null when the key held nothing', () => {
    const s = store();
    let received: unknown = 'not called';

    updateJson<unknown[]>(s, 'k', (data) => {
      received = data;
      return [];
    });
    assert.equal(received, null);
  });

  // which branch runs is decided by the argument, not by what is stored, so the two disagreeing is
  // an error instead of a silent replacement. the shapes are never the same key in practice
  it('throws when an array is appended to a key holding an object', () => {
    assert.throws(() => updateJson(store({ k: '{"x":1}' }), 'k', [1]), /error on update JSON/);
  });

  it('throws on a scalar, which it has no way to merge', () => {
    assert.throws(() => updateJson(store(), 'k', 1), /error on update JSON/);
    assert.throws(() => updateJson(store(), 'k', 'a'), /error on update JSON/);
  });

  it('spreads an object over a key holding a scalar, since only the argument decides', () => {
    const s = store({ k: '1' });

    updateJson(s, 'k', { a: 1 });
    assert.deepEqual(JSON.parse(s.items.k), { a: 1 });
  });
});
