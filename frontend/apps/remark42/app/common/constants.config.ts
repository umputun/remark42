export const NODE_ID = process.env.REMARK_NODE!;
export const API_BASE = '/api/v1';
export const COMMENT_NODE_CLASSNAME_PREFIX = 'remark42__comment-';
export const BASE_URL = getBaseUrl();

export function getBaseUrl() {
  const baseUrl = window.remark_config.host ?? process.env.REMARK_URL;

  if (!baseUrl) {
    throw new Error(`Remark42: remark_config.host wasn't configured.`);
  }

  // Validate host
  try {
    const { protocol } = new URL(baseUrl);
    // Show error if protocol of iframe doesn't match protocol of current page
    if (protocol !== window.location.protocol) {
      console.error('Remark42: Protocol mismatch.');
    }
    // Check if host has valid protocol and prevent XSS vurnuality
    if (!protocol.startsWith('http')) {
      console.error('Remark42: Wrong protocol in host URL.');
      throw new Error();
    }
  } catch (e) {
    throw new Error('Remark42: Invalid host URL.');
  }

  return baseUrl;
}
