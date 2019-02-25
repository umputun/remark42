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
  (acc, method) => {
    acc[method] = <T = unknown>(data: FetcherInit): Promise<T> => {
      const { url, body = undefined, withCredentials = false, overriddenApiBase = API_BASE } =
        typeof data === 'string' ? { url: data } : data;
      const basename = `${BASE_URL}${overriddenApiBase}`;

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

        if (res.status >= 400) {
          return res.text().then(text => {
            let err;
            try {
              err = JSON.parse(text);
            } catch (e) {
              throw text;
            }
            throw err;
          });
        }

        if (res.headers.has('Content-Type') && res.headers.get('Content-Type')!.indexOf('application/json') === 0) {
          return res.json();
        }

        return res.text();
      });
    };
    return acc;
  },
  {} as { [K in FetcherMethod]: <T = unknown>(data: FetcherInit) => Promise<T> }
);

export default fetcher;
