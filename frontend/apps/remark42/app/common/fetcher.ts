import { errorMessages, RequestError } from 'utils/errorUtils';

import { siteId } from './settings';
import { getCookie, setAuthCookie, clearAuthCookie } from './cookies';
import { StaticStore } from './static-store';
import { BASE_URL, API_BASE, MAX_CLOCK_SKEW_MS } from './constants';
import {
  authHeaders,
  clockSkewMs,
  JWT_HEADER,
  XSRF_HEADER,
  jwtPayload,
  requestBody,
  requestURL,
  type Payload,
  type QueryParams,
} from './fetcher-core';

export { JWT_HEADER, XSRF_HEADER };

/** Cookie name for JWT token when using AUTH_SEND_JWT_HEADER */
export const JWT_COOKIE_NAME = 'JWT';
/** Cookie field with XSRF token */
export const XSRF_COOKIE = 'XSRF-TOKEN';
/**
 * Cookie TTL in seconds - matches backend's auth.ttl.cookie default of 200 hours
 * The JWT token itself expires in 5 minutes, but the cookie persists longer
 * to match server-side behavior when not using AUTH_SEND_JWT_HEADER
 */
export const AUTH_COOKIE_TTL_SECONDS = 200 * 60 * 60;

type BodylessMethod = <T>(url: string, query?: QueryParams) => Promise<T>;
type BodyMethod = <T>(url: string, query?: QueryParams, body?: Payload) => Promise<T>;
type Methods = {
  get: BodylessMethod;
  put: BodyMethod;
  post: BodyMethod;
  delete: BodylessMethod;
};

/** JWT token received from server and will be sent by each request, if it is present */
let activeJwtToken: string | undefined;

const createFetcher = (baseUrl: string = ''): Methods => {
  /**
   * Fetcher is abstraction on top of fetch
   *
   * @method - a string to set http method
   * @uri – uri to API endpoint
   * @query - collection of query params. They will be concatenated to URL. `siteId` will be added automatically.
   * @body - data for sending to the server. If you pass object it will be stringified. If you pass form data it will be sent as is. Content type headers will be added automatically.
   */
  const request = async (method: string, uri: string, query: QueryParams = {}, body?: Payload) => {
    const url = requestURL(baseUrl, uri, query, siteId);
    const sending = requestBody(body, siteId);
    // the jwt is kept in memory as well as in a cookie, so a request still carries it where
    // storing cookies is disabled
    const headers = { ...authHeaders(activeJwtToken, getCookie(XSRF_COOKIE)), ...sending.headers };
    const params: RequestInit = { method, body: sending.body };

    try {
      const res = await fetch(url, { ...params, headers });
      const skew = clockSkewMs(res.headers.get('date'), new Date().getTime(), MAX_CLOCK_SKEW_MS);

      if (skew !== null) {
        StaticStore.serverClientTimeDiffMs = skew;
      }

      // backend could update jwt in any time. so, we should handle it
      if (res.headers.has(JWT_HEADER)) {
        activeJwtToken = res.headers.get(JWT_HEADER) as string;

        // Store the JWT token in cookies for persistence across page reloads
        try {
          const payload = jwtPayload(activeJwtToken);
          if (payload && payload.jti) {
            // Set XSRF cookie with the JWT ID using enhanced security
            setAuthCookie(XSRF_COOKIE, payload.jti as string, {
              expires: AUTH_COOKIE_TTL_SECONDS,
            });

            // Store the JWT in cookie for persistence with enhanced security
            setAuthCookie(JWT_COOKIE_NAME, activeJwtToken, {
              expires: AUTH_COOKIE_TTL_SECONDS,
            });
          }
        } catch (e) {
          console.error('Failed to process JWT token', e);
        }
      }

      if ([401, 403].includes(res.status)) {
        activeJwtToken = undefined;
        clearAuthCookie(JWT_COOKIE_NAME);
        clearAuthCookie(XSRF_COOKIE);
      }

      if (res.status >= 400) {
        const descriptor = errorMessages[res.status];

        if (descriptor) {
          throw new RequestError(descriptor.defaultMessage as string, res.status);
        }

        return res.text().then((text) => {
          let err;
          try {
            err = JSON.parse(text);
          } catch (e) {
            throw new RequestError(errorMessages[500].defaultMessage as string, 0);
          }
          throw err;
        });
      }

      if (res.headers.get('Content-Type')?.indexOf('application/json') === 0) {
        return res.json();
      }

      return res.text();
    } catch (e) {
      if (e instanceof Error && e.message === 'Failed to fetch') {
        throw new RequestError(e.message, 'fetch-error');
      }

      throw e;
    }
  };

  return {
    get: (uri: string, query: QueryParams, body: Payload) => request('get', uri, query, body),
    put: (uri: string, query: QueryParams, body: Payload) => request('put', uri, query, body),
    post: (uri: string, query: QueryParams, body: Payload) => request('post', uri, query, body),
    delete: (uri: string, query: QueryParams, body: Payload) => request('delete', uri, query, body),
  } as Methods;
};

export const apiFetcher = createFetcher(`${BASE_URL}${API_BASE}`);
export const authFetcher = createFetcher(`${BASE_URL}/auth`);
export const adminFetcher = createFetcher(`${BASE_URL}${API_BASE}/admin`);
