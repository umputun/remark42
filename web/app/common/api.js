import { siteId, url } from './settings';

import fetcher from './fetcher'

/* common */

export const getConfig = () => fetcher.get(`/config`);

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

export const unpin = ({ id, url }) => fetcher.put(`/admin/pin/${id}?url=${url}&pin=0`);

export const remove = ({ id }) => fetcher.delete(`/admin/comment/${id}?url=${url}`);

export const blockUser = ({ id }) => fetcher.put(`/admin/user/${id}?block=1`);

export const unblockUser = ({ id }) => fetcher.put(`/admin/user/${id}?block=0`);

export default {
  getConfig,
  find,
  getComment,
  vote,
  send,
  getUser,

  pin,
  unpin,
  remove,
  blockUser,
  unblockUser,
};
