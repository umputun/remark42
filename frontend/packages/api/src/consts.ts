/** Base path to API */
export const API_BASE = '/api/v1' as const

/** Header name for JWT token */
export const JWT_HEADER = 'X-JWT' as const

/** Cookie field with XSRF token */
export const XSRF_COOKIE = 'XSRF-TOKEN' as const

/** Header name for XSRF token */
export const XSRF_HEADER = `X-${XSRF_COOKIE}` as const

/** Header name for site identificator */
export const SITE_HEADER = 'X-SITE-ID' as const
