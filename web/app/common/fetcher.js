import axios from 'axios';

import { BASE_URL, API_BASE } from './constants';
import { siteId } from './settings';
import store from './store';

const fetcher = {};
const methods = ['get', 'post', 'put', 'patch', 'delete', 'head'];

methods.forEach(method => {
  fetcher[method] = data => {
    const { url, body = {}, withCredentials = false, overriddenApiBase = API_BASE } =
      typeof data === 'string' ? { url: data } : data;
    const basename = `${BASE_URL}${overriddenApiBase}`;

    return new Promise((resolve, reject) => {
      const headers = {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      };

      const parameters = {
        method,
        headers,
        withCredentials,
      };

      if (Object.keys(body).length) {
        parameters.data = body;
      }

      parameters.url = `${basename}${url}`;

      if (siteId && method !== 'post' && !parameters.url.includes('?site=') && !parameters.url.includes('&site=')) {
        parameters.url += (parameters.url.includes('?') ? '&' : '?') + `site=${siteId}`;
      }

      axios(parameters)
        .then(res => {
          const date = ('date' in res.headers && res.headers.date) || '';
          const timestamp = isNaN(Date.parse(date)) ? 0 : Date.parse(date);
          const timeDiff = (new Date() - timestamp) / 1000;
          store.set('serverClientTimeDiff', timeDiff);

          resolve(res.data);
        })
        .catch(error => reject(error));
    });
  };
});

export default fetcher;
