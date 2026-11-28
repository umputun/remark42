import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { parseMaxShownComments } from './max-shown-comments.ts';

/**
 * `max_shown_comments` comes from the host page's `remark_config` as a query parameter, so anything
 * can arrive; only a positive whole number may limit the comments shown.
 */
describe('parseMaxShownComments', () => {
  it('accepts a positive whole number', () => {
    assert.equal(parseMaxShownComments('1'), 1);
    assert.equal(parseMaxShownComments('5'), 5);
    assert.equal(parseMaxShownComments('100'), 100);
  });

  it('rejects anything else', () => {
    for (const value of [undefined, '', ' ', '0', '-3', '2.5', 'ten', 'Infinity', 'NaN', 5]) {
      assert.equal(parseMaxShownComments(value), undefined, `${JSON.stringify(value)} was taken as a limit`);
    }
  });
});
