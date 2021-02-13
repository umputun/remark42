import { httpErrorMap, httpMessages, RequestError } from 'utils/errorUtils';

import { siteId } from './settings';
import { getCookie } from './cookies';
import { StaticStore } from './static-store';
import { BASE_URL, API_BASE } from './constants';

/** Header name for JWT token */
export const JWT_HEADER = 'X-JWT';
/** Header name for XSRF token */
export const XSRF_HEADER = 'X-XSRF-TOKEN';
/** Cookie field with XSRF token */
export const XSRF_COOKIE = 'XSRF-TOKEN';

type QueryParams = Record<string, string | number | undefined>;
type Payload = BodyInit | Record<string, unknown> | null;
type BodylessMethod = <T>(url: string, query?: QueryParams) => Promise<T>;
type BodyMethod = <T>(url: string, query?: QueryParams, body?: Payload) => Promise<T>;
type Methods = {
  get: BodylessMethod;
  put: BodyMethod;
  post: BodyMethod;
  delete: BodylessMethod;
};

/** JWT token received from server and will be send by each request, if it present */
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
    const queryString = new URLSearchParams({ site: siteId, ...query });
    const url = `${baseUrl}${uri}?${queryString}`;
    const headers: Record<string, string> = {};
    const params: RequestInit = { method };

    // Save token in memory and pass it into headers in case if storing cookies is disabled
    if (activeJwtToken) {
      headers[JWT_HEADER] = activeJwtToken;
    }
    headers[XSRF_HEADER] = getCookie(XSRF_COOKIE) || '';

    if (body instanceof FormData) {
      // Shouldn't add any kind of `Content-Type` if we send `FormData`
      // Now FormData is sent only in case of uploading file
      params.body = body;
    } else if (typeof body === 'object' && body !== null) {
      headers['Content-Type'] = 'application/json';
      params.body = JSON.stringify(body);
    } else {
      params.body = body;
    }

    try {
      const res = await fetch(url, { ...params, headers });
      // TODO: it should be clarified when frontend gets this header and what could be in it to simplify this logic and cover by tests
      const date = (res.headers.has('date') && res.headers.get('date')) || '';
      const timestamp = isNaN(Date.parse(date)) ? 0 : Date.parse(date);
      const timeDiff = (new Date().getTime() - timestamp) / 1000;

      StaticStore.serverClientTimeDiff = timeDiff;

      // backend could update jwt in any time. so, we should handle it
      if (res.headers.has(JWT_HEADER)) {
        activeJwtToken = res.headers.get(JWT_HEADER) as string;
      }

      if ([401, 403].includes(res.status)) {
        activeJwtToken = undefined;
      }

      if (res.status >= 400) {
        if (httpErrorMap.has(res.status)) {
          const descriptor = httpErrorMap.get(res.status) || httpMessages.unexpectedError;

          throw new RequestError(descriptor.defaultMessage, res.status);
        }

        return res.text().then((text) => {
          let err;
          try {
            err = JSON.parse(text);
          } catch (e) {
            throw new RequestError(httpMessages.unexpectedError.defaultMessage, 0);
          }
          throw err;
        });
      }

      if (res.headers.get('Content-Type')?.indexOf('application/json') === 0) {
        return res.json();
      }

      return res.text();
    } catch (e) {
      if (e?.message === 'Failed to fetch') {
        throw new RequestError(e.message, -2);
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
