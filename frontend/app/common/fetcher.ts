import { httpErrorMap, httpMessages, RequestError } from 'utils/errorUtils';

import { BASE_URL, API_BASE } from './constants';
import { StaticStore } from './static-store';
import { getCookie } from './cookies';

/** List of fetcher’s supported methods */
const METHODS = ['get', 'post', 'put', 'delete'] as const;
/** Header name for JWT token */
export const JWT_HEADER = 'X-JWT';
/** Header name for XSRF token */
export const XSRF_HEADER = 'X-XSRF-TOKEN';
/** Cookie field with XSRF token */
export const XSRF_COOKIE = 'XSRF-TOKEN';

type Methods = typeof METHODS;
type FetchInit = Omit<RequestInit, 'headers'> & {
  headers?: Record<string, string>;
  json?: unknown;
  query?: Record<string, string | number | undefined>;
};
type FetcherObject = Record<Methods[number], <T>(url: string, params?: FetchInit) => Promise<T>>;

/** JWT token received from server and will be send by each request, if it present */
let activeJwtToken: string | undefined;

const fetcher = METHODS.reduce((acc, method) => {
  acc[method] = async (uri: string, params: FetchInit = {}) => {
    const { headers = {}, json, ...fetchParams } = params;
    // add api base if it's not auth request
    // we use `indexOf` instead of `startsWidth` because we don't want to have another one polyfill for no reason
    const baseUrl = uri.indexOf('/auth') === 0 ? BASE_URL : `${BASE_URL}${API_BASE}`;
    const url = `${baseUrl}${uri}`;

    // Save token in memory and pass it into headers in case if storing cookies is disabled
    if (activeJwtToken) {
      headers[JWT_HEADER] = activeJwtToken;
    }
    headers[XSRF_HEADER] = getCookie(XSRF_COOKIE) || '';

    if (json) {
      headers['Content-Type'] = 'application/json';
      fetchParams.body = JSON.stringify(json);
    }

    try {
      const res = await fetch(url, { ...fetchParams, method, headers });
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
}, {} as FetcherObject);

export default fetcher;
