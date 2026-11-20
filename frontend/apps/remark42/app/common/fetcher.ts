import { errorMessages, RequestError } from 'utils/errorUtils';

import { siteId } from './settings';
import { getCookie, setAuthCookie, clearAuthCookie } from './cookies';
import { StaticStore } from './static-store';
import { BASE_URL, API_BASE, IS_THIRD_PARTY, MAX_CLOCK_SKEW_MS } from './constants';
import {
  authHeaders,
  clockSkewMs,
  JWT_HEADER,
  XSRF_HEADER,
  frameHeaders,
  jwtLifetimeMs,
  jwtPayload,
  sameOrigin,
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

/**
 * How long before its expiry the JWT in memory stops being sent beside a cookie session, in
 * milliseconds. The backend refuses an expired token sent in a header without looking at the
 * cookie, so stopping late ends the session and stopping early only hands the request to the cookie.
 */
export const JWT_HEADER_MARGIN_MS = 10 * 1000;

/** JWT token received from server and will be sent by each request, if it is present */
let activeJwtToken: string | undefined;
/** Client time after which `activeJwtToken` is left out of a request that has a cookie to go on */
let activeJwtStaleAt = Infinity;

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
    // storing cookies is disabled. Only the cookie can be renewed, so the copy in memory steps
    // aside before it expires and the request reaches the backend on the cookie alone. The xsrf
    // cookie is the sign that there is a cookie session to step aside for: without one the copy
    // in memory is all the request has, and it is sent for as long as it lasts
    const xsrf = getCookie(XSRF_COOKIE);
    const jwt = xsrf && Date.now() >= activeJwtStaleAt ? undefined : activeJwtToken;
    const headers = {
      ...frameHeaders(
        IS_THIRD_PARTY,
        window.location.protocol === 'https:',
        sameOrigin(BASE_URL, window.location.origin)
      ),
      ...authHeaders(jwt, xsrf),
      ...sending.headers,
    };
    const params: RequestInit = { method, body: sending.body };
    // a token this request brings back was issued after this moment, so it cannot expire sooner
    // than its lifetime from here, however long the response takes to arrive
    const sentAt = Date.now();

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
          const lifetime = jwtLifetimeMs(payload);
          activeJwtStaleAt = lifetime === null ? Infinity : sentAt + lifetime - JWT_HEADER_MARGIN_MS;
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

      // only 401 says the session itself is gone. A 403 comes with a valid session that may not do
      // this, such as one belonging to another site served from the same remark42 host, and
      // clearing on it signs the reader out of every site sharing these cookies
      if (res.status === 401) {
        dropSession();
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

/** Forgets the session this document holds: the in-memory JWT and the auth cookies it can reach */
export function dropSession(): void {
  activeJwtToken = undefined;
  activeJwtStaleAt = Infinity;
  clearAuthCookie(JWT_COOKIE_NAME);
  clearAuthCookie(XSRF_COOKIE);
}

export const apiFetcher = createFetcher(`${BASE_URL}${API_BASE}`);
export const authFetcher = createFetcher(`${BASE_URL}/auth`);
export const adminFetcher = createFetcher(`${BASE_URL}${API_BASE}/admin`);
