import { siteId, url } from './settings';

import fetcher from './fetcher'

export const find = ({ url }) => fetcher.get(`/find?url=${url}&sort=time&format=tree`);

export const send = ({ text }) => fetcher.post('/comment', { text, locator: { site: siteId, url } });

export const vote = ({ id, url, value }) => fetcher.put(`/vote/${id}?url=${url}&vote=${value}`);

export default {
  find,
  send,
  vote,
};
