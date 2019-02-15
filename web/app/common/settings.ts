import { Theme } from './types';
import { THEMES, MAX_SHOWN_ROOT_COMMENTS } from './constants';

export interface QuerySettingsType {
  site_id?: string;
  page_title?: string;
  url?: string;
  max_shown_comments?: number;
  theme: Theme;
  /* used in delete users data page */
  token?: string;
}

export const querySettings: Partial<QuerySettingsType> =
  window.location.search
    .substr(1)
    .split('&')
    .reduce((acc, param) => {
      const pair = param.split('=');
      (acc as any)[pair[0]] = decodeURIComponent(pair[1]);
      return acc;
    }, {}) || {};

if (querySettings.max_shown_comments) {
  querySettings.max_shown_comments = parseInt((querySettings.max_shown_comments as any) as string, 10);
} else {
  querySettings.max_shown_comments = MAX_SHOWN_ROOT_COMMENTS;
}

if (!querySettings.theme || THEMES.indexOf(querySettings.theme) == -1) {
  querySettings.theme = THEMES[0];
}

export const siteId = querySettings.site_id;
export const pageTitle = querySettings.page_title;
export const url = querySettings.url;
export const maxShownComments = querySettings.max_shown_comments;
export const token = querySettings.token;
export const theme = querySettings.theme;
