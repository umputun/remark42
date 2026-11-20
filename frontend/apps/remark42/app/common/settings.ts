import { parseQuery } from 'utils/parse-query';
import { THEMES, MAX_SHOWN_ROOT_COMMENTS, DEFAULT_SORT, IS_MOBILE } from './constants';
import { parseMaxShownComments } from './max-shown-comments';
import { parseSorting } from './sorting';

function includes<T extends U, U>(coll: ReadonlyArray<T>, el: U): el is T {
  return coll.includes(el as T);
}

export const rawParams = parseQuery();
export const defaultSort = parseSorting(rawParams.default_sort) ?? DEFAULT_SORT;
/**
 * How many top-level comments are shown before the Show more button. A site that sets
 * max_shown_comments gets it on every device; without it, mobile shows the first
 * MAX_SHOWN_ROOT_COMMENTS and desktop shows every comment, as before the setting reached desktop.
 */
export const maxShownComments =
  parseMaxShownComments(rawParams.max_shown_comments) ?? (IS_MOBILE ? MAX_SHOWN_ROOT_COMMENTS : Infinity);
export const isEmailSubscription = rawParams.show_email_subscription !== 'false';
export const isTelegramSubscription = rawParams.show_telegram_subscription !== 'false';
export const isRssSubscription =
  rawParams.show_rss_subscription === undefined || rawParams.show_rss_subscription !== 'false';
export const theme = (rawParams.theme = includes(THEMES, rawParams.theme) ? rawParams.theme : THEMES[0]);
export const siteId = rawParams.site_id || 'remark';
export const pageTitle = rawParams.page_title;
export const url = rawParams.url;
export const token = rawParams.token;
export const locale = rawParams.locale || 'en';
export const noFooter = rawParams.no_footer === 'true';
