import { siteId, url } from './settings';

import fetcher from './fetcher'

/* common */

export const find = ({ url }) => fetcher.get(`/find?url=${url}&sort=-score&format=tree`);

export const getComment = ({ id }) => fetcher.get(`/id/${id}?url=${url}`);

export const vote = ({ id, url, value }) => fetcher.put(`/vote/${id}?url=${url}&vote=${value}`);

export const send = ({ text, pid }) => fetcher.post('/comment', {
  text,
  locator: {
    site: siteId,
    url
  },
  ...(pid ? { pid } : {}),
});

export const getUser = () => fetcher.get('/user');

/* admin */
export const pin = ({ id, url }) => fetcher.put(`/admin/pin/${id}?url=${url}&pin=1`);

export const unpin = ({ id, url }) => fetcher.get(`/admin/pin/${id}?url=${url}&pin=0`);

export default {
  find,
  getComment,
  pin,
  unpin,
  vote,
  send,
  getUser,
};
