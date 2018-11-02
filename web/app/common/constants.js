export const BASE_URL = process.env.REMARK_URL;
export const API_BASE = '/api/v1';
export const NODE_ID = process.env.REMARK_NODE;
export const COUNTER_NODE_CLASSNAME = 'remark42__counter';
export const COMMENT_NODE_CLASSNAME_PREFIX = 'remark42__comment-';
export const LAST_COMMENTS_NODE_CLASSNAME = 'remark42__last-comments';
export const DEFAULT_LAST_COMMENTS_MAX = 15;
export const DEFAULT_MAX_COMMENT_SIZE = 1000;
export const MAX_SHOWN_ROOT_COMMENTS = 10;
export const DEFAULT_SORT = '-active';
export const PROVIDER_NAMES = {
  google: 'Google',
  facebook: 'Facebook',
  github: 'GitHub',
  yandex: 'Yandex',
  dev: 'Dev',
};
export const LS_COLLAPSE_KEY = '__remarkCollapsed';
export const LS_SORT_KEY = '__remarkSort';

export const BLOCKING_DURATIONS = [
  {
    label: 'Permanently',
    value: 'permanently',
  },
  {
    label: 'For a month',
    value: `${30 * 60 * 24}m`,
  },
  {
    label: 'For a week',
    value: `${7 * 60 * 24}m`,
  },
  {
    label: 'For a day',
    value: `${60 * 24}m`,
  },
];
