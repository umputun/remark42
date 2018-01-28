import 'common/promises';

// TODO: i think we need to use unfetch here instead of heavy axios
import axios from 'axios';

import { baseUrl, apiBase, siteId } from './settings';

const fetcher = {};
const methods = ['get', 'post', 'put', 'patch', 'delete', 'head'];
const basename = `${baseUrl}${apiBase}`;

const { CancelToken } = axios;
let cancelHandler = [];

fetcher.cancel = (mask) => {
  cancelHandler.forEach(req => {
    if (req.url.includes(mask)) {
      req.executor('Operation canceled by the user.');
    }
  });
};

methods.forEach(method => {
  fetcher[method] = (url, body = {}, heads) => new Promise((resolve, reject) => {
    const headers = Object.assign({
      Accept: 'application/json',
      'Content-Type': 'application/json',
    }, heads);

    const parameters = {
      method,
      headers,
    };

    if (Object.keys(body).length) {
      parameters.data = body;
    }

    parameters.url = `${basename}${url}`;
    if (method !== 'post') parameters.url += (parameters.url.includes('?') ? '&' : '?') + `site=${siteId}`;
    parameters.cancelToken = new CancelToken(executor => {
      cancelHandler.push({
        executor,
        url: parameters.url,
      });
    });

    axios(parameters)
      .then(res => resolve(res.data))
      .catch(error => {
        if (!axios.isCancel(error)) {
          reject(error);
        }
      })
      .finally(() => {
        cancelHandler = cancelHandler.filter(req => req.url !== parameters.url);
      });
  });
});

export default fetcher;
