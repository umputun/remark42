import { Sorting, AuthProvider, BlockingDuration, Theme } from './types';

export const BASE_URL: string = process.env.REMARK_URL!;
export const API_BASE = '/api/v1';
export const NODE_ID: string = process.env.REMARK_NODE!;
export const COUNTER_NODE_CLASSNAME = 'remark42__counter';
export const COMMENT_NODE_CLASSNAME_PREFIX = 'remark42__comment-';
export const LAST_COMMENTS_NODE_CLASSNAME = 'remark42__last-comments';
export const DEFAULT_LAST_COMMENTS_MAX = 15;
export const DEFAULT_MAX_COMMENT_SIZE = 1000;
export const MAX_SHOWN_ROOT_COMMENTS = 10;

export const DEFAULT_SORT: Sorting = '-active';

/* matches auth providers to UI label */
export const PROVIDER_NAMES: { [P in AuthProvider['name']]: string } = {
  google: 'Google',
  facebook: 'Facebook',
  github: 'GitHub',
  yandex: 'Yandex',
  dev: 'Dev',
  anonymous: 'Anonymous',
};

/** locastorage key for collapsed comments */
export const LS_COLLAPSE_KEY = '__remarkCollapsed';

/** cookie key under which sort preference resides */
export const COOKIE_SORT_KEY = 'remarkSort';

export const BLOCKING_DURATIONS: BlockingDuration[] = [
  {
    label: 'Permanently',
    value: 'permanently',
  },
  {
    label: 'For a month',
    value: '43200m',
  },
  {
    label: 'For a week',
    value: '10080m',
  },
  {
    label: 'For a day',
    value: '1440m',
  },
];

export const THEMES: Theme[] = ['light', 'dark'];

export const IS_MOBILE = /Android|webOS|iPhone|iPad|iPod|Opera Mini|Windows Phone/i.test(navigator.userAgent);

/**
 * Defines if browser storage features (cookies, localsrotage)
 * are available or blocked via browser preferences
 */
export const IS_STORAGE_AVAILABLE: boolean = (() => {
  try {
    localStorage.setItem('localstorage_availability_test', '');
    localStorage.removeItem('localstorage_availability_test');
  } catch (e) {
    return false;
  }
  return true;
})();

/**
 * Defines whether iframe loaded in cross origin environment
 * Usefull for checking if some privacy restriction may be applied
 */
export const IS_THIRD_PARTY: boolean = (() => {
  try {
    return window.parent.location.host !== window.location.host;
  } catch (e) {
    return true;
  }
})();
