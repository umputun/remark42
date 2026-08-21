import { siteId, url } from 'common/settings';
import { LS_COLLAPSE_KEY } from 'common/constants';
import { setItem as localStorageSetItem, getItem as localStorageGetItem } from 'common/local-storage';
import type { Comment } from 'common/types';

/**
 * collapsed comment ids, keyed by site and then by page url.
 *
 * nested rather than joined into one string: any separator can also occur inside a site id or
 * a url, and then "site_url_id" cannot be taken apart again. it is what made a page whose url
 * contains an underscore lose its collapsed threads on reload, and what let one page's entries
 * be read or deleted as another's
 */
type CollapsedComments = Record<string, Record<string, Comment['id'][]>>;

function getFromLocalStorage(): CollapsedComments {
  const stored: unknown = JSON.parse(localStorageGetItem(LS_COLLAPSE_KEY) || '{}');

  // anything of another shape, including the flat list this used to keep, reads as empty:
  // collapsed threads are a view preference, so re-expanding them once costs the reader
  // nothing worth a migration
  if (typeof stored !== 'object' || stored === null || Array.isArray(stored)) {
    return {};
  }
  return stored as CollapsedComments;
}

/**
 * returns the ids of the collapsed comments on the current page
 */
export const getCollapsedComments = (): Comment['id'][] => {
  const ids = getFromLocalStorage()[siteId]?.[url];

  // the check above covers the top level only, and a leaf of another type would reach the
  // reducer, which reduces over it
  return Array.isArray(ids) ? ids : [];
};

/**
 * @param siteId site id
 * @param url url of the page with comments
 * @param info ids of the comments collapsed on that page
 */
export const saveCollapsedComments = (siteId: string, url: string, info: Comment['id'][]): void => {
  const stored = getFromLocalStorage();
  const site = { ...stored[siteId], [url]: info };
  const next = { ...stored, [siteId]: site };

  // an empty list is dropped rather than stored: every page a reader collapses and expands
  // again would otherwise keep an entry of its own for good
  if (info.length === 0) {
    delete site[url];
  }
  if (Object.keys(site).length === 0) {
    delete next[siteId];
  }

  localStorageSetItem(LS_COLLAPSE_KEY, JSON.stringify(next));
};
