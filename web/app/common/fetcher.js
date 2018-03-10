import 'common/promises';

// TODO: i think we need to use unfetch here instead of heavy axios
import fetch from 'unfetch';

import { BASE_URL, API_BASE } from './constants';
import { siteId } from './settings';

const fetcher = {};
const methods = ['get', 'post', 'put', 'patch', 'delete', 'head'];

methods.forEach(method => {
  fetcher[method] = data => {
    const {
      url,
      body = {},
      withCredentials = false,
      overriddenApiBase = API_BASE,
    } = (typeof data === 'string' ? { url: data } : data);
    const basename = `${BASE_URL}${overriddenApiBase}`;

    // TODO: try to rewrite without promises
    return new Promise((resolve, reject) => {
      const parameters = {
        method: method.toUpperCase(),
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
        credentials: withCredentials && 'include',
      };

      if (Object.keys(body).length) {
        parameters.body = JSON.stringify(body);
      }

      let requestUrl = `${basename}${url}`;
      if (method !== 'post') requestUrl += (requestUrl.includes('?') ? '&' : '?') + `site=${siteId}`; // TODO: rewrite it

      fetch(requestUrl, parameters)
        .then(res => res.json())
        .then(data => resolve(data))
        .catch(error => reject(error));
    });
  }
});

export default fetcher;
