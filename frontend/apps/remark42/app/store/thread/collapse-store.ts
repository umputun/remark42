import type { Comment } from 'common/types';

/**
 * The shape of the collapsed-threads record, and the reading and writing over it.
 *
 * Kept apart from utils.ts so it can be tested without a browser: that module binds localStorage at
 * import time through common/local-storage, which is enough to make the file unimportable outside
 * one however pure the arithmetic here is. The cases that matter are the ones a browser test
 * cannot reach on purpose, since they are about what anything else with access to the key might
 * have left in storage.
 *
 * Collapsed comment ids are keyed by site and then by page url, nested instead of joined into one
 * string: any separator can also occur inside a site id or a url, and then "site_url_id" cannot be
 * taken apart again. Joined, a page whose url contains the separator loses its collapsed threads
 * on reload, and one page's entries can be read or deleted as another's.
 */
export type CollapsedComments = Record<string, Record<string, Comment['id'][]>>;

/**
 * Coerces whatever storage held into the nested shape.
 *
 * Anything else, a flat list included, reads as empty: collapsed threads are a view preference, so
 * re-expanding them once costs the reader nothing worth a migration.
 */
export function normalizeCollapsed(stored: unknown): CollapsedComments {
  if (typeof stored !== 'object' || stored === null || Array.isArray(stored)) {
    return {};
  }

  return stored as CollapsedComments;
}

/** The ids collapsed on one page. */
export function readCollapsed(stored: CollapsedComments, siteId: string, url: string): Comment['id'][] {
  const ids = stored[siteId]?.[url];

  // normalizeCollapsed checks the top level only, and a leaf of another type would otherwise
  // reach the reducer, which reduces over it
  return Array.isArray(ids) ? ids : [];
}

/**
 * The record to store after collapsing `info` on one page.
 *
 * An empty list is dropped instead of stored: every page a reader collapses and expands again
 * would otherwise keep an entry of its own for good, and so would every site.
 */
export function writeCollapsed(
  stored: CollapsedComments,
  siteId: string,
  url: string,
  info: Comment['id'][]
): CollapsedComments {
  const site = { ...stored[siteId], [url]: info };
  const next = { ...stored, [siteId]: site };

  if (info.length === 0) {
    delete site[url];
  }
  if (Object.keys(site).length === 0) {
    delete next[siteId];
  }

  return next;
}
