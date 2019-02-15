import { BASE_URL, API_BASE } from './constants';
import { siteId } from './settings';
import { StaticStore } from './static_store';
import { getCookie } from './cookies';

export type FetcherMethod = 'get' | 'post' | 'put' | 'patch' | 'delete' | 'head';
const methods: FetcherMethod[] = ['get', 'post', 'put', 'patch', 'delete', 'head'];

type FetcherInit =
  | string
  | {
      url: string;
      body?: string | object | Blob | ArrayBuffer;
      overriddenApiBase?: string;
      withCredentials?: boolean;
    };

const fetcher = methods.reduce(
  (acc: Partial<{ [K in FetcherMethod]: (data: FetcherInit) => Promise<unknown> }>, method: FetcherMethod) => {
    acc[method] = (data: FetcherInit): Promise<unknown> => {
      const { url, body = undefined, withCredentials = false, overriddenApiBase = API_BASE } =
        typeof data === 'string' ? { url: data } : data;
      const basename = `${BASE_URL}${overriddenApiBase}`;

      {
        const headers = new Headers({
          Accept: 'application/json',
          'Content-Type': 'application/json',
          'X-XSRF-TOKEN': getCookie('XSRF-TOKEN') || '',
        });

        let rurl = `${basename}${url}`;

        const parameters: RequestInit = {
          method,
          headers,
          mode: 'cors',
          credentials: withCredentials ? 'include' : 'omit',
        };

        if (body) {
          if (typeof body === 'object') {
            parameters.body = JSON.stringify(body);
          } else {
            parameters.body = body;
          }
        }

        if (siteId && method !== 'post' && !rurl.includes('?site=') && !rurl.includes('&site=')) {
          rurl += (rurl.includes('?') ? '&' : '?') + `site=${siteId}`;
        }

        return fetch(rurl, parameters).then(res => {
          const date = (res.headers.has('date') && res.headers.get('date')) || '';
          const timestamp = isNaN(Date.parse(date)) ? 0 : Date.parse(date);
          const timeDiff = (new Date().getTime() - timestamp) / 1000;
          StaticStore.serverClientTimeDiff = timeDiff;

          return res.json();
        });
      }
    };
    return acc;
  },
  {}
) as { [K in FetcherMethod]: (data: FetcherInit) => Promise<unknown> };

export default fetcher;
