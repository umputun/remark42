import 'common/promises';

// TODO: i think we need to use unfetch here instead of heavy axios
// import axios from 'axios';
import fetch from 'unfetch';

import { BASE_URL, API_BASE } from './constants';
import { siteId } from './settings';

const fetcher = {};
const methods = ['get', 'post', 'put', 'patch', 'delete', 'head'];

// const { CancelToken } = axios;
// let cancelHandler = [];

// fetcher.cancel = (mask) => {
//   cancelHandler.forEach(req => {
//     if (req.url.includes(mask)) {
//       req.executor('Operation canceled by the user.');
//     }
//   });
// };

methods.forEach(method => {
  fetcher[method] = data => {
    const {
      url,
      body = {},
      withCredentials = false,
      overriddenApiBase = API_BASE,
    } = (typeof data === 'string' ? { url: data } : data);
    const basename = `${BASE_URL}${overriddenApiBase}`;

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
        parameters.data = body;
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
