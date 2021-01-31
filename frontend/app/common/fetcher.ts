import { httpErrorMap, httpMessages, RequestError } from 'utils/errorUtils';

import { BASE_URL, API_BASE } from './constants';
import { StaticStore } from './static-store';
import { getCookie } from './cookies';
import { siteId } from 'common/settings';

/** List of fetcher’s supported methods */
const METHODS = ['get', 'post', 'put', 'delete'] as const;
/** Header name for JWT token */
export const JWT_HEADER = 'X-JWT';
/** Header name for XSRF token */
export const XSRF_HEADER = 'X-XSRF-TOKEN';
/** Cookie field with XSRF token */
export const XSRF_COOKIE = 'XSRF-TOKEN';

type QueryParams = Record<string, string | number | undefined>;
type Payload = BodyInit | Record<string, unknown> | null;
type FetcherMethods = Record<'get' | 'delete', <T>(url: string, query?: QueryParams) => Promise<T>> &
  Record<'put' | 'post', <T>(url: string, query?: QueryParams, body?: Payload) => Promise<T>>;

/** JWT token received from server and will be send by each request, if it present */
let activeJwtToken: string | undefined;

const createFetcher = (baseUrl: string = '') =>
  METHODS.reduce<FetcherMethods>((acc, method) => {
    /**
     * Fetcher is abstraction on top of fetch
     *
     * @uri – uri to API endpoint
     * @query - collection of query params. They will be concatenated to URL. `siteId` will be added automatically.
     * @body - data for sending to the server. If you pass object it will be stringified. If you pass form data it will be sent as is. Content type headers will be added automatically.
     */
    acc[method] = async (uri: string, query: QueryParams = {}, body?: Payload) => {
      const queryString = new URLSearchParams({ ...query, site: siteId });
      const url = `${baseUrl}${uri}?${queryString}`;
      const headers: Record<string, string> = {};
      const params: RequestInit = { method };

      // Save token in memory and pass it into headers in case if storing cookies is disabled
      if (activeJwtToken) {
        headers[JWT_HEADER] = activeJwtToken;
      }
      headers[XSRF_HEADER] = getCookie(XSRF_COOKIE) || '';

      if (body instanceof FormData) {
        headers['Content-Type'] = 'multipart/form-data';
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
    return acc;
  }, {} as FetcherMethods);

export const apiFetcher = createFetcher(`${BASE_URL}${API_BASE}`);
export default createFetcher;
