const BASE_URL = 'https://remark42.radio-t.com';
const API_BASE = '/api/v1';
const NODE_ID = 'remark42';
const COUNTER_NODE_CLASSNAME = 'remark42__counter';
const COMMENT_NODE_CLASSNAME_PREFIX = 'remark42__comment-';
const LAST_COMMENTS_NODE_CLASSNAME = 'remark42__last-comments';
const DEFAULT_LAST_COMMENTS_MAX = 15;
const DEFAULT_MAX_COMMENT_SIZE = 1000;
const MAX_SHOWN_ROOT_COMMENTS = 10;
const DEFAULT_SORT = '-score';
const PROVIDER_NAMES = {
  google: 'Google',
  facebook: 'Facebook',
  github: 'GitHub',
  yandex: 'Yandex',
  dev: 'Dev',
};
const LS_COLLAPSE_KEY = '__remarkCollapsed';
const LS_SORT_KEY = '__remarkSort';

const BLOCKING_DURATIONS = [
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

module.exports = {
  BASE_URL,
  API_BASE,
  NODE_ID,
  COUNTER_NODE_CLASSNAME,
  LAST_COMMENTS_NODE_CLASSNAME,
  COMMENT_NODE_CLASSNAME_PREFIX,
  DEFAULT_LAST_COMMENTS_MAX,
  DEFAULT_MAX_COMMENT_SIZE,
  MAX_SHOWN_ROOT_COMMENTS,
  PROVIDER_NAMES,
  DEFAULT_SORT,
  LS_COLLAPSE_KEY,
  LS_SORT_KEY,
  BLOCKING_DURATIONS,
};
