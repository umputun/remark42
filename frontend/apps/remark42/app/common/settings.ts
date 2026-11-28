import { parseQuery } from 'utils/parse-query';
import { THEMES, DEFAULT_SORT } from './constants';
import { parseMaxShownComments } from './max-shown-comments';
import { parseSorting } from './sorting';
import { parseCommentsPageSize } from './comments-page-size';

function includes<T extends U, U>(coll: ReadonlyArray<T>, el: U): el is T {
  return coll.includes(el as T);
}

export const rawParams = parseQuery();
export const defaultSort = parseSorting(rawParams.default_sort) ?? DEFAULT_SORT;
/**
 * How many top-level comments the site set to show before the Show more button, on every device;
 * undefined when it set none, and the root then picks a default for the device
 */
export const maxShownComments = parseMaxShownComments(rawParams.max_shown_comments);
/** comments loaded at a time, whole top-level threads only; 0 loads the thread in one request */
export const commentsPageSize = parseCommentsPageSize(rawParams.comments_page_size);
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
