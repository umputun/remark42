import { resolveBaseUrl } from './base-url';

export const NODE_ID = 'remark42';
export const API_BASE = '/api/v1';
export const COMMENT_NODE_CLASSNAME_PREFIX = 'remark42__comment-';

// the address the bundle falls back to when a page sets no remark_config.host. The server fills
// the placeholder in when it serves the bundle; the dev server fills nothing in, and there the
// bundle and the instance share an origin. Declared above BASE_URL, which reads it on import
const INSTANCE_URL = process.env.NODE_ENV === 'development' ? window.location.origin : '{% REMARK_URL %}';

export const BASE_URL = getBaseUrl();

export function getBaseUrl() {
  return resolveBaseUrl(window.remark_config.host ?? INSTANCE_URL, window.location.protocol);
}
