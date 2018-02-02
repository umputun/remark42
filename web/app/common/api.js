import { siteId, url } from './settings';

import fetcher from './fetcher'

export const find = ({ url }) => fetcher.get(`/find?url=${url}&sort=-score&format=tree`);

export const getComment = ({ id }) => fetcher.get(`/id/${id}?url=${url}`);

export const getUser = () => fetcher.get('/user');

export const send = ({ text, pid }) => fetcher.post('/comment', {
  text,
  locator: {
    site: siteId,
    url
  },
  ...(pid ? { pid } : {}),
});

export const vote = ({ id, url, value }) => fetcher.put(`/vote/${id}?url=${url}&vote=${value}`);

export default {
  find,
  getComment,
  send,
  vote,
  getUser,
};
