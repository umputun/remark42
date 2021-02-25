import { Sorting, Theme } from './types';

export { BASE_URL, API_BASE, NODE_ID, COMMENT_NODE_CLASSNAME_PREFIX } from './constants.config';
export const LAST_COMMENTS_NODE_CLASSNAME = 'remark42__last-comments';
export const MAX_SHOWN_ROOT_COMMENTS = 10;

export const DEFAULT_SORT: Sorting = '-active';

/** locastorage key for collapsed comments */
export const LS_COLLAPSE_KEY = '__remarkCollapsed';

/** locastorage key for comment form value */
export const LS_SAVED_COMMENT_VALUE = '__remark_comment_value';

/** locastorage key for hidden users */
export const LS_HIDDEN_USERS_KEY = '__remarkHiddenUsers';

/** localstorage key under which sort preference resides */
export const LS_SORT_KEY = '__remarkSort';

/** localstorage key for email of logged in user */
export const LS_EMAIL_KEY = '__remarkEmail';

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
