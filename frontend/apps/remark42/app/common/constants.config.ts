import { resolveBaseUrl } from './base-url';

export const NODE_ID = process.env.REMARK_NODE!;
export const API_BASE = '/api/v1';
export const COMMENT_NODE_CLASSNAME_PREFIX = 'remark42__comment-';
export const BASE_URL = getBaseUrl();

export function getBaseUrl() {
  return resolveBaseUrl(window.remark_config.host ?? process.env.REMARK_URL, window.location.protocol);
}
