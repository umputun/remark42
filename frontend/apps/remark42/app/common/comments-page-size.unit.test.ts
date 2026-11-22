import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { DEFAULT_COMMENTS_PAGE_SIZE, parseCommentsPageSize } from './comments-page-size.ts';

/**
 * `comments_page_size` comes from the host page's `remark_config` as a query parameter, so anything
 * can arrive; `0` is a valid value and loads the whole thread at once.
 */
describe('parseCommentsPageSize', () => {
  it('takes a whole number from zero up', () => {
    assert.equal(parseCommentsPageSize('0'), 0);
    assert.equal(parseCommentsPageSize('1'), 1);
    assert.equal(parseCommentsPageSize('250'), 250);
  });

  it('falls back to the default for anything else', () => {
    for (const value of [undefined, '', ' ', '-1', '1.5', 'ten', 'Infinity', '1000000000000000000000', 20]) {
      assert.equal(parseCommentsPageSize(value), DEFAULT_COMMENTS_PAGE_SIZE);
    }
  });
});
