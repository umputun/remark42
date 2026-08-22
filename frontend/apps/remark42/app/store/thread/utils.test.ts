import { LS_COLLAPSE_KEY } from 'common/constants';
import { siteId, url } from 'common/settings';

import { getCollapsedComments, saveCollapsedComments } from './utils';

jest.mock('common/settings', () => ({ siteId: 'remark', url: 'https://example.com/a_b/' }));

function stored(): unknown {
  return JSON.parse(localStorage.getItem(LS_COLLAPSE_KEY) as string);
}

describe('collapsed comments storage', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('restores what it saved for the current page', () => {
    saveCollapsedComments(siteId, url, ['c1', 'c2']);

    expect(getCollapsedComments()).toEqual(['c1', 'c2']);
  });

  it('restores ids containing an underscore', () => {
    saveCollapsedComments(siteId, url, ['id_with_underscores']);

    expect(getCollapsedComments()).toEqual(['id_with_underscores']);
  });

  it('leaves another page of the same site untouched', () => {
    saveCollapsedComments(siteId, 'https://example.com/other/', ['other']);
    saveCollapsedComments(siteId, url, ['mine']);

    expect(getCollapsedComments()).toEqual(['mine']);
    expect(stored()).toMatchObject({ remark: { 'https://example.com/other/': ['other'] } });
  });

  // the shape these two exercise is what a single joined key cannot represent: the separator
  // occurs inside the values, so the pieces cannot be told apart again
  it('keeps a url that extends the current one with the separator', () => {
    saveCollapsedComments(siteId, `${url}_extra/`, ['deeper']);
    saveCollapsedComments(siteId, url, ['mine']);

    expect(getCollapsedComments()).toEqual(['mine']);
    expect(stored()).toMatchObject({ remark: { [`${url}_extra/`]: ['deeper'] } });
  });

  it('does not confuse a site and url pair with one split differently', () => {
    saveCollapsedComments('remark_https://example.com', '/a_b/', ['elsewhere']);
    saveCollapsedComments(siteId, url, ['mine']);

    expect(getCollapsedComments()).toEqual(['mine']);
    expect(stored()).toMatchObject({ 'remark_https://example.com': { '/a_b/': ['elsewhere'] } });
  });

  it('forgets the page once its last thread is expanded', () => {
    saveCollapsedComments(siteId, 'https://example.com/other/', ['other']);
    saveCollapsedComments(siteId, url, ['mine']);
    saveCollapsedComments(siteId, url, []);

    expect(getCollapsedComments()).toEqual([]);
    expect(stored()).toEqual({ remark: { 'https://example.com/other/': ['other'] } });
  });

  it('forgets the site once its last page is expanded', () => {
    saveCollapsedComments(siteId, url, ['mine']);
    saveCollapsedComments(siteId, url, []);

    expect(stored()).toEqual({});
  });

  it('reads as empty when the stored value is of another shape', () => {
    localStorage.setItem(LS_COLLAPSE_KEY, JSON.stringify(['remark_https://example.com/a_b/_c1']));

    expect(getCollapsedComments()).toEqual([]);
  });

  it('reads as empty when the stored value is not json', () => {
    localStorage.setItem(LS_COLLAPSE_KEY, '{oops');

    expect(getCollapsedComments()).toEqual([]);
  });

  it('starts a fresh entry when the stored value is not json', () => {
    localStorage.setItem(LS_COLLAPSE_KEY, '{oops');

    saveCollapsedComments(siteId, url, ['c1']);

    expect(getCollapsedComments()).toEqual(['c1']);
  });

  it('reads as empty when the entry for the page is not a list of ids', () => {
    localStorage.setItem(LS_COLLAPSE_KEY, JSON.stringify({ [siteId]: { [url]: 'c1' } }));

    expect(getCollapsedComments()).toEqual([]);
  });
});
