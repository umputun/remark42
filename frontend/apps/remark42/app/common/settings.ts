import { parseQuery } from 'utils/parse-query';
import { THEME_SETTINGS, MAX_SHOWN_ROOT_COMMENTS, DEFAULT_SORT } from './constants';
import { parseSorting } from './sorting';
import type { ThemeSetting } from './types';

function parseNumber(value: unknown) {
  if (typeof value !== 'string') {
    return undefined;
  }

  const parsed = +value;

  return isNaN(parsed) ? undefined : parsed;
}

function includes<T extends U, U>(coll: ReadonlyArray<T>, el: U): el is T {
  return coll.includes(el as T);
}

export const rawParams = parseQuery();
export const defaultSort = parseSorting(rawParams.default_sort) ?? DEFAULT_SORT;
export const maxShownComments = parseNumber(rawParams.max_shown_comments) ?? MAX_SHOWN_ROOT_COMMENTS;
export const isEmailSubscription = rawParams.show_email_subscription !== 'false';
export const isTelegramSubscription = rawParams.show_telegram_subscription !== 'false';
export const isRssSubscription =
  rawParams.show_rss_subscription === undefined || rawParams.show_rss_subscription !== 'false';
export const theme = (rawParams.theme = includes(THEME_SETTINGS, rawParams.theme) ? rawParams.theme : THEME_SETTINGS[0]);
export const siteId = rawParams.site_id || 'remark';
export const pageTitle = rawParams.page_title;
export const url = rawParams.url;
export const token = rawParams.token;
export const locale = rawParams.locale || 'en';
export const noFooter = rawParams.no_footer === 'true';
