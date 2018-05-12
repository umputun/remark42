const BASE_URL = 'https://remark42.radio-t.com';
const API_BASE = '/api/v1';
const NODE_ID = 'remark42';
const COUNTER_NODE_CLASSNAME = 'remark42__counter';
const COMMENT_NODE_CLASSNAME_PREFIX = 'remark42__comment-';
const LAST_COMMENTS_NODE_CLASSNAME = 'remark42__last-comments';
const DEFAULT_LAST_COMMENTS_MAX = 15;
const DEFAULT_MAX_COMMENT_SIZE = 1000;
const DEFAULT_SORT = '-score';
const PROVIDER_NAMES = {
  google: 'Google',
  facebook: 'Facebook',
  github: 'GitHub',
};

module.exports = {
  BASE_URL,
  API_BASE,
  NODE_ID,
  COUNTER_NODE_CLASSNAME,
  LAST_COMMENTS_NODE_CLASSNAME,
  COMMENT_NODE_CLASSNAME_PREFIX,
  DEFAULT_LAST_COMMENTS_MAX,
  DEFAULT_MAX_COMMENT_SIZE,
  PROVIDER_NAMES,
  DEFAULT_SORT,
};
