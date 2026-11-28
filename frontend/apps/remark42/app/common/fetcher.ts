import { asApiError, backendErrorCode, errorMessages, RequestError } from 'utils/errorUtils';

import { siteId } from './settings';
import { getCookie, clearAuthCookie } from './cookies';
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

/**
 * Cookie the backend keeps the JWT in. It is HttpOnly, so the widget never reads or writes it; the
 * name is here to clear what script can reach, which is the copy older versions of the widget wrote
 */
export const JWT_COOKIE_NAME = 'JWT';
/** Cookie field with XSRF token */
export const XSRF_COOKIE = 'XSRF-TOKEN';
/**
 * The XSRF cookie of an embedded session: the session the backend hands a frame on another site
 * after an OAuth sign-in in a popup, kept apart from the pair above. Its JWT is HttpOnly, like the
 * ordinary one
 */
export const EMBEDDED_XSRF_COOKIE = '__Host-XSRF-TOKEN-EMBEDDED';
/**
 * Set by the backend with an embedded session and kept after a sign-out. While a frame holds it, or
 * the embedded XSRF cookie, the backend reads the embedded session alone, so the widget reads the
 * embedded XSRF cookie alone as well
 */
export const EMBEDDED_MARKER_COOKIE = '__Host-REMARK42-EMBEDDED';

/** Whether this frame is on its embedded session, see EMBEDDED_MARKER_COOKIE */
export function isEmbeddedSession(): boolean {
  return getCookie(EMBEDDED_MARKER_COOKIE) === '1' || Boolean(getCookie(EMBEDDED_XSRF_COOKIE));
}

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
    // the jwt is kept in memory as well as in the backend's cookie, so a request still carries it
    // where storing cookies is disabled. Only the cookie can be renewed, so the copy in memory steps
    // aside before it expires and the request reaches the backend on the cookie alone. The xsrf
    // cookie is the sign that there is a cookie session to step aside for: without one the copy
    // in memory is all the request has, and it is sent for as long as it lasts
    // an embedded session has its own XSRF cookie, and no token in memory: the backend sends none
    // for it and would not read one
    const embedded = isEmbeddedSession();
    const xsrf = getCookie(embedded ? EMBEDDED_XSRF_COOKIE : XSRF_COOKIE);
    const jwt = embedded || (xsrf && Date.now() >= activeJwtStaleAt) ? undefined : activeJwtToken;
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

        // nothing is written to a cookie here: the cookies that carry the session across a reload
        // are the backend's own, set with this same response, partitioned as well for a frame
        // that asked. The copy kept here only serves a browser that stores no cookie
        const lifetime = jwtLifetimeMs(jwtPayload(activeJwtToken));
        activeJwtStaleAt = lifetime === null ? Infinity : sentAt + lifetime - JWT_HEADER_MARGIN_MS;
      }

      // only 401 says the session itself is gone. A 403 comes with a valid session that may not do
      // this, such as one belonging to another site served from the same remark42 host, and
      // clearing on it signs the reader out of every site sharing these cookies
      if (res.status === 401) {
        dropSession();
      }

      if (res.status >= 400) {
        throw await responseError(res);
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

/**
 * What a failed request rejects with.
 *
 * The body is read first, since the backend explains most failures with a code of its own, and a
 * code the catalogue has a message for wins over the HTTP status: a blocked user, a read-only
 * thread and a refused token all arrive as 403, and only the code tells them apart. Without one,
 * a status the catalogue knows picks the message, and any other status rejects with the parsed
 * body, which is how the auth endpoints, which send no code, get their own text shown.
 */
async function responseError(res: Response): Promise<unknown> {
  let body: unknown;
  let isJSON = false;

  try {
    body = JSON.parse(await res.text());
    isJSON = true;
  } catch (e) {
    // not JSON, or the body could not be read: the status is all there is to go on
  }

  const code = backendErrorCode(body);
  const apiError = asApiError(body);

  if (code !== undefined) {
    return new RequestError(errorMessages[code].defaultMessage as string, code, res.status, apiError);
  }

  const descriptor = errorMessages[res.status];

  if (descriptor) {
    return new RequestError(descriptor.defaultMessage as string, res.status, res.status, apiError);
  }

  if (!isJSON) {
    return new RequestError(errorMessages[500].defaultMessage as string, 0, res.status);
  }

  return body;
}

/**
 * Forgets the session this document holds: the in-memory JWT and the auth cookies it can reach.
 * @param bothForms Clear the cookies in the form this context does not address as well, see
 * `clearAuthCookie`. For a sign-out, which has to hold whatever the backend answered
 */
export function dropSession(bothForms = false): void {
  forgetToken();
  clearSessionCookies(bothForms);
}

/**
 * Forgets the JWT kept in memory, and nothing else. For the moment a frame moves to its embedded
 * session, when the token it holds is the other session's
 */
export function forgetToken(): void {
  activeJwtToken = undefined;
  activeJwtStaleAt = Infinity;
}

/**
 * Clears the auth cookies of the session this frame is on that script can reach. On an embedded
 * session that is its XSRF cookie only: the ordinary pair belongs to remark42's own site, which the
 * frame may share it with, and the marker stays, so the frame stays on its own session
 */
export function clearSessionCookies(bothForms = false): void {
  if (isEmbeddedSession()) {
    clearAuthCookie(EMBEDDED_XSRF_COOKIE, bothForms);
    return;
  }
  clearAuthCookie(JWT_COOKIE_NAME, bothForms);
  clearAuthCookie(XSRF_COOKIE, bothForms);
}

/**
 * Asks the API who is signed in, and leaves the session alone whatever the answer: no token from
 * memory is sent, and a refusal clears nothing. For a sign-in that is still settling, where a 401
 * that comes back late would clear the XSRF cookie the sign-in has just set, and a token in memory
 * may be another session's. The status says how it went; the user comes with a 2xx
 */
export async function probeSignedInUser<T>(): Promise<{ status: number; user?: T }> {
  const embedded = isEmbeddedSession();
  const xsrf = getCookie(embedded ? EMBEDDED_XSRF_COOKIE : XSRF_COOKIE);
  const headers = {
    ...frameHeaders(
      IS_THIRD_PARTY,
      window.location.protocol === 'https:',
      sameOrigin(BASE_URL, window.location.origin)
    ),
    ...authHeaders(undefined, xsrf),
  };
  const res = await fetch(requestURL(`${BASE_URL}${API_BASE}`, '/user', {}, siteId), { headers });
  if (!res.ok) {
    return { status: res.status };
  }
  return { status: res.status, user: (await res.json()) as T };
}

export const apiFetcher = createFetcher(`${BASE_URL}${API_BASE}`);
export const authFetcher = createFetcher(`${BASE_URL}/auth`);
export const adminFetcher = createFetcher(`${BASE_URL}${API_BASE}/admin`);
