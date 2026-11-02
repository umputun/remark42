import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { normalizeCollapsed, readCollapsed, writeCollapsed } from './collapse-store.ts';

/**
 * What comes back from storage is whatever is under the key, which is not always what this widget
 * wrote: anything else on the page can write it, and a hand-edited value can be any JSON at all.
 * Those are the cases here, and none of them can be produced through a browser on purpose.
 */
describe('normalizeCollapsed', () => {
  it('keeps a record of sites', () => {
    const stored = { site: { 'https://example.com/': ['a'] } };

    assert.deepEqual(normalizeCollapsed(stored), stored);
  });

  it('reads the flat list an older version kept as empty', () => {
    assert.deepEqual(normalizeCollapsed(['a', 'b']), {});
  });

  it('reads a missing value as empty', () => {
    assert.deepEqual(normalizeCollapsed(null), {});
  });

  it('reads a value of the wrong type as empty', () => {
    assert.deepEqual(normalizeCollapsed('a'), {});
    assert.deepEqual(normalizeCollapsed(42), {});
    assert.deepEqual(normalizeCollapsed(undefined), {});
  });
});

describe('readCollapsed', () => {
  const stored = { site: { 'https://example.com/a_b/': ['x', 'y'] } };

  it('finds the ids for a page', () => {
    assert.deepEqual(readCollapsed(stored, 'site', 'https://example.com/a_b/'), ['x', 'y']);
  });

  it('finds nothing for another page on the same site', () => {
    assert.deepEqual(readCollapsed(stored, 'site', 'https://example.com/other/'), []);
  });

  // the separator is the point: the keys are nested and never joined, so a url carrying the
  // character a joined key would use must not read as a boundary
  it('does not confuse a url containing an underscore with another key', () => {
    const two = { site: { 'https://example.com/a_b/': ['x'], 'https://example.com/a': ['y'] } };

    assert.deepEqual(readCollapsed(two, 'site', 'https://example.com/a_b/'), ['x']);
    assert.deepEqual(readCollapsed(two, 'site', 'https://example.com/a'), ['y']);
  });

  it('finds nothing for an unknown site', () => {
    assert.deepEqual(readCollapsed(stored, 'other-site', 'https://example.com/a_b/'), []);
  });

  it('reads a leaf of the wrong type as nothing collapsed', () => {
    const broken = { site: { 'https://example.com/': 'x' } } as unknown as typeof stored;

    assert.deepEqual(readCollapsed(broken, 'site', 'https://example.com/'), []);
  });
});

describe('writeCollapsed', () => {
  it('records a page on a site that had none', () => {
    assert.deepEqual(writeCollapsed({}, 'site', 'https://example.com/', ['a']), {
      site: { 'https://example.com/': ['a'] },
    });
  });

  it('leaves the other pages of the site alone', () => {
    const stored = { site: { '/one': ['a'] } };

    assert.deepEqual(writeCollapsed(stored, 'site', '/two', ['b']), { site: { '/one': ['a'], '/two': ['b'] } });
  });

  it('leaves the other sites alone', () => {
    const stored = { one: { '/p': ['a'] } };

    assert.deepEqual(writeCollapsed(stored, 'two', '/p', ['b']), { one: { '/p': ['a'] }, two: { '/p': ['b'] } });
  });

  it('does not mutate what it was given', () => {
    const stored = { site: { '/one': ['a'] } };

    writeCollapsed(stored, 'site', '/two', ['b']);
    assert.deepEqual(stored, { site: { '/one': ['a'] } });
  });

  it('drops a page whose last thread was expanded', () => {
    const stored = { site: { '/one': ['a'], '/two': ['b'] } };

    assert.deepEqual(writeCollapsed(stored, 'site', '/one', []), { site: { '/two': ['b'] } });
  });

  it('drops the site once its last page goes', () => {
    const stored = { site: { '/one': ['a'] }, other: { '/p': ['b'] } };

    assert.deepEqual(writeCollapsed(stored, 'site', '/one', []), { other: { '/p': ['b'] } });
  });

  it('stores nothing at all instead of an empty husk', () => {
    assert.deepEqual(writeCollapsed({}, 'site', '/one', []), {});
  });

  it('replaces the ids for a page instead of appending to them', () => {
    const stored = { site: { '/one': ['a', 'b'] } };

    assert.deepEqual(writeCollapsed(stored, 'site', '/one', ['c']), { site: { '/one': ['c'] } });
  });
});
