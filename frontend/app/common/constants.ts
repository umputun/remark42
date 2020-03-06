import { Sorting, AuthProvider, Theme } from './types';
import * as configConstant from './constants.config';

export const BASE_URL = configConstant.BASE_URL;
export const API_BASE = configConstant.API_BASE;
export const NODE_ID = configConstant.NODE_ID;
export const COMMENT_NODE_CLASSNAME_PREFIX = configConstant.COMMENT_NODE_CLASSNAME_PREFIX;
export const LAST_COMMENTS_NODE_CLASSNAME = 'remark42__last-comments';
export const MAX_SHOWN_ROOT_COMMENTS = 10;

export const DEFAULT_SORT: Sorting = '-active';

/* matches auth providers to UI label */
export const PROVIDER_NAMES: { [P in AuthProvider['name']]: string } = {
  google: 'Google',
  twitter: 'Twitter',
  facebook: 'Facebook',
  github: 'GitHub',
  yandex: 'Yandex',
  dev: 'Dev',
  anonymous: 'Anonymous',
  email: 'Email',
};

/** locastorage key for collapsed comments */
export const LS_COLLAPSE_KEY = '__remarkCollapsed';

/** locastorage key for hidden users */
export const LS_HIDDEN_USERS_KEY = '__remarkHiddenUsers';

/** localstorage key under which sort preference resides */
export const LS_SORT_KEY = '__remarkSort';

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
