import { siteId, url } from 'common/settings';

import { getCollapsedComments, saveCollapsedComments } from './utils';

jest.mock('common/settings', () => ({ siteId: 'remark', url: 'https://example.com/a_b/' }));

/**
 * The record's shape, its nesting and what it does with anything else in storage live in
 * `collapse-store.ts` and are covered there. What only a browser can show is that these two reach
 * the real `localStorage`, and that the reader takes the site and url from settings.
 */
describe('collapsed comments storage', () => {
  afterEach(() => {
    localStorage.clear();
  });

  it('restores for the current page what it saved', () => {
    saveCollapsedComments(siteId, url, ['a', 'b']);

    expect(getCollapsedComments()).toEqual(['a', 'b']);
  });
});
