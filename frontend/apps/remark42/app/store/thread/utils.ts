import { siteId, url } from 'common/settings';
import { LS_COLLAPSE_KEY } from 'common/constants';
import { setItem as localStorageSetItem, getJsonItem } from 'common/local-storage';
import type { Comment } from 'common/types';

import { normalizeCollapsed, readCollapsed, writeCollapsed } from './collapse-store';

/**
 * Where the collapsed-threads record meets localStorage. The record's own shape and rules live in
 * collapse-store.ts, which carries the cases worth testing.
 */
function getFromLocalStorage() {
  // getJsonItem rather than a bare parse: the value is whatever is in the browser's storage,
  // and a throw here would take down the restore this runs from, leaving the widget with no
  // thread at all over a view preference
  return normalizeCollapsed(getJsonItem<unknown>(LS_COLLAPSE_KEY));
}

/**
 * returns the ids of the collapsed comments on the current page
 */
export const getCollapsedComments = (): Comment['id'][] => readCollapsed(getFromLocalStorage(), siteId, url);

/**
 * @param siteId site id
 * @param url url of the page with comments
 * @param info ids of the comments collapsed on that page
 */
export const saveCollapsedComments = (siteId: string, url: string, info: Comment['id'][]): void => {
  localStorageSetItem(LS_COLLAPSE_KEY, JSON.stringify(writeCollapsed(getFromLocalStorage(), siteId, url, info)));
};
