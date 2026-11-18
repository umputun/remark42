import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { parseSorting } from './sorting.ts';

/**
 * `default_sort` comes from the host page's `remark_config` as a query parameter, so anything can
 * arrive; only the picker's own values may become the default order.
 */
describe('parseSorting', () => {
  it('accepts every sort order', () => {
    for (const value of ['-score', '+score', '-time', '+time', '-active', '+active', '-controversy', '+controversy']) {
      assert.equal(parseSorting(value), value);
    }
  });

  it('rejects anything else', () => {
    for (const value of [undefined, '', 'time', ' -time', '-TIME', 'newest', 1]) {
      assert.equal(parseSorting(value), undefined);
    }
  });
});
